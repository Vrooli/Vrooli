package opsbridge

import (
	"context"
	"errors"
	"testing"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/operatingmode"
	"swarm-manager/internal/opsrunner"
)

type fakeLoader struct {
	w     agentops.WorkflowInstance
	found bool
	err   error
	// byRunID, when set, is what FindByRunID returns; loadNil forces Load to miss
	// (found=false) so a test can exercise the run-id correlation fallback that a
	// plan-execution round needs (its round scope id is not the workflow key).
	byRunID    *agentops.WorkflowInstance
	loadMisses bool
}

func (f fakeLoader) Load(agentops.TargetKind, string) (agentops.WorkflowInstance, bool, error) {
	if f.loadMisses {
		return agentops.WorkflowInstance{}, false, f.err
	}
	return f.w, f.found, f.err
}

func (f fakeLoader) FindByRunID(runID string) (agentops.WorkflowInstance, bool, error) {
	if f.byRunID == nil {
		return agentops.WorkflowInstance{}, false, f.err
	}
	if _, ok := opsrunner.FindOperationByRunID(*f.byRunID, runID); !ok {
		return agentops.WorkflowInstance{}, false, nil
	}
	return *f.byRunID, true, nil
}

type fakeCommitter struct {
	calls []opsrunner.CommitRequest
	err   error
}

func (f *fakeCommitter) CommitResult(_ context.Context, req opsrunner.CommitRequest) (opsrunner.OperationResult, error) {
	f.calls = append(f.calls, req)
	return opsrunner.OperationResult{}, f.err
}

func ownedWorkflow(runID, executionID string) agentops.WorkflowInstance {
	return agentops.WorkflowInstance{
		// A backlog-item workflow is keyed by the same id its round carries as the
		// scope id, so the fast-path commit targets this domain id.
		Domain: agentops.WorkflowDomain{Kind: "backlog-item", ID: "fix/flaky"},
		Operations: []agentops.OperationExecutionRecord{
			{Operation: "workshop-round", ExecutionID: executionID, RunID: runID, State: "running"},
		},
	}
}

func terminalRound(runID string, resolved map[string]any) operatingmode.RoundEnvelope {
	round := operatingmode.RoundEnvelope{
		Status: operatingmode.RoundStatusCompleted, ScopeKind: string(agentops.TargetBacklogItem),
		ScopeID: "fix/flaky", RunID: runID, Mode: "backlog-workshop",
	}
	view := operatingmode.MutableRoundPayload(&round)
	view.SetPhaseResult(operatingmode.PhaseResult{}, resolved)
	view.SetProgress(operatingmode.ProgressState{Decision: operatingmode.ProgressComplete})
	return round
}

func TestCompletionRouterCommitsOwnedRound(t *testing.T) {
	loader := fakeLoader{w: ownedWorkflow("run-1", "exec-1"), found: true}
	committer := &fakeCommitter{}
	router := NewCompletionRouter(loader, committer, nil)

	router.Observe(context.Background(), terminalRound("run-1", map[string]any{"handoff": map[string]any{"summary": "ok"}}))

	if len(committer.calls) != 1 {
		t.Fatalf("expected one CommitResult call, got %d", len(committer.calls))
	}
	got := committer.calls[0]
	if got.ExecutionID != "exec-1" {
		t.Fatalf("wrong execution id: %q", got.ExecutionID)
	}
	if got.Outcome != OutcomeCompleted {
		t.Fatalf("wrong outcome: %q", got.Outcome)
	}
	if got.Target.Kind != agentops.TargetBacklogItem || got.Target.ID != "fix/flaky" {
		t.Fatalf("wrong target: %+v", got.Target)
	}
}

