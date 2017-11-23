package main

import "context"

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
