package main

import (
	"context"
	"log/slog"
	"os"
)

// InitStructuredLogging configures the global slog logger to output
// JSON to stderr for journalctl compatibility. [REQ:OBS-003]
func InitStructuredLogging(level slog.Level) {
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))
}

// StructuredLog emits a structured log entry with standard fields. [REQ:OBS-003]
func StructuredLog(level slog.Level, component, action, result string, durationMs int64, err error) {
	attrs := []any{
		slog.String("component", component),
		slog.String("action", action),
		slog.String("result", result),
		slog.Int64("duration_ms", durationMs),
	}
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}
	slog.Log(context.Background(), level, action, attrs...)
}
