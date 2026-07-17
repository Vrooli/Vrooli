package runmanager

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	sharedartifacts "test-genie/internal/shared/artifacts"
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
	// returnErr simulates the orchestrator erroring AFTER it appended the
	// in_progress record but BEFORE it finalized (e.g. the target scenario never
	// came up). The terminal durable Update is skipped, exactly like the real
	// early-return path, so a test can prove drive() reconciles the record itself.
	returnErr error

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

	if f.returnErr != nil {
		// Early return without finalizing the durable record (skipped-finalize
		// path). drive() is responsible for reconciling it to a terminal state.
		return nil, f.returnErr
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
	if catalog, err := sharedartifacts.ReadArtifactCatalog(scenarioDir, runID); err != nil || catalog.LegacyDiscovered {
		t.Fatalf("aborted run artifact catalog = %+v err=%v", catalog, err)
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

// waitForStatus polls a run's in-memory status until it equals want or the
// deadline elapses. Used where promotion happens on a background goroutine.
func waitForStatus(t *testing.T, m *Manager, scenario, runID, want string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if st, err := m.Status(scenario, runID); err == nil && st.Status == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	st, _ := m.Status(scenario, runID)
	t.Fatalf("run %s/%s status = %q, want %q", scenario, runID, st.Status, want)
}

func waitForDriveCount(t *testing.T, exec *fakeExecutor, want int32) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if exec.driveCount() == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("drive count = %d, want %d", exec.driveCount(), want)
}

// TestGlobalCapQueuesBeyondLimit verifies that once the global concurrency cap is
// reached, a run for a DIFFERENT scenario is admitted as queued (not rejected,
// not driven) and persisted so it is visible in the durable index.
func TestGlobalCapQueuesBeyondLimit(t *testing.T) {
	root := t.TempDir()
	exec := newFakeExecutor("")
	exec.blockOnCtx = true
	m := New(exec, root)
	m.maxConcurrentRuns = 1
	defer m.Shutdown()

	aID := startRun(t, m, StartOptions{Input: startInput("a")})
	if st, _ := m.Status("a", aID); st.Status != sharedruns.StatusInProgress {
		t.Fatalf("run a status = %q, want in_progress", st.Status)
	}

	bID := startRun(t, m, StartOptions{Input: startInput("b")})
	if st, _ := m.Status("b", bID); st.Status != sharedruns.StatusQueued {
		t.Fatalf("run b status = %q, want queued", st.Status)
	}
	// Only one suite has actually been driven.
	if c := exec.driveCount(); c != 1 {
		t.Fatalf("drive count = %d, want 1 (queued run must not drive)", c)
	}
	// The queued run is persisted for visibility in `runs list`.
	rec, err := sharedruns.NewIndex(root + "/b").Find(bID)
	if err != nil {
		t.Fatalf("queued run not in durable index: %v", err)
	}
	if rec.Status != sharedruns.StatusQueued {
		t.Fatalf("durable status = %q, want queued", rec.Status)
	}
}

func TestPreviewAdmissionIsNonBlockingAndBounded(t *testing.T) {
	m := New(newFakeExecutor(""), t.TempDir())
	m.previewTokens = make(chan struct{}, 1)
	m.maxPreviewPerCaller = 2 // exercise the global gate in this test
	release, err := m.TryAcquirePreview()
	if err != nil {
		t.Fatalf("first TryAcquirePreview: %v", err)
	}
	if _, err := m.TryAcquirePreview(); err == nil {
		t.Fatal("second TryAcquirePreview succeeded with a full gate")
	} else {
		var saturated *SaturatedError
		if !errors.As(err, &saturated) || saturated.Limit != "preview capacity" {
			t.Fatalf("preview saturation = %v, want preview-capacity SaturatedError", err)
		}
	}
	release()
	if release, err := m.TryAcquirePreview(); err != nil {
		t.Fatalf("acquire after release: %v", err)
	} else {
		release()
	}
}

func TestAdmissionStatusReportsOccupancyLimitsAndOutcomes(t *testing.T) {
	fake := newFakeExecutor(t.TempDir())
	m := New(fake, t.TempDir())
	m.maxConcurrentRuns = 1
	m.maxQueuedRuns = 1
	m.previewTokens = make(chan struct{}, 1)
	defer m.Shutdown()

	if _, err := m.Start(StartOptions{Input: inputWith("alpha", "")}); err != nil {
		t.Fatalf("start alpha: %v", err)
	}
	<-fake.started
	if _, err := m.Start(StartOptions{Input: inputWith("beta", "")}); err != nil {
		t.Fatalf("start beta: %v", err)
	}
	if _, err := m.Start(StartOptions{Input: inputWith("gamma", "")}); err == nil {
		t.Fatal("start gamma succeeded despite full queue")
	}
	release, err := m.TryAcquirePreview()
	if err != nil {
		t.Fatalf("acquire preview: %v", err)
	}
	defer release()
	if _, err := m.TryAcquirePreview(); err == nil {
		t.Fatal("second preview acquire succeeded despite full gate")
	}
	if _, err := m.Start(StartOptions{Input: inputWith("alpha", "")}); err != nil {
		t.Fatalf("coalesced alpha: %v", err)
	}

	status := m.AdmissionStatus()
	if status.Running != 1 || status.Queued != 1 || status.PreviewInFlight != 1 {
		t.Fatalf("occupancy = %#v, want running=1 queued=1 preview=1", status)
	}
	if status.MaxConcurrentRuns != 1 || status.MaxQueuedRuns != 1 || status.MaxPreviewRuns != 1 {
		t.Fatalf("limits = %#v, want all 1", status)
	}
	if status.QueueRejectedTotal != 1 || status.PreviewRejectedTotal != 1 || status.CoalescedTotal != 1 {
		t.Fatalf("outcomes = %#v, want one queue reject, preview reject, and coalesce", status)
	}
	close(fake.release)
}

func TestCoalescedRunBypassesPreviewSaturation(t *testing.T) {
	fake := newFakeExecutor(t.TempDir())
	m := New(fake, t.TempDir())
	firstInput := inputWith("alpha", "")
	first, err := m.Start(StartOptions{Input: firstInput})
	if err != nil {
		t.Fatalf("start first: %v", err)
	}
	<-fake.started
	m.previewTokens = make(chan struct{}, 1)
	release, err := m.TryAcquirePreview()
	if err != nil {
		t.Fatalf("fill preview gate: %v", err)
	}
	defer release()
	if got := m.CoalescedRunID(firstInput.Request); got != first.RunID {
		t.Fatalf("CoalescedRunID = %q, want %q while preview gate is full", got, first.RunID)
	}
	close(fake.release)
	m.Shutdown()
}

func TestGlobalQueuedAdmissionIsBounded(t *testing.T) {
	ctx := context.Background()
	fake := newFakeExecutor(t.TempDir())
	m := New(fake, t.TempDir())
	m.maxConcurrentRuns = 1
	m.maxQueuedRuns = 1
	if _, err := m.Start(StartOptions{Input: inputWith("alpha", "")}); err != nil {
		t.Fatalf("start alpha: %v", err)
	}
	<-fake.started
	if _, err := m.Start(StartOptions{Input: inputWith("beta", "")}); err != nil {
		t.Fatalf("start beta queued: %v", err)
	}
	if _, err := m.Start(StartOptions{Input: inputWith("gamma", "")}); err == nil {
		t.Fatal("start gamma succeeded despite a full queued admission cap")
	} else {
		var saturated *SaturatedError
		if !errors.As(err, &saturated) || saturated.Limit != "queued run capacity" {
			t.Fatalf("queue saturation = %v", err)
		}
	}
	_ = ctx
	close(fake.release)
	m.Shutdown()
}

func TestCallerQueueLimitDoesNotStarveOtherCallers(t *testing.T) {
	fake := newFakeExecutor(t.TempDir())
	m := New(fake, t.TempDir())
	m.maxConcurrentRuns = 1
	m.maxQueuedRuns = 3
	m.maxQueuedPerCaller = 1
	defer m.Shutdown()

	if _, err := m.Start(StartOptions{Input: inputWith("running", ""), Caller: "alice"}); err != nil {
		t.Fatalf("start running: %v", err)
	}
	<-fake.started
	if _, err := m.Start(StartOptions{Input: inputWith("alice-queued", ""), Caller: "alice"}); err != nil {
		t.Fatalf("queue alice: %v", err)
	}
	if _, err := m.Start(StartOptions{Input: inputWith("alice-rejected", ""), Caller: "alice"}); err == nil {
		t.Fatal("second queued run for alice succeeded")
	} else {
		var saturated *SaturatedError
		if !errors.As(err, &saturated) || saturated.Limit != "caller queued run capacity" {
			t.Fatalf("alice saturation = %v", err)
		}
	}
	if _, err := m.Start(StartOptions{Input: inputWith("bob-queued", ""), Caller: "bob"}); err != nil {
		t.Fatalf("queue bob despite alice cap: %v", err)
	}
	close(fake.release)
}

func TestCallerPreviewLimitDoesNotStarveOtherCallers(t *testing.T) {
	m := New(newFakeExecutor(""), t.TempDir())
	m.previewTokens = make(chan struct{}, 2)
	m.maxPreviewPerCaller = 1
	aliceRelease, err := m.TryAcquirePreviewFor("alice")
	if err != nil {
		t.Fatalf("acquire alice preview: %v", err)
	}
	defer aliceRelease()
	if _, err := m.TryAcquirePreviewFor("alice"); err == nil {
		t.Fatal("second alice preview succeeded")
	}
	bobRelease, err := m.TryAcquirePreviewFor("bob")
	if err != nil {
		t.Fatalf("bob preview was starved by alice: %v", err)
	}
	defer bobRelease()
}

func TestMixedBurstKeepsAdmissionBounded(t *testing.T) {
	fake := newFakeExecutor(t.TempDir())
	m := New(fake, t.TempDir())
	m.maxConcurrentRuns = 2
	m.maxQueuedRuns = 16
	m.maxQueuedPerCaller = 16
	defer m.Shutdown()

	const requests = 288
	var wg sync.WaitGroup
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Unique scenarios prevent per-scenario coalescing from masking the
			// global queue bound; callers rotate to exercise fair-share state.
			_, _ = m.Start(StartOptions{Input: inputWith(fmt.Sprintf("burst-%03d", i), ""), Caller: fmt.Sprintf("caller-%d", i%8)})
		}(i)
	}
	wg.Wait()
	status := m.AdmissionStatus()
	if status.Running > m.maxConcurrentRuns || status.Queued > m.maxQueuedRuns {
		t.Fatalf("burst exceeded admission bounds: %#v", status)
	}
	if status.QueueRejectedTotal < uint64(requests-m.maxConcurrentRuns-m.maxQueuedRuns) {
		t.Fatalf("queue rejections = %d, want at least %d", status.QueueRejectedTotal, requests-m.maxConcurrentRuns-m.maxQueuedRuns)
	}
	close(fake.release)
}

