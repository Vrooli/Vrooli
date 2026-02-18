package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// mockCommandRunner is a test double for CommandRunner.
type mockCommandRunner struct {
	stdout   string
	stderr   string
	exitCode int
	err      error
}

func (m *mockCommandRunner) Run(_ context.Context, _ string, _ []string, _ string) (string, string, int, error) {
	return m.stdout, m.stderr, m.exitCode, m.err
}

func TestExecuteScript_Success(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test-script.sh"), []byte("#!/bin/bash\necho hello"), 0o755); err != nil {
		t.Fatal(err)
	}

	svc := NewScriptService(dir, &mockCommandRunner{
		stdout:   "OK",
		exitCode: 0,
	})

	result, err := svc.ExecuteScript(context.Background(), "test-script", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "completed" {
		t.Errorf("expected status 'completed', got %q", result.Status)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if result.Stdout != "OK" {
		t.Errorf("expected stdout 'OK', got %q", result.Stdout)
	}
}

func TestExecuteScript_Failure(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test-script.sh"), []byte("#!/bin/bash\nexit 1"), 0o755); err != nil {
		t.Fatal(err)
	}

	svc := NewScriptService(dir, &mockCommandRunner{
		stderr:   "failed",
		exitCode: 1,
		err:      fmt.Errorf("exit status 1"),
	})

	result, err := svc.ExecuteScript(context.Background(), "test-script", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "failed" {
		t.Errorf("expected status 'failed', got %q", result.Status)
	}
	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}
	if result.Stderr != "failed" {
		t.Errorf("expected stderr 'failed', got %q", result.Stderr)
	}
}

func TestExecuteScript_Timeout(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test-script.sh"), []byte("#!/bin/bash\nsleep 999"), 0o755); err != nil {
		t.Fatal(err)
	}

	svc := NewScriptService(dir, &mockCommandRunner{
		exitCode: -1,
		err:      context.DeadlineExceeded,
	})

	// Use a context with an already-passed deadline so the derived timeout
	// context's Err() returns DeadlineExceeded.
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	result, err := svc.ExecuteScript(ctx, "test-script", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.TimedOut {
		t.Error("expected TimedOut to be true")
	}
	if result.Status != "failed" {
		t.Errorf("expected status 'failed', got %q", result.Status)
	}
	if result.ExitCode != -1 {
		t.Errorf("expected exit code -1, got %d", result.ExitCode)
	}
}
