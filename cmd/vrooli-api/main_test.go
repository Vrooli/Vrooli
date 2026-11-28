package main

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/buildinfo"
)

func TestInstallAPILoggerEmitsStartupWarnings(t *testing.T) {
	t.Setenv("VROOLI_LOG_LEVEL", "trace")
	t.Setenv("VROOLI_LOG_FORMAT", "yaml")

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

	originalDefault := slog.Default()
	logger, restore := installAPILogger()
	t.Cleanup(func() {
		restore()
		slog.SetDefault(originalDefault)
	})

	logger.Info("api bootstrap ready")
	_ = writer.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "invalid_log_level") {
		t.Fatalf("missing invalid_log_level warning: %q", got)
	}
	if !strings.Contains(got, "invalid_log_format") {
		t.Fatalf("missing invalid_log_format warning: %q", got)
	}
	if !strings.Contains(got, "api bootstrap ready") {
		t.Fatalf("missing info log: %q", got)
	}
}

func TestEnforceStrictFingerprint(t *testing.T) {
	originalFingerprint := buildinfo.Fingerprint
	t.Cleanup(func() {
		buildinfo.Fingerprint = originalFingerprint
	})

	t.Run("Disabled", func(t *testing.T) {
		t.Setenv("VROOLI_STRICT_FINGERPRINT", "")
		if err := enforceStrictFingerprint(); err != nil {
			t.Fatalf("enforceStrictFingerprint disabled: %v", err)
		}
	})

	t.Run("Match", func(t *testing.T) {
		root := t.TempDir()
		writeGoTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
		writeGoTestFile(t, root, "cmd/vrooli-api/main.go", "package main\n")
		writeGoTestFile(t, root, "internal/logx/logx.go", "package logx\n")

		t.Setenv("VROOLI_STRICT_FINGERPRINT", "1")
		t.Setenv(buildinfo.SourceRootEnvVar, root)
		t.Setenv(buildinfo.FingerprintPathsEnvVar, "cmd/vrooli-api,internal")

		current, err := buildinfo.CurrentFingerprint()
		if err != nil {
			t.Fatalf("CurrentFingerprint: %v", err)
		}
		buildinfo.Fingerprint = current

		if err := enforceStrictFingerprint(); err != nil {
			t.Fatalf("enforceStrictFingerprint match: %v", err)
		}
	})

	t.Run("Mismatch", func(t *testing.T) {
		root := t.TempDir()
		writeGoTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
		writeGoTestFile(t, root, "cmd/vrooli-api/main.go", "package main\n")
		writeGoTestFile(t, root, "internal/logx/logx.go", "package logx\n")

		t.Setenv("VROOLI_STRICT_FINGERPRINT", "1")
		t.Setenv(buildinfo.SourceRootEnvVar, root)
		t.Setenv(buildinfo.FingerprintPathsEnvVar, "cmd/vrooli-api,internal")

		buildinfo.Fingerprint = "stale-fingerprint"
		err := enforceStrictFingerprint()
		if err == nil {
			t.Fatalf("expected mismatch error")
		}
		if !strings.Contains(err.Error(), "stale-fingerprint") {
			t.Fatalf("mismatch error %q does not include embedded fingerprint", err)
		}
	})
}

func writeGoTestFile(t *testing.T, root, rel, contents string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
