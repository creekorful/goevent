# goevent

Go library to build event driven applications easily using AMQP as a backend.

## Publisher

```go
package main

import (
	"github.com/creekorful/goevent"
	"log"
	"os"
)

type GreetingsEvent struct {
	Name string `json:"name"`
}

func (evt *GreetingsEvent) Exchange() string {
	return "greetings"
}

func main() {
	pub, err := event.NewPublisher(os.Getenv("PUBLISHER_URI"))
	if err != nil {
		log.Fatal(err)
	}
	defer pub.Close()

	if err := pub.PublishEvent(&GreetingsEvent{Name: "John Doe"}); err != nil {
		log.Fatal(err)
	}
}
```

## Subscriber

```go
package main

import (
	"github.com/creekorful/goevent"
	"log"
	"os"
	"os/signal"
	"syscall"
)

type GreetingsEvent struct {
	Name string `json:"name"`
}

func (evt *GreetingsEvent) Exchange() string {
	return "greetings"
}

func main() {
	sub, err := goevent.NewSubscriber(os.Getenv("SUBSCRIBER_URI"), 1)
	if err != nil {
		log.Fatal(err)
	}

	if err := sub.Subscribe("greetings", "greetingQueue", "", greetingsHandler); err != nil {
		log.Fatal(err)
	}

	// Handle graceful shutdown
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Waiting for events...")

	// Block until we receive our signal.
	<-ch

	log.Println("Terminating connection...")

	if err := sub.Close(); err != nil {
		log.Fatal(err)
	}
}

func greetingsHandler(sub goevent.Subscriber, msg *goevent.RawMessage) error {
	var evt GreetingsEvent

	if err := sub.Read(msg, &evt); err != nil {
		return err
	}

	log.Printf("Hello %s", evt.Name)

	return nil
}
```

## Who's using the library?

- [creekorful/bathyscaphe](https://github.com/creekorful/bathyscaphe) : Fast, highly configurable, cloud native dark
  web crawler.