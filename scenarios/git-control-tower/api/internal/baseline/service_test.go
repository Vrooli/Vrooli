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
		Snaps:      newTestSnapshotStore(t),
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