// TestQueuedRunPromotedOnSlotFree verifies the dispatcher promotes a queued run
// to in_progress (and drives it) when a running run completes.
func TestQueuedRunPromotedOnSlotFree(t *testing.T) {
	root := t.TempDir()
	exec := newFakeExecutor("")
	exec.blockOnCtx = true
	m := New(exec, root)
	m.maxConcurrentRuns = 1
	defer m.Shutdown()

	aID := startRun(t, m, StartOptions{Input: startInput("a")})
	bID := startRun(t, m, StartOptions{Input: startInput("b")})
	if st, _ := m.Status("b", bID); st.Status != sharedruns.StatusQueued {
		t.Fatalf("run b status = %q, want queued", st.Status)
	}

	// Free the only slot; dispatch must promote b.
	if _, err := m.Abort("a", aID); err != nil {
		t.Fatalf("abort a: %v", err)
	}
	waitForStatus(t, m, "b", bID, sharedruns.StatusInProgress)
	waitForDriveCount(t, exec, 2)
}

// TestAbortQueuedRun verifies a queued run can be aborted directly (it has no
// executor goroutine) without consuming a slot or blocking.
func TestAbortQueuedRun(t *testing.T) {
	root := t.TempDir()
	exec := newFakeExecutor("")
	exec.blockOnCtx = true
	m := New(exec, root)
	m.maxConcurrentRuns = 1
	defer m.Shutdown()

	_ = startRun(t, m, StartOptions{Input: startInput("a")})
	bID := startRun(t, m, StartOptions{Input: startInput("b")})

	st, err := m.Abort("b", bID)
	if err != nil {
		t.Fatalf("abort queued: %v", err)
	}
	if st.Status != sharedruns.StatusAborted {
		t.Fatalf("aborted-queued status = %q, want aborted", st.Status)
	}
	rec, err := sharedruns.NewIndex(root + "/b").Find(bID)
	if err != nil {
		t.Fatalf("find b: %v", err)
	}
	if rec.Status != sharedruns.StatusAborted {
		t.Fatalf("durable status = %q, want aborted", rec.Status)
	}
	// The queued run never drove, and a was untouched (still the lone running run).
	if c := exec.driveCount(); c != 1 {
		t.Fatalf("drive count = %d, want 1", c)
	}
}

