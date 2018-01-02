package main

import (
	"bufio"
	"bytes"
	"context"

	"github.com/pkg/errors"
)

// A Message represents a message which is handled by gantry
type Message interface {
	ID() string
	Payload() []byte
	Delete() error
}

// A MessageQueue represents a queue to receive and publish messages
type MessageQueue interface {
	MessageSource
	MessageSink
}

// A MessageSource represents a source to retrieve a Message
type MessageSource interface {
	ReceiveMessageWithContext(context.Context) (Message, error)
}

// A MessageSink represents a channel to publish messages to
type MessageSink interface {
	PublishPayload([]byte) error
}

// LogWriter represents a logger which can be used as io.Writer
type LogWriter struct {
	len     int
	writeFn func(...interface{})
}

func (lw LogWriter) Write(b []byte) (int, error) {
	lw.len += len(b)
	// scan for newlines, else the structured logging blows up
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		lw.writeFn(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return 0, errors.Wrap(err, "logwriter: err reading from buffer")
	}
	return len(b), nil
}
