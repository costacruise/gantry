package main

import (
	"context"
)

type mockMsg struct{}

func (mm mockMsg) Id() string      { return "mock-msg-id-123" }
func (mm mockMsg) Payload() []byte { return []byte("mock message payload bytes") }
func (mm mockMsg) Delete() error   { return nil }

type happyPathMessage struct {
	mockMsg
}

type mockSrc struct {
	mockErr  error
	messages []Message
}

func (ms mockSrc) ReceiveMessageWithContext(context.Context) (Message, error) {
	if returnErr := ms.mockErr; returnErr != nil {
		ms.mockErr = nil
		return nil, returnErr
	}
	return ms.messages[0], nil
}
