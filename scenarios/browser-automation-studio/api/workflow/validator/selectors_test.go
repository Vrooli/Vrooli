package validator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverSelectorRootFromScenarioSubdir(t *testing.T) {
	dir := t.TempDir()
	scenarioDir := filepath.Join(dir, "scenarios", "browser-automation-studio")
	if err := os.MkdirAll(filepath.Join(scenarioDir, "api"), 0o755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioDir, "ui", "src"), 0o755); err != nil {
		t.Fatalf("failed to create ui/src dir: %v", err)
	}

	root := discoverSelectorRootFrom(filepath.Join(scenarioDir, "api"))
	if root != filepath.Join(scenarioDir, "ui", "src") {
		t.Fatalf("expected selector root %s, got %s", filepath.Join(scenarioDir, "ui", "src"), root)
	}
}

func TestDiscoverSelectorRootFromRepoRoot(t *testing.T) {
	dir := t.TempDir()
	scenarioDir := filepath.Join(dir, "scenarios", "browser-automation-studio")
	if err := os.MkdirAll(filepath.Join(scenarioDir, "ui", "src"), 0o755); err != nil {
		t.Fatalf("failed to create ui/src dir: %v", err)
	}

	root := discoverSelectorRootFrom(dir)
	if root != filepath.Join(scenarioDir, "ui", "src") {
		t.Fatalf("expected selector root %s, got %s", filepath.Join(scenarioDir, "ui", "src"), root)
	}
}

func TestDiscoverSelectorRootReturnsEmptyWhenMissing(t *testing.T) {
	dir := t.TempDir()
	root := discoverSelectorRootFrom(dir)
	if root != "" {
		t.Fatalf("expected empty selector root, got %s", root)
	}
}
