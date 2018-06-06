package main

import (
	"fmt"
)

var _ Logger = &LogRecorder{}

// A LogEntry represents an entry in the logs
type LogEntry map[string]interface{}

// Level returns the level of the LogEntry
func (l LogEntry) Level() string {
	if l == nil {
		return ""
	}
	return fmt.Sprint(l["level"])
}

// Message returns the message of the LogEntry
func (l LogEntry) Message() string {
	if l == nil {
		return ""
	}
	return fmt.Sprint(l["msg"])
}

// Fields return the fields added to the entry
func (l LogEntry) Fields() Fields {
	fields := Fields{}
	for k, v := range l {
		if k == "level" {
			continue
		}
		if k == "msg" {
			continue
		}
		fields[k] = v
	}
	return fields
}

// Logs represent multiple log entries
type Logs []LogEntry

// ByLevel aggregates the entries by log level
func (l Logs) ByLevel() map[string][]LogEntry {
	byLevel := map[string][]LogEntry{
		"debug": []LogEntry{},
		"info":  []LogEntry{},
		"warn":  []LogEntry{},
		"error": []LogEntry{},
		"fatal": []LogEntry{},
	}
	for _, entry := range l {
		byLevel[entry.Level()] = append(
			byLevel[entry.Level()], entry,
		)
	}
	return byLevel
}

// NewRecorder returns an initialized LogRecorder.
func NewRecorder() *LogRecorder {
	return &LogRecorder{}
}

// LogRecorder is an implementation of Logger that records its
// mutations for later inspection in tests.
type LogRecorder struct {
	Logs       []LogEntry
	fields     map[string]interface{}
	writeLogFn func(entry map[string]interface{})
}

func (lr *LogRecorder) writeLogEntry(entry map[string]interface{}) {
	if lr.writeLogFn != nil {
		lr.writeLogFn(entry)
		return
	}

	lr.Logs = append(lr.Logs, entry)
}

func (lr *LogRecorder) log(level string, v ...interface{}) {
	logEntry := map[string]interface{}{
		"level": level,
		"msg":   fmt.Sprint(v...),
	}
	for k, v := range lr.fields {
		if k != "level" && k != "msg" {
			logEntry[k] = v
		}
	}
	lr.writeLogEntry(logEntry)
}

func (lr *LogRecorder) logf(level string, template string, v ...interface{}) {
	logEntry := map[string]interface{}{
		"level": level,
		"msg":   fmt.Sprintf(template, v...),
	}
	for k, v := range lr.fields {
		if k != "level" && k != "msg" {
			logEntry[k] = v
		}
	}
	lr.writeLogEntry(logEntry)
}

// Debug logs the values v at DEBUG level
func (lr *LogRecorder) Debug(v ...interface{}) {
	lr.log("debug", v...)
}

// Debugf logs the values v at DEBUG level by interpolating them into template
func (lr *LogRecorder) Debugf(template string, v ...interface{}) {
	lr.logf("debug", template, v...)
}

// Info logs the values v at INFO level
func (lr *LogRecorder) Info(v ...interface{}) {
	lr.log("info", v...)
}

// Infof logs the values v at INFO level by interpolating them into template
func (lr *LogRecorder) Infof(template string, v ...interface{}) {
	lr.logf("info", template, v...)
}

// Warn logs the values v at WARN level
func (lr *LogRecorder) Warn(v ...interface{}) {
	lr.log("warn", v...)
}

// Warnf logs the values v at WARN level by interpolating them into template
func (lr *LogRecorder) Warnf(template string, v ...interface{}) {
	lr.logf("warn", template, v...)
}

// Error logs the values v at ERROR level
func (lr *LogRecorder) Error(v ...interface{}) {
	lr.log("error", v...)
}

// Errorf logs the values v at ERROR level by interpolating them into template
func (lr *LogRecorder) Errorf(template string, v ...interface{}) {
	lr.logf("error", template, v...)
}

// Fatal logs the values v at FATAL level
func (lr *LogRecorder) Fatal(v ...interface{}) {
	lr.log("fatal", v...)
}

// Fatalf logs the values v at FATAL level by interpolating them into template
func (lr *LogRecorder) Fatalf(template string, v ...interface{}) {
	lr.logf("fatal", template, v...)
}

// WithFields returns lr
func (lr *LogRecorder) WithFields(fields Fields) Logger {
	newFields := make(map[string]interface{})
	for k, v := range lr.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}
	l := &LogRecorder{
		writeLogFn: lr.writeLogEntry,
		fields:     newFields,
	}
	return l
}
