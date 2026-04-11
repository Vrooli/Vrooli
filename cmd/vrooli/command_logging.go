package main

import (
	"io"
	"log/slog"

	"github.com/vrooli/vrooli/internal/logx"
)

func createCommandLogger(verbose bool, stderr io.Writer) *slog.Logger {
	level := logx.LevelFromEnv()
	if verbose && level > slog.LevelDebug {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(stderr, &slog.HandlerOptions{
		Level: level,
	})
	logger := slog.New(handler).With("component", "vrooli")
	return logger
}

func debugLog(logger *slog.Logger, msg string, args ...any) {
	if logger == nil {
		return
	}
	logger.Debug(msg, args...)
}
