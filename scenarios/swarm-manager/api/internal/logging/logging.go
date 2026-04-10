// Package logging provides structured logging via log/slog with context propagation.
//
// Usage:
//
//	logger := logging.FromContext(ctx)
//	logger.Info("item created", "kind", kind, "name", name)
//	logger.Warn("retrying", "attempt", n, "err", err)
//	logger.Error("failed", "err", err)
package logging

import (
	"context"
	"log/slog"
	"os"
)

type contextKey struct{}

// Init configures the default slog logger with JSON output for production.
func Init() {
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
}

// WithLogger returns a new context carrying the given logger.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, logger)
}

// FromContext returns the logger attached to the context, or the default logger.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(contextKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// With returns a child logger with the given key-value pairs attached.
// This is a convenience wrapper for FromContext(ctx).With(args...).
func With(ctx context.Context, args ...any) *slog.Logger {
	return FromContext(ctx).With(args...)
}
