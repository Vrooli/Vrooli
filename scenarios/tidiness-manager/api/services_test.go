package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewScenarioLocator_FailsWithoutRepoContractContext(t *testing.T) {
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("VROOLI_SOURCE_ROOT", "")

	nonRepo := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(nonRepo); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	if _, err := NewScenarioLocator(time.Minute); err == nil {
		t.Fatal("expected NewScenarioLocator to fail outside repo-contract context")
	}
}

func TestScenarioLocatorResolveRequestedPath_AllowsAbsoluteOverride(t *testing.T) {
	repoRoot := t.TempDir()
	writeRepoContractFixture(t, repoRoot)

	locator := &ScenarioLocator{
		repoRoot:     repoRoot,
		scenariosDir: filepath.Join(repoRoot, "scenarios"),
		cacheTTL:     time.Minute,
	}

	overrideDir := filepath.Join(t.TempDir(), "custom-sandbox-scenario")
	if err := os.MkdirAll(overrideDir, 0o755); err != nil {
		t.Fatalf("mkdir override dir: %v", err)
	}

	absPath, scenarioName, err := locator.ResolveRequestedPath(overrideDir)
	if err != nil {
		t.Fatalf("ResolveRequestedPath returned error: %v", err)
	}
	if absPath != overrideDir {
		t.Fatalf("ResolveRequestedPath absPath = %q, want %q", absPath, overrideDir)
	}
	if scenarioName != filepath.Base(overrideDir) {
		t.Fatalf("ResolveRequestedPath scenarioName = %q, want %q", scenarioName, filepath.Base(overrideDir))
	}
}
