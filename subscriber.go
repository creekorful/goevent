package goevent

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/rabbitmq/amqp091-go"
)

// Handler represent an event handler
type Handler func(Subscriber, *RawMessage) error

// Subscriber is something that read msg from an event queue
type Subscriber interface {
	Publisher

	// Read RawMessage and deserialize it into proper Event
	Read(msg *RawMessage, event Event) error

	// Subscribe to named exchange with unique consuming guaranty
	Subscribe(exchange, queue, subscriptionId string, handler Handler) error

	// SubscribeDLQ subscribe to named exchange and add DLQ functionality
	SubscribeDLQ(exchange, queue, subscriptionId string, handler Handler) error

	// SubscribeAll subscribe to given exchange but ensure everyone on the exchange receive the messages
	SubscribeAll(exchange, subscriptionId string, handler Handler) error
}

// Subscriber represent a subscriber
type subscriber struct {
	channel            *amqp091.Channel
	subscriptions      map[string]struct{}
	subscriptionsMutex sync.Mutex
}

// NewSubscriber create a new subscriber and connect it to given server
func NewSubscriber(amqpURI string, prefetch int) (Subscriber, error) {
	conn, err := amqp091.Dial(amqpURI)
	if err != nil {
		return nil, err
	}

	c, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := c.Qos(prefetch, 0, false); err != nil {
		return nil, err
	}

	return &subscriber{
		channel:       c,
		subscriptions: map[string]struct{}{},
	}, nil
}

func (s *subscriber) PublishEvent(event Event) error {
	return publishEvent(s.channel, event)
}

func (s *subscriber) PublishRaw(exchange string, msg *RawMessage) error {
	return publishRaw(s.channel, exchange, msg)
}

func (s *subscriber) Close() error {
	for subscription := range s.subscriptions {
		// close the deliveries chan
		_ = s.channel.Cancel(subscription, false)
	}

	return s.channel.Close()
}

func (s *subscriber) Read(msg *RawMessage, event Event) error {
	if err := json.Unmarshal(msg.Body, event); err != nil {
		return err
	}

	return nil
}

func (s *subscriber) Subscribe(exchange, queue, subscriptionId string, handler Handler) error {
	// First declare the exchange
	if err := s.channel.ExchangeDeclare(exchange, amqp091.ExchangeFanout, true, false, false, false, nil); err != nil {
		return err
	}

	// Then declare the queue
	q, err := s.channel.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		return err
	}

	// Bind the queue to the exchange
	if err := s.channel.QueueBind(q.Name, "", exchange, false, nil); err != nil {
		return err
	}

	if err := s.startConsuming(q.Name, subscriptionId, handler, false); err != nil {
		return err
	}

	return nil
}

func (s *subscriber) SubscribeDLQ(exchange, queue, subscriptionId string, handler Handler) error {
	// First declare the exchange
	if err := s.channel.ExchangeDeclare(exchange, amqp091.ExchangeFanout, true, false, false, false, nil); err != nil {
		return err
	}

	dlqName := queue + "DLQ"

	// Declare the DLQ
	_, err := s.channel.QueueDeclare(dlqName, true, false, false, false, nil)
	if err != nil {
		return err
	}

	// Then declare the queue
	q, err := s.channel.QueueDeclare(queue, true, false, false, false, map[string]interface{}{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": dlqName,
	})
	if err != nil {
		return err
	}

	// Bind the queue to the exchange
	if err := s.channel.QueueBind(q.Name, "", exchange, false, nil); err != nil {
		return err
	}

	if err := s.startConsuming(q.Name, subscriptionId, handler, true); err != nil {
		return err
	}

	return nil
}

func (s *subscriber) SubscribeAll(exchange, subscriptionId string, handler Handler) error {
	// First declare the exchange
	if err := s.channel.ExchangeDeclare(exchange, amqp091.ExchangeFanout, true, false, false, false, nil); err != nil {
		return err
	}

	// Then declare the queue
	q, err := s.channel.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		return err
	}

	// Bind the queue to the exchange
	if err := s.channel.QueueBind(q.Name, "", exchange, false, nil); err != nil {
		return err
	}

	if err := s.startConsuming(q.Name, subscriptionId, handler, false); err != nil {
		return err
	}

	return nil
}

func (s *subscriber) startConsuming(queueName, subscriptionId string, handler Handler, dlq bool) error {
	s.subscriptionsMutex.Lock()
	defer s.subscriptionsMutex.Unlock()

	if _, exists := s.subscriptions[subscriptionId]; exists {
		return fmt.Errorf("a subscription already exists with id %s", subscriptionId)
	}

	// Start consuming asynchronously
	deliveries, err := s.channel.Consume(queueName, subscriptionId, false, false, false, false, nil)
	if err != nil {
		return err
	}

	s.subscriptions[subscriptionId] = struct{}{}

	go func() {
		for delivery := range deliveries {
			msg := &RawMessage{
				Body:    delivery.Body,
				Headers: delivery.Headers,
			}

			// If an error happens and DLQ is configured reject the message
			if err := handler(s, msg); err != nil && dlq {
				_ = delivery.Reject(false)
			} else {
				_ = delivery.Ack(false)
			}
		}
	}()

	return nil
}
