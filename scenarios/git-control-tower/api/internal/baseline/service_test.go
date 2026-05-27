package baseline

import (
	"context"
	"errors"
	"testing"

	"git-control-tower/internal/git"
)

func newTestService(t *testing.T, adapters map[string]SurfaceAdapter, runs RunsClient, gitState git.State) (*Service, *Storage) {
	t.Helper()
	st := newTestStorage(t)
	svc := NewService(Deps{
		Storage:    st,
		Adapters:   adapters,
		Runs:       runs,
		CaptureGit: fixedGit(gitState),
	})
	return svc, st
}

func TestServiceCreateCapturesSurfacesAndWarnsDirty(t *testing.T) {
	exec := &fakeExecutor{result: ExecResult{Success: true, Phases: []PhaseStatus{{"unit", "passed"}}}}
	runs := &fakeRuns{}
	adapters := map[string]SurfaceAdapter{
		SurfaceTests: NewTestsAdapter(exec, runs),
	}
	svc, _ := newTestService(t, adapters, runs, git.State{Branch: "agi", Sha: "abc", Dirty: true, DirtySummary: "3 modified"})

	res, err := svc.Create(context.Background(), CreateRequest{
		RepoID: 1, RepoDir: "/repo", Scenario: "foo", Name: "plan-1",
		Include: []string{SurfaceTests}, Fast: true, Capture: true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if res.Manifest.Branch != "agi" {
		t.Fatalf("branch = %q", res.Manifest.Branch)
	}
	if _, ok := res.Manifest.Surfaces[SurfaceTests]; !ok {
		t.Fatalf("tests surface not captured: %+v", res.Manifest.Surfaces)
	}
	if res.DirtyWarning == "" {
		t.Fatal("expected dirty-tree warning")
	}
	if len(runs.pins) != 1 {
		t.Fatalf("expected pin, got %+v", runs.pins)
	}
}

func TestServiceCreateDuplicate(t *testing.T) {
	svc, _ := newTestService(t, map[string]SurfaceAdapter{}, &fakeRuns{}, git.State{Branch: "agi"})
	req := CreateRequest{RepoID: 1, Scenario: "foo", Name: "p", Capture: false}
	if _, err := svc.Create(context.Background(), req); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := svc.Create(context.Background(), req); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestServiceDiffAggregatesVerdict(t *testing.T) {
	exec := &fakeExecutor{result: ExecResult{}}
	runs := &fakeRuns{compare: CompareResult{
		Verdict: "regression",
		Phases:  []PhaseDiff{{Phase: "unit", Verdict: "regression", Regressions: []string{"TestX"}}},
	}}
	adapters := map[string]SurfaceAdapter{SurfaceTests: NewTestsAdapter(exec, runs)}
	svc, _ := newTestService(t, adapters, runs, git.State{Branch: "agi", Sha: "abc"})

	if _, err := svc.Create(context.Background(), CreateRequest{
		RepoID: 1, Scenario: "foo", Name: "p", Include: []string{SurfaceTests}, Capture: true,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	res, err := svc.Diff(context.Background(), 1, "/repo", "foo", "agi", "p", "")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if res.Verdict != VerdictRegression {
		t.Fatalf("expected overall regression, got %s", res.Verdict)
	}
}

func TestServicePartialBaselineDoesNotMasqueradeAsComplete(t *testing.T) {
	// Request tests + workflows, but only register a tests adapter. workflows
	// must be persisted as skipped and reported not-comparable on diff so the
	// baseline can't read as "workflows clean".
	exec := &fakeExecutor{result: ExecResult{Success: true, Phases: []PhaseStatus{{"unit", "passed"}}}}
	runs := &fakeRuns{}
	adapters := map[string]SurfaceAdapter{SurfaceTests: NewTestsAdapter(exec, runs)}
	svc, _ := newTestService(t, adapters, runs, git.State{Branch: "agi", Sha: "abc"})

	res, err := svc.Create(context.Background(), CreateRequest{
		RepoID: 1, Scenario: "foo", Name: "p",
		Include: []string{SurfaceTests, SurfaceWorkflows}, Capture: true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, ok := res.Manifest.Skipped[SurfaceWorkflows]; !ok {
		t.Fatalf("workflows should be persisted as skipped, got skipped=%v", res.Manifest.Skipped)
	}

	// Reload to confirm the skip survived persistence (not just the in-memory result).
	reloaded, err := svc.Get(context.Background(), 1, "foo", "agi", "p")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if _, ok := reloaded.Skipped[SurfaceWorkflows]; !ok {
		t.Fatalf("persisted manifest lost the skipped surface: %+v", reloaded)
	}

	diff, err := svc.Diff(context.Background(), 1, "/repo", "foo", "agi", "p", "")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	var sawWorkflowsNotComparable bool
	for _, d := range diff.Surfaces {
		if d.SurfaceID == SurfaceWorkflows {
			if d.Verdict != VerdictNotComparable {
				t.Fatalf("uncaptured workflows should be not-comparable, got %s", d.Verdict)
			}
			sawWorkflowsNotComparable = true
		}
	}
	if !sawWorkflowsNotComparable {
		t.Fatal("diff omitted the uncaptured workflows surface — partial baseline masquerading as complete")
	}
	if diff.Verdict != VerdictNotComparable {
		t.Fatalf("overall verdict should be not-comparable when a surface was never captured, got %s", diff.Verdict)
	}
}

func TestServiceDeleteUnpinsRuns(t *testing.T) {
	exec := &fakeExecutor{result: ExecResult{}}
	runs := &fakeRuns{}
	adapters := map[string]SurfaceAdapter{SurfaceTests: NewTestsAdapter(exec, runs)}
	svc, _ := newTestService(t, adapters, runs, git.State{Branch: "agi", Sha: "abc"})

	if _, err := svc.Create(context.Background(), CreateRequest{
		RepoID: 1, Scenario: "foo", Name: "p", Include: []string{SurfaceTests}, Capture: true,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.Delete(context.Background(), 1, "foo", "agi", "p"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(runs.unpins) != 1 {
		t.Fatalf("expected one unpin on delete, got %+v", runs.unpins)
	}
	if _, err := svc.Get(context.Background(), 1, "foo", "agi", "p"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected baseline gone, got %v", err)
	}
}

// Deleting a baseline must release its exclusively-owned visuals snapshot so
// pinned screenshots don't leak (Plan C Phase 4).
func TestServiceDeleteReleasesVisualsSnapshot(t *testing.T) {
	runs := &fakeRuns{}
	vis := &fakeVisual{capture: []VisualSnapshot{{SnapshotID: "snap-77", ScreenshotCount: 2, Pages: []string{"/"}}}}
	adapters := map[string]SurfaceAdapter{SurfaceVisuals: NewVisualsAdapter(vis)}
	svc, _ := newTestService(t, adapters, runs, git.State{Branch: "agi", Sha: "abc"})

	if _, err := svc.Create(context.Background(), CreateRequest{
		RepoID: 1, Scenario: "foo", Name: "p", Include: []string{SurfaceVisuals}, Capture: true,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.Delete(context.Background(), 1, "foo", "agi", "p"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(vis.deleted) != 1 || vis.deleted[0] != "snap-77" {
		t.Fatalf("expected pinned visuals snapshot snap-77 released, got %+v", vis.deleted)
	}
}
