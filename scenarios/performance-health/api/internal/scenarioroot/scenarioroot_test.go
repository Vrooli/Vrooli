package scenarioroot

import (
	"path/filepath"
	"testing"
)

func TestResolveExplicitPathWins(t *testing.T) {
	dir := t.TempDir()
	got, err := Resolve("", "ignored-scenario", dir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	abs, _ := filepath.Abs(dir)
	if got != abs {
		t.Fatalf("expected explicit path %q, got %q", abs, got)
	}
}

func TestResolveRequiresScenarioOrPath(t *testing.T) {
	if _, err := Resolve("/some/repo", "", ""); err == nil {
		t.Fatal("expected error when neither scenario nor path is set")
	}
}
