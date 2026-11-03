package logx

import "log/slog"

// seam: Logger carries structured diagnostics into application domains.
// Production wires slog through Default or the server composition root; tests
// can substitute a recording implementation without changing domain behavior.
type Logger interface {
	Debug(message string, args ...any)
	Info(message string, args ...any)
	Warn(message string, args ...any)
	Error(message string, args ...any)
}

var _ Logger = (*slog.Logger)(nil)

// Default adapts the process logger at the composition edge for optional
// constructors. Runtime composition should pass its configured logger.
func Default() Logger { return slog.Default() }
