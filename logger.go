package main

type Logger interface {
	Debug(...interface{})
	Debugf(string, ...interface{})
	Info(...interface{})
	Infof(string, ...interface{})
	Fatal(...interface{})
	Fatalf(string, ...interface{})
}

type NoopLogger struct{}

func (nl NoopLogger) Debug(...interface{})          {}
func (nl NoopLogger) Debugf(string, ...interface{}) {}
func (nl NoopLogger) Info(...interface{})           {}
func (nl NoopLogger) Infof(string, ...interface{})  {}
func (nl NoopLogger) Fatal(...interface{})          {}
func (nl NoopLogger) Fatalf(string, ...interface{}) {}
