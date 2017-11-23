package main

import (
	"bufio"
	"bytes"
	"context"

	"github.com/pkg/errors"
)

type Message interface {
	Id() string
	Payload() []byte
	Delete() error
}

type MessageSource interface {
	ReceiveMessageWithContext(context.Context) (Message, error)
}

type MessageSink interface {
	PublishMessage(Message) error
}

type LogWriter struct {
	logger Logger
}

func (lw LogWriter) Write(b []byte) (int, error) {
	// scan for newlines, else the structured logging blows up
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		lw.logger.Info(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return 0, errors.Wrap(err, "logwriter: err reading from buffer")
	}
	return len(b), nil
}
