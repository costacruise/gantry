package main

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

type mockMsg struct{}

func (mm mockMsg) Id() string      { return "mock-msg-id-123" }
func (mm mockMsg) Payload() []byte { return []byte("mock message payload bytes") }
func (mm mockMsg) Delete() error   { return nil }

type fixtureMessage struct {
	mockMsg
	payload []byte
}

func (fm fixtureMessage) Payload() []byte { return fm.payload }

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

func Test_Gantry_RunsEntrypointScriptInMessagesWithSanePayloads(t *testing.T) {

	payload, err := Payloader{}.DirToBase64EncTarGz("./fixtures/greet")
	if err != nil {
		t.Fatal(err)
	}

	g := Gantry{
		ctx:    context.TODO(),
		src:    mockSrc{messages: []Message{fixtureMessage{payload: payload}}},
		logger: logrus.StandardLogger(),
	}

	out, err := g.HandleMessageIfExists()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("output from handlemsg", out)

	if strings.Count(string(out), "Hello Fixture") == 0 {
		t.Fatal("expected output to include the fixture greeting")
	}

}
