package scenarios

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestTestingRunnerExecutesPreferredCommand(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "run.sh")
	writeExecutable(t, script, "echo ok")

	caps := TestingCapabilities{
		Phased:    true,
		HasTests:  true,
		Preferred: "phased",
		Commands: []TestingCommand{
			{Type: "phased", Command: []string{script}, WorkingDir: dir},
		},
	}

	runner := TestingRunner{Timeout: time.Second}
	if _, err := runner.Run(context.Background(), caps, ""); err != nil {
		t.Fatalf("expected command to succeed, got %v", err)
	}
}

func TestTestingRunnerFallsBackToLifecycle(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "run.sh")
	writeExecutable(t, script, "exit 0")

	caps := TestingCapabilities{
		Lifecycle: true,
		Commands: []TestingCommand{
			{Type: "lifecycle", Command: []string{script}},
		},
	}

	runner := TestingRunner{Timeout: time.Second}
	if _, err := runner.Run(context.Background(), caps, ""); err != nil {
		t.Fatalf("expected fallback command to succeed, got %v", err)
	}
}

func TestTestingRunnerErrorsWhenMissingCommands(t *testing.T) {
	runner := TestingRunner{}
	if _, err := runner.Run(context.Background(), TestingCapabilities{}, ""); err == nil {
		t.Fatal("expected error when no commands available")
	}
}

func TestTestingRunnerRespectsTimeout(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "sleep.sh")
	writeExecutable(t, script, "sleep 2")

	caps := TestingCapabilities{
		Phased: true,
		Commands: []TestingCommand{
			{Type: "phased", Command: []string{script}},
		},
	}
	runner := TestingRunner{Timeout: 500 * time.Millisecond}
	_, err := runner.Run(context.Background(), caps, "")
	if err == nil || err.Error() == "" || err.Error() == "exit status 1" {
		t.Fatalf("expected timeout error, got %v", err)
	}
}

func writeExecutable(t *testing.T, path string, script string) {
	t.Helper()
	content := "#!/usr/bin/env bash\nset -euo pipefail\n" + script + "\n"
	mode := os.FileMode(0o755)
	if runtime.GOOS == "windows" {
		mode = 0o644
		content = "@echo off\r\n" + script + "\r\n"
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("failed to write script: %v", err)
	}
}
