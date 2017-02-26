package eventually_test

import (
	"fmt"
	events "github.com/erkkah/eventually"
	"testing"
)

func TestOnce(t *testing.T) {
	b := events.NewBus()

	done := make(chan bool)
	result := 0

	_, err := b.Once("ping", func(msg int) {
		result += msg
		done <- true
	})

	if err != nil {
		t.Fail()
	}

	b.Post("ping", 99)
	b.Post("ping", 99)
	<-done

	if result != 99 {
		t.Fail()
	}
}

func TestUnsubscribe(t *testing.T) {
	b := events.NewBus()

	done := make(chan bool)
	result := 0

	listener, err := b.On("ping", func(msg int) {
		result += msg
		done <- true
	})

	if err != nil {
		t.Fail()
	}

	b.Post("ping", 99)
	b.Unsubscribe("ping", listener)
	b.Post("ping", 99)
	<-done

	if result != 99 {
		t.Fail()
	}
}

func TestEventMap_OKListenerAndEvent(t *testing.T) {
	topics := events.EventMap{
		"hello": {"string", 42},
	}

	b := events.NewBus(events.WithEventMap(topics))

	_, err := b.Once("hello", func(s string, i int) {
		//
	})

	if err != nil {
		t.Fatalf("Failed to register callback: %v", err)
	}

	err = b.Post("hello", "foo", 12)

	if err != nil {
		t.Fatalf("Failed to post event: %v", err)
	}
}

func TestEventMap_BadListener(t *testing.T) {
	topics := events.EventMap{
		"hello": {"string", 42},
	}

	b := events.NewBus(events.WithEventMap(topics))
	_, err := b.Once("hello", func(s string, f float32) {
		//
	})

	if err == nil {
		t.Fatal("Registering mistyped callback should fail")
	}
}

func TestEventMap_BadEvent(t *testing.T) {
	topics := events.EventMap{
		"hello": {"string", 42},
	}

	b := events.NewBus(events.WithEventMap(topics))
	_, err := b.Once("hello", func(s string, i int) {
		//
	})

	if err != nil {
		t.Fatalf("Failed to register callback: %v", err)
	}

	err = b.Post("hello", "foo", "splat")

	if err == nil {
		t.Fatal("Posting mistyped event should fail")
	}
}

func TestEventMap_UnknownTopic(t *testing.T) {
	topics := events.EventMap{
		"hello": {"string", 42},
	}

	b := events.NewBus(events.WithEventMap(topics))
	_, err := b.Once("hällo", func(s string, i int) {
		//
	})

	if err == nil {
		t.Fatal("Registering unknown topic listener should fail")
	}
}

func Example() {
	foo := func(msg string) {
		fmt.Printf("foo: %v\n", msg)
	}

	splat := func(msg string) {
		fmt.Printf("splat: %v\n", msg)
	}

	b := events.NewBus()

	b.Once("heartbeat", foo)

	l, _ := b.On("heartbeat", splat)

	b.Post("heartbeat", "1")
	b.Post("heartbeat", "2")
	b.Unsubscribe("heartbeat", l)

	b.Post("heartbeat", "3")
}

func Example_EventMap() {
	// EventMap maps from topic to a template of instances
	// describing the expected event properties.
	topics := events.EventMap{
		"hello": {"", int(42)},
	}

	done := make(chan bool)

	listener := func(name string, age int) {
		fmt.Printf("Name: %q, age: %v\n", name, age)
		done <- true
	}

	b := events.NewBus(events.WithEventMap(topics))

	b.Once("hello", listener)

	// Post to unknown topic
	err := b.Post("ping")
	if err != nil {
		fmt.Println("Ping event not defined")
	}

	// Post to known topic with wrong arguments
	err = b.Post("hello", 3.14)
	if err != nil {
		fmt.Println("Hello event expects other arguments")
	}

	b.Post("hello", "Fred", 9)

	<-done

	// Output:
	// Ping event not defined
	// Hello event expects other arguments
	// Name: "Fred", age: 9
}
