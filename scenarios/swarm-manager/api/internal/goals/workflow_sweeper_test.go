package goals

import (
	"context"
	"fmt"
	"testing"

	"swarm-manager/internal/backlog"

	"google.golang.org/protobuf/types/known/structpb"
)

// notReadyWorkflow stands in for a run that has not finished, reporting the
// sentinel wrapped in transport context — the shape the real adapter produces.
type notReadyWorkflow struct{ captureGoalWorkflow }

func (f *notReadyWorkflow) CollectWorkflow(_ context.Context, _ string) (WorkflowCompletion, error) {
	return WorkflowCompletion{}, fmt.Errorf("%w: workflow execution is not terminal", ErrWorkflowNotReady)
}

type sentinelNotReadyWorkflow struct{ captureGoalWorkflow }

func (f *sentinelNotReadyWorkflow) CollectWorkflow(_ context.Context, _ string) (WorkflowCompletion, error) {
	return WorkflowCompletion{}, ErrWorkflowNotReady
}

// unavailableWorkflow stands in for agent-manager being unreachable — an
// outage, not a defect in the correlation record.
type unavailableWorkflow struct{ captureGoalWorkflow }

func (f *unavailableWorkflow) CollectWorkflow(_ context.Context, _ string) (WorkflowCompletion, error) {
	return WorkflowCompletion{}, fmt.Errorf("%w: dial tcp: connection refused", ErrWorkflowUnavailable)
}

// sweeperFixture builds a goal whose workflow result is terminal and ready to
// apply, and returns the handler holding it.
func sweeperFixture(t *testing.T) (*Handler, *captureGoalProposalRecorder, string) {
	t.Helper()
	svc := newTestService(t, []backlog.BacklogItem{item("execute", "a", "ready", nil)})
	created, err := svc.Create(CreateRequest{Name: "delivery", Targets: []string{"execute/a"}})
	if err != nil {
		t.Fatal(err)
	}
	input, err := structpb.NewValue(map[string]any{"entity": map[string]any{"kind": "goal", "name": "delivery", "version": created.Goal.Updated}})
	if err != nil {
		t.Fatal(err)
	}
	output, err := structpb.NewValue(map[string]any{"result": map[string]any{
		"outcome": "proposed", "summary": "Create the delivery milestone", "proposals": []any{},
	}})
	if err != nil {
		t.Fatal(err)
	}
	recorder := &captureGoalProposalRecorder{}
	handler := NewHandler(svc)
	handler.SetWorkflow(&completedGoalWorkflow{completion: WorkflowCompletion{
		ExecutionID: "wf-ready", DefinitionDigest: "sha256:test", Succeeded: true, Input: input, Output: output,
	}}, goalWorkflowRegistry(t))
	handler.SetWorkflowProposalRecorder(recorder)
	if err := handler.writeWorkflowPending("delivery", workflowPending{
		ExecutionID: "wf-ready", DefinitionDigest: "sha256:test", Transition: "goal.plan", GoalVersion: created.Goal.Updated,
	}); err != nil {
		t.Fatal(err)
	}
	return handler, recorder, created.Goal.Updated
}

// The whole point of the sweeper: a terminal result must land without anyone
// calling the apply endpoint by hand. Before it existed, results sat unapplied
// on disk indefinitely.
func TestWorkflowSweeperAppliesTerminalResult(t *testing.T) {
	handler, recorder, _ := sweeperFixture(t)

	applied, err := NewWorkflowSweeper(handler).RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if applied != 1 {
		t.Fatalf("applied = %d, want 1", applied)
	}
	if len(recorder.proposals) != 1 {
		t.Fatalf("recorded proposals = %d, want 1", len(recorder.proposals))
	}
	pending, err := handler.ListPendingWorkflows()
	if err != nil {
		t.Fatalf("ListPendingWorkflows: %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending after apply = %#v, want empty", pending)
	}
}

// A second pass must not re-record the proposals. Apply is idempotent, and the
// sweeper runs on a ticker, so this is the common path rather than an edge case.
func TestWorkflowSweeperIsIdempotentAcrossPasses(t *testing.T) {
	handler, recorder, _ := sweeperFixture(t)
	sweeper := NewWorkflowSweeper(handler)

	if _, err := sweeper.RunOnce(context.Background()); err != nil {
		t.Fatalf("first RunOnce: %v", err)
	}
	applied, err := sweeper.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("second RunOnce: %v", err)
	}
	if applied != 0 {
		t.Fatalf("second pass applied = %d, want 0", applied)
	}
	if len(recorder.proposals) != 1 {
		t.Fatalf("recorded proposals = %d, want 1 — the second pass duplicated them", len(recorder.proposals))
	}
}

