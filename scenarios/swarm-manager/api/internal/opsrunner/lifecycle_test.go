package opsrunner

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/opscatalog"
)

// fakeStarter records the live start and returns a fixed run association.
type fakeStarter struct {
	started int
	handle  StartHandle
	err     error
}

func (f *fakeStarter) Start(context.Context, Prepared, RunHandle) (StartHandle, error) {
	f.started++
	if f.err != nil {
		return StartHandle{}, f.err
	}
	// Default to a non-empty run id so a zero-value fakeStarter still models a
	// successful live start; tests that need an empty run id set handle explicitly.
	if f.handle.RunID == "" && f.handle.TaskID == "" && f.handle.CreatedAt == "" {
		return StartHandle{RunID: "run-default"}, nil
	}
	return f.handle, nil
}

// newRunnerWithStarter assembles a runner whose live path routes through a
// starter (non-blocking Invoke) instead of the synchronous driver.
func newRunnerWithStarter(t *testing.T, catalog *opscatalog.Catalog, storeRoot string, prep ModePreparer, starter RunStarter, owners RunOwnerIndex) (*Runner, *WorkflowRepo, *ExecutionStore) {
	t.Helper()
	loc := memLocator{root: storeRoot}
	repo := NewWorkflowRepo(loc)
	execStore := NewExecutionStore(loc)
	resolver := NewBindingResolver(catalog, NewFSOverrideStore(loc), preparerChecker{prep})
	dispatcher := NewDispatcher(NewActionRegistry(), repo)
	r, err := New(Config{
		Catalog: catalog, Resolver: resolver, Preparer: prep,
		Driver: fakeDriver{outcome: "accepted", disposition: "success"}, Starter: starter,
		Repo: repo, Executions: execStore, Dispatcher: dispatcher, RunOwners: owners,
	})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	return r, repo, execStore
}

func reviewTarget() TargetRef {
	return TargetRef{Kind: agentops.TargetBacklogItem, ID: "feature/foo"}
}

func liveInvoke(t *testing.T, r *Runner) OperationResult {
	t.Helper()
	res, err := r.Invoke(context.Background(), InvokeRequest{
		Target: reviewTarget(), Operation: agentops.OpReviewRound,
		CallerInputs: map[string]any{}, RequestedBy: "test",
	})
	if err != nil {
		t.Fatalf("live Invoke: %v", err)
	}
	return res
}

// TestLiveInvokeIsNonBlockingStart proves that a live Invoke starts the run,
// returns a StartHandle, leaves the workflow running, and fires NO transition.
func TestLiveInvokeIsNonBlockingStart(t *testing.T) {
	dir := t.TempDir()
	catalog := writeCatalog(t, dir, "rev-1")
	starter := &fakeStarter{handle: StartHandle{RunID: "run-xyz", TaskID: "task-xyz"}}
	r, repo, _ := newRunnerWithStarter(t, catalog, dir, fakePreparer{}, starter, &memRunOwners{})

	res := liveInvoke(t, r)

	if starter.started != 1 {
		t.Fatalf("expected exactly one live start, got %d", starter.started)
	}
	if res.StartHandle == nil || res.StartHandle.RunID != "run-xyz" {
		t.Fatalf("expected start handle with run id, got %+v", res.StartHandle)
	}
	if res.Outcome != "" || res.Action != "" {
		t.Fatalf("live start must not classify or act: outcome=%q action=%q", res.Outcome, res.Action)
	}
	if res.WorkflowState != agentops.WorkflowRunning {
		t.Fatalf("expected running workflow, got %q", res.WorkflowState)
	}
	w, _, err := repo.Load(reviewTarget().Kind, reviewTarget().ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Operations) != 1 || w.Operations[0].State != "running" {
		t.Fatalf("expected one running operation record, got %+v", w.Operations)
	}
}

