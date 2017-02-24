### Eventually - yet another in-process asynchronous event bus for Go

Eventually (https://github.com/erkkah/eventually) is an event bus for Go, providing
an asynchronous in-process decoupled event pub/sub system.

Providing an optional event map makes listener registration and event publishing
fail early, making it easier to track down event type mismatches.

## Getting Started

Get the source using go get:

`go get github.com/erkkah/eventually`

or - even better, use [glide](https://glide.sh):

`glide get github.com/erkkah/eventually`

## Example

```go
import "github.com/erkkah/eventually"
import "fmt"

func listener(msg string) {
    fmt.Printf("Received: %v\n", msg)
}

func main() {
    bus := eventually.NewBus()
    bus.On("call", listener)
    bus.Post("call", "Hello?")
}
```