func TestCompletionRouterCorrelatesPlanExecutionByRunID(t *testing.T) {
	// A plan-execution round is keyed by the engine's resolved execution id, which
	// differs from the workflow key (the plan handle). Load-by-scope-id therefore
	// misses; the router must correlate by the agent run id and commit against the
	// workflow's OWN domain id (the handle), not the round scope id.
	const handle = "plan-abc"
	const resolvedExecID = "pm-exec-999"
	wf := agentops.WorkflowInstance{
		Domain: agentops.WorkflowDomain{Kind: "plan-execution", ID: handle},
		Operations: []agentops.OperationExecutionRecord{
			{Operation: "execution-run", ExecutionID: "op-1", RunID: "run-plan", State: "running"},
		},
	}
	loader := fakeLoader{loadMisses: true, byRunID: &wf}
	committer := &fakeCommitter{}
	round := operatingmode.RoundEnvelope{
		Status: operatingmode.RoundStatusCompleted, ScopeKind: string(agentops.TargetPlanExecution),
		ScopeID: resolvedExecID, RunID: "run-plan", Mode: "execution-drain",
	}
	view := operatingmode.MutableRoundPayload(&round)
	view.SetPhaseResult(operatingmode.PhaseResult{}, map[string]any{"handoff": map[string]any{"summary": "done"}})
	view.SetProgress(operatingmode.ProgressState{Decision: operatingmode.ProgressComplete})

	NewCompletionRouter(loader, committer, nil).Observe(context.Background(), round)

	if len(committer.calls) != 1 {
		t.Fatalf("expected one CommitResult call, got %d", len(committer.calls))
	}
	got := committer.calls[0]
	if got.Target.Kind != agentops.TargetPlanExecution || got.Target.ID != handle {
		t.Fatalf("commit must target the workflow key (handle), got %+v", got.Target)
	}
	if got.ExecutionID != "op-1" {
		t.Fatalf("wrong execution id: %q", got.ExecutionID)
	}
}

func TestCompletionRouterIgnoresRoundOwnedByNoOperation(t *testing.T) {
	// The workflow exists but no operation matches the round's run id (a legacy
	// initiative round, or a run the runner never started).
	loader := fakeLoader{w: ownedWorkflow("some-other-run", "exec-1"), found: true}
	committer := &fakeCommitter{}
	NewCompletionRouter(loader, committer, nil).Observe(context.Background(), terminalRound("run-1", map[string]any{"handoff": map[string]any{}}))
	if len(committer.calls) != 0 {
		t.Fatalf("must not commit a round no runner operation owns, got %d calls", len(committer.calls))
	}
}

func TestCompletionRouterIgnoresWhenNoWorkflow(t *testing.T) {
	committer := &fakeCommitter{}
	NewCompletionRouter(fakeLoader{found: false}, committer, nil).Observe(context.Background(), terminalRound("run-1", map[string]any{"handoff": map[string]any{}}))
	if len(committer.calls) != 0 {
		t.Fatalf("must not commit when no runner workflow exists, got %d calls", len(committer.calls))
	}
}

func TestCompletionRouterIgnoresRoundWithoutRunAssociation(t *testing.T) {
	committer := &fakeCommitter{}
	router := NewCompletionRouter(fakeLoader{w: ownedWorkflow("run-1", "exec-1"), found: true}, committer, nil)
	round := terminalRound("", map[string]any{"handoff": map[string]any{}}) // no run id
	router.Observe(context.Background(), round)
	if len(committer.calls) != 0 {
		t.Fatalf("must not commit a round with no run association, got %d calls", len(committer.calls))
	}
}

func TestCompletionRouterFailSoftOnCommitError(t *testing.T) {
	// A commit error (e.g. invalid result) must not panic or propagate; the router
	// swallows it (the refresh driver re-delivers idempotently).
	loader := fakeLoader{w: ownedWorkflow("run-1", "exec-1"), found: true}
	committer := &fakeCommitter{err: errors.New("boom")}
	router := NewCompletionRouter(loader, committer, nil)
	router.Observe(context.Background(), terminalRound("run-1", map[string]any{"handoff": map[string]any{}}))
	if len(committer.calls) != 1 {
		t.Fatalf("expected the commit attempt, got %d", len(committer.calls))
	}
}

func TestCompletionRouterIgnoresNonTerminalRound(t *testing.T) {
	loader := fakeLoader{w: ownedWorkflow("run-1", "exec-1"), found: true}
	committer := &fakeCommitter{}
	router := NewCompletionRouter(loader, committer, nil)
	round := operatingmode.RoundEnvelope{
		Status: operatingmode.RoundStatusAgentRunning, ScopeKind: string(agentops.TargetBacklogItem),
		ScopeID: "fix/flaky", RunID: "run-1",
	}
	router.Observe(context.Background(), round)
	if len(committer.calls) != 0 {
		t.Fatalf("must not commit a non-terminal round, got %d calls", len(committer.calls))
	}
}
