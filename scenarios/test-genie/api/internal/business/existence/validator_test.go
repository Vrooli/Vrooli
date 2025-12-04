package existence

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRequirementsDir_Success(t *testing.T) {
	tmpDir := t.TempDir()
	reqDir := filepath.Join(tmpDir, "requirements")
	if err := os.MkdirAll(reqDir, 0o755); err != nil {
		t.Fatalf("failed to create requirements dir: %v", err)
	}

	v := New(tmpDir, io.Discard)
	result := v.ValidateRequirementsDir()

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if len(result.Observations) == 0 {
		t.Error("expected observations to be recorded")
	}
}

func TestValidateRequirementsDir_Missing(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't create requirements directory

	v := New(tmpDir, io.Discard)
	result := v.ValidateRequirementsDir()

	if result.Success {
		t.Fatal("expected failure when requirements directory missing")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration failure class, got %s", result.FailureClass)
	}
	if result.Remediation == "" {
		t.Error("expected remediation guidance")
	}
	if !strings.Contains(result.Error.Error(), "directory not found") {
		t.Errorf("expected 'directory not found' error, got: %v", result.Error)
	}
}

func TestValidateRequirementsDir_IsFile(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a file instead of directory
	reqPath := filepath.Join(tmpDir, "requirements")
	if err := os.WriteFile(reqPath, []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	v := New(tmpDir, io.Discard)
	result := v.ValidateRequirementsDir()

	if result.Success {
		t.Fatal("expected failure when requirements is a file")
	}
	if !strings.Contains(result.Error.Error(), "not a directory") {
		t.Errorf("expected 'not a directory' error, got: %v", result.Error)
	}
}

func TestValidateIndexFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	reqDir := filepath.Join(tmpDir, "requirements")
	if err := os.MkdirAll(reqDir, 0o755); err != nil {
		t.Fatalf("failed to create requirements dir: %v", err)
	}
	indexPath := filepath.Join(reqDir, "index.json")
	if err := os.WriteFile(indexPath, []byte(`{"imports":[]}`), 0o644); err != nil {
		t.Fatalf("failed to create index file: %v", err)
	}

	v := New(tmpDir, io.Discard)
	result := v.ValidateIndexFile()

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if len(result.Observations) == 0 {
		t.Error("expected observations to be recorded")
	}
}

func TestValidateIndexFile_Missing(t *testing.T) {
	tmpDir := t.TempDir()
	reqDir := filepath.Join(tmpDir, "requirements")
	if err := os.MkdirAll(reqDir, 0o755); err != nil {
		t.Fatalf("failed to create requirements dir: %v", err)
	}
	// Don't create index.json

	v := New(tmpDir, io.Discard)
	result := v.ValidateIndexFile()

	if result.Success {
		t.Fatal("expected failure when index file missing")
	}
	if result.FailureClass != FailureClassMisconfiguration {
		t.Errorf("expected misconfiguration failure class, got %s", result.FailureClass)
	}
	if result.Remediation == "" {
		t.Error("expected remediation guidance")
	}
}

func TestValidateIndexFile_IsDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	reqDir := filepath.Join(tmpDir, "requirements")
	if err := os.MkdirAll(reqDir, 0o755); err != nil {
		t.Fatalf("failed to create requirements dir: %v", err)
	}
	// Create a directory instead of file
	indexPath := filepath.Join(reqDir, "index.json")
	if err := os.MkdirAll(indexPath, 0o755); err != nil {
		t.Fatalf("failed to create index directory: %v", err)
	}

	v := New(tmpDir, io.Discard)
	result := v.ValidateIndexFile()

	if result.Success {
		t.Fatal("expected failure when index.json is a directory")
	}
	if !strings.Contains(result.Error.Error(), "directory, expected file") {
		t.Errorf("expected 'directory, expected file' error, got: %v", result.Error)
	}
}

// TestValidate_WithNilLogWriter tests that nil log writer is handled.
func TestValidate_WithNilLogWriter(t *testing.T) {
	tmpDir := t.TempDir()
	reqDir := filepath.Join(tmpDir, "requirements")
	if err := os.MkdirAll(reqDir, 0o755); err != nil {
		t.Fatalf("failed to create requirements dir: %v", err)
	}

	// Create validator with nil writer
	v := New(tmpDir, nil)
	result := v.ValidateRequirementsDir()

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
}

// TestValidate_WithRealLogWriter tests that logging works with real writer.
func TestValidate_WithRealLogWriter(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't create requirements directory - should fail

	var buf strings.Builder
	v := New(tmpDir, &buf)
	_ = v.ValidateRequirementsDir()

	output := buf.String()
	// Should have logged the error
	if !strings.Contains(output, "Missing") && !strings.Contains(output, "ERROR") {
		t.Errorf("expected error logging output, got: %s", output)
	}
}

// TestValidate_LogsSuccessMessages tests that success messages are logged.
func TestValidate_LogsSuccessMessages(t *testing.T) {
	tmpDir := t.TempDir()
	reqDir := filepath.Join(tmpDir, "requirements")
	if err := os.MkdirAll(reqDir, 0o755); err != nil {
		t.Fatalf("failed to create requirements dir: %v", err)
	}

	var buf strings.Builder
	v := New(tmpDir, &buf)
	result := v.ValidateRequirementsDir()

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}

	output := buf.String()
	if !strings.Contains(output, "SUCCESS") {
		t.Errorf("expected success logging output, got: %s", output)
	}
}

// Ensure validator satisfies the interface at compile time
var _ Validator = (*validator)(nil)

// Benchmarks

func BenchmarkValidateRequirementsDir(b *testing.B) {
	tmpDir := b.TempDir()
	reqDir := filepath.Join(tmpDir, "requirements")
	if err := os.MkdirAll(reqDir, 0o755); err != nil {
		b.Fatalf("failed to create requirements dir: %v", err)
	}

	v := New(tmpDir, io.Discard)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.ValidateRequirementsDir()
	}
}

func BenchmarkValidateIndexFile(b *testing.B) {
	tmpDir := b.TempDir()
	reqDir := filepath.Join(tmpDir, "requirements")
	if err := os.MkdirAll(reqDir, 0o755); err != nil {
		b.Fatalf("failed to create requirements dir: %v", err)
	}
	indexPath := filepath.Join(reqDir, "index.json")
	if err := os.WriteFile(indexPath, []byte(`{"imports":[]}`), 0o644); err != nil {
		b.Fatalf("failed to create index file: %v", err)
	}

	v := New(tmpDir, io.Discard)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.ValidateIndexFile()
	}
}