// TestLiveInvokeEmptyRunIDFailsClosedAndReaps proves that a live start reporting
// success with no run id fails closed (ErrNoRunID) and reaps the operation so it
// does not linger "running" — the runner never returns a phantom untrackable
// operation.
func TestLiveInvokeEmptyRunIDFailsClosedAndReaps(t *testing.T) {
	dir := t.TempDir()
	catalog := writeCatalog(t, dir, "rev-1")
	// handle explicitly carries a non-run-id field so the default is not applied,
	// but RunID stays empty — the case a degraded engine/agent produces.
	starter := &fakeStarter{handle: StartHandle{CreatedAt: "2026-07-14T00:00:00Z"}}
	r, repo, _ := newRunnerWithStarter(t, catalog, dir, fakePreparer{}, starter, &memRunOwners{})

	_, err := r.Invoke(context.Background(), InvokeRequest{
		Target: reviewTarget(), Operation: agentops.OpReviewRound,
		CallerInputs: map[string]any{}, RequestedBy: "test",
	})
	if !errors.Is(err, ErrNoRunID) {
		t.Fatalf("expected ErrNoRunID for a run-id-less live start, got %v", err)
	}
	w, _, lerr := repo.Load(reviewTarget().Kind, reviewTarget().ID)
	if lerr != nil {
		t.Fatal(lerr)
	}
	if len(w.Operations) != 1 || w.Operations[0].State != "failed" {
		t.Fatalf("expected the run-id-less operation reaped to failed, got %+v", w.Operations)
	}
}

// TestCancelExecutionReapsRunningOperation proves that CancelExecution marks a
// running operation canceled (so it does not linger after its agent run was
// stopped) and is idempotent / a no-op for unknown executions.
func TestCancelExecutionReapsRunningOperation(t *testing.T) {
	dir := t.TempDir()
	catalog := writeCatalog(t, dir, "rev-1")
	starter := &fakeStarter{handle: StartHandle{RunID: "run-c"}}
	r, repo, _ := newRunnerWithStarter(t, catalog, dir, fakePreparer{}, starter, &memRunOwners{})

	start := liveInvoke(t, r)

	if err := r.CancelExecution(context.Background(), reviewTarget(), start.ExecutionID); err != nil {
		t.Fatalf("CancelExecution: %v", err)
	}
	w, _, err := repo.Load(reviewTarget().Kind, reviewTarget().ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Operations) != 1 || w.Operations[0].State != "canceled" {
		t.Fatalf("expected the operation reaped to canceled, got %+v", w.Operations)
	}
	// Idempotent: a second cancel of the now-terminal execution is a no-op.
	if err := r.CancelExecution(context.Background(), reviewTarget(), start.ExecutionID); err != nil {
		t.Fatalf("second CancelExecution: %v", err)
	}
	// Unknown execution id is a no-op, not an error.
	if err := r.CancelExecution(context.Background(), reviewTarget(), "does-not-exist"); err != nil {
		t.Fatalf("unknown CancelExecution: %v", err)
	}
}

// TestCommitResultRecordsOutcomeAndTransitions proves that delivering a valid
// result finalizes the running operation and fires the policy transition.
func TestCommitResultRecordsOutcomeAndTransitions(t *testing.T) {
	dir := t.TempDir()
	catalog := writeCatalog(t, dir, "rev-1")
	starter := &fakeStarter{handle: StartHandle{RunID: "run-1"}}
	r, repo, _ := newRunnerWithStarter(t, catalog, dir, fakePreparer{}, starter, &memRunOwners{})

	start := liveInvoke(t, r)

	commit, err := r.CommitResult(context.Background(), CommitRequest{
		Target: reviewTarget(), ExecutionID: start.ExecutionID, Outcome: "accepted",
		DeliveredResult: json.RawMessage(`{"verdict":"accepted","handoff":{"summary":"ok"}}`),
	})
	if err != nil {
		t.Fatalf("CommitResult: %v", err)
	}
	if commit.Action != agentops.ActionOpenReview {
		t.Fatalf("expected open-review action, got %q", commit.Action)
	}
	if commit.WorkflowState != agentops.WorkflowAwaitingDecision {
		t.Fatalf("expected awaiting-decision, got %q", commit.WorkflowState)
	}
	w, _, err := repo.Load(reviewTarget().Kind, reviewTarget().ID)
	if err != nil {
		t.Fatal(err)
	}
	if w.Operations[0].State != "completed" || w.Operations[0].Outcome != "accepted" {
		t.Fatalf("expected completed/accepted op record, got %+v", w.Operations[0])
	}
}

