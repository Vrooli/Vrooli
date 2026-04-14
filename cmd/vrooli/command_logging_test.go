package main

import (
	"bytes"
	"io"
	"log/slog"
	"testing"
)

func TestRunEmitsDebugLogsWhenVerbose(t *testing.T) {
	var captured bytes.Buffer
	app := configuredApp()
	app.NewLoggerFn = func(globals globalOptions, _ io.Writer) (*slog.Logger, func()) {
		level := slog.LevelInfo
		if globals.Verbose {
			level = slog.LevelDebug
		}
		return slog.New(slog.NewTextHandler(&captured, &slog.HandlerOptions{
			Level: level,
		})), func() {}
	}
	app.ResolveSourceRootFn = func() (string, error) { return "/repo", nil }
	app.CheckStalenessFn = nil
	t.Setenv("VROOLI_LOG_LEVEL", "info")

	var stdout bytes.Buffer
	code := app.Run([]string{"--verbose", "version"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("run exit code = %d", code)
	}
	if !bytes.Contains(captured.Bytes(), []byte("Parsed command")) {
		t.Fatalf("missing debug log for parsed command: %q", captured.String())
	}
	if !bytes.Contains(captured.Bytes(), []byte("Resolved root")) {
		t.Fatalf("missing debug log for root resolution: %q", captured.String())
	}
}
