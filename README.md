[![GoDoc](https://godoc.org/github.com/erkkah/eventually?status.svg)](https://godoc.org/github.com/erkkah/eventually)
[![GitHub release](https://img.shields.io/github/release/erkkah/eventually.svg)](https://github.com/erkkah/eventually/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/erkkah/eventually)](https://goreportcard.com/report/github.com/erkkah/eventually)
[![Build Status](https://travis-ci.org/erkkah/eventually.svg?branch=master)](https://travis-ci.org/erkkah/eventually)

# Eventually - yet another in-process asynchronous event bus for Go

__Eventually__ (https://github.com/erkkah/eventually) is an event bus for Go, providing
an asynchronous in-process decoupled event pub/sub system.

Providing an optional event map makes listener registration and event publishing
fail early, making it easier to track down event type mismatches.

## Getting Started

Get the source using go get:

`go get github.com/erkkah/eventually`

or, if you use modules, just import and go!

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