// TestCommitResultIsIdempotent proves a second commit for the same execution is
// a no-op replay that does not re-fire the transition.
func TestCommitResultIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	catalog := writeCatalog(t, dir, "rev-1")
	r, repo, _ := newRunnerWithStarter(t, catalog, dir, fakePreparer{}, &fakeStarter{}, &memRunOwners{})
	start := liveInvoke(t, r)

	req := CommitRequest{
		Target: reviewTarget(), ExecutionID: start.ExecutionID, Outcome: "accepted",
		DeliveredResult: json.RawMessage(`{"verdict":"accepted","handoff":{}}`),
	}
	if _, err := r.CommitResult(context.Background(), req); err != nil {
		t.Fatalf("first commit: %v", err)
	}
	w1, _, _ := repo.Load(reviewTarget().Kind, reviewTarget().ID)

	second, err := r.CommitResult(context.Background(), req)
	if err != nil {
		t.Fatalf("second commit: %v", err)
	}
	if !second.Replayed {
		t.Fatalf("expected replayed second commit")
	}
	w2, _, _ := repo.Load(reviewTarget().Kind, reviewTarget().ID)
	if w2.Version != w1.Version {
		t.Fatalf("idempotent commit mutated workflow: v%d -> v%d", w1.Version, w2.Version)
	}
}

// TestCommitResultUndeclaredOutcomeFailsClosed proves an outcome the contract
// does not declare is rejected without mutating workflow state.
func TestCommitResultUndeclaredOutcomeFailsClosed(t *testing.T) {
	dir := t.TempDir()
	catalog := writeCatalog(t, dir, "rev-1")
	r, repo, _ := newRunnerWithStarter(t, catalog, dir, fakePreparer{}, &fakeStarter{}, &memRunOwners{})
	start := liveInvoke(t, r)
	before, _, _ := repo.Load(reviewTarget().Kind, reviewTarget().ID)

	_, err := r.CommitResult(context.Background(), CommitRequest{
		Target: reviewTarget(), ExecutionID: start.ExecutionID, Outcome: "not-a-real-outcome",
		DeliveredResult: json.RawMessage(`{"verdict":"accepted","handoff":{}}`),
	})
	if !errors.Is(err, ErrUndeclaredOutcome) {
		t.Fatalf("expected ErrUndeclaredOutcome, got %v", err)
	}
	after, _, _ := repo.Load(reviewTarget().Kind, reviewTarget().ID)
	if after.Version != before.Version || after.State != before.State {
		t.Fatalf("undeclared outcome mutated state: %+v -> %+v", before.State, after.State)
	}
}

// TestCommitResultMalformedResultFailsClosed proves a non-abstaining outcome
// with a missing required field is rejected and preserves the running state
// (the round's domain artifacts survive for recovery).
func TestCommitResultMalformedResultFailsClosed(t *testing.T) {
	dir := t.TempDir()
	catalog := writeCatalog(t, dir, "rev-1")
	r, repo, _ := newRunnerWithStarter(t, catalog, dir, fakePreparer{}, &fakeStarter{}, &memRunOwners{})
	start := liveInvoke(t, r)

	// "accepted" requires verdict+handoff; omit handoff.
	_, err := r.CommitResult(context.Background(), CommitRequest{
		Target: reviewTarget(), ExecutionID: start.ExecutionID, Outcome: "accepted",
		DeliveredResult: json.RawMessage(`{"verdict":"accepted"}`),
	})
	if !errors.Is(err, ErrInvalidResult) {
		t.Fatalf("expected ErrInvalidResult, got %v", err)
	}
	w, _, _ := repo.Load(reviewTarget().Kind, reviewTarget().ID)
	if w.Operations[0].State != "running" {
		t.Fatalf("malformed result must leave op running, got %q", w.Operations[0].State)
	}
}

