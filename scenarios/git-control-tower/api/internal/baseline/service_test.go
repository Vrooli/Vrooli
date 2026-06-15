package baseline

import (
	"context"
	"errors"
	"strings"
	"testing"

	"git-control-tower/internal/git"
)

func newTestService(t *testing.T, exec Executor, runs RunsClient, gitState git.State) (*Service, *Storage) {
	t.Helper()
	return newTestServiceWith(t, Deps{Exec: exec, Runs: runs, CaptureGit: fixedGit(gitState)})
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

	res, err := svc.Diff(context.Background(), 1, "/repo", "foo", "agi", "p", "")
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

// A diff restricted to one surface still buckets correctly and returns only it.
func TestServiceDiffSingleSurface(t *testing.T) {
	exec := &fakeExecutor{result: ExecResult{Phases: []PhaseStatus{{"standards", "passed"}}}}
	runs := &fakeRuns{compare: CompareResult{Phases: []PhaseDiff{
		{Phase: "standards", Verdict: "regression", Regressions: []string{"PRD-001"}},
	}}}
	svc, _ := newTestService(t, exec, runs, git.State{Branch: "agi", Sha: "abc"})

	if _, err := svc.Create(context.Background(), CreateRequest{
		RepoID: 1, Scenario: "foo", Name: "p", Include: []string{SurfaceRules}, Capture: true,
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	res, err := svc.Diff(context.Background(), 1, "/repo", "foo", "agi", "p", SurfaceRules)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(res.Surfaces) != 1 || res.Surfaces[0].SurfaceID != SurfaceRules {
		t.Fatalf("expected only the rules surface, got %+v", res.Surfaces)
	}
	if res.Surfaces[0].Verdict != VerdictRegression {
		t.Fatalf("rules should be regression, got %s", res.Surfaces[0].Verdict)
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
	res, err := svc.Diff(context.Background(), 1, "/repo", "foo", "agi", "p", SurfaceVisuals)
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
	// The comprehensive run skips the smoke phase (browser unavailable). The
	// tests surface (unit+integration+smoke) still buckets from the phases that
	// did run; a surface whose phases all went missing in the compare is
	// not-comparable, not silently clean.
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
	res, err := svc.Diff(context.Background(), 1, "/repo", "foo", "agi", "p", "")
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

// Diff fast-skips an unreachable backend: it marks captured surfaces
// not-comparable with the probe reason instead of blocking on a fresh run.
func TestServiceDiffFastSkipsWhenUnreachable(t *testing.T) {
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

	res, err := svc.Diff(context.Background(), 1, "/repo", "foo", "agi", "p", "")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if exec.calls != callsBefore {
		t.Fatalf("unreachable diff must not trigger a current run, calls went %d→%d", callsBefore, exec.calls)
	}
	if len(res.Surfaces) == 0 {
		t.Fatal("expected surfaces marked not-comparable")
	}
	for _, d := range res.Surfaces {
		if d.Verdict != VerdictNotComparable {
			t.Fatalf("surface %q should be not-comparable when unreachable, got %s", d.SurfaceID, d.Verdict)
		}
		if !strings.Contains(d.Summary, "connection refused") {
			t.Fatalf("not-comparable reason should carry the probe error, got %q", d.Summary)
		}
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
