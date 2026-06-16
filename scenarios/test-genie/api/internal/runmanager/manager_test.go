package runmanager

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	sharedruns "test-genie/internal/shared/runs"
)

// fakeExecutor is a scriptable stand-in for *execution.SuiteExecutionService. It
// records an in-progress index entry (like the real orchestrator), emits the
// scripted events, then either blocks until released or until the context is
// cancelled — letting tests exercise decouple/fan-out/abort deterministically.
type fakeExecutor struct {
	scenarioDir string
	events      []orchestrator.ExecutionEvent
	result      *orchestrator.SuiteExecutionResult

	blockOnCtx bool
	release    chan struct{}

	startedOnce sync.Once
	started     chan struct{}
	// calls counts how many times ExecuteWithEvents was invoked — i.e. how many
	// suites were actually driven. The one-run-per-scenario guard's whole point
	// is that coalesced/rejected requests never increment this.
	calls int32
}

func (f *fakeExecutor) driveCount() int32 { return atomic.LoadInt32(&f.calls) }

func newFakeExecutor(scenarioDir string) *fakeExecutor {
	return &fakeExecutor{
		scenarioDir: scenarioDir,
		release:     make(chan struct{}),
		started:     make(chan struct{}),
	}
}

func (f *fakeExecutor) ExecuteWithEvents(ctx context.Context, input execution.SuiteExecutionInput, emit orchestrator.ExecutionEventCallback) (*orchestrator.SuiteExecutionResult, error) {
	atomic.AddInt32(&f.calls, 1)
	runID := input.Request.RunID
	if f.scenarioDir != "" {
		_ = sharedruns.NewIndex(f.scenarioDir).Append(sharedruns.RunRecord{
			RunID:     runID,
			Scenario:  input.Request.ScenarioName,
			StartedAt: time.Now().UTC(),
			Status:    sharedruns.StatusInProgress,
		})
	}
	f.startedOnce.Do(func() { close(f.started) })

	for _, ev := range f.events {
		if emit != nil {
			emit(ev)
		}
	}

	if f.blockOnCtx {
		<-ctx.Done()
	} else {
		<-f.release
	}

	if f.scenarioDir != "" {
		status := sharedruns.StatusPassed
		if f.result != nil && !f.result.Success {
			status = sharedruns.StatusFailed
		}
		_ = sharedruns.NewIndex(f.scenarioDir).Update(runID, func(r *sharedruns.RunRecord) error {
			r.Status = status
			r.CompletedAt = time.Now().UTC()
			return nil
		})
	}
	return f.result, nil
}

func phaseEvent(t orchestrator.ExecutionEventType, phase, status string) orchestrator.ExecutionEvent {
	return orchestrator.ExecutionEvent{Type: t, Phase: phase, Status: status, PhaseIndex: 1, PhaseTotal: 1, Timestamp: time.Now()}
}

func startInput(scenario string) execution.SuiteExecutionInput {
	return execution.SuiteExecutionInput{Request: orchestrator.SuiteExecutionRequest{ScenarioName: scenario}}
}

// startRun starts a run and fails the test on a (non-busy) error, returning the
// run id. Tests that assert coalescing/rejection call m.Start directly.
func startRun(t *testing.T, m *Manager, opts StartOptions) string {
	t.Helper()
	res, err := m.Start(opts)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	return res.RunID
}

func kindSeq(replay []Event, ch <-chan Event) []string {
	kinds := make([]string, 0, len(replay)+8)
	for _, e := range replay {
		kinds = append(kinds, e.Kind)
	}
	for e := range ch {
		kinds = append(kinds, e.Kind)
	}
	return kinds
}

func contains(seq []string, want string) bool {
	for _, s := range seq {
		if s == want {
			return true
		}
	}
	return false
}

// TestStartRejectsEmptyScenario guards the one synchronous validation Start does.
func TestStartRejectsEmptyScenario(t *testing.T) {
	m := New(newFakeExecutor(""), "")
	if _, err := m.Start(StartOptions{Input: startInput("")}); err == nil {
		t.Fatal("expected error for empty scenarioName")
	}
}

