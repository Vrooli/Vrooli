package baseline

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"git-control-tower/internal/git"
)

func newTestService(t *testing.T, exec Executor, runs RunsClient, gitState git.State) (*Service, *Storage) {
	t.Helper()
	return newTestServiceWith(t, Deps{Exec: exec, Runs: runs, CaptureGit: fixedGit(gitState)})
}

// runDiff drives the durable diff to completion (StartDiff → FinalizeDiff) and
// returns the computed result — the test bridge for the now-async diff flow.
func runDiff(t *testing.T, svc *Service, repoID int64, repoDir, scenario, branch, name, surface string) (DiffResult, error) {
	t.Helper()
	out, err := svc.StartDiff(context.Background(), StartDiffRequest{
		RepoID: repoID, RepoDir: repoDir, Scenario: scenario, Branch: branch, Name: name, Surface: surface,
	})
	if err != nil {
		return DiffResult{}, err
	}
	cd, err := svc.FinalizeDiff(context.Background(), out.Pending)
	if err != nil {
		return DiffResult{}, err
	}
	if cd.Result == nil {
		return DiffResult{}, errors.New("finalized diff has no result")
	}
	return *cd.Result, nil
}

// newTestServiceWith builds a Service from a partial Deps, filling in a fresh
// temp Storage. Lets reachability/cancel tests inject extra seams.
func newTestServiceWith(t *testing.T, d Deps) (*Service, *Storage) {
	t.Helper()
	st := newTestStorage(t)
	d.Storage = st
	if d.CaptureGit == nil {
		d.CaptureGit = fixedGit(git.State{Branch: "agi", Sha: "abc"})
	}
	return NewService(d), st
}

