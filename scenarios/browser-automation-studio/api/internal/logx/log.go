// Package logx provides the structured logging boundary used by components
// that need deterministic error reporting in tests.
package logx

import "log"

// Logger is the subset needed for encoding and response failures.
type Logger interface {
	Errorf(format string, args ...any)
}

// Standard adapts the process standard logger for production composition.
type Standard struct{}

func (Standard) Errorf(format string, args ...any) { log.Printf(format, args...) }