// TestAbortOrphanDowngradesIndex is the regression guard for the 2026-06-21
// incident: an in_progress durable record with NO live registry counterpart must
// be downgraded to aborted by Abort, not read back as a permanently-stuck
// in_progress (the startup Sweep otherwise only fires at boot).
func TestAbortOrphanDowngradesIndex(t *testing.T) {
	root := t.TempDir()
	scenario := "demo"
	runID := "20260621-163500-6f7a5722"
	if err := sharedruns.NewIndex(root + "/" + scenario).Append(sharedruns.RunRecord{
		RunID:     runID,
		Scenario:  scenario,
		StartedAt: time.Now().UTC(),
		Status:    sharedruns.StatusInProgress,
	}); err != nil {
		t.Fatalf("seed orphan: %v", err)
	}

	m := New(nil, root)
	st, err := m.Abort(scenario, runID)
	if err != nil {
		t.Fatalf("abort orphan: %v", err)
	}
	if st.Status != sharedruns.StatusAborted {
		t.Fatalf("orphan abort status = %q, want aborted", st.Status)
	}
	rec, err := sharedruns.NewIndex(root + "/" + scenario).Find(runID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if rec.Status != sharedruns.StatusAborted {
		t.Fatalf("durable status = %q, want aborted (orphan must be recoverable without restart)", rec.Status)
	}
}

func TestWaitOrphanDowngradesIndexInsteadOfReturningInProgress(t *testing.T) {
	root := t.TempDir()
	scenario := "demo"
	runID := "20260716-190415-371abe2e"
	if err := sharedruns.NewIndex(root + "/" + scenario).Append(sharedruns.RunRecord{
		RunID:     runID,
		Scenario:  scenario,
		StartedAt: time.Now().UTC(),
		Status:    sharedruns.StatusInProgress,
	}); err != nil {
		t.Fatalf("seed orphan: %v", err)
	}

	m := New(nil, root)
	st, err := m.Wait(context.Background(), scenario, runID)
	if err != nil {
		t.Fatalf("wait orphan: %v", err)
	}
	if st.Status != sharedruns.StatusAborted {
		t.Fatalf("wait orphan status = %q, want aborted", st.Status)
	}
	rec, err := sharedruns.NewIndex(root + "/" + scenario).Find(runID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if rec.Status != sharedruns.StatusAborted {
		t.Fatalf("durable status = %q, want aborted (wait must not return a stranded in-progress run)", rec.Status)
	}
}

func TestStatusOrphanDowngradesIndexInsteadOfReturningInProgress(t *testing.T) {
	root := t.TempDir()
	scenario := "demo"
	runID := "20260716-200000-371abe2e"
	if err := sharedruns.NewIndex(root + "/" + scenario).Append(sharedruns.RunRecord{
		RunID:     runID,
		Scenario:  scenario,
		StartedAt: time.Now().UTC(),
		Status:    sharedruns.StatusInProgress,
	}); err != nil {
		t.Fatalf("seed orphan: %v", err)
	}

	m := New(nil, root)
	st, err := m.Status(scenario, runID)
	if err != nil {
		t.Fatalf("status orphan: %v", err)
	}
	if st.Status != sharedruns.StatusAborted {
		t.Fatalf("orphan status = %q, want aborted", st.Status)
	}
}

// TestFailedRunReconcilesDurableRecord is the regression guard for the orphan
// factory: when the executor returns an error before the orchestrator finalizes
// (leaving the durable record at in_progress), drive() must reconcile it to a
// terminal status — not leave it to linger as an orphan after retire.
func TestFailedRunReconcilesDurableRecord(t *testing.T) {
	root := t.TempDir()
	scenarioDir := root + "/demo"
	exec := newFakeExecutor(scenarioDir)
	exec.returnErr = errors.New("timeout waiting for target scenario runtime URLs")
	m := New(exec, root)
	defer m.Shutdown()

	runID := startRun(t, m, StartOptions{Input: startInput("demo")})
	// Wait for the run to leave in_progress (drive sets failed, reconciles record).
	waitForStatus(t, m, "demo", runID, sharedruns.StatusFailed)

	rec, err := sharedruns.NewIndex(scenarioDir).Find(runID)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if rec.Status != sharedruns.StatusFailed {
		t.Fatalf("durable status = %q, want failed (must not orphan at in_progress)", rec.Status)
	}
	if rec.CompletedAt.IsZero() {
		t.Fatal("reconciled record must carry a CompletedAt")
	}
}

func TestWaitHydratesCanonicalTerminalSnapshotAfterRestart(t *testing.T) { // [REQ:TESTGENIE-RUN-SNAPSHOT-P0]
	root := t.TempDir()
	scenario := "demo"
	runID := "20260710-142937-ae6a753e"
	started := time.Now().UTC().Add(-20 * time.Second).Truncate(time.Second)
	completed := started.Add(15 * time.Second)
	idx := sharedruns.NewIndex(root + "/" + scenario)
	descriptorSnapshot, err := sharedruns.NewDescriptorSnapshot([]sharedruns.PhaseDescriptorSnapshot{{
		Phase: "unit", DisplayName: "Unit", Applicability: sharedruns.ApplicabilityDecisionSnapshot{Status: "applies", Planned: true},
	}})
	if err != nil {
		t.Fatalf("build descriptor snapshot: %v", err)
	}
	if err := sharedruns.WriteDescriptorSnapshot(root+"/"+scenario, runID, descriptorSnapshot); err != nil {
		t.Fatalf("write descriptor snapshot: %v", err)
	}
	if err := idx.Append(sharedruns.RunRecord{
		RunID:                           runID,
		Scenario:                        scenario,
		StartedAt:                       started,
		Status:                          sharedruns.StatusInProgress,
		DescriptorSnapshotSchemaVersion: descriptorSnapshot.SchemaVersion,
		DescriptorSnapshotDigest:        descriptorSnapshot.Digest,
	}); err != nil {
		t.Fatalf("append: %v", err)
	}
	result := &orchestrator.SuiteExecutionResult{
		RunID:        runID,
		ScenarioName: scenario,
		StartedAt:    started,
		CompletedAt:  completed,
		Success:      false,
		Verdict:      "FAIL",
		Phases: []orchestrator.PhaseExecutionResult{
			{Name: "unit", Status: "failed", DurationSeconds: 15},
		},
	}
	if err := idx.Finalize(runID, result, func(r *sharedruns.RunRecord) error {
		r.Status = sharedruns.StatusFailed
		r.CompletedAt = completed
		r.Phases = []sharedruns.PhaseRecord{{Name: "unit", Status: "failed", DurationSeconds: 15}}
		return nil
	}); err != nil {
		t.Fatalf("finalize: %v", err)
	}

	// A new manager has no live registry state; Wait must hydrate the same full
	// terminal truth from the persisted snapshot rather than return zero phases.
	m := New(nil, root)
	status, err := m.Wait(context.Background(), scenario, runID)
	if err != nil {
		t.Fatalf("wait after restart: %v", err)
	}
	if status.Result == nil || len(status.Result.Phases) != 1 {
		t.Fatalf("terminal result phases = %+v, want one persisted phase", status.Result)
	}
	if got := status.Result.Phases[0]; got.Name != "unit" || got.Status != "failed" || got.DurationSeconds != 15 {
		t.Fatalf("persisted phase = %+v", got)
	}
	if status.Verdict != "FAIL" || status.ElapsedSeconds != 15 || len(status.DegradedReasons) != 0 {
		t.Fatalf("hydrated status = %+v", status)
	}
}

func TestWaitMarksLegacyTerminalProjectionDegraded(t *testing.T) {
	root := t.TempDir()
	scenario := "demo"
	runID := "legacy-run"
	if err := sharedruns.NewIndex(root + "/" + scenario).Append(sharedruns.RunRecord{
		RunID: runID, Scenario: scenario, StartedAt: time.Now().UTC(), Status: sharedruns.StatusPassed,
	}); err != nil {
		t.Fatalf("append: %v", err)
	}
	status, err := New(nil, root).Wait(context.Background(), scenario, runID)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if status.Result != nil || len(status.DegradedReasons) != 2 {
		t.Fatalf("legacy status must be explicit degraded evidence: %+v", status)
	}
}
