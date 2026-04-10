package logx

import (
	"bytes"
	"log"
	"log/slog"
	"strings"
	"testing"
)

func TestLevelFromEnv(t *testing.T) {
	t.Setenv(logLevelEnvVar, "debug")
	if got := LevelFromEnv(); got != slog.LevelDebug {
		t.Fatalf("LevelFromEnv debug = %v", got)
	}

	t.Setenv(logLevelEnvVar, "warn")
	if got := LevelFromEnv(); got != slog.LevelWarn {
		t.Fatalf("LevelFromEnv warn = %v", got)
	}

	t.Setenv(logLevelEnvVar, "error")
	if got := LevelFromEnv(); got != slog.LevelError {
		t.Fatalf("LevelFromEnv error = %v", got)
	}

	t.Setenv(logLevelEnvVar, "")
	if got := LevelFromEnv(); got != slog.LevelInfo {
		t.Fatalf("LevelFromEnv default = %v", got)
	}
}

func TestNewIncludesComponent(t *testing.T) {
	var buffer bytes.Buffer
	logger := New(Options{
		Name:   "vrooli-api",
		Writer: &buffer,
	})
	logger.Info("hello")

	output := buffer.String()
	if !strings.Contains(output, "component=vrooli-api") {
		t.Fatalf("expected component field in log output, got %q", output)
	}
}

func TestRedirectStandardLibrary(t *testing.T) {
	originalWriter := log.Writer()
	originalFlags := log.Flags()
	t.Cleanup(func() {
		log.SetOutput(originalWriter)
		log.SetFlags(originalFlags)
	})

	var buffer bytes.Buffer
	logger := New(Options{
		Name:   "test",
		Writer: &buffer,
	})
	RedirectStandardLibrary(logger)

	log.Printf("redirected log")

	if !strings.Contains(buffer.String(), "redirected log") {
		t.Fatalf("expected redirected log output, got %q", buffer.String())
	}
}
