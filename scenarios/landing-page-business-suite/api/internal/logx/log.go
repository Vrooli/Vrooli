// Package logx owns the application logging boundary.
package logx

import (
	"context"
	"fmt"
	"log"
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

// Structured emits a JSON log record through slog's attribute encoder. The
// standard logger's current writer remains the sink so tests and operators
// can redirect it without changing the application logger.
func Structured(level slog.Level, message string, fields map[string]interface{}) {
	attrs := make([]slog.Attr, 0, len(fields))
	for key, value := range fields {
		attrs = append(attrs, slog.Any(key, value))
	}
	handler := slog.NewJSONHandler(log.Writer(), &slog.HandlerOptions{Level: level})
	slog.New(handler).LogAttrs(context.Background(), level, message, attrs...)
}

// Info emits a structured informational event for dependency injection sites.
func Info(message string, fields map[string]interface{}) {
	Structured(slog.LevelInfo, message, fields)
}

// Error emits a structured error event for dependency injection sites.
func Error(message string, fields map[string]interface{}) {
	Structured(slog.LevelError, message, fields)
}

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
