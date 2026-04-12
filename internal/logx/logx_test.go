package logx

import (
	"bytes"
	"log"
	"log/slog"
	"os"
	"strings"
	"testing"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=1 | LAST: 2026-04-11

func TestLevelFromEnv(t *testing.T) {
	t.Setenv(LogLevelEnvVar, "debug")
	got, err := LevelFromEnv()
	if err != nil {
		t.Fatalf("LevelFromEnv debug returned error: %v", err)
	}
	if got != slog.LevelDebug {
		t.Fatalf("LevelFromEnv debug = %v", got)
	}

	t.Setenv(LogLevelEnvVar, "warn")
	got, err = LevelFromEnv()
	if err != nil {
		t.Fatalf("LevelFromEnv warn returned error: %v", err)
	}
	if got != slog.LevelWarn {
		t.Fatalf("LevelFromEnv warn = %v", got)
	}

	t.Setenv(LogLevelEnvVar, "error")
	got, err = LevelFromEnv()
	if err != nil {
		t.Fatalf("LevelFromEnv error returned error: %v", err)
	}
	if got != slog.LevelError {
		t.Fatalf("LevelFromEnv error = %v", got)
	}

	t.Setenv(LogLevelEnvVar, "")
	got, err = LevelFromEnv()
	if err != nil {
		t.Fatalf("LevelFromEnv default returned error: %v", err)
	}
	if got != slog.LevelInfo {
		t.Fatalf("LevelFromEnv default = %v", got)
	}
}

func TestLevelFromEnvRejectsInvalidValues(t *testing.T) {
	t.Setenv(LogLevelEnvVar, "trace")
	got, err := LevelFromEnv()
	if err == nil {
		t.Fatal("expected invalid value error")
	}
	if got != slog.LevelInfo {
		t.Fatalf("invalid values should fall back to info, got %v", got)
	}
}

func TestNewIncludesComponent(t *testing.T) {
	var buffer bytes.Buffer
	logger, diagnostics := New(Options{
		Component: "vrooli-api",
		Writer:    &buffer,
	})
	if len(diagnostics.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", diagnostics.Warnings)
	}
	logger.Info("hello")

	output := buffer.String()
	if !strings.Contains(output, "component=vrooli-api") {
		t.Fatalf("expected component field in log output, got %q", output)
	}
}

func TestNewSupportsJSONOutput(t *testing.T) {
	var buffer bytes.Buffer
	logger, diagnostics := New(Options{
		Component: "vrooli-api",
		Writer:    &buffer,
		JSON:      true,
	})
	if len(diagnostics.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", diagnostics.Warnings)
	}

	logger.Info("hello", "port", 8092)
	output := buffer.String()
	if !strings.Contains(output, `"component":"vrooli-api"`) {
		t.Fatalf("expected json component field, got %q", output)
	}
	if !strings.Contains(output, `"port":8092`) {
		t.Fatalf("expected json attribute, got %q", output)
	}
}

func TestNewVerboseOverridesInfoEnv(t *testing.T) {
	t.Setenv(LogLevelEnvVar, "info")

	var buffer bytes.Buffer
	logger, diagnostics := New(Options{
		Component: "vrooli",
		Writer:    &buffer,
		Verbose:   true,
	})
	if got := diagnostics.Level; got != slog.LevelDebug {
		t.Fatalf("verbose level = %v, want debug", got)
	}

	logger.Debug("visible")
	if !strings.Contains(buffer.String(), "visible") {
		t.Fatalf("expected verbose logger to emit debug logs, got %q", buffer.String())
	}
}

func TestNewDefaultsWriterToStderr(t *testing.T) {
	originalStderr := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = writer
	t.Cleanup(func() {
		os.Stderr = originalStderr
		_ = reader.Close()
		_ = writer.Close()
	})

	logger, _ := New(Options{Component: "vrooli"})
	logger.Info("stderr default")
	_ = writer.Close()

	data, err := ioReadAll(reader)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	if !strings.Contains(string(data), "stderr default") {
		t.Fatalf("expected stderr output, got %q", string(data))
	}
}

func TestInstallWarnsOnceForInvalidEnvValue(t *testing.T) {
	t.Setenv(LogLevelEnvVar, "trace")

	var buffer bytes.Buffer
	logger, diagnostics, restore := Install(Options{
		Component:      "vrooli",
		Writer:         &buffer,
		SetDefault:     true,
		RedirectStdlib: true,
	})
	defer restore()
	if logger == nil {
		t.Fatal("expected logger")
	}
	if len(diagnostics.Warnings) != 1 {
		t.Fatalf("warnings = %v, want 1 warning", diagnostics.Warnings)
	}
	if !strings.Contains(buffer.String(), "Invalid "+LogLevelEnvVar+" value; using info level") {
		t.Fatalf("expected invalid env warning in log output, got %q", buffer.String())
	}
}

func TestRedirectStandardLibrary(t *testing.T) {
	restore := func() {}

	{
		var buffer bytes.Buffer
		logger, diagnostics := New(Options{
			Component: "test",
			Writer:    &buffer,
		})
		restore = RedirectStandardLibrary(logger, diagnostics.Level)
		log.Printf("redirected log")

		if !strings.Contains(buffer.String(), "redirected log") {
			t.Fatalf("expected redirected log output, got %q", buffer.String())
		}
	}
	restore()
}

func TestRedirectStandardLibraryRestoreReinstatesPriorState(t *testing.T) {
	originalWriter := log.Writer()
	originalFlags := log.Flags()
	originalPrefix := log.Prefix()

	var buffer bytes.Buffer
	logger, diagnostics := New(Options{Writer: &buffer})
	restore := RedirectStandardLibrary(logger, diagnostics.Level)

	if log.Flags() != 0 {
		t.Fatalf("expected redirected flags to be cleared, got %d", log.Flags())
	}
	restore()

	if log.Writer() != originalWriter {
		t.Fatal("expected original writer to be restored")
	}
	if log.Flags() != originalFlags {
		t.Fatalf("expected original flags %d, got %d", originalFlags, log.Flags())
	}
	if log.Prefix() != originalPrefix {
		t.Fatalf("expected original prefix %q, got %q", originalPrefix, log.Prefix())
	}
}

func ioReadAll(r *os.File) ([]byte, error) {
	var buffer bytes.Buffer
	_, err := buffer.ReadFrom(r)
	return buffer.Bytes(), err
}