// TestStartedRunSurvivesWaiterCancellation is the keystone proof: cancelling the
// context of a caller waiting on the run does NOT cancel the run.
func TestStartedRunSurvivesWaiterCancellation(t *testing.T) {
	exec := newFakeExecutor("")
	exec.result = &orchestrator.SuiteExecutionResult{ScenarioName: "demo", Success: true, Verdict: "PASS", CompletedAt: time.Now().UTC()}
	m := New(exec, "")

	runID := startRun(t, m, StartOptions{Input: startInput("demo")})
	<-exec.started

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	st, err := m.Wait(ctx, "demo", runID)
	if err == nil {
		t.Fatal("expected ctx error from cancelled waiter")
	}
	if st.Status != sharedruns.StatusInProgress {
		t.Fatalf("run must stay in_progress after waiter cancel, got %q", st.Status)
	}
	if m.lookup("demo", runID) == nil {
		t.Fatal("run must remain active after waiter cancel")
	}

	close(exec.release)
	final, err := m.Wait(context.Background(), "demo", runID)
	if err != nil {
		t.Fatalf("final wait: %v", err)
	}
	if final.Status != sharedruns.StatusPassed {
		t.Fatalf("final status = %q, want passed", final.Status)
	}
}

// TestMultipleFollowersSeeFullSequence verifies two followers of one run both
// receive the complete canonical event sequence.
func TestMultipleFollowersSeeFullSequence(t *testing.T) {
	exec := newFakeExecutor("")
	exec.events = []orchestrator.ExecutionEvent{
		phaseEvent(orchestrator.EventPhaseStart, "standards", ""),
		phaseEvent(orchestrator.EventPhaseEnd, "standards", "passed"),
	}
	exec.result = &orchestrator.SuiteExecutionResult{ScenarioName: "demo", Success: true, Verdict: "PASS", CompletedAt: time.Now().UTC()}
	m := New(exec, "")

	runID := startRun(t, m, StartOptions{Input: startInput("demo")})
	<-exec.started

	r1, c1, err := m.Follow(context.Background(), "demo", runID)
	if err != nil {
		t.Fatalf("follow 1: %v", err)
	}
	r2, c2, err := m.Follow(context.Background(), "demo", runID)
	if err != nil {
		t.Fatalf("follow 2: %v", err)
	}

	close(exec.release)

	seq1 := kindSeq(r1, c1)
	seq2 := kindSeq(r2, c2)

	for _, want := range []string{EventRunStarted, EventPhaseStarted, EventPhaseCompleted, EventRunCompleted} {
		if !contains(seq1, want) {
			t.Errorf("follower 1 missing %s: %v", want, seq1)
		}
		if !contains(seq2, want) {
			t.Errorf("follower 2 missing %s: %v", want, seq2)
		}
	}
	if seq1[len(seq1)-1] != EventRunCompleted {
		t.Errorf("follower 1 last event = %s, want run_completed", seq1[len(seq1)-1])
	}
	if seq2[len(seq2)-1] != EventRunCompleted {
		t.Errorf("follower 2 last event = %s, want run_completed", seq2[len(seq2)-1])
	}
}

