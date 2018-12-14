package main

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type temporary interface {
	Temporary() bool
}

type timeout interface {
	Timeout() bool
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

// Fields contains structured data added to the logs
type Fields map[string]interface{}

func (f Fields) logError(err error) Fields {
	errorFields := map[string]interface{}{
		"message":   err.Error(),
		"name":      fmt.Sprintf("%T", errors.Cause(err)),
		"stack":     nil,
		"temporary": false,
		"timeout":   false,
	}
	if st, ok := err.(stackTracer); ok {
		errorFields["stack"] = fmt.Sprintf("%v", st.StackTrace())
	}
	if t, ok := err.(temporary); ok {
		errorFields["temporary"] = t.Temporary()
	}
	if t, ok := err.(timeout); ok {
		errorFields["timeout"] = t.Timeout()
	}
	f["err"] = errorFields
	f["error"] = err
	return f
}

// ErrorFields returns fields which include the error
func ErrorFields(err error) Fields {
	return Fields{}.logError(err)
}

// Logger represents a leveled logger interface
type Logger interface {
	Debug(...interface{})
	Debugf(string, ...interface{})
	Info(...interface{})
	Infof(string, ...interface{})
	Warn(...interface{})
	Warnf(string, ...interface{})
	Error(...interface{})
	Errorf(string, ...interface{})
	Fatal(...interface{})
	Fatalf(string, ...interface{})
	WithFields(fields Fields) Logger
}

// noopLogger implements a Logger. It does nothing on each level.
type noopLogger struct{}

func (nl noopLogger) Debug(...interface{})            {}
func (nl noopLogger) Debugf(string, ...interface{})   {}
func (nl noopLogger) Info(...interface{})             {}
func (nl noopLogger) Infof(string, ...interface{})    {}
func (nl noopLogger) Warn(...interface{})             {}
func (nl noopLogger) Warnf(string, ...interface{})    {}
func (nl noopLogger) Error(...interface{})            {}
func (nl noopLogger) Errorf(string, ...interface{})   {}
func (nl noopLogger) Fatal(...interface{})            {}
func (nl noopLogger) Fatalf(string, ...interface{})   {}
func (nl noopLogger) WithFields(fields Fields) Logger { return nl }

// NewLogrusLogger returns a new Logger from a logrus.Entry
func NewLogrusLogger(logEntry *logrus.Entry) Logger {
	return &logger{logEntry}
}

type logger struct {
	*logrus.Entry
}

func (l *logger) WithFields(fields Fields) Logger {
	var logrusFields = logrus.Fields{}
	for k, v := range fields {
		logrusFields[k] = v
	}

	return &logger{l.Entry.WithFields(logrusFields)}
}
