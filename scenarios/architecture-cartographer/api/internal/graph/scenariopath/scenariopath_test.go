package scenariopath_test

import (
	"os"
	"path/filepath"
	"testing"

	"architecture-cartographer/internal/graph/scenariopath"
)

func writeMarker(t *testing.T, dir, marker string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	if err := os.WriteFile(filepath.Join(dir, marker), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write marker: %v", err)
	}
}

func tsCandidates() []scenariopath.Candidate {
	return []scenariopath.Candidate{
		{Subdir: "ui", Marker: "tsconfig.json"},
		{Subdir: ".", Marker: "tsconfig.json"},
	}
}

func TestResolve_PrefersUIThenRoot(t *testing.T) {
	repoRoot := t.TempDir()
	scenarioRoot := filepath.Join(repoRoot, "scenarios", "demo")
	writeMarker(t, filepath.Join(scenarioRoot, "ui"), "tsconfig.json")

	r := scenariopath.NewResolver(repoRoot, tsCandidates())
	path, found, err := r.Resolve("demo")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !found {
		t.Fatal("expected found=true when ui/tsconfig.json exists")
	}
	if want := filepath.Join(scenarioRoot, "ui"); path != want {
		t.Fatalf("path=%q want %q", path, want)
	}
}

func TestResolve_FallsBackToRoot(t *testing.T) {
	repoRoot := t.TempDir()
	scenarioRoot := filepath.Join(repoRoot, "scenarios", "demo")
	writeMarker(t, scenarioRoot, "tsconfig.json") // root-level TS project, no ui/

	r := scenariopath.NewResolver(repoRoot, tsCandidates())
	path, found, err := r.Resolve("demo")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !found {
		t.Fatal("expected found=true for root-level tsconfig.json")
	}
	if path != filepath.Clean(scenarioRoot) {
		t.Fatalf("path=%q want %q", path, scenarioRoot)
	}
}

func TestResolve_NoProjectFound(t *testing.T) {
	repoRoot := t.TempDir()
	// Create the scenario dir but no tsconfig.json anywhere.
	if err := os.MkdirAll(filepath.Join(repoRoot, "scenarios", "demo", "api"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	r := scenariopath.NewResolver(repoRoot, tsCandidates())
	_, found, err := r.Resolve("demo")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if found {
		t.Fatal("expected found=false when no candidate marker exists")
	}
}

func TestResolve_RejectsEmptyInputs(t *testing.T) {
	if _, _, err := scenariopath.NewResolver("/repo", tsCandidates()).Resolve("  "); err == nil {
		t.Fatal("expected error for empty scenario name")
	}
	if _, _, err := scenariopath.NewResolver("", tsCandidates()).Resolve("demo"); err == nil {
		t.Fatal("expected error for empty repo root")
	}
}