// TestAbortTransitionsToAborted verifies AbortRun cancels the run, transitions
// the durable record to aborted, and stops the goroutine.
func TestAbortTransitionsToAborted(t *testing.T) {
	root := t.TempDir()
	scenarioDir := root + "/demo"
	exec := newFakeExecutor(scenarioDir)
	exec.blockOnCtx = true
	exec.result = &orchestrator.SuiteExecutionResult{ScenarioName: "demo", Success: false, Verdict: "FAIL", CompletedAt: time.Now().UTC()}
	m := New(exec, root)

	runID := startRun(t, m, StartOptions{Input: execution.SuiteExecutionInput{Request: orchestrator.SuiteExecutionRequest{ScenarioName: "demo"}}})
	<-exec.started

	st, err := m.Abort("demo", runID)
	if err != nil {
		t.Fatalf("abort: %v", err)
	}
	if st.Status != sharedruns.StatusAborted {
		t.Fatalf("abort status = %q, want aborted", st.Status)
	}

	rec, err := sharedruns.NewIndex(scenarioDir).Find(runID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if rec.Status != sharedruns.StatusAborted {
		t.Fatalf("durable status = %q, want aborted", rec.Status)
	}
}

// TestSweepDowngradesOrphansOnce verifies the startup sweep marks an orphaned
// in_progress record aborted exactly once (idempotent).
func TestSweepDowngradesOrphansOnce(t *testing.T) {
	root := t.TempDir()
	scenario := "demo"
	dir := root + "/" + scenario
	if err := sharedruns.NewIndex(dir).Append(sharedruns.RunRecord{
		RunID:     "20260101-000000-deadbeef",
		Scenario:  scenario,
		StartedAt: time.Now().UTC(),
		Status:    sharedruns.StatusInProgress,
	}); err != nil {
		t.Fatalf("seed index: %v", err)
	}

	m := New(nil, root)
	n, err := m.Sweep()
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if n != 1 {
		t.Fatalf("sweep count = %d, want 1", n)
	}
	rec, err := sharedruns.NewIndex(dir).Find("20260101-000000-deadbeef")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if rec.Status != sharedruns.StatusAborted {
		t.Fatalf("status = %q, want aborted", rec.Status)
	}

	n2, err := m.Sweep()
	if err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	if n2 != 0 {
		t.Fatalf("second sweep count = %d, want 0 (idempotent)", n2)
	}
}

// inputWith builds a run input with a preset/phases so admission-key divergence
// can be exercised.
func inputWith(scenario, preset string, phases ...string) execution.SuiteExecutionInput {
	return execution.SuiteExecutionInput{Request: orchestrator.SuiteExecutionRequest{
		ScenarioName: scenario,
		Preset:       preset,
		Phases:       phases,
	}}
}

// blockingManager returns a manager whose runs block until released, plus the
// fake executor (so tests can assert drive counts) and a cleanup that releases.
func blockingManager(t *testing.T) (*Manager, *fakeExecutor, func()) {
	t.Helper()
	exec := newFakeExecutor("")
	exec.result = &orchestrator.SuiteExecutionResult{ScenarioName: "demo", Success: true, Verdict: "PASS", CompletedAt: time.Now().UTC()}
	m := New(exec, "")
	released := false
	return m, exec, func() {
		if !released {
			released = true
			close(exec.release)
		}
	}
}

// TestStartCoalescesIdenticalInFlight proves an identical re-request rides the
// in-flight run instead of stacking a second suite (§6b row 3).
func TestStartCoalescesIdenticalInFlight(t *testing.T) {
	m, exec, release := blockingManager(t)
	defer release()

	first := startRun(t, m, StartOptions{Input: inputWith("demo", "comprehensive")})
	<-exec.started

	res, err := m.Start(StartOptions{Input: inputWith("demo", "comprehensive")})
	if err != nil {
		t.Fatalf("second start: %v", err)
	}
	if !res.Coalesced {
		t.Fatal("second identical start must coalesce")
	}
	if res.RunID != first {
		t.Fatalf("coalesced run id = %q, want in-flight %q", res.RunID, first)
	}
	if n := exec.driveCount(); n != 1 {
		t.Fatalf("drive count = %d, want 1 (no second suite)", n)
	}
}

// TestStartRejectsDivergentInFlight proves a divergent request (different
// preset) is rejected with a typed BusyError carrying the in-flight run id +
// preset, and no second suite is driven (§6b row 4).
func TestStartRejectsDivergentInFlight(t *testing.T) {
	m, exec, release := blockingManager(t)
	defer release()

	first := startRun(t, m, StartOptions{Input: inputWith("demo", "comprehensive")})
	<-exec.started

	_, err := m.Start(StartOptions{Input: inputWith("demo", "quick")})
	var busy *BusyError
	if !errors.As(err, &busy) {
		t.Fatalf("divergent start error = %v, want *BusyError", err)
	}
	if busy.RunID != first {
		t.Fatalf("busy run id = %q, want %q", busy.RunID, first)
	}
	if busy.Preset != "comprehensive" {
		t.Fatalf("busy preset = %q, want comprehensive", busy.Preset)
	}
	if busy.Scenario != "demo" {
		t.Fatalf("busy scenario = %q, want demo", busy.Scenario)
	}
	if n := exec.driveCount(); n != 1 {
		t.Fatalf("drive count = %d, want 1 (divergent rejected, no second suite)", n)
	}
}

// TestStartDifferentScenariosRunConcurrently proves the cap is per-scenario:
// different scenarios are never serialized against each other.
func TestStartDifferentScenariosRunConcurrently(t *testing.T) {
	m, exec, release := blockingManager(t)
	defer release()

	a := startRun(t, m, StartOptions{Input: inputWith("alpha", "comprehensive")})
	b := startRun(t, m, StartOptions{Input: inputWith("beta", "comprehensive")})
	if a == b {
		t.Fatal("different scenarios must get distinct runs")
	}
	// Both drives must run; wait until both ExecuteWithEvents have been entered.
	deadline := time.After(2 * time.Second)
	for exec.driveCount() < 2 {
		select {
		case <-deadline:
			t.Fatalf("drive count = %d, want 2 concurrent runs", exec.driveCount())
		case <-time.After(2 * time.Millisecond):
		}
	}
}

// TestRetireGraceLingererDoesNotBlock proves a just-finished (terminal) run
// lingering in the registry within retireGrace neither blocks nor coalesces a
// fresh start (§6b row 13).
func TestRetireGraceLingererDoesNotBlock(t *testing.T) {
	exec := newFakeExecutor("")
	exec.result = &orchestrator.SuiteExecutionResult{ScenarioName: "demo", Success: true, Verdict: "PASS", CompletedAt: time.Now().UTC()}
	m := New(exec, "")

	first := startRun(t, m, StartOptions{Input: inputWith("demo", "comprehensive")})
	<-exec.started
	close(exec.release)
	if _, err := m.Wait(context.Background(), "demo", first); err != nil {
		t.Fatalf("wait first: %v", err)
	}
	// The terminal run still lingers in the registry (retireGrace is 60s).
	if m.lookup("demo", first) == nil {
		t.Fatal("precondition: terminal run should still linger in registry")
	}

	// A fresh identical request must NOT coalesce onto the terminal lingerer.
	exec.release = make(chan struct{}) // re-arm so the second run blocks
	res, err := m.Start(StartOptions{Input: inputWith("demo", "comprehensive")})
	if err != nil {
		t.Fatalf("second start: %v", err)
	}
	if res.Coalesced {
		t.Fatal("must not coalesce onto a terminal lingering run")
	}
	if res.RunID == first {
		t.Fatal("fresh start must mint a new run id, not reuse the terminal one")
	}
	close(exec.release)
	if _, err := m.Wait(context.Background(), "demo", res.RunID); err != nil {
		t.Fatalf("wait second: %v", err)
	}
}

// TestStartAdmissionIsRaceFree fires many identical Starts concurrently and
// proves exactly one suite is driven and every caller observes the same run id
// (the scan+decide+insert is atomic under m.mu — no TOCTOU). Run under -race.
func TestStartAdmissionIsRaceFree(t *testing.T) {
	m, exec, release := blockingManager(t)
	defer release()

	const n = 24
	var wg sync.WaitGroup
	ids := make([]string, n)
	coalesced := make([]bool, n)
	errs := make([]error, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			res, err := m.Start(StartOptions{Input: inputWith("demo", "comprehensive")})
			ids[i] = res.RunID
			coalesced[i] = res.Coalesced
			errs[i] = err
		}(i)
	}
	wg.Wait()
	<-exec.started

	fresh := 0
	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("start %d errored: %v", i, errs[i])
		}
		if !coalesced[i] {
			fresh++
		}
		if ids[i] != ids[0] {
			t.Fatalf("start %d id = %q, want all equal to %q", i, ids[i], ids[0])
		}
	}
	if fresh != 1 {
		t.Fatalf("fresh (non-coalesced) starts = %d, want exactly 1", fresh)
	}
	if c := exec.driveCount(); c != 1 {
		t.Fatalf("drive count = %d, want exactly 1 suite", c)
	}
}

