package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services/mocks"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/testutil"
)

type nativeRunnerStub struct {
	query string
	data  []byte
}

func (r *nativeRunnerStub) RunNative(_ context.Context, query string) ([]byte, error) {
	r.query = query
	return r.data, nil
}

func writeExecutionPolicy(t *testing.T, scriptsDir string, policy string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(filepath.Dir(scriptsDir), "execution-policy.json"), []byte(policy), 0o600); err != nil {
		t.Fatal(err)
	}
}

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

func TestUpdateScriptPersistsContent(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteExecutableFile(t, dir, "test-script.sh", "#!/bin/bash\n# NAME: Test script\necho old")
	svc := NewScriptService(dir)
	updated := "#!/bin/bash\n# NAME: Test script\necho updated"

	meta, content, err := svc.UpdateScript("test-script", updated)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.ID != "test-script" {
		t.Fatalf("updated metadata ID = %q, want test-script", meta.ID)
	}
	if content != updated {
		t.Fatalf("returned content = %q, want %q", content, updated)
	}

	onDisk, err := os.ReadFile(filepath.Join(dir, "test-script.sh"))
	if err != nil {
		t.Fatalf("reading updated script: %v", err)
	}
	if string(onDisk) != updated {
		t.Fatalf("on-disk content = %q, want %q", string(onDisk), updated)
	}
}

func TestUpdateScriptRejectsBlankContent(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteExecutableFile(t, dir, "test-script.sh", "#!/bin/bash\necho old")
	svc := NewScriptService(dir)

	if _, _, err := svc.UpdateScript("test-script", " \n\t"); err == nil {
		t.Fatal("expected blank content to be rejected")
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

func TestNativeInvestigationUsesTypedRunner(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "active")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	testutil.WriteExecutableFile(t, dir, "cpu-analyzer.sh", "#!/bin/bash\nexit 99")
	writeExecutionPolicy(t, dir, `{"entries":{"cpu-analyzer":{"mode":"native","query":"cpu"}}}`)
	native := &nativeRunnerStub{data: []byte(`{"execution_mode":"native"}`)}
	svc := NewScriptService(dir)
	svc.SetNativeRunner(native)

	result, err := svc.ExecuteScript(context.Background(), "cpu-analyzer", "#!/bin/bash\nexit 1")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "completed" || result.ExecutionMode != "native" {
		t.Fatalf("native result = %#v", result)
	}
	if native.query != "cpu" {
		t.Fatalf("native query = %q, want cpu", native.query)
	}
}

func TestShellInvestigationReportsMissingTools(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "active")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	testutil.WriteExecutableFile(t, dir, "linux-only.sh", "#!/bin/bash\necho should-not-run")
	writeExecutionPolicy(t, dir, `{"entries":{"linux-only":{"mode":"shell","required_tools":["tool-that-is-not-installed-for-system-monitor-tests"]}}}`)
	svc := NewScriptService(dir)
	result, err := svc.ExecuteScript(context.Background(), "linux-only", "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "skipped" || result.SkipReason == "" {
		t.Fatalf("skip result = %#v", result)
	}
}
