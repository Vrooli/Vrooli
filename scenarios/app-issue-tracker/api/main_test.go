package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetVrooliRootCanonicalizesContractDescendantOverride(t *testing.T) {
	root := newMainContractFixtureRepo(t)
	nested := filepath.Join(root, "scenarios", "app-issue-tracker", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	t.Setenv("APP_ISSUE_TRACKER_ROOT", nested)
	t.Setenv("VROOLI_ROOT", "")

	if got := getVrooliRoot(); got != root {
		t.Fatalf("getVrooliRoot() = %q, want %q", got, root)
	}
}

func TestLoadConfigUsesContractScenarioPath(t *testing.T) {
	root := newMainContractFixtureRepo(t)
	nested := filepath.Join(root, "scenarios", "app-issue-tracker", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	t.Setenv("APP_ISSUE_TRACKER_ROOT", nested)
	t.Setenv("VROOLI_ROOT", "")

	cfg := loadConfig()
	wantRoot := filepath.Join(root, "scenarios", "app-issue-tracker")
	if cfg.VrooliRoot != root {
		t.Fatalf("config.VrooliRoot = %q, want %q", cfg.VrooliRoot, root)
	}
	if cfg.ScenarioRoot != wantRoot {
		t.Fatalf("config.ScenarioRoot = %q, want %q", cfg.ScenarioRoot, wantRoot)
	}
}

func newMainContractFixtureRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	repoRoot := appIssueTrackerRepoRoot(t)
	contractData, err := os.ReadFile(filepath.Join(repoRoot, ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("read repo contract: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), contractData, 0o644); err != nil {
		t.Fatalf("write repo contract: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/app-issue-tracker-test\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for _, dir := range []string{"scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	return root
}

func appIssueTrackerRepoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}
