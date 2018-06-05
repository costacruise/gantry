package main

import (
	"context"
	"testing"
)

type mockMsg struct{}

func (mm mockMsg) ID() string      { return "mock-msg-id-123" }
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

type warnLoggerSpy struct {
	Logger
	warnCalledWith []string
}

func (wls *warnLoggerSpy) Warn(args ...interface{}) {
	for _, a := range args {
		if s, ok := a.(string); ok {
			wls.warnCalledWith = append(wls.warnCalledWith, s)
		}
	}
}

func Test_Gantry_DeletesMessagesWithNoEntrypoint(t *testing.T) {
	t.Skip("not implemented yet")
}

func Test_Gantry_DeletesMalformedMessages(t *testing.T) {
	t.Skip("not implemented yet")
}

func Test_Gantry_RunsEntrypointScriptInMessagesWithSanePayloads(t *testing.T) {

	payload, err := Payloader{}.DirToBase64EncTarGz("./fixtures/greet")
	if err != nil {
		t.Fatal(err)
	}

	g := Gantry{
		ctx: context.TODO(),
		src: mockSrc{messages: []Message{fixtureMessage{payload: payload}}},
		// logger: logrus.StandardLogger(),
		logger: noopLogger{},
	}

	err = g.HandleMessageIfExists()
	if err != nil {
		t.Fatal(err)
	}
}

func Test_Gantry_RunsExecutableEntrypointScriptWithoutShebang(t *testing.T) {

	payload, err := Payloader{}.DirToBase64EncTarGz("./fixtures/executable-script-no-shebang")
	if err != nil {
		t.Fatal(err)
	}

	wls := &warnLoggerSpy{Logger: noopLogger{}}

	g := Gantry{
		ctx:    context.TODO(),
		src:    mockSrc{messages: []Message{fixtureMessage{payload: payload}}},
		logger: wls,
	}

	err = g.HandleMessageIfExists()
	if err != nil {
		t.Fatal(err)
	}

	if len(wls.warnCalledWith) != 1 {
		t.Fatalf("expected one warning from gantry, got %d", len(wls.warnCalledWith))
	}

	if wls.warnCalledWith[0] != "entrypoint.sh produced no output on stdout" {
		t.Fatalf("expected warning from gantry didn't match assertion")
	}

}

func Test_Gantry_RaisesErrOnNonExecutableEntrypointScript(t *testing.T) {

	payload, err := Payloader{}.DirToBase64EncTarGz("./fixtures/non-executable-entrypoint")
	if err != nil {
		t.Fatal(err)
	}

	g := Gantry{
		ctx:    context.TODO(),
		src:    mockSrc{messages: []Message{fixtureMessage{payload: payload}}},
		logger: noopLogger{},
	}

	err = g.HandleMessageIfExists()
	if err == nil {
		t.Fatalf("expected non executable entrypoint to raise an error")
	}
	if err.Error() != "expected payload to contain executable entrypoint.sh check the filemode" {
		t.Fatalf("expected one error from gantry, got another")
	}
}
