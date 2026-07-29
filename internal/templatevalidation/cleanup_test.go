package templatevalidation

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A scenario's proto footprint spans six outputs across two name forms. Missing
// any one of them leaves residue in the SHARED proto tree that outlives the
// scenario, so this pins the whole set rather than spot-checking.
func TestRelocationArtifactPathsCoversEveryCodegenOutput(t *testing.T) {
	protoRoot := filepath.Join("repo", "packages", "proto")
	target := filepath.Join(protoRoot, "schemas", "throwaway-probe")

	got := RelocationArtifactPaths([]string{target})
	gotSet := make(map[string]struct{}, len(got))
	for _, p := range got {
		gotSet[p] = struct{}{}
	}

	gen := filepath.Join(protoRoot, "gen")
	want := []string{
		target,
		filepath.Join(gen, "go", "throwaway-probe"),
		filepath.Join(gen, "typescript", "throwaway-probe"),
		filepath.Join(gen, "typescript", "js", "throwaway-probe"),
		filepath.Join(gen, "python", "throwaway-probe"),
		// protoc-gen-python rewrites hyphens: module names disallow "-".
		filepath.Join(gen, "python", "throwaway_probe"),
		// A file beside the gen trees, not a directory inside one.
		filepath.Join(gen, "manifests", "throwaway-probe.lock.json"),
	}
	for _, w := range want {
		if _, ok := gotSet[w]; !ok {
			t.Errorf("missing codegen path %q\ngot: %v", w, got)
		}
	}
}

// A target that is not under schemas/ carries no derivable codegen footprint;
// inventing gen paths for it would delete unrelated directories.
func TestRelocationArtifactPathsIgnoresNonSchemaTargets(t *testing.T) {
	target := filepath.Join("repo", "scenarios", "demo")
	got := RelocationArtifactPaths([]string{target})
	if len(got) != 1 || got[0] != filepath.Clean(target) {
		t.Fatalf("expected only the target itself, got %v", got)
	}
}

func TestPlanCleanupSkipsRetainedByDefault(t *testing.T) {
	repoRoot := t.TempDir()
	searchRoot := t.TempDir()
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	oldRun := writeCleanupTestRun(t, repoRoot, searchRoot, "old", false, now.Add(-48*time.Hour))
	retainedRun := writeCleanupTestRun(t, repoRoot, searchRoot, "retained", true, now.Add(-48*time.Hour))
	_ = retainedRun
	youngRun := writeCleanupTestRun(t, repoRoot, searchRoot, "young", false, now.Add(-time.Hour))
	_ = youngRun

	plan := PlanCleanup(CleanupOptions{RepoRoot: repoRoot, SearchRoots: []string{searchRoot}, Now: now, OlderThan: 24 * time.Hour})
	if len(plan.Eligible) != 1 || plan.Eligible[0].Marker.RunID != oldRun.RunID {
		t.Fatalf("eligible = %#v, want only old run", plan.Eligible)
	}
	if len(plan.Skipped) != 2 {
		t.Fatalf("skipped = %#v, want retained and young", plan.Skipped)
	}
}

func TestPlanCleanupIncludesRetainedExplicitly(t *testing.T) {
	repoRoot := t.TempDir()
	searchRoot := t.TempDir()
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	retainedRun := writeCleanupTestRun(t, repoRoot, searchRoot, "retained", true, now.Add(-48*time.Hour))

	plan := PlanCleanup(CleanupOptions{RepoRoot: repoRoot, SearchRoots: []string{searchRoot}, Now: now, OlderThan: 24 * time.Hour, IncludeRetained: true})
	if len(plan.Eligible) != 1 || plan.Eligible[0].Marker.RunID != retainedRun.RunID {
		t.Fatalf("eligible = %#v, want retained run", plan.Eligible)
	}
}

