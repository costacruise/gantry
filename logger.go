package main

// Logger represents a leveled logger interface
type Logger interface {
	Debug(...interface{})
	Debugf(string, ...interface{})
	Info(...interface{})
	Infof(string, ...interface{})
	Warn(...interface{})
	Warnf(string, ...interface{})
	Fatal(...interface{})
	Fatalf(string, ...interface{})
}

// noopLogger implements a Logger. It does nothing on each level.
type noopLogger struct{}

func (nl noopLogger) Debug(...interface{})          {}
func (nl noopLogger) Debugf(string, ...interface{}) {}
func (nl noopLogger) Info(...interface{})           {}
func (nl noopLogger) Infof(string, ...interface{})  {}
func (nl noopLogger) Warn(...interface{})           {}
func (nl noopLogger) Warnf(string, ...interface{})  {}
func (nl noopLogger) Fatal(...interface{})          {}
func (nl noopLogger) Fatalf(string, ...interface{}) {}
