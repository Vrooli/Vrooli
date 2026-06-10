package services

import (
	"context"
	"testing"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services/mocks"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/testutil"
)

func TestExecuteScript_Success(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteExecutableFile(t, dir, "test-script.sh", "#!/bin/bash\necho hello")

	runner := mocks.NewCommandRunner().WithStdout("OK").WithExitCode(0)
	svc := NewScriptService(dir, runner)

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
	testutil.WriteExecutableFile(t, dir, "test-script.sh", "#!/bin/bash\nexit 1")

	runner := mocks.NewCommandRunner().
		WithStderr("failed").
		WithExitCode(1).
		WithErrorf("exit status 1")
	svc := NewScriptService(dir, runner)

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
	testutil.WriteExecutableFile(t, dir, "test-script.sh", "#!/bin/bash\nsleep 999")

	runner := mocks.NewCommandRunner().
		WithExitCode(-1).
		WithError(context.DeadlineExceeded)
	svc := NewScriptService(dir, runner)

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
