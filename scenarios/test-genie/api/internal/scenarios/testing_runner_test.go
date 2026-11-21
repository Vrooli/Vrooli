package scenarios

import (
	"context"
	"fmt"
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

func TestTestingRunnerFallsBackToLegacy(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "run.sh")
	writeExecutable(t, script, "exit 0")

	caps := TestingCapabilities{
		Legacy: true,
		Commands: []TestingCommand{
			{Type: "legacy", Command: []string{script}},
		},
	}

	runner := TestingRunner{Timeout: time.Second}
	if _, err := runner.Run(context.Background(), caps, ""); err != nil {
		t.Fatalf("expected fallback command to succeed, got %v", err)
	}
}

func TestTestingRunnerUsesAndReapsGovernedGoWorkDir(t *testing.T) {
	runtimeHome := t.TempDir()
	t.Setenv("VROOLI_HOME", runtimeHome)
	dir := t.TempDir()
	script := filepath.Join(dir, "run.sh")
	writeExecutable(t, script, "test -n \"$GOTMPDIR\"; test -d \"$GOTMPDIR\"")
	caps := TestingCapabilities{Phased: true, Commands: []TestingCommand{{Type: "phased", Command: []string{script}, WorkingDir: dir}}}
	if _, err := (TestingRunner{Timeout: time.Second}).Run(context.Background(), caps, ""); err != nil {
		t.Fatalf("expected command to succeed, got %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(runtimeHome, "tmp", "go-work"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("reaped Go work directory still has entries: %v", entries)
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

func TestTestingRunnerRecordsPlatformSkipsAndEnforcesBudget(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	platform := normalizedPlatform(runtime.GOOS)
	budget := fmt.Sprintf(`{"budgets":{"%s":1}}`, platform)
	if err := os.WriteFile(filepath.Join(dir, ".vrooli", "skip-budgets.json"), []byte(budget), 0o644); err != nil {
		t.Fatalf("write budget: %v", err)
	}
	script := filepath.Join(dir, "run.sh")
	writeExecutable(t, script, `printf '{"kind":"platform_skip","platform":"`+platform+`"}\n' >> "$VROOLI_SKIP_RECORD_PATH"`)
	caps := TestingCapabilities{Phased: true, HasTests: true, Commands: []TestingCommand{{Type: "phased", Command: []string{script}, WorkingDir: dir}}}
	runner := TestingRunner{Timeout: time.Second}
	result, err := runner.Run(context.Background(), caps, "")
	if err != nil {
		t.Fatalf("expected one skip at budget to pass, got %v", err)
	}
	if result.SkipSummary.Skipped != 1 || result.SkipSummary.Budget != 1 || !result.SkipSummary.WithinBudget {
		t.Fatalf("skip summary = %#v", result.SkipSummary)
	}

	if err := os.WriteFile(filepath.Join(dir, ".vrooli", "skip-budgets.json"), []byte(fmt.Sprintf(`{"budgets":{"%s":0}}`, platform)), 0o644); err != nil {
		t.Fatalf("lower budget: %v", err)
	}
	if _, err := runner.Run(context.Background(), caps, ""); err == nil {
		t.Fatal("expected over-budget run to fail")
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
