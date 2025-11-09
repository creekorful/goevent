package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/creekorful/goevent"
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