// TestAdmissionKeyCoalescingIdentity locks the run-key semantics: phase order is
// irrelevant (sorted), but a different phase set or preset is a distinct key.
func TestAdmissionKeyCoalescingIdentity(t *testing.T) {
	base := orchestrator.SuiteExecutionRequest{ScenarioName: "demo", Preset: "comprehensive", Phases: []string{"unit", "smoke"}}
	reordered := orchestrator.SuiteExecutionRequest{ScenarioName: "demo", Preset: "comprehensive", Phases: []string{"smoke", "unit"}}
	if admissionKey(base) != admissionKey(reordered) {
		t.Fatal("phase order must not change the admission key")
	}
	diffPhases := orchestrator.SuiteExecutionRequest{ScenarioName: "demo", Preset: "comprehensive", Phases: []string{"unit"}}
	if admissionKey(base) == admissionKey(diffPhases) {
		t.Fatal("a different phase set must produce a different admission key")
	}
	diffPreset := orchestrator.SuiteExecutionRequest{ScenarioName: "demo", Preset: "quick", Phases: []string{"unit", "smoke"}}
	if admissionKey(base) == admissionKey(diffPreset) {
		t.Fatal("a different preset must produce a different admission key")
	}
	diffScenario := orchestrator.SuiteExecutionRequest{ScenarioName: "other", Preset: "comprehensive", Phases: []string{"unit", "smoke"}}
	if admissionKey(base) == admissionKey(diffScenario) {
		t.Fatal("a different scenario must produce a different admission key")
	}
}
