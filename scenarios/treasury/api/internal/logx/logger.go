// Package logx owns Treasury's replaceable structured logging boundary.
package logx

import "log"

// Logger records an operational error without coupling callers to a logger implementation.
type Logger interface {
	Error(message string, args ...any)
}

// System adapts the process logger at the application boundary.
type System struct{ Logger *log.Logger }

func (s System) Error(message string, args ...any) {
	logger := s.Logger
	if logger == nil {
		logger = log.Default()
	}
	logger.Printf(message, args...)
}

var _ Logger = System{}
