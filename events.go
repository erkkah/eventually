package eventually

import (
	"fmt"
	"reflect"
)

// Bus is a simple channel based event bus.
// Events are distributed asynchronously on named topic channels, with
// a list of arbitrary arguments.
type Bus interface {
	// Once registers a callback that will receive at most one event
	Once(topic string, callback interface{}) (Listener, error)
	// On registers a callback that will receive all events until unsubscribed
	On(topic string, callback interface{}) (Listener, error)
	// Post sends an event to all listeners for a specific topic
	Post(topic string, data ...interface{}) error
	// Unsubscribe removes previously registered topic callbacks
	Unsubscribe(topic string, listener Listener)
}

type event struct {
	topic string
	data  []interface{}
}

// Listener is returned from Once and On calls and is used in Unsubscribe
// calls to refer to registered callbacks.
type Listener struct {
	topic    string
	once     bool
	channel  chan []interface{}
	callback reflect.Value
}

type listenerRequest struct {
	listener Listener
	errors   chan error
}

type requestType int

const (
	addListenerReq requestType = iota
	removeListenerReq
	sendEventReq
)

type busRequest struct {
	request  requestType
	event    event
	listener Listener
	errors   chan error
}

type bus struct {
	queueLength    int
	requests       chan busRequest
	topicListeners map[string][]Listener
	eventMap       *EventMap
}

func prepareArguments(generic []interface{}) (specific []reflect.Value) {
	specific = make([]reflect.Value, 0)
	for _, arg := range generic {
		specific = append(specific, reflect.ValueOf(arg))
	}
	return
}

func callListener(callback reflect.Value, evnt []interface{}) (err error) {
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("Failed to call listener %#v with %#v: %v", callback, evnt, x)
		}
	}()
	args := prepareArguments(evnt)
	callback.Call(args)
	return nil
}

func (b *bus) registerListener(topic string, callback interface{}, callOnce bool) (Listener, error) {
	if !(reflect.TypeOf(callback).Kind() == reflect.Func) {
		panic("Listeners must be functions")
	}

	ch := make(chan []interface{})

	l := Listener{
		topic:    topic,
		once:     callOnce,
		channel:  ch,
		callback: reflect.ValueOf(callback),
	}

	go func(l Listener) {
		for {
			evnt, alive := <-l.channel
			if evnt != nil {
				if err := callListener(l.callback, evnt); err != nil {
					// ??? Replace with error on error channel?
					fmt.Println(err)
				}
			}
			if !alive {
				break
			}
		}
	}(l)

	errors := make(chan error)

	b.requests <- busRequest{
		request:  addListenerReq,
		listener: l,
		errors:   errors,
	}

	return l, <-errors
}

func (b *bus) Once(topic string, callback interface{}) (Listener, error) {
	return b.registerListener(topic, callback, true)
}

func (b *bus) On(topic string, callback interface{}) (Listener, error) {
	return b.registerListener(topic, callback, false)
}

func (b *bus) Unsubscribe(topic string, listener Listener) {
	b.requests <- busRequest{
		request:  removeListenerReq,
		listener: listener,
	}
}

func (b *bus) Post(topic string, data ...interface{}) error {
	evnt := event{
		topic, data,
	}

	errors := make(chan error)

	b.requests <- busRequest{
		request: sendEventReq,
		event:   evnt,
		errors:  errors,
	}
	return <-errors
}

// Option is the type of optional arguments to NewBus.
type Option func(*bus)

// EventMap describes event topics and associated argument types.
type EventMap map[string][]interface{}

// WithEventMap sets the event map to check listeners and events against.
func WithEventMap(eventMap EventMap) Option {
	return func(b *bus) {
		b.eventMap = &eventMap
	}
}

// WithQueueLength sets the internal queue length for bus communications.
// When the queue is full, requests to the bus start to block.
// Defaults to 10.
func WithQueueLength(length int) Option {
	return func(b *bus) {
		b.queueLength = length
	}
}

func typesOf(args []interface{}) []reflect.Type {
	result := []reflect.Type{}
	for _, arg := range args {
		argType := reflect.TypeOf(arg)
		result = append(result, argType)
	}
	return result
}

func (b *bus) verifyListener(l Listener) error {
	if b.eventMap == nil {
		return nil
	}
	if eventType, found := (*b.eventMap)[l.topic]; found {
		argTypes := typesOf(eventType)
		expected := reflect.FuncOf(argTypes, []reflect.Type{}, false)
		if l.callback.Type() != expected {
			return fmt.Errorf("Argument mismatch")
		}
		return nil
	}
	return fmt.Errorf("No such topic, %q", l.topic)
}

func (b *bus) addListener(l Listener) error {
	if err := b.verifyListener(l); err != nil {
		return err
	}
	existing, exists := b.topicListeners[l.topic]
	if !exists {
		existing = make([]Listener, 0)
	}
	existing = append(existing, l)
	b.topicListeners[l.topic] = existing
	return nil
}

func (b *bus) removeListener(l Listener) {
	if listeners, exists := b.topicListeners[l.topic]; exists {
		keepList := []Listener{}
		for _, l := range listeners {
			if l.callback == l.callback {
				close(l.channel)
			} else {
				keepList = append(keepList, l)
			}
		}
		b.topicListeners[l.topic] = keepList
	}
}

func (b *bus) verifyEvent(evnt event) error {
	if b.eventMap == nil {
		return nil
	}
	if eventType, found := (*b.eventMap)[evnt.topic]; found {
		argTypes := typesOf(eventType)
		expected := typesOf(evnt.data)
		if !reflect.DeepEqual(argTypes, expected) {
			return fmt.Errorf("Message data mismatch")
		}
		return nil
	}
	return fmt.Errorf("No such topic, %q", evnt.topic)
}

func (b *bus) broadcast(evnt event) error {
	if err := b.verifyEvent(evnt); err != nil {
		return err
	}
	if listeners, exists := b.topicListeners[evnt.topic]; exists {
		keepList := []Listener{}
		for _, l := range listeners {
			l.channel <- evnt.data
			if !l.once {
				keepList = append(keepList, l)
			} else {
				close(l.channel)
			}
		}
		b.topicListeners[evnt.topic] = keepList
	}
	return nil
}

// NewBus creates a new event bus.
//
// If no event map is specified (see WithEventMap), events to
// topics with no registered listeners are ignored, and events
// sent with erronous arguments will panic at the listener callback.
//
// Specifying an event map makes listener registration and event
// posting fail as early as possible.
func NewBus(options ...Option) Bus {
	b := &bus{queueLength: 10}

	for _, o := range options {
		o(b)
	}

	b.requests = make(chan busRequest, b.queueLength)
	b.topicListeners = make(map[string][]Listener)

	go func(b *bus) {
		for {
			request := <-b.requests
			switch request.request {
			case addListenerReq:
				request.errors <- b.addListener(request.listener)
			case removeListenerReq:
				b.removeListener(request.listener)
			case sendEventReq:
				request.errors <- b.broadcast(request.event)
			}
		}
	}(b)

	return b
}
