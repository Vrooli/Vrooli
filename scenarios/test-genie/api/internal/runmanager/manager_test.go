package runmanager

import (
	"context"
	"sync"
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
}

func newFakeExecutor(scenarioDir string) *fakeExecutor {
	return &fakeExecutor{
		scenarioDir: scenarioDir,
		release:     make(chan struct{}),
		started:     make(chan struct{}),
	}
}

func (f *fakeExecutor) ExecuteWithEvents(ctx context.Context, input execution.SuiteExecutionInput, emit orchestrator.ExecutionEventCallback) (*orchestrator.SuiteExecutionResult, error) {
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

	runID, err := m.Start(StartOptions{Input: startInput("demo")})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
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

	runID, err := m.Start(StartOptions{Input: startInput("demo")})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
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

	runID, err := m.Start(StartOptions{Input: execution.SuiteExecutionInput{Request: orchestrator.SuiteExecutionRequest{ScenarioName: "demo"}}})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
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
