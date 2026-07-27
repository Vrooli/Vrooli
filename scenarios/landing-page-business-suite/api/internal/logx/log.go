// Package logx owns the application logging boundary.
package logx

import (
	"fmt"
	"log/slog"
	"os"
)

// Logger is the injectable boundary for application logging.
//
// seam: Logger
type Logger interface {
	Printf(format string, args ...any)
	Fatalf(format string, args ...any)
}

// System writes structured events through the process logger.
type System struct{}

// Printf writes a formatted log event.
func (System) Printf(format string, args ...any) {
	slog.Default().Info(fmt.Sprintf(format, args...))
}

// Fatalf writes a formatted log event and terminates the process.
func (System) Fatalf(format string, args ...any) {
	slog.Default().Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}

var _ Logger = System{}

// Printf is the production convenience entry point. Dependencies that need
// deterministic logging should accept a Logger instead.
func Printf(format string, args ...any) { System{}.Printf(format, args...) }

// Fatalf logs a fatal event and terminates the process.
func Fatalf(format string, args ...any) { System{}.Fatalf(format, args...) }
