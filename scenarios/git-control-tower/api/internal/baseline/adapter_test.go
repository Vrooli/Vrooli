package baseline

import (
	"context"
	"testing"
)

func TestTestGenieAdapterCapturePins(t *testing.T) {
	exec := &fakeExecutor{result: ExecResult{Success: true, Phases: []PhaseStatus{{"playbooks", "passed"}}}}
	runs := &fakeRuns{}
	a := NewWorkflowsAdapter(exec, runs)

	ptr, err := a.Capture(context.Background(), Target{Scenario: "foo"}, CaptureOptions{Fast: true, PinnedBy: "gct:baseline:p1"})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if ptr.Kind != KindTestGenieRun || ptr.Ref == "" {
		t.Fatalf("unexpected pointer: %+v", ptr)
	}
	if exec.lastDiag != "light" {
		t.Fatalf("fast workflows should use light diagnostics, got %q", exec.lastDiag)
	}
	if len(runs.pins) != 1 || runs.pins[0].by != "gct:baseline:p1" {
		t.Fatalf("expected one pin by baseline owner, got %+v", runs.pins)
	}
}

func TestTestGenieAdapterDiffRegression(t *testing.T) {
	exec := &fakeExecutor{result: ExecResult{}}
	runs := &fakeRuns{compare: CompareResult{
		Verdict: "regression",
		Phases:  []PhaseDiff{{Phase: "playbooks", Verdict: "regression", Regressions: []string{"login-smoke"}}},
	}}
	a := NewWorkflowsAdapter(exec, runs)
	d, err := a.Diff(context.Background(), Target{Scenario: "foo"}, SurfacePointer{Ref: "base-run"})
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if d.Verdict != VerdictRegression || len(d.Regressions) != 1 {
		t.Fatalf("expected regression verdict, got %+v", d)
	}
}

func TestTestGenieAdapterDiffNoBaselineRef(t *testing.T) {
	a := NewTestsAdapter(&fakeExecutor{}, &fakeRuns{})
	d, _ := a.Diff(context.Background(), Target{Scenario: "foo"}, SurfacePointer{})
	if d.Verdict != VerdictNotComparable {
		t.Fatalf("expected not-comparable when baseline has no ref, got %s", d.Verdict)
	}
}

// TestStructureAndRulesAdaptersTargetPhases asserts the structure and rules
// surfaces route through test-genie (Decision 3) — structure→"structure",
// rules→"standards" — rather than calling scenario-auditor directly.
func TestStructureAndRulesAdaptersTargetPhases(t *testing.T) {
	for _, tc := range []struct {
		name    string
		build   func(Executor, RunsClient) SurfaceAdapter
		surface string
		phase   string
	}{
		{"structure", NewStructureAdapter, SurfaceStructure, "structure"},
		{"rules", NewRulesAdapter, SurfaceRules, "standards"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			exec := &fakeExecutor{result: ExecResult{Success: true, Phases: []PhaseStatus{{tc.phase, "passed"}}}}
			runs := &fakeRuns{}
			a := tc.build(exec, runs)

			ptr, err := a.Capture(context.Background(), Target{Scenario: "foo"}, CaptureOptions{PinnedBy: "gct:baseline:p1"})
			if err != nil {
				t.Fatalf("Capture: %v", err)
			}
			if ptr.Kind != KindTestGenieRun || ptr.Ref == "" {
				t.Fatalf("%s should pin a test-genie run, got %+v", tc.name, ptr)
			}
			if len(exec.lastPh) != 1 || exec.lastPh[0] != tc.phase {
				t.Fatalf("%s should run the %q phase, ran %v", tc.name, tc.phase, exec.lastPh)
			}
			if exec.lastDiag != "none" {
				t.Fatalf("%s is a static surface and should request no diagnostics, got %q", tc.name, exec.lastDiag)
			}
			if len(runs.pins) != 1 {
				t.Fatalf("%s should pin its run, got %+v", tc.name, runs.pins)
			}
		})
	}
}

func TestDiffVisuals(t *testing.T) {
	base := VisualSnapshot{Pages: []string{"/", "/dashboard"}, ScreenshotCount: 2}

	// Page removed -> regression.
	d := diffVisuals(base, VisualSnapshot{Pages: []string{"/"}, ScreenshotCount: 1})
	if d.Verdict != VerdictRegression {
		t.Fatalf("expected regression on removed page, got %+v", d)
	}
	// Page added -> new-failure (drift).
	d = diffVisuals(base, VisualSnapshot{Pages: []string{"/", "/dashboard", "/new"}, ScreenshotCount: 3})
	if d.Verdict != VerdictNewFailure {
		t.Fatalf("expected new-failure on added page, got %+v", d)
	}
	// Identical -> clean.
	d = diffVisuals(base, base)
	if d.Verdict != VerdictClean {
		t.Fatalf("expected clean on identical, got %+v", d)
	}
	// Same pages, different count -> drift (new-failure).
	d = diffVisuals(base, VisualSnapshot{Pages: []string{"/", "/dashboard"}, ScreenshotCount: 4})
	if d.Verdict != VerdictNewFailure {
		t.Fatalf("expected new-failure on count change, got %+v", d)
	}
}

func TestVisualsAdapterCaptureDiff(t *testing.T) {
	vis := &fakeVisual{
		capture: []VisualSnapshot{
			{SnapshotID: "s1", Pages: []string{"/"}, ScreenshotCount: 1},         // capture
			{SnapshotID: "s2", Pages: []string{"/", "/new"}, ScreenshotCount: 2}, // diff: page added
		},
		getSnap: VisualSnapshot{SnapshotID: "s1", Pages: []string{"/"}, ScreenshotCount: 1},
		getOK:   true,
	}
	a := NewVisualsAdapter(vis)
	ptr, err := a.Capture(context.Background(), Target{RepoID: 1, Scenario: "foo"}, CaptureOptions{})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if ptr.Ref != "s1" || ptr.Kind != KindGCTLocalSnapshot {
		t.Fatalf("unexpected pointer: %+v", ptr)
	}
	d, err := a.Diff(context.Background(), Target{RepoID: 1, Scenario: "foo"}, ptr)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if d.Verdict != VerdictNewFailure || len(d.NewFailures) != 1 {
		t.Fatalf("expected new-failure on added page, got %+v", d)
	}
}

func TestRulesAdapterDiffClassifiesStandardsRegression(t *testing.T) {
	// rules diffs by running the standards phase now and asking test-genie's
	// CompareRuns to classify the delta — same path as workflows/tests.
	exec := &fakeExecutor{result: ExecResult{}}
	runs := &fakeRuns{compare: CompareResult{
		Verdict: "regression",
		Phases:  []PhaseDiff{{Phase: "standards", Verdict: "regression", Regressions: []string{"PRD-001"}}},
	}}
	a := NewRulesAdapter(exec, runs)
	d, err := a.Diff(context.Background(), Target{Scenario: "foo"}, SurfacePointer{Ref: "base-run"})
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if d.Verdict != VerdictRegression || len(d.Regressions) != 1 {
		t.Fatalf("expected standards regression, got %+v", d)
	}
}
