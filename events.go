package eventually

import (
	"fmt"
	"reflect"
)

// Bus is a simple channel based event bus.
// Events are distributed on topic channels.
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

type message struct {
	topic string
	data  []interface{}
}

// Listener is used in Unsubscribe calls to refer to a registered listener
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
	sendMessageReq
)

type busRequest struct {
	request  requestType
	message  message
	listener Listener
	errors   chan error
}

type bus struct {
	queueLength    int
	requests       chan busRequest
	topicListeners map[string][]Listener
	messageMap     *MessageMap
}

func prepareArguments(generic []interface{}) (specific []reflect.Value) {
	specific = make([]reflect.Value, 0)
	for _, arg := range generic {
		specific = append(specific, reflect.ValueOf(arg))
	}
	return
}

func callListener(callback reflect.Value, msg []interface{}) (err error) {
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("Failed to call listener %#v with %#v: %v", callback, msg, x)
		}
	}()
	args := prepareArguments(msg)
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
			msg, alive := <-l.channel
			if msg != nil {
				if err := callListener(l.callback, msg); err != nil {
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
	msg := message{
		topic, data,
	}

	errors := make(chan error)

	b.requests <- busRequest{
		request: sendMessageReq,
		message: msg,
		errors:  errors,
	}
	return <-errors
}

// Option is the type of optional arguments to NewBus
type Option func(*bus)

// MessageMap describes message topics and associated argument types
type MessageMap map[string][]interface{}

// WithMessageMap sets the message map to check listeners and messages against
func WithMessageMap(msgMap MessageMap) Option {
	return func(b *bus) {
		b.messageMap = &msgMap
	}
}

// WithQueueLength sets the internal queue length for bus communications
func WithQueueLength(length int) Option {
	return func(b *bus) {
		b.queueLength = length
	}
}

func typesOf(args []interface{}) []reflect.Type {
	result := []reflect.Type{}
	for _, arg := range args {
		result = append(result, reflect.TypeOf(arg))
	}
	return result
}

func (b *bus) verifyListener(l Listener) error {
	if b.messageMap == nil {
		return nil
	}
	if messageType, found := (*b.messageMap)[l.topic]; found {
		argTypes := typesOf(messageType)
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

func (b *bus) verifyMessage(msg message) error {
	if b.messageMap == nil {
		return nil
	}
	if messageType, found := (*b.messageMap)[msg.topic]; found {
		argTypes := typesOf(messageType)
		expected := typesOf(msg.data)
		if !reflect.DeepEqual(argTypes, expected) {
			return fmt.Errorf("Message data mismatch")
		}
		return nil
	}
	return fmt.Errorf("No such topic, %q", msg.topic)
}

func (b *bus) broadcast(msg message) error {
	if err := b.verifyMessage(msg); err != nil {
		return err
	}
	if listeners, exists := b.topicListeners[msg.topic]; exists {
		keepList := []Listener{}
		for _, l := range listeners {
			l.channel <- msg.data
			if !l.once {
				keepList = append(keepList, l)
			} else {
				close(l.channel)
			}
		}
		b.topicListeners[msg.topic] = keepList
	}
	return nil
}

// NewBus creates a new event bus
func NewBus(options ...Option) Bus {
	b := &bus{}

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
			case sendMessageReq:
				request.errors <- b.broadcast(request.message)
			}
		}
	}(b)

	return b
}
