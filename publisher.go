package goevent

import (
	"encoding/json"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
)

// RawMessage is a raw message as viewed by the messaging system
type RawMessage struct {
	Body    []byte
	Headers map[string]interface{}
}

// Publisher is something that push an event
type Publisher interface {
	// PublishEvent publish the given event
	PublishEvent(event Event) error

	// PublishRaw publish given RawMessage into given exchange
	PublishRaw(exchange string, msg *RawMessage) error

	// Close the underlying connection gracefully
	Close() error
}

type publisher struct {
	channel *amqp091.Channel
}

// NewPublisher create a new Publisher instance
func NewPublisher(amqpURI string) (Publisher, error) {
	conn, err := amqp091.Dial(amqpURI)
	if err != nil {
		return nil, err
	}

	c, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	return &publisher{
		channel: c,
	}, nil
}

func (p *publisher) PublishEvent(event Event) error {
	return publishEvent(p.channel, event)
}

func (p *publisher) PublishRaw(exchange string, msg *RawMessage) error {
	return publishRaw(p.channel, exchange, msg)
}

func (p *publisher) Close() error {
	return p.channel.Close()
}

func publishEvent(ch *amqp091.Channel, event Event) error {
	evtBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("error while encoding event: %s", err)
	}

	return publishRaw(ch, event.Exchange(), &RawMessage{Body: evtBytes})
}

func publishRaw(ch *amqp091.Channel, exchange string, raw *RawMessage) error {
	return ch.Publish(exchange, "", false, false, amqp091.Publishing{
		ContentType:  "application/json",
		Body:         raw.Body,
		Headers:      raw.Headers,
		DeliveryMode: amqp091.Persistent,
	})
}
