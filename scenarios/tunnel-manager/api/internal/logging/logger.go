// Package logging provides the logger seam shared by long-lived API loops.
package logging

import "log"

// Logger is the printf-compatible surface used by middleware and schedulers.
// The standard library logger satisfies it directly, while tests can inject a
// buffer-backed logger without touching process-global state.
type Logger interface {
	Printf(string, ...any)
}

// StructuredLogger documents the richer seam available to future structured
// event emitters without coupling current code to a logging implementation.
type StructuredLogger interface {
	Info(...any)
	Error(...any)
	Debug(...any)
	Warn(...any)
}

// Default returns the process logger at the composition boundary.
func Default() Logger {
	return log.Default()
}
