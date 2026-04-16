package cliutil

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	checker.FingerprintFunc = func(spec FreshnessSpec) (string, error) {
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

func TestStaleCheckerUsesDeclaredInputsFromSourceContext(t *testing.T) {
	temp := t.TempDir()
	contextRoot := filepath.Join(temp, "resources", "demo")
	srcRoot := filepath.Join(contextRoot, "cli")
	if err := os.MkdirAll(filepath.Join(srcRoot, "internal"), 0o755); err != nil {
		t.Fatalf("mkdir src root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcRoot, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(contextRoot, "resource.json"), []byte("{\"name\":\"demo\"}\n"), 0o644); err != nil {
		t.Fatalf("write resource manifest: %v", err)
	}

	var gotRoot string
	var gotInputs []string
	checker := NewStaleChecker("demo", "old", "ts", srcRoot)
	checker.SourceContextPath = ".."
	checker.FreshnessInputs = []string{"cli/**", "resource.json"}
	checker.FingerprintFunc = func(spec FreshnessSpec) (string, error) {
		gotRoot = spec.ContextRoot
		gotInputs = append([]string(nil), spec.Inputs...)
		return "match", nil
	}
	checker.LookPathFunc = func(file string) (string, error) { return "/usr/bin/go", nil }

	if restarted := checker.CheckAndMaybeRebuild(); restarted {
		t.Fatalf("expected no restart when fingerprints match")
	}
	if gotRoot != contextRoot {
		t.Fatalf("fingerprint root = %q, want %q", gotRoot, contextRoot)
	}
	if strings.Join(gotInputs, ",") != "cli/**,resource.json" {
		t.Fatalf("fingerprint inputs = %v", gotInputs)
	}
}

func TestStaleCheckerPassesContextRelativeManifestToInstaller(t *testing.T) {
	temp := t.TempDir()
	repoRoot := temp
	installerPath := filepath.Join(repoRoot, "packages", "cli-core")
	if err := os.MkdirAll(installerPath, 0o755); err != nil {
		t.Fatalf("mkdir installer path: %v", err)
	}
	contextRoot := filepath.Join(repoRoot, "resources", "demo")
	srcRoot := filepath.Join(contextRoot, "cli")
	if err := os.MkdirAll(srcRoot, 0o755); err != nil {
		t.Fatalf("mkdir src root: %v", err)
	}

	var ranArgs []string
	checker := NewStaleChecker("demo", "old", "ts", srcRoot)
	checker.SourceContextPath = ".."
	checker.ManifestSourcePath = "resource.json"
	checker.FingerprintFunc = func(spec FreshnessSpec) (string, error) { return "new", nil }
	checker.LookPathFunc = func(file string) (string, error) { return "/usr/bin/go", nil }
	checker.CommandRunner = func(cmd *exec.Cmd) error {
		ranArgs = append([]string(nil), cmd.Args...)
		return nil
	}
	checker.Reexec = func(executable string, args []string) error { return nil }

	if restarted := checker.CheckAndMaybeRebuild(); !restarted {
		t.Fatalf("expected restart to be triggered")
	}
	args := strings.Join(ranArgs, " ")
	if !strings.Contains(args, "--manifest") || !strings.Contains(args, filepath.Join(contextRoot, "resource.json")) {
		t.Fatalf("installer args missing context-relative manifest path: %v", ranArgs)
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
		FingerprintFunc: func(spec FreshnessSpec) (string, error) {
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
	checker.FingerprintFunc = func(spec FreshnessSpec) (string, error) {
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

func TestComputeFreshnessFingerprintUsesDeclaredInputs(t *testing.T) {
	root := t.TempDir()
	contextRoot := filepath.Join(root, "scenario")
	sourceRoot := filepath.Join(contextRoot, "cli")
	if err := os.MkdirAll(sourceRoot, 0o755); err != nil {
		t.Fatalf("mkdir source root: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(contextRoot, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir manifest dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(contextRoot, ".vrooli", "service.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	spec := FreshnessSpec{
		SourceRoot:  sourceRoot,
		ContextRoot: contextRoot,
		Inputs:      []string{"cli/**", ".vrooli/service.json"},
	}
	first, err := ComputeFreshnessFingerprint(spec)
	if err != nil {
		t.Fatalf("first fingerprint: %v", err)
	}

	if err := os.WriteFile(filepath.Join(contextRoot, "README.md"), []byte("ignored\n"), 0o644); err != nil {
		t.Fatalf("write unrelated file: %v", err)
	}
	second, err := ComputeFreshnessFingerprint(spec)
	if err != nil {
		t.Fatalf("second fingerprint: %v", err)
	}
	if first != second {
		t.Fatalf("unrelated file changed declared-input fingerprint: %q vs %q", first, second)
	}

	if err := os.WriteFile(filepath.Join(contextRoot, ".vrooli", "service.json"), []byte("{\"name\":\"demo\"}\n"), 0o644); err != nil {
		t.Fatalf("mutate manifest: %v", err)
	}
	third, err := ComputeFreshnessFingerprint(spec)
	if err != nil {
		t.Fatalf("third fingerprint: %v", err)
	}
	if third == second {
		t.Fatalf("declared file change should affect fingerprint")
	}
}
