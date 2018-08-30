package main

import (
	"context"
	"os"
	"reflect"
	"testing"
	"time"
)

type mockMsg struct{}

func (mm mockMsg) ID() string        { return "mock-msg-id-123" }
func (mm mockMsg) SentAt() time.Time { return time.Now() }
func (mm mockMsg) Body() messageBody { return messageBody{} }
func (mm mockMsg) Payload() []byte   { return []byte("mock message payload bytes") }
func (mm mockMsg) Delete() error     { return nil }

type fixtureMessage struct {
	mockMsg
	payload []byte
	body    messageBody
	sentAt  time.Time
}

func (fm fixtureMessage) Payload() []byte   { return fm.payload }
func (fm fixtureMessage) Body() messageBody { return fm.body }
func (fm fixtureMessage) SentAt() time.Time { return fm.sentAt }

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

type logSpy struct {
	Logger
	infoCalledWith  []string
	errorCalledWith []string
}

func (wls *logSpy) WithFields(fields Fields) Logger {
	return wls
}

func (wls *logSpy) Info(args ...interface{}) {
	for _, a := range args {
		if s, ok := a.(string); ok {
			wls.infoCalledWith = append(wls.infoCalledWith, s)
		}
	}
}

func (wls *logSpy) Error(args ...interface{}) {
	for _, a := range args {
		if s, ok := a.(string); ok {
			wls.errorCalledWith = append(wls.errorCalledWith, s)
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
	payload, err := Payloader{}.DirToTarGz("./fixtures/greet")
	if err != nil {
		t.Fatal(err)
	}
	logger := NewRecorder()

	g := Gantry{
		ctx: context.TODO(),
		src: mockSrc{messages: []Message{
			fixtureMessage{
				payload: payload,
				body: messageBody{
					Env: map[string]string{"test": "out"},
				},
				sentAt: time.Date(2018, time.June, 01, 0, 0, 0, 0, time.UTC),
			},
		}},
		logger: logger,
	}

	err = g.HandleMessageIfExists()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("logs the status", func(t *testing.T) {
		byLevel := Logs(logger.Logs).ByLevel()

		infoLogs := byLevel["info"]

		if len(infoLogs) != 2 {
			t.Fatalf("expected 2 info log, got %d", len(infoLogs))
		}

		t.Run("on message receive", func(t *testing.T) {
			fields := infoLogs[0].Fields()
			expectedFields := map[string]interface{}{
				"status": "message received",
				"message": map[string]interface{}{
					"id":        "mock-msg-id-123",
					"body":      messageBody{Env: env{"test": "out"}},
					"queued_at": "2018-06-01T00:00:00Z",
				},
			}

			for k, v := range expectedFields {
				if !reflect.DeepEqual(v, fields[k]) {
					t.Errorf("expected log field %q to equal '%+#v', got '%+#v'", k, v, fields[k])
				}
			}
		})

		t.Run("on completion", func(t *testing.T) {
			fields := infoLogs[1].Fields()
			expectedFields := map[string]interface{}{
				"success":           true,
				"status":            "completed",
				"command_env":       map[string]string{"test": "out"},
				"command_stderr":    "Hello stderr\n",
				"message_queued_at": "2018-06-01T00:00:00Z",
			}

			for k, v := range expectedFields {
				if !reflect.DeepEqual(v, fields[k]) {
					t.Errorf("expected log field %q to equal '%v', got '%v'", k, v, fields[k])
				}
			}
		})
	})
}

func Test_Gantry_PropagatesEnvToEntrypoint(t *testing.T) {
	payload, err := Payloader{}.DirToTarGz("./fixtures/env-propagation")
	if err != nil {
		t.Fatal(err)
	}
	logger := NewRecorder()

	g := Gantry{
		ctx: context.TODO(),
		src: mockSrc{messages: []Message{
			fixtureMessage{
				payload: payload,
				body: messageBody{
					Env: env{"TEST_VAR": "is set"},
				},
			},
		}},
		logger: logger,
	}

	err = g.HandleMessageIfExists()
	if err != nil {
		t.Fatal(err)
	}
}

func Test_Gantry_RunsExecutableEntrypointScriptWithoutShebang(t *testing.T) {
	payload, err := Payloader{}.DirToTarGz("./fixtures/executable-script-no-shebang")
	if err != nil {
		t.Fatal(err)
	}

	logger := NewRecorder()

	g := Gantry{
		ctx: context.TODO(),
		src: mockSrc{messages: []Message{
			fixtureMessage{
				payload: payload,
				body: messageBody{
					Env: map[string]string{"test": "out"},
				},
				sentAt: time.Date(2018, time.June, 01, 0, 0, 0, 0, time.UTC),
			},
		}},
		logger: logger,
	}

	err = g.HandleMessageIfExists()
	pathErr, ok := err.(*os.PathError)
	if !ok {
		t.Fatalf("expected path error, got %v", err)
	}
	if pathErr.Op != "fork/exec" {
		t.Fatalf("expected path error operation to equal 'fork/exec', got %q", pathErr.Op)
	}

	t.Run("logs the error", func(t *testing.T) {
		byLevel := Logs(logger.Logs).ByLevel()

		errorLogs := byLevel["error"]

		if len(errorLogs) != 1 {
			t.Fatalf("expected 1 error log, got %d", len(errorLogs))
		}

		fields := errorLogs[0].Fields()
		expectedFields := map[string]interface{}{
			"success":           false,
			"status":            "completed",
			"command_env":       map[string]string{"test": "out"},
			"command_stderr":    "",
			"message_queued_at": "2018-06-01T00:00:00Z",
		}

		for k, v := range expectedFields {
			if !reflect.DeepEqual(v, fields[k]) {
				t.Errorf("expected log field %q to equal '%v', got '%v'", k, v, fields[k])
			}
		}
		if fields["success"] != false {
			t.Errorf("expected log field 'success' to equal 'false', got '%v'", fields["success"])
		}
		if fields["status"] != "completed" {
			t.Errorf("expected log field 'status' to equal 'completed', got '%v'", fields["status"])
		}
		if _, ok := fields["error"]; !ok {
			t.Errorf("expected entry to contain 'error' field")
		}
	})
}

func Test_Gantry_RaisesErrOnNonExecutableEntrypointScript(t *testing.T) {
	payload, err := Payloader{}.DirToTarGz("./fixtures/non-executable-entrypoint")
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