// TestCommitResultAbstainAcceptsPartialResult proves the abstain (needs-attention)
// path records even when the round output is unparseable/partial — the whole
// point of the abstain outcome.
func TestCommitResultAbstainAcceptsPartialResult(t *testing.T) {
	dir := t.TempDir()
	catalog := writeCatalog(t, dir, "rev-1")
	r, repo, _ := newRunnerWithStarter(t, catalog, dir, fakePreparer{}, &fakeStarter{}, &memRunOwners{})
	start := liveInvoke(t, r)

	commit, err := r.CommitResult(context.Background(), CommitRequest{
		Target: reviewTarget(), ExecutionID: start.ExecutionID, Outcome: "needs-attention",
		DeliveredResult: nil,
	})
	if err != nil {
		t.Fatalf("abstain commit: %v", err)
	}
	if commit.Disposition != "abstain" {
		t.Fatalf("expected abstain disposition, got %q", commit.Disposition)
	}
	w, _, _ := repo.Load(reviewTarget().Kind, reviewTarget().ID)
	if w.Operations[0].State != "needs-attention" {
		t.Fatalf("expected needs-attention op state, got %q", w.Operations[0].State)
	}
}

// TestCommitResultUnknownExecutionFailsClosed proves a result for an execution
// the workflow never started is rejected.
func TestCommitResultUnknownExecutionFailsClosed(t *testing.T) {
	dir := t.TempDir()
	catalog := writeCatalog(t, dir, "rev-1")
	r, _, _ := newRunnerWithStarter(t, catalog, dir, fakePreparer{}, &fakeStarter{}, &memRunOwners{})
	liveInvoke(t, r)

	_, err := r.CommitResult(context.Background(), CommitRequest{
		Target: reviewTarget(), ExecutionID: "exec-does-not-exist", Outcome: "accepted",
		DeliveredResult: json.RawMessage(`{"verdict":"accepted","handoff":{}}`),
	})
	if !errors.Is(err, ErrUnknownExecution) {
		t.Fatalf("expected ErrUnknownExecution, got %v", err)
	}
}

// TestLiveStartPersistsRunIDForCompletionMapping proves the non-blocking live
// start records its agent run id on the operation record, and that a delivered
// round carrying only that run id resolves back to the execution (the completion
// seam's mapping) and finalizes via CommitResult.
func TestLiveStartPersistsRunIDForCompletionMapping(t *testing.T) {
	dir := t.TempDir()
	catalog := writeCatalog(t, dir, "rev-1")
	starter := &fakeStarter{handle: StartHandle{RunID: "run-owner-7"}}
	r, repo, _ := newRunnerWithStarter(t, catalog, dir, fakePreparer{}, starter, &memRunOwners{})

	liveInvoke(t, r)

	w, _, err := repo.Load(reviewTarget().Kind, reviewTarget().ID)
	if err != nil {
		t.Fatal(err)
	}
	if w.Operations[0].RunID != "run-owner-7" {
		t.Fatalf("expected run id persisted on op record, got %q", w.Operations[0].RunID)
	}
	// The completion seam only knows the delivered round's run id; it must map
	// that back to the owning execution.
	op, ok := FindOperationByRunID(w, "run-owner-7")
	if !ok {
		t.Fatalf("FindOperationByRunID did not resolve the live run id")
	}
	if _, ok := FindOperationByRunID(w, "some-other-run"); ok {
		t.Fatalf("FindOperationByRunID must not match an unrelated run id")
	}

	commit, err := r.CommitResult(context.Background(), CommitRequest{
		Target: reviewTarget(), ExecutionID: op.ExecutionID, Outcome: "accepted",
		DeliveredResult: json.RawMessage(`{"verdict":"accepted","handoff":{"summary":"ok"}}`),
	})
	if err != nil {
		t.Fatalf("CommitResult via run-id-resolved execution: %v", err)
	}
	if commit.Action != agentops.ActionOpenReview {
		t.Fatalf("expected open-review action, got %q", commit.Action)
	}
}

// TestLiveStartFailurePropagates proves a starter failure surfaces as an error
// (the caller maps it to a typed API error; the item is not corrupted).
func TestLiveStartFailurePropagates(t *testing.T) {
	dir := t.TempDir()
	catalog := writeCatalog(t, dir, "rev-1")
	starter := &fakeStarter{err: errors.New("agent-manager unavailable")}
	r, _, _ := newRunnerWithStarter(t, catalog, dir, fakePreparer{}, starter, &memRunOwners{})

	_, err := r.Invoke(context.Background(), InvokeRequest{
		Target: reviewTarget(), Operation: agentops.OpReviewRound, CallerInputs: map[string]any{},
	})
	if err == nil {
		t.Fatalf("expected start failure to propagate")
	}
}