func TestPlanCleanupRunIDSelectsRetainedRun(t *testing.T) {
	repoRoot := t.TempDir()
	searchRoot := t.TempDir()
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	retainedRun := writeCleanupTestRun(t, repoRoot, searchRoot, "retained", true, now)

	plan := PlanCleanup(CleanupOptions{RepoRoot: repoRoot, SearchRoots: []string{searchRoot}, Now: now, RunID: retainedRun.RunID})
	if len(plan.Eligible) != 1 || plan.Eligible[0].Marker.RunID != retainedRun.RunID {
		t.Fatalf("eligible = %#v, want retained run by id", plan.Eligible)
	}
}

func TestExecuteCleanupRemovesMarkedRunAndArtifacts(t *testing.T) {
	repoRoot := t.TempDir()
	searchRoot := t.TempDir()
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	marker := writeCleanupTestRun(t, repoRoot, searchRoot, "old", false, now.Add(-48*time.Hour))
	artifact := marker.RelocationArtifacts[0]
	if err := os.MkdirAll(artifact, 0o755); err != nil {
		t.Fatalf("mkdir artifact: %v", err)
	}

	result := ExecuteCleanup(PlanCleanup(CleanupOptions{RepoRoot: repoRoot, SearchRoots: []string{searchRoot}, Now: now, OlderThan: 24 * time.Hour}))
	if len(result.Failures) != 0 {
		t.Fatalf("failures = %#v", result.Failures)
	}
	if len(result.Removed) != 1 {
		t.Fatalf("removed = %#v", result.Removed)
	}
	if _, err := os.Stat(marker.TempRoot); !os.IsNotExist(err) {
		t.Fatalf("temp root exists after cleanup: %v", err)
	}
	if _, err := os.Stat(artifact); !os.IsNotExist(err) {
		t.Fatalf("artifact exists after cleanup: %v", err)
	}
	if !result.NeedsProtoGenerate {
		t.Fatalf("NeedsProtoGenerate = false, want true")
	}
}

func TestPlanCleanupIgnoresUnmarkedGlobMatch(t *testing.T) {
	repoRoot := t.TempDir()
	searchRoot := t.TempDir()
	unmarked := filepath.Join(searchRoot, "vrooli-template-deep-unmarked")
	if err := os.MkdirAll(unmarked, 0o755); err != nil {
		t.Fatalf("mkdir unmarked: %v", err)
	}

	result := ExecuteCleanup(PlanCleanup(CleanupOptions{RepoRoot: repoRoot, SearchRoots: []string{searchRoot}, Now: time.Now().UTC(), OlderThan: 24 * time.Hour}))
	if len(result.Eligible) != 0 || len(result.Removed) != 0 {
		t.Fatalf("result = %#v", result)
	}
	if _, err := os.Stat(unmarked); err != nil {
		t.Fatalf("unmarked directory should remain: %v", err)
	}
}

func writeCleanupTestRun(t *testing.T, repoRoot, searchRoot, suffix string, retained bool, createdAt time.Time) RunMarker {
	t.Helper()
	scenarioID := "template-validation-react-vite-deep"
	tempRoot := filepath.Join(searchRoot, "vrooli-template-deep-"+suffix)
	marker := RunMarker{
		Version:      MarkerVersion,
		RunID:        scenarioID + "-" + suffix,
		RepoRoot:     repoRoot,
		Template:     "react-vite",
		ScenarioID:   scenarioID,
		ScenarioPath: filepath.Join(tempRoot, "scenarios", scenarioID),
		TempRoot:     tempRoot,
		CreatedAt:    createdAt,
		Retained:     retained,
		CreatorPID:   os.Getpid(),
		Completed:    true,
		RelocationArtifacts: []string{
			filepath.Join(repoRoot, "packages", "proto", "schemas", scenarioID),
		},
	}
	if err := os.MkdirAll(marker.ScenarioPath, 0o755); err != nil {
		t.Fatalf("mkdir scenario path: %v", err)
	}
	if err := WriteMarker(marker); err != nil {
		t.Fatalf("WriteMarker: %v", err)
	}
	return marker
}
