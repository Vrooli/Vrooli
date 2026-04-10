package main

import (
	"errors"
	"log/slog"
	"testing"
)

// [REQ:OBS-003] Additional structured logging edge-case tests

func TestInitStructuredLogging_AllLevels(t *testing.T) {
	for _, level := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
		InitStructuredLogging(level)
	}
}

func TestStructuredLog_ZeroDuration(t *testing.T) {
	InitStructuredLogging(slog.LevelDebug)
	StructuredLog(slog.LevelInfo, "probe", "check", "success", 0, nil)
}

func TestStructuredLog_LargeDuration(t *testing.T) {
	InitStructuredLogging(slog.LevelDebug)
	StructuredLog(slog.LevelWarn, "recovery", "restart", "slow", 999999, nil)
}

func TestStructuredLog_EmptyStrings(t *testing.T) {
	InitStructuredLogging(slog.LevelDebug)
	StructuredLog(slog.LevelInfo, "", "", "", 0, nil)
}

func TestStructuredLog_WrappedError(t *testing.T) {
	InitStructuredLogging(slog.LevelDebug)
	wrapped := errors.Join(errors.New("outer"), errors.New("inner"))
	StructuredLog(slog.LevelError, "tunnel", "check", "failure", 100, wrapped)
}
