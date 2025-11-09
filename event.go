package goevent

// Event represent an event
type Event interface {
	// Exchange returns the exchange where event should be push
	Exchange() string
}
