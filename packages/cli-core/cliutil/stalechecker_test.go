package cliutil

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFindRepositoryRoot(t *testing.T) {
	temp := t.TempDir()
	target := filepath.Join(temp, "packages", "cli-core")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	start := filepath.Join(temp, "scenarios", "example")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatalf("mkdir scenarios: %v", err)
	}

	root, ok := findRepositoryRoot(start, filepath.Join("packages", "cli-core"))
	if !ok || root != temp {
		t.Fatalf("expected repo root %s, got %s (ok=%v)", temp, root, ok)
	}
}

func TestStaleCheckerSkipsWhenFingerprintUnknown(t *testing.T) {
	checker := &StaleChecker{
		BuildFingerprint: "unknown",
		BuildSourceRoot:  t.TempDir(),
	}
	if restarted := checker.CheckAndMaybeRebuild(); restarted {
		t.Fatalf("expected no restart when fingerprint unknown")
	}
}

func TestStaleCheckerRebuildsAndReexecs(t *testing.T) {
	temp := t.TempDir()
	repoRoot := temp
	installerPath := filepath.Join(repoRoot, "packages", "cli-core")
	if err := os.MkdirAll(installerPath, 0o755); err != nil {
		t.Fatalf("mkdir installer path: %v", err)
	}
	srcRoot := filepath.Join(repoRoot, "scenarios", "demo")
	if err := os.MkdirAll(srcRoot, 0o755); err != nil {
		t.Fatalf("mkdir src root: %v", err)
	}

	ranCommand := false
	restarted := false
	checker := NewStaleChecker("demo", "old", "ts", srcRoot, "ENV")
	checker.ReexecArgs = []string{"foo"}
	checker.FingerprintFunc = func(root string, skip ...string) (string, error) {
		return "new", nil
	}
	checker.LookPathFunc = func(file string) (string, error) {
		if file == "go" {
			return "/usr/bin/go", nil
		}
		return "", errors.New("not found")
	}
	checker.CommandRunner = func(cmd *exec.Cmd) error {
		ranCommand = true
		return nil
	}
	checker.Reexec = func(executable string, args []string) error {
		restarted = true
		return nil
	}

	if restartedFlag := checker.CheckAndMaybeRebuild(); !restartedFlag {
		t.Fatalf("expected restart to be triggered")
	}
	if !ranCommand {
		t.Fatalf("expected installer command to run")
	}
	if !restarted {
		t.Fatalf("expected reexec to be called")
	}
}

func TestStaleCheckerNoRebuildWhenFingerprintMatches(t *testing.T) {
	srcRoot := filepath.Join(t.TempDir(), "scenarios", "demo")
	if err := os.MkdirAll(srcRoot, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	checker := &StaleChecker{
		BuildFingerprint: "match",
		BuildSourceRoot:  srcRoot,
		FingerprintFunc: func(root string, skip ...string) (string, error) {
			return "match", nil
		},
		LookPathFunc: func(file string) (string, error) {
			return "/usr/bin/go", nil
		},
		CommandRunner: func(cmd *exec.Cmd) error {
			t.Fatalf("command should not run")
			return nil
		},
		Reexec: func(executable string, args []string) error {
			t.Fatalf("reexec should not run")
			return nil
		},
	}

	if restarted := checker.CheckAndMaybeRebuild(); restarted {
		t.Fatalf("expected no restart when fingerprints match")
	}
}

func TestStaleCheckerRebuildsOutsideScenarioPath(t *testing.T) {
	temp := t.TempDir()
	repoRoot := temp
	installerPath := filepath.Join(repoRoot, "packages", "cli-core")
	if err := os.MkdirAll(installerPath, 0o755); err != nil {
		t.Fatalf("mkdir installer path: %v", err)
	}
	srcRoot := filepath.Join(repoRoot, "packages", "custom-cli")
	if err := os.MkdirAll(srcRoot, 0o755); err != nil {
		t.Fatalf("mkdir src root: %v", err)
	}

	ranCommand := false
	restarted := false
	checker := NewStaleChecker("custom-cli", "old", "ts", srcRoot)
	checker.FingerprintFunc = func(root string, skip ...string) (string, error) {
		return "new", nil
	}
	checker.LookPathFunc = func(file string) (string, error) {
		return "/usr/bin/go", nil
	}
	checker.CommandRunner = func(cmd *exec.Cmd) error {
		ranCommand = true
		return nil
	}
	checker.Reexec = func(executable string, args []string) error {
		restarted = true
		return nil
	}

	if restartedFlag := checker.CheckAndMaybeRebuild(); !restartedFlag {
		t.Fatalf("expected restart to be triggered for packages-based CLI")
	}
	if !ranCommand {
		t.Fatalf("expected installer command to run")
	}
	if !restarted {
		t.Fatalf("expected reexec to be called")
	}
}
