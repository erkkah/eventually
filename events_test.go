package eventually_test

import (
	"fmt"
	events "github.com/erkkah/eventually"
	"testing"
	"time"
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

func TestMessageMap_OKListenerAndMessage(t *testing.T) {
	topics := events.MessageMap{
		"hello": {"string", 42},
	}

	b := events.NewBus(events.WithMessageMap(topics))
	_, err := b.Once("hello", func(s string, i int) {
		//
	})

	if err != nil {
		t.Fatalf("Failed to register callback: %v", err)
	}

	err = b.Post("hello", "foo", 12)

	if err != nil {
		t.Fatalf("Failed to post message: %v", err)
	}
}

func TestMessageMap_BadListener(t *testing.T) {
	topics := events.MessageMap{
		"hello": {"string", 42},
	}

	b := events.NewBus(events.WithMessageMap(topics))
	_, err := b.Once("hello", func(s string, f float32) {
		//
	})

	if err == nil {
		t.Fatal("Registering mistyped callback should fail")
	}
}

func TestMessageMap_BadMessage(t *testing.T) {
	topics := events.MessageMap{
		"hello": {"string", 42},
	}

	b := events.NewBus(events.WithMessageMap(topics))
	_, err := b.Once("hello", func(s string, i int) {
		//
	})

	if err != nil {
		t.Fatalf("Failed to register callback: %v", err)
	}

	err = b.Post("hello", "foo", "splat")

	if err == nil {
		t.Fatal("Posting mistyped message should fail")
	}
}

func TestMessageMap_UnknownTopic(t *testing.T) {
	topics := events.MessageMap{
		"hello": {"string", 42},
	}

	b := events.NewBus(events.WithMessageMap(topics))
	_, err := b.Once("hällo", func(s string, i int) {
		//
	})

	if err == nil {
		t.Fatal("Registering unknown topic listener should fail")
	}
}

func foo(msg string) {
	fmt.Printf("foo: %v\n", msg)
}

func splat(msg string) {
	fmt.Printf("splat: %v\n", msg)
}

func Example() {
	h := events.NewBus()
	h.Once("heartbeat", foo)
	l, _ := h.On("heartbeat", splat)

	h.Post("heartbeat", "1")
	h.Post("heartbeat", "2")
	h.Unsubscribe("heartbeat", l)

	h.Post("heartbeat", "3")

	time.Sleep(time.Second * 2)
}