// Apply refuses a result whose goal moved on. The sweeper must recognise that
// as permanent and leave it alone — without letting it block the healthy
// records in the same pass.
func TestWorkflowSweeperSkipsStaleCorrelation(t *testing.T) {
	handler, recorder, _ := sweeperFixture(t)
	if err := handler.writeWorkflowPending("delivery", workflowPending{
		ExecutionID: "wf-stale", DefinitionDigest: "sha256:test", Transition: "goal.plan",
		GoalVersion: "1999-01-01T00:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}

	pending, err := handler.ListPendingWorkflows()
	if err != nil {
		t.Fatalf("ListPendingWorkflows: %v", err)
	}
	if len(pending) != 2 {
		t.Fatalf("pending = %#v, want two records", pending)
	}
	stale, ok := findPending(pending, "wf-stale")
	if !ok || !stale.Stale {
		t.Fatalf("wf-stale = %#v, want a record marked stale", stale)
	}

	applied, err := NewWorkflowSweeper(handler).RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if applied != 1 || len(recorder.proposals) != 1 {
		t.Fatalf("applied = %d, proposals = %d, want 1 and 1 — the stale record blocked the healthy one", applied, len(recorder.proposals))
	}

	after, err := handler.ListPendingWorkflows()
	if err != nil {
		t.Fatalf("ListPendingWorkflows: %v", err)
	}
	if len(after) != 1 || after[0].ExecutionID != "wf-stale" {
		t.Fatalf("pending after sweep = %#v, want only wf-stale", after)
	}
	if after[0].Attempts != 0 {
		t.Fatalf("stale record was retried: attempts = %d, want 0", after[0].Attempts)
	}
}

func findPending(records []PendingWorkflow, executionID string) (PendingWorkflow, bool) {
	for _, record := range records {
		if record.ExecutionID == executionID {
			return record, true
		}
	}
	return PendingWorkflow{}, false
}

// A run still in flight — or an unreachable engine — is the sweeper's normal
// case. Neither may be recorded as a failure, or an agent-manager outage would
// stamp every healthy correlation with a permanent-looking error.
func TestWorkflowSweeperLeavesTransientFailuresUnmarked(t *testing.T) {
	for _, tc := range []struct {
		name     string
		workflow WorkflowInvoker
	}{
		{name: "run not finished", workflow: &sentinelNotReadyWorkflow{}},
		{name: "run not finished, wrapped", workflow: &notReadyWorkflow{}},
		{name: "engine unreachable", workflow: &unavailableWorkflow{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler, _, _ := sweeperFixture(t)
			handler.SetWorkflow(tc.workflow, goalWorkflowRegistry(t))

			applied, err := NewWorkflowSweeper(handler).RunOnce(context.Background())
			if err != nil {
				t.Fatalf("RunOnce: %v", err)
			}
			if applied != 0 {
				t.Fatalf("applied = %d, want 0", applied)
			}
			pending, err := handler.ListPendingWorkflows()
			if err != nil {
				t.Fatalf("ListPendingWorkflows: %v", err)
			}
			if len(pending) != 1 {
				t.Fatalf("pending = %#v, want the record retained", pending)
			}
			if pending[0].LastError != "" || pending[0].Attempts != 0 {
				t.Fatalf("transient failure was recorded against the record: %#v", pending[0])
			}
		})
	}
}

// A hard apply failure must be written onto the correlation record, so the
// operator listing can say why a record is stuck instead of only that it is.
func TestWorkflowSweeperRecordsHardFailureOnTheRecord(t *testing.T) {
	handler, _, _ := sweeperFixture(t)
	// A terminal-but-unsuccessful run is a permanent failure, not a retry.
	handler.SetWorkflow(&completedGoalWorkflow{completion: WorkflowCompletion{
		ExecutionID: "wf-ready", DefinitionDigest: "sha256:test", Succeeded: false,
	}}, goalWorkflowRegistry(t))

	if _, err := NewWorkflowSweeper(handler).RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	pending, err := handler.ListPendingWorkflows()
	if err != nil {
		t.Fatalf("ListPendingWorkflows: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("pending = %#v, want one record", pending)
	}
	if pending[0].Attempts != 1 || pending[0].LastError == "" || pending[0].LastAttemptAt == "" {
		t.Fatalf("diagnostics not recorded: %#v", pending[0])
	}
}
