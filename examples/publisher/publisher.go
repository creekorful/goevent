package main

import (
	"log"
	"os"

	"github.com/creekorful/goevent"
)

type GreetingsEvent struct {
	Name string `json:"name"`
}

func (evt *GreetingsEvent) Exchange() string {
	return "greetings"
}

func main() {
	pub, err := goevent.NewPublisher(os.Getenv("PUBLISHER_URI"))
	if err != nil {
		log.Fatal(err)
	}
	defer pub.Close()

	if err := pub.PublishEvent(&GreetingsEvent{Name: "John Doe"}); err != nil {
		log.Fatal(err)
	}
}
