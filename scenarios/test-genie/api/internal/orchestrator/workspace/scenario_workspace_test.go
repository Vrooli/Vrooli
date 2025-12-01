package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewScenarioWorkspace(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "demo")
	required := []string{
		filepath.Join("test", "phases"),
	}
	for _, rel := range required {
		if err := os.MkdirAll(filepath.Join(scenarioDir, rel), 0o755); err != nil {
			t.Fatalf("failed to create %s: %v", rel, err)
		}
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "test", "run-tests.sh"), []byte("echo ok"), 0o644); err != nil {
		t.Fatalf("failed to seed runner: %v", err)
	}

	workspace, err := New(root, "demo")
	if err != nil {
		t.Fatalf("expected workspace, got error: %v", err)
	}
	if workspace.ScenarioDir != scenarioDir {
		t.Fatalf("unexpected scenario dir %s", workspace.ScenarioDir)
	}
	if workspace.TestDir != filepath.Join(scenarioDir, "test") {
		t.Fatalf("unexpected test dir %s", workspace.TestDir)
	}
	if workspace.PhaseDir != filepath.Join(scenarioDir, "test", "phases") {
		t.Fatalf("unexpected phase dir %s", workspace.PhaseDir)
	}
	if workspace.AppRoot == "" {
		t.Fatalf("expected app root to be set")
	}

	env := workspace.Environment()
	if env.ScenarioName != "demo" || env.ScenarioDir == "" || env.TestDir == "" {
		t.Fatalf("environment missing data: %#v", env)
	}

	artifactDir, err := workspace.EnsureArtifactDir()
	if err != nil {
		t.Fatalf("artifact dir error: %v", err)
	}
	if info, err := os.Stat(artifactDir); err != nil || !info.IsDir() {
		t.Fatalf("artifact dir missing: %v", err)
	}
}

func TestNewScenarioWorkspaceValidatesNames(t *testing.T) {
	root := t.TempDir()
	if _, err := New(root, ""); err == nil {
		t.Fatalf("expected error for empty scenario")
	}
	if _, err := New(root, "invalid name"); err == nil {
		t.Fatalf("expected error for invalid characters")
	}
}