// Create triggers exactly ONE comprehensive run and pins it ONCE, regardless of
// how many surfaces are requested — surfaces are views over the single run.
func TestServiceCreateOneRunOnePin(t *testing.T) {
	exec := &fakeExecutor{result: ExecResult{Success: true, Phases: []PhaseStatus{
		{"structure", "passed"},
		{"standards", "passed"},
		{"unit", "passed"},
		{"integration", "passed"},
		{"smoke", "passed"},
		{"playbooks", "passed"},
	}}}
	runs := &fakeRuns{}
	svc, _ := newTestService(t, exec, runs, git.State{Branch: "agi", Sha: "abc", Dirty: true, DirtySummary: "3 modified"})

	res, err := svc.Create(context.Background(), CreateRequest{
		RepoID: 1, RepoDir: "/repo", Scenario: "foo", Name: "plan-1", Capture: true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if exec.calls != 1 {
		t.Fatalf("expected exactly one comprehensive run, got %d", exec.calls)
	}
	if len(runs.pins) != 1 {
		t.Fatalf("expected exactly one pin, got %+v", runs.pins)
	}
	if res.DirtyWarning == "" {
		t.Fatal("expected dirty-tree warning")
	}
	// Every surface points at the SAME shared run.
	runID := runs.pins[0].runID
	for _, id := range AllSurfaces {
		ptr, ok := res.Manifest.Surfaces[id]
		if !ok {
			t.Fatalf("surface %q not captured: %+v", id, res.Manifest.Surfaces)
		}
		if ptr.Ref != runID || ptr.Kind != KindTestGenieRun {
			t.Fatalf("surface %q should reference shared run %s, got ref=%q kind=%q", id, runID, ptr.Ref, ptr.Kind)
		}
	}
}

// StartCapture returns the run handle WITHOUT pinning or writing a manifest;
// FinalizeCapture (run on a server-owned context) does the pin + manifest. This
// is the return-fast durable flow.
func TestServiceStartThenFinalizeCapture(t *testing.T) {
	exec := &fakeExecutor{result: ExecResult{Success: true, Phases: []PhaseStatus{{"unit", "passed"}}}}
	runs := &fakeRuns{}
	svc, _ := newTestService(t, exec, runs, git.State{Branch: "agi", Sha: "abc"})

	pending, err := svc.StartCapture(context.Background(), CreateRequest{
		RepoID: 1, RepoDir: "/repo", Scenario: "foo", Name: "p", Capture: true,
	})
	if err != nil {
		t.Fatalf("StartCapture: %v", err)
	}
	if pending.Run.RunID == "" || !pending.Run.EtaKnown {
		t.Fatalf("StartCapture must return a run handle with an ETA, got %+v", pending.Run)
	}
	// Nothing pinned or persisted yet.
	if len(runs.pins) != 0 {
		t.Fatalf("StartCapture must not pin, got %+v", runs.pins)
	}
	if _, err := svc.Get(context.Background(), 1, "foo", "agi", "p"); err == nil {
		t.Fatal("manifest must not exist until FinalizeCapture")
	}

	if _, err := svc.FinalizeCapture(context.Background(), pending); err != nil {
		t.Fatalf("FinalizeCapture: %v", err)
	}
	if len(runs.pins) != 1 {
		t.Fatalf("FinalizeCapture must pin exactly once, got %+v", runs.pins)
	}
	got, err := svc.Get(context.Background(), 1, "foo", "agi", "p")
	if err != nil {
		t.Fatalf("manifest must be persisted after FinalizeCapture: %v", err)
	}
	if got.RunID() != pending.Run.RunID {
		t.Fatalf("manifest pins run %q, want %q", got.RunID(), pending.Run.RunID)
	}
}

func TestSnapshotStatusRecoversPendingIntentAfterRestart(t *testing.T) {
	exec := &fakeExecutor{result: ExecResult{Success: true, Phases: []PhaseStatus{{"unit", "passed"}}}}
	runs := &fakeRuns{}
	svc, store := newTestService(t, exec, runs, git.State{Branch: "agi", Sha: "abc"})

	pending, err := svc.StartCapture(context.Background(), CreateRequest{
		RepoID: 1, RepoDir: "/repo", Scenario: "foo", Name: "p", Capture: true,
	})
	if err != nil {
		t.Fatalf("StartCapture: %v", err)
	}
	if _, err := svc.Get(context.Background(), 1, "foo", "agi", "p"); err == nil {
		t.Fatal("manifest must not exist before recovery")
	}

	// Simulate a GCT API restart: the in-memory finalize goroutine is gone, but
	// storage still carries the pending snapshot intent.
	recovered := NewService(Deps{
		Storage:    store,
		Exec:       exec,
		Runs:       runs,
		CaptureGit: fixedGit(git.State{Branch: "agi", Sha: "abc"}),
	})
	st, err := recovered.GetSnapshotStatus(context.Background(), SnapshotStatusRequest{
		RepoID: 1, RepoDir: "/repo", Scenario: "foo", Branch: "agi", Name: "p", RunID: pending.Run.RunID,
	})
	if err != nil {
		t.Fatalf("GetSnapshotStatus: %v", err)
	}
	if st.Status != "ready" {
		t.Fatalf("status = %q, want ready (err=%q)", st.Status, st.Error)
	}
	if st.Baseline == nil || st.Baseline.RunID() != pending.Run.RunID {
		t.Fatalf("baseline not recovered from pending intent: %+v", st.Baseline)
	}
	if len(runs.pins) != 1 {
		t.Fatalf("recovery should pin exactly once, got %+v", runs.pins)
	}
}

func TestSnapshotStatusMarksAbortedRunFailed(t *testing.T) {
	exec := &fakeExecutor{
		err:        errors.New("run run-1 ended without comparable baseline artifacts (status=aborted)"),
		statusInfo: &RunStatusInfo{Status: "aborted", Terminal: true, Success: false},
	}
	runs := &fakeRuns{}
	svc, _ := newTestService(t, exec, runs, git.State{Branch: "agi", Sha: "abc"})

	pending, err := svc.StartCapture(context.Background(), CreateRequest{
		RepoID: 1, RepoDir: "/repo", Scenario: "foo", Name: "p", Capture: true,
	})
	if err != nil {
		t.Fatalf("StartCapture: %v", err)
	}
	st, err := svc.GetSnapshotStatus(context.Background(), SnapshotStatusRequest{
		RepoID: 1, RepoDir: "/repo", Scenario: "foo", Branch: "agi", Name: "p", RunID: pending.Run.RunID,
	})
	if err != nil {
		t.Fatalf("GetSnapshotStatus: %v", err)
	}
	if st.Status != "failed" || !strings.Contains(st.Error, "aborted") {
		t.Fatalf("status = %+v, want failed aborted", st)
	}
	if _, err := svc.Get(context.Background(), 1, "foo", "agi", "p"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("aborted snapshot must not create baseline manifest, got %v", err)
	}
}

// StartCapture fails fast (no run started) when the baseline already exists.
func TestServiceStartCaptureRejectsDuplicate(t *testing.T) {
	exec := &fakeExecutor{result: ExecResult{Success: true, Phases: []PhaseStatus{{"unit", "passed"}}}}
	runs := &fakeRuns{}
	svc, _ := newTestService(t, exec, runs, git.State{Branch: "agi", Sha: "abc"})

	if _, err := svc.Create(context.Background(), CreateRequest{RepoID: 1, Scenario: "foo", Name: "dup", Branch: "agi"}); err != nil {
		t.Fatalf("seed create: %v", err)
	}
	startsBefore := exec.calls
	if _, err := svc.StartCapture(context.Background(), CreateRequest{RepoID: 1, Scenario: "foo", Name: "dup", Capture: true}); err != ErrAlreadyExists {
		t.Fatalf("StartCapture on duplicate = %v, want ErrAlreadyExists", err)
	}
	if exec.calls != startsBefore {
		t.Fatal("StartCapture must not start a run when the baseline already exists")
	}
}

// A subset of surfaces still triggers exactly one run + one pin.
func TestServiceCreateIncludeSubset(t *testing.T) {
	exec := &fakeExecutor{result: ExecResult{Success: true, Phases: []PhaseStatus{{"unit", "passed"}}}}
	runs := &fakeRuns{}
	svc, _ := newTestService(t, exec, runs, git.State{Branch: "agi", Sha: "abc"})

	res, err := svc.Create(context.Background(), CreateRequest{
		RepoID: 1, Scenario: "foo", Name: "p", Include: []string{SurfaceTests}, Capture: true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if exec.calls != 1 || len(runs.pins) != 1 {
		t.Fatalf("expected one run + one pin, got calls=%d pins=%d", exec.calls, len(runs.pins))
	}
	if _, ok := res.Manifest.Surfaces[SurfaceTests]; !ok {
		t.Fatalf("tests surface not captured: %+v", res.Manifest.Surfaces)
	}
	if _, ok := res.Manifest.Surfaces[SurfaceWorkflows]; ok {
		t.Fatalf("workflows should not be captured when not requested")
	}
}

func TestServiceCreateDuplicate(t *testing.T) {
	svc, _ := newTestService(t, &fakeExecutor{}, &fakeRuns{}, git.State{Branch: "agi"})
	req := CreateRequest{RepoID: 1, Scenario: "foo", Name: "p", Capture: false}
	if _, err := svc.Create(context.Background(), req); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := svc.Create(context.Background(), req); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

// When test-genie can't be reached, every requested surface is recorded as
// skipped so the baseline can't masquerade as complete.
func TestServiceCreateRunFailureSkipsAllSurfaces(t *testing.T) {
	exec := &fakeExecutor{err: errors.New("test-genie down")}
	runs := &fakeRuns{}
	svc, _ := newTestService(t, exec, runs, git.State{Branch: "agi", Sha: "abc"})

	res, err := svc.Create(context.Background(), CreateRequest{
		RepoID: 1, Scenario: "foo", Name: "p", Include: []string{SurfaceTests, SurfaceWorkflows}, Capture: true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if len(runs.pins) != 0 {
		t.Fatalf("a failed run should pin nothing, got %+v", runs.pins)
	}
	for _, id := range []string{SurfaceTests, SurfaceWorkflows} {
		if _, ok := res.Manifest.Skipped[id]; !ok {
			t.Fatalf("surface %q should be skipped on run failure, skipped=%v", id, res.Manifest.Skipped)
		}
	}
}

func TestServiceDiffEmptyBaselineExplainsSkippedSurfaces(t *testing.T) {
	svc, st := newTestService(t, nil, nil, git.State{Branch: "agi", Sha: "abc"})
	err := st.Save(1, BaselineManifest{
		Name:      "stt-scaling-analysis-plan",
		Scenario:  "audio-tools",
		Branch:    "agi",
		CreatedAt: time.Now(),
		Git:       git.State{Branch: "agi", Sha: "abc"},
		Surfaces:  map[string]SurfacePointer{},
		Skipped: map[string]string{
			SurfaceTests: "test-genie unreachable",
			SurfaceRules: "api-core discovery failed",
		},
		SchemaVersion: SchemaVersion,
	}, CreateOnly)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	_, err = svc.StartDiff(context.Background(), StartDiffRequest{
		RepoID: 1, RepoDir: "/repo", Scenario: "audio-tools", Branch: "agi", Name: "stt-scaling-analysis-plan",
	})
	if err == nil {
		t.Fatal("StartDiff succeeded, want empty-baseline error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "has no captured run") ||
		!strings.Contains(msg, "rules: api-core discovery failed") ||
		!strings.Contains(msg, "tests: test-genie unreachable") ||
		!strings.Contains(msg, "snapshot again") {
		t.Fatalf("StartDiff error = %q, want skipped surface reasons", msg)
	}
}

// Diff issues exactly ONE current run + ONE empty-phase compare and buckets the
// per-phase deltas into surfaces locally (option-c). The multi-phase `tests`
// surface aggregates unit+integration+smoke.
func TestServiceDiffOptionCBucketing(t *testing.T) {
	exec := &fakeExecutor{result: ExecResult{Success: true, Phases: []PhaseStatus{
		{"unit", "passed"}, {"integration", "passed"}, {"smoke", "passed"}, {"playbooks", "passed"},
	}}}
	runs := &fakeRuns{}
	svc, _ := newTestService(t, exec, runs, git.State{Branch: "agi", Sha: "abc"})

	if _, err := svc.Create(context.Background(), CreateRequest{
		RepoID: 1, Scenario: "foo", Name: "p",
		Include: []string{SurfaceTests, SurfaceWorkflows}, Capture: true,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if exec.calls != 1 {
		t.Fatalf("create should trigger one run, got %d", exec.calls)
	}

	// One empty-phase compare returns deltas for all phases at once.
	runs.compare = CompareResult{
		Verdict: "regression",
		Phases: []PhaseDiff{
			{Phase: "unit", Verdict: "clean"},
			{Phase: "integration", Verdict: "preexisting", Preexisting: []string{"TestFlaky"}},
			{Phase: "smoke", Verdict: "regression", Regressions: []string{"login-smoke"}},
			{Phase: "playbooks", Verdict: "clean"},
		},
	}

	res, err := runDiff(t, svc, 1, "/repo", "foo", "agi", "p", "")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if exec.calls != 2 {
		t.Fatalf("diff should trigger exactly one more run (total 2), got %d", exec.calls)
	}

	byID := map[string]SurfaceDiff{}
	for _, d := range res.Surfaces {
		byID[d.SurfaceID] = d
	}
	// tests = unit(clean) + integration(preexisting) + smoke(regression) ⇒ worst = regression.
	tests := byID[SurfaceTests]
	if tests.Verdict != VerdictRegression {
		t.Fatalf("tests surface should aggregate to regression, got %s", tests.Verdict)
	}
	if len(tests.Regressions) != 1 || tests.Regressions[0] != "login-smoke" {
		t.Fatalf("tests regressions not aggregated: %+v", tests.Regressions)
	}
	if len(tests.Preexisting) != 1 || tests.Preexisting[0] != "TestFlaky" {
		t.Fatalf("tests preexisting not aggregated: %+v", tests.Preexisting)
	}
	// workflows = playbooks(clean).
	if byID[SurfaceWorkflows].Verdict != VerdictClean {
		t.Fatalf("workflows should be clean, got %s", byID[SurfaceWorkflows].Verdict)
	}
	if res.Verdict != VerdictRegression {
		t.Fatalf("overall verdict should be regression, got %s", res.Verdict)
	}
}

func TestServiceDiffIncludesUnmappedPhaseInOverallVerdict(t *testing.T) {
	exec := &fakeExecutor{result: ExecResult{Success: true, Phases: []PhaseStatus{
		{"unit", "passed"}, {"architecture", "failed"},
	}}}
	runs := &fakeRuns{compare: CompareResult{
		Verdict: "regression",
		Phases: []PhaseDiff{
			{Phase: "unit", Verdict: "clean"},
			{Phase: "architecture", Verdict: "regression", Regressions: []string{"arch-drift"}},
		},
	}}
	svc, _ := newTestService(t, exec, runs, git.State{Branch: "agi", Sha: "abc"})

	if _, err := svc.Create(context.Background(), CreateRequest{
		RepoID: 1, Scenario: "foo", Name: "p", Include: []string{SurfaceTests}, Capture: true,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	res, err := runDiff(t, svc, 1, "/repo", "foo", "agi", "p", "")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if res.Verdict != VerdictRegression {
		t.Fatalf("overall verdict = %s, want regression from unmapped architecture phase", res.Verdict)
	}
	if len(res.Phases) != 2 {
		t.Fatalf("expected all phase diffs to be exposed, got %+v", res.Phases)
	}
	var found bool
	for _, p := range res.Phases {
		if p.Phase == "architecture" {
			found = true
			if p.SurfaceID != "" {
				t.Fatalf("unmapped phase SurfaceID = %q, want empty", p.SurfaceID)
			}
			if p.Verdict != VerdictRegression {
				t.Fatalf("architecture verdict = %s, want regression", p.Verdict)
			}
		}
	}
	if !found {
		t.Fatalf("architecture phase missing from diff details: %+v", res.Phases)
	}
}

// A diff restricted to one surface still buckets correctly and returns only it.
func TestServiceDiffSingleSurface(t *testing.T) {
	exec := &fakeExecutor{result: ExecResult{Phases: []PhaseStatus{{"standards", "passed"}}}}
	runs := &fakeRuns{compare: CompareResult{Phases: []PhaseDiff{
		{Phase: "standards", Verdict: "regression", Regressions: []string{"PRD-001"}},
		{Phase: "architecture", Verdict: "regression", Regressions: []string{"arch-drift"}},
	}}}
	svc, _ := newTestService(t, exec, runs, git.State{Branch: "agi", Sha: "abc"})

	if _, err := svc.Create(context.Background(), CreateRequest{
		RepoID: 1, Scenario: "foo", Name: "p", Include: []string{SurfaceRules}, Capture: true,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	res, err := runDiff(t, svc, 1, "/repo", "foo", "agi", "p", SurfaceRules)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(res.Surfaces) != 1 || res.Surfaces[0].SurfaceID != SurfaceRules {
		t.Fatalf("expected only the rules surface, got %+v", res.Surfaces)
	}
	if res.Surfaces[0].Verdict != VerdictRegression {
		t.Fatalf("rules should be regression, got %s", res.Surfaces[0].Verdict)
	}
	if len(res.Phases) != 1 || res.Phases[0].Phase != "standards" || res.Phases[0].SurfaceID != SurfaceRules {
		t.Fatalf("surface-filtered phase details should include only rules phases, got %+v", res.Phases)
	}
}

// The visuals surface is advisory: a pixel difference (or an added/removed page)
// is the neutral `changed` tier, never a failing verdict, and the overall diff
// stays exit-0. test-genie owns the pixel comparison (CompareRunVisuals); GCT
// only renders the neutral deltas.
func TestServiceDiffVisualsAdvisory(t *testing.T) {
	exec := &fakeExecutor{result: ExecResult{Phases: []PhaseStatus{{"smoke", "passed"}}}}
	runs := &fakeRuns{
		visualDeltas: []VisualDelta{
			{Page: "/", Status: "identical"},
			{Page: "/dashboard", Status: "changed", ChangedFraction: 0.18},
			{Page: "/backlog", Status: "removed"},
		},
	}
	svc, _ := newTestService(t, exec, runs, git.State{Branch: "agi", Sha: "abc"})

	if _, err := svc.Create(context.Background(), CreateRequest{
		RepoID: 1, Scenario: "foo", Name: "p", Include: []string{SurfaceVisuals}, Capture: true,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	res, err := runDiff(t, svc, 1, "/repo", "foo", "agi", "p", SurfaceVisuals)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(res.Surfaces) != 1 {
		t.Fatalf("expected one surface, got %+v", res.Surfaces)
	}
	surface := res.Surfaces[0]
	if surface.Verdict != VerdictChanged {
		t.Fatalf("visual differences should be the advisory `changed` tier, got %s (%+v)", surface.Verdict, surface)
	}
	if len(surface.Regressions) != 0 || len(surface.NewFailures) != 0 {
		t.Fatalf("visuals must never populate failure buckets: %+v", surface)
	}
	if len(surface.Changed) != 2 { // changed page + removed page; identical omitted
		t.Fatalf("expected 2 review entries, got %v", surface.Changed)
	}
	// The overall diff verdict must not gate.
	if res.Verdict != VerdictChanged {
		t.Fatalf("overall verdict = %s, want changed (advisory)", res.Verdict)
	}
}

func TestServicePartialBaselineDoesNotMasqueradeAsComplete(t *testing.T) {
	// The comprehensive run skips browser-dependent validation. The tests
	// surface still buckets from the phases that did run; a surface whose phases
	// all went missing in the compare is not-comparable, not silently clean.
	exec := &fakeExecutor{result: ExecResult{Phases: []PhaseStatus{{"unit", "passed"}}}}
	runs := &fakeRuns{compare: CompareResult{Phases: []PhaseDiff{
		{Phase: "unit", Verdict: "clean"},
		// workflows' playbooks phase did not appear in the compare.
	}}}
	svc, _ := newTestService(t, exec, runs, git.State{Branch: "agi", Sha: "abc"})

	if _, err := svc.Create(context.Background(), CreateRequest{
		RepoID: 1, Scenario: "foo", Name: "p",
		Include: []string{SurfaceTests, SurfaceWorkflows}, Capture: true,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	res, err := runDiff(t, svc, 1, "/repo", "foo", "agi", "p", "")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	byID := map[string]SurfaceDiff{}
	for _, d := range res.Surfaces {
		byID[d.SurfaceID] = d
	}
	if byID[SurfaceWorkflows].Verdict != VerdictNotComparable {
		t.Fatalf("workflows with no comparable phase should be not-comparable, got %s", byID[SurfaceWorkflows].Verdict)
	}
}

// When the reachability probe fails fast, every requested surface is skipped
// with the probe's reason and NO comprehensive run / pin is attempted — the
// fast-skip that replaces the multi-minute silent hang.
func TestServiceCaptureFastSkipsWhenUnreachable(t *testing.T) {
	exec := &fakeExecutor{}
	runs := &fakeRuns{}
	reach := &fakeReachability{err: errors.New("connection refused")}
	svc, _ := newTestServiceWith(t, Deps{Exec: exec, Runs: runs, Reachable: reach})

	res, err := svc.Create(context.Background(), CreateRequest{
		RepoID: 1, Scenario: "foo", Name: "p", Capture: true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if reach.probes != 1 {
		t.Fatalf("expected exactly one reachability probe, got %d", reach.probes)
	}
	if exec.calls != 0 {
		t.Fatalf("unreachable backend must not trigger a run, got %d calls", exec.calls)
	}
	if len(runs.pins) != 0 {
		t.Fatalf("unreachable backend must not pin, got %+v", runs.pins)
	}
	for _, id := range AllSurfaces {
		reason, ok := res.Manifest.Skipped[id]
		if !ok {
			t.Fatalf("surface %q should be skipped when unreachable, skipped=%v", id, res.Manifest.Skipped)
		}
		if !strings.Contains(reason, "connection refused") {
			t.Fatalf("skip reason should carry the probe error, got %q", reason)
		}
	}
}

// A reachable backend proceeds normally: the probe is consulted, then the run
// runs and pins.
func TestServiceCaptureProceedsWhenReachable(t *testing.T) {
	exec := &fakeExecutor{result: ExecResult{Phases: []PhaseStatus{{"unit", "passed"}}}}
	runs := &fakeRuns{}
	reach := &fakeReachability{}
	svc, _ := newTestServiceWith(t, Deps{Exec: exec, Runs: runs, Reachable: reach})

	if _, err := svc.Create(context.Background(), CreateRequest{
		RepoID: 1, Scenario: "foo", Name: "p", Include: []string{SurfaceTests}, Capture: true,
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if reach.probes != 1 || exec.calls != 1 || len(runs.pins) != 1 {
		t.Fatalf("reachable path should probe once, run once, pin once; got probes=%d calls=%d pins=%d", reach.probes, exec.calls, len(runs.pins))
	}
}

// StartDiff fast-fails an unreachable backend up front: it returns an error
// carrying the probe reason and starts NO run — the durable contract mirrors
// StartCapture (an unreachable test-genie can't produce a verdict, so the CLI
// reports it rather than rendering a misleading all-not-comparable diff).
func TestServiceStartDiffFailsFastWhenUnreachable(t *testing.T) {
	exec := &fakeExecutor{result: ExecResult{Phases: []PhaseStatus{{"unit", "passed"}}}}
	runs := &fakeRuns{}
	reach := &fakeReachability{}
	svc, _ := newTestServiceWith(t, Deps{Exec: exec, Runs: runs, Reachable: reach})

	// Capture while reachable so the baseline has a comparable run.
	if _, err := svc.Create(context.Background(), CreateRequest{
		RepoID: 1, Scenario: "foo", Name: "p", Include: []string{SurfaceTests}, Capture: true,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	// Now the backend goes unreachable before the diff.
	reach.err = errors.New("connection refused")
	callsBefore := exec.calls

	_, err := svc.StartDiff(context.Background(), StartDiffRequest{
		RepoID: 1, RepoDir: "/repo", Scenario: "foo", Branch: "agi", Name: "p",
	})
	if err == nil {
		t.Fatal("StartDiff should fail fast when test-genie is unreachable")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("error should carry the probe reason, got %v", err)
	}
	if exec.calls != callsBefore {
		t.Fatalf("unreachable diff must not start a run, calls went %d→%d", callsBefore, exec.calls)
	}
}

// The orchestration tail (pin + manifest write) must complete even when the
// context passed to Create is already canceled — modeling a client that
// disconnected after the heavy run finished. The handler detaches the tail via
// context.WithoutCancel; this asserts the service itself does not abandon the
// pin/manifest on a canceled context (the seams here are synchronous fakes, so
// the run + pin + Save still run to completion and the baseline is durable).
func TestServiceCaptureTailSurvivesCanceledContext(t *testing.T) {
	exec := &fakeExecutor{result: ExecResult{Phases: []PhaseStatus{{"unit", "passed"}}}}
	runs := &fakeRuns{}
	// A reachability probe that ignores cancellation (mirrors the production
	// probe, which bounds itself independently of the caller ctx).
	reach := &fakeReachability{}
	svc, st := newTestServiceWith(t, Deps{Exec: exec, Runs: runs, Reachable: reach})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already canceled before the call

	res, err := svc.Create(ctx, CreateRequest{
		RepoID: 1, Scenario: "foo", Name: "p", Include: []string{SurfaceTests}, Capture: true,
	})
	if err != nil {
		t.Fatalf("Create on canceled ctx: %v", err)
	}
	if len(runs.pins) != 1 {
		t.Fatalf("pin must still be written despite canceled ctx, got %+v", runs.pins)
	}
	if _, ok := res.Manifest.Surfaces[SurfaceTests]; !ok {
		t.Fatalf("manifest surface must still be written, got %+v", res.Manifest.Surfaces)
	}
	// Manifest is durably persisted.
	if _, err := st.Load(1, "foo", "agi", "p"); err != nil {
		t.Fatalf("manifest must be durable on disk: %v", err)
	}
}

// seedBaseline captures a baseline (one shared run) so a diff has a base run.
func seedBaseline(t *testing.T, svc *Service, name string) {
	t.Helper()
	if _, err := svc.Create(context.Background(), CreateRequest{
		RepoID: 1, RepoDir: "/repo", Scenario: "foo", Name: name, Branch: "agi", Capture: true,
	}); err != nil {
		t.Fatalf("seed baseline %q: %v", name, err)
	}
}

// A clean working tree at a sha with an existing comprehensive run reuses that
// run — no second suite (the redundant-run fix, §6b rows 8/9).
func TestServiceDiffReusesCleanTreeRun(t *testing.T) {
	exec := &fakeExecutor{
		result:      ExecResult{Phases: []PhaseStatus{{"unit", "passed"}}},
		reusableHit: true,
		reusable:    ReusableRun{RunID: "reused-run", CompletedAt: time.Now().UTC()},
	}
	runs := &fakeRuns{compare: CompareResult{Phases: []PhaseDiff{{Phase: "unit", Verdict: "clean"}}}}
	svc, _ := newTestServiceWith(t, Deps{Exec: exec, Runs: runs, ReuseTTL: 15 * time.Minute, CaptureGit: fixedGit(git.State{Branch: "agi", Sha: "abc"})})
	seedBaseline(t, svc, "p")
	callsAfterSeed := exec.calls

	out, err := svc.StartDiff(context.Background(), StartDiffRequest{RepoID: 1, RepoDir: "/repo", Scenario: "foo", Branch: "agi", Name: "p"})
	if err != nil {
		t.Fatalf("StartDiff: %v", err)
	}
	if !out.ReusedRun {
		t.Fatal("clean tree with an existing run should reuse it")
	}
	if out.RunID != "reused-run" {
		t.Fatalf("reused run id = %q, want reused-run", out.RunID)
	}
	if out.ReusedSha != "abc" {
		t.Fatalf("reused sha = %q, want abc", out.ReusedSha)
	}
	if exec.calls != callsAfterSeed {
		t.Fatalf("reuse must not start a new run, calls went %d→%d", callsAfterSeed, exec.calls)
	}
}

// A dirty working tree never reuses a run (uncommitted edits aren't captured by
// sha) — a fresh run starts (§6b row 10).
func TestServiceDiffDirtyTreeStartsFresh(t *testing.T) {
	exec := &fakeExecutor{
		result:      ExecResult{Phases: []PhaseStatus{{"unit", "passed"}}},
		reusableHit: true, // a match exists, but the dirty tree must ignore it
		reusable:    ReusableRun{RunID: "reused-run", CompletedAt: time.Now().UTC()},
	}
	runs := &fakeRuns{}
	svc, _ := newTestServiceWith(t, Deps{Exec: exec, Runs: runs, ReuseTTL: 15 * time.Minute, CaptureGit: fixedGit(git.State{Branch: "agi", Sha: "abc", Dirty: true, DirtySummary: "1 modified"})})
	seedBaseline(t, svc, "p")
	callsAfterSeed := exec.calls

	out, err := svc.StartDiff(context.Background(), StartDiffRequest{RepoID: 1, RepoDir: "/repo", Scenario: "foo", Branch: "agi", Name: "p"})
	if err != nil {
		t.Fatalf("StartDiff: %v", err)
	}
	if out.ReusedRun {
		t.Fatal("a dirty tree must never reuse a run")
	}
	if exec.calls != callsAfterSeed+1 {
		t.Fatalf("dirty tree should start exactly one fresh run, calls went %d→%d", callsAfterSeed, exec.calls)
	}
	if out.DirtyWarning == "" {
		t.Fatal("dirty tree should surface a warning")
	}
}

// A reuse candidate older than the TTL is not reused — a fresh run starts.
func TestServiceDiffReuseTTLExpired(t *testing.T) {
	exec := &fakeExecutor{
		result:      ExecResult{Phases: []PhaseStatus{{"unit", "passed"}}},
		reusableHit: true,
		reusable:    ReusableRun{RunID: "stale-run", CompletedAt: time.Now().UTC().Add(-1 * time.Hour)},
	}
	runs := &fakeRuns{}
	svc, _ := newTestServiceWith(t, Deps{Exec: exec, Runs: runs, ReuseTTL: 15 * time.Minute, CaptureGit: fixedGit(git.State{Branch: "agi", Sha: "abc"})})
	seedBaseline(t, svc, "p")
	callsAfterSeed := exec.calls

	out, err := svc.StartDiff(context.Background(), StartDiffRequest{RepoID: 1, RepoDir: "/repo", Scenario: "foo", Branch: "agi", Name: "p"})
	if err != nil {
		t.Fatalf("StartDiff: %v", err)
	}
	if out.ReusedRun {
		t.Fatal("a run older than the TTL must not be reused")
	}
	if exec.calls != callsAfterSeed+1 {
		t.Fatalf("expired reuse should start a fresh run, calls went %d→%d", callsAfterSeed, exec.calls)
	}
}

// GetDiffResult returns in_progress (with a backoff) when the run is still
// executing and no result is cached yet.
func TestServiceGetDiffResultInProgress(t *testing.T) {
	exec := &fakeExecutor{statusInfo: &RunStatusInfo{Status: "in_progress", Terminal: false, RecommendedNextCheckSeconds: 12}}
	runs := &fakeRuns{}
	svc, _ := newTestServiceWith(t, Deps{Exec: exec, Runs: runs, CaptureGit: fixedGit(git.State{Branch: "agi", Sha: "abc"})})
	seedBaseline(t, svc, "p")

	cd, next, err := svc.GetDiffResult(context.Background(), GetDiffResultRequest{RepoID: 1, RepoDir: "/repo", Scenario: "foo", Branch: "agi", Name: "p", RunID: "run-running"})
	if err != nil {
		t.Fatalf("GetDiffResult: %v", err)
	}
	if cd.Status != "in_progress" {
		t.Fatalf("status = %q, want in_progress", cd.Status)
	}
	if next != 12 {
		t.Fatalf("recommended next check = %d, want 12", next)
	}
}

func TestServiceGetDiffResultLatestUsesStartDiffIntent(t *testing.T) {
	exec := &fakeExecutor{statusInfo: &RunStatusInfo{Status: "in_progress", Terminal: false, RecommendedNextCheckSeconds: 12}}
	runs := &fakeRuns{}
	svc, _ := newTestServiceWith(t, Deps{Exec: exec, Runs: runs, CaptureGit: fixedGit(git.State{Branch: "agi", Sha: "abc"})})
	seedBaseline(t, svc, "p")

	start, err := svc.StartDiff(context.Background(), StartDiffRequest{
		RepoID: 1, RepoDir: "/repo", Scenario: "foo", Branch: "agi", Name: "p",
	})
	if err != nil {
		t.Fatalf("StartDiff: %v", err)
	}

	cd, next, err := svc.GetDiffResult(context.Background(), GetDiffResultRequest{
		RepoID: 1, RepoDir: "/repo", Scenario: "foo", Branch: "agi", Name: "p", Latest: true,
	})
	if err != nil {
		t.Fatalf("GetDiffResult latest: %v", err)
	}
	if cd.RunID != start.RunID {
		t.Fatalf("latest resolved run %q, want %q", cd.RunID, start.RunID)
	}
	if cd.Status != "in_progress" {
		t.Fatalf("status = %q, want in_progress", cd.Status)
	}
	if next != 12 {
		t.Fatalf("recommended next check = %d, want 12", next)
	}
}

// FinalizeDiff caches the verdict; GetDiffResult then returns it instantly.
// Two baseline names sharing ONE current run each cache their own result (§6.3).
func TestServiceTwoNamesShareOneRunTwoCachedResults(t *testing.T) {
	exec := &fakeExecutor{result: ExecResult{Phases: []PhaseStatus{{"unit", "passed"}}}}
	runs := &fakeRuns{compare: CompareResult{Phases: []PhaseDiff{{Phase: "unit", Verdict: "clean"}}}}
	svc, _ := newTestServiceWith(t, Deps{Exec: exec, Runs: runs, CaptureGit: fixedGit(git.State{Branch: "agi", Sha: "abc"})})
	seedBaseline(t, svc, "alpha")
	seedBaseline(t, svc, "beta")

	const sharedRun = "shared-current-run"
	for _, name := range []string{"alpha", "beta"} {
		manifest, err := svc.Get(context.Background(), 1, "foo", "agi", name)
		if err != nil {
			t.Fatalf("load %q: %v", name, err)
		}
		pending := PendingDiff{
			RepoID: 1, Scenario: "foo", Branch: "agi", Name: name,
			Manifest: manifest, BaseRunID: manifest.RunID(), CurRunID: sharedRun,
		}
		if _, err := svc.FinalizeDiff(context.Background(), pending); err != nil {
			t.Fatalf("FinalizeDiff %q: %v", name, err)
		}
	}
	// Both names resolve their own cached result against the one shared run.
	for _, name := range []string{"alpha", "beta"} {
		cd, _, err := svc.GetDiffResult(context.Background(), GetDiffResultRequest{RepoID: 1, RepoDir: "/repo", Scenario: "foo", Branch: "agi", Name: name, RunID: sharedRun})
		if err != nil {
			t.Fatalf("GetDiffResult %q: %v", name, err)
		}
		if cd.Status != "ready" || cd.Result == nil {
			t.Fatalf("name %q: status=%q result=%v, want ready+result", name, cd.Status, cd.Result)
		}
		if cd.Result.Manifest.Name != name {
			t.Fatalf("cached result for %q has manifest name %q", name, cd.Result.Manifest.Name)
		}
	}
}

// When the run is terminal but no result was cached (finalize lost to a crash),
// GetDiffResult recomputes on demand.
func TestServiceGetDiffResultRecomputesWhenTerminalUncached(t *testing.T) {
	exec := &fakeExecutor{result: ExecResult{Phases: []PhaseStatus{{"unit", "passed"}}}} // RunStatus defaults to terminal
	runs := &fakeRuns{compare: CompareResult{Phases: []PhaseDiff{{Phase: "unit", Verdict: "clean"}}}}
	svc, _ := newTestServiceWith(t, Deps{Exec: exec, Runs: runs, CaptureGit: fixedGit(git.State{Branch: "agi", Sha: "abc"})})
	seedBaseline(t, svc, "p")

	cd, _, err := svc.GetDiffResult(context.Background(), GetDiffResultRequest{RepoID: 1, RepoDir: "/repo", Scenario: "foo", Branch: "agi", Name: "p", RunID: "terminal-uncached"})
	if err != nil {
		t.Fatalf("GetDiffResult: %v", err)
	}
	if cd.Status != "ready" || cd.Result == nil {
		t.Fatalf("terminal-uncached should recompute to ready, got status=%q result=%v", cd.Status, cd.Result)
	}
}

func TestServiceDeleteUnpinsRunOnce(t *testing.T) {
	exec := &fakeExecutor{result: ExecResult{Phases: []PhaseStatus{{"unit", "passed"}, {"playbooks", "passed"}}}}
	runs := &fakeRuns{}
	svc, _ := newTestService(t, exec, runs, git.State{Branch: "agi", Sha: "abc"})

	if _, err := svc.Create(context.Background(), CreateRequest{
		RepoID: 1, Scenario: "foo", Name: "p", Capture: true,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.Delete(context.Background(), 1, "foo", "agi", "p"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	// One shared run ⇒ exactly one unpin even though every surface referenced it.
	if len(runs.unpins) != 1 {
		t.Fatalf("expected exactly one unpin on delete, got %+v", runs.unpins)
	}
	if _, err := svc.Get(context.Background(), 1, "foo", "agi", "p"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected baseline gone, got %v", err)
	}
}
