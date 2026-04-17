package main

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/vrooli/vrooli/internal/logx"
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

func TestResolveQuietExplicitFlags(t *testing.T) {
	t.Setenv(logx.LogLevelEnvVar, "")
	// Pass a plain buffer (non-TTY) to isolate the explicit-flag branches
	// from the TTY auto-quiet path.
	buf := &bytes.Buffer{}

	if got := resolveQuiet(globalOptions{Verbose: true}, buf); got {
		t.Fatalf("verbose must suppress quiet, got true")
	}
	if got := resolveQuiet(globalOptions{Quiet: true}, buf); !got {
		t.Fatalf("explicit --quiet must return true")
	}
	if got := resolveQuiet(globalOptions{Quiet: true, Verbose: true}, buf); got {
		t.Fatalf("verbose must beat quiet, got true")
	}
}

func TestResolveQuietRespectsExplicitLogLevel(t *testing.T) {
	t.Setenv(logx.LogLevelEnvVar, "info")
	// Use a real /dev/null file to make sure a non-nil *os.File that is
	// not a TTY does not trigger auto-quiet — only the env var override
	// should matter here.
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open /dev/null: %v", err)
	}
	defer devnull.Close()

	if got := resolveQuiet(globalOptions{}, devnull); got {
		t.Fatalf("VROOLI_LOG_LEVEL set must disable auto-quiet, got true")
	}
}

func TestResolveQuietNonTTYWriterStaysLoud(t *testing.T) {
	t.Setenv(logx.LogLevelEnvVar, "")
	// bytes.Buffer is not a TTY — non-interactive consumers (CI, file
	// capture) want the full info stream.
	if got := resolveQuiet(globalOptions{}, &bytes.Buffer{}); got {
		t.Fatalf("non-TTY writer must not auto-quiet")
	}
}

func TestIsTerminalRejectsNonFileWriters(t *testing.T) {
	if isTerminal(nil) {
		t.Fatalf("nil writer reported as terminal")
	}
	if isTerminal(&bytes.Buffer{}) {
		t.Fatalf("bytes.Buffer reported as terminal")
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	defer r.Close()
	defer w.Close()
	if isTerminal(w) {
		t.Fatalf("pipe writer reported as terminal")
	}
}
