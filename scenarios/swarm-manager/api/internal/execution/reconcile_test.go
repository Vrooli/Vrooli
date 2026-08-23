package execution

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/promptmanager"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type workflowStateReaderStub struct {
	states map[string]agentmanager.WorkflowExecutionState
	calls  []string
}

func (s *workflowStateReaderStub) GetWorkflowExecutionState(_ context.Context, workflowID string) (agentmanager.WorkflowExecutionState, error) {
	s.calls = append(s.calls, workflowID)
	state, ok := s.states[workflowID]
	if !ok {
		return agentmanager.WorkflowExecutionState{}, fmt.Errorf("unknown workflow %q", workflowID)
	}
	return state, nil
}

func TestReconcileWorkflowExecutionsRepairsTerminalCallbacksIdempotently(t *testing.T) {
	root := t.TempDir()
	service := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: testPlanRenderer(),
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
	})
	reader := &workflowStateReaderStub{states: map[string]agentmanager.WorkflowExecutionState{
		"wf-succeeded": {Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, UpdatedAt: "2026-08-22T12:00:00Z", TerminalEvidence: true},
		"wf-failed":    {Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_FAILED, UpdatedAt: "2026-08-22T12:01:00Z", TerminalCode: "agent_error"},
		"wf-cancelled": {Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_CANCELLED, UpdatedAt: "2026-08-22T12:02:00Z"},
		"wf-no-output": {Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, UpdatedAt: "2026-08-22T12:03:00Z"},
		"wf-running":   {Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_RUNNING},
	}}
	service.SetWorkflowStateReader(reader)
	seed := []Record{
		{ExecutionID: "exec-succeeded", Status: StatusRunning, RunID: "run-succeeded", OpWorkflowID: "wf-succeeded", Mode: ModeManual},
		{ExecutionID: "exec-failed", Status: StatusStarting, OpWorkflowID: "wf-failed", Mode: ModeManual},
		{ExecutionID: "exec-cancelled", Status: StatusNeedsReview, OpWorkflowID: "wf-cancelled", Mode: ModeManual},
		{ExecutionID: "exec-no-output", Status: StatusRunning, RunID: "run-no-output", OpWorkflowID: "wf-no-output", Mode: ModeManual},
		{ExecutionID: "exec-running", Status: StatusRunning, RunID: "run-running", OpWorkflowID: "wf-running", Mode: ModeManual},
	}
	if err := service.store.Save(seed); err != nil {
		t.Fatalf("save seed: %v", err)
	}

	report, err := service.ReconcileWorkflowExecutions(context.Background())
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if report.Observed != 4 || len(report.Reconciled) != 4 {
		t.Fatalf("report=%+v, want four terminal observations and repairs", report)
	}
	after, err := service.store.Load()
	if err != nil {
		t.Fatalf("load after reconcile: %v", err)
	}
	byID := map[string]Record{}
	for _, record := range after {
		byID[record.ExecutionID] = record
	}
	if byID["exec-succeeded"].Status != StatusCompleted {
		t.Fatalf("successful workflow status=%q, want completed", byID["exec-succeeded"].Status)
	}
	if byID["exec-failed"].Status != StatusFailed || byID["exec-failed"].FailureReason == "" {
		t.Fatalf("failed workflow record=%+v", byID["exec-failed"])
	}
	if byID["exec-cancelled"].Status != StatusCanceled {
		t.Fatalf("cancelled workflow status=%q, want canceled", byID["exec-cancelled"].Status)
	}
	if byID["exec-no-output"].Status != StatusNeedsReview || byID["exec-no-output"].FailureReason == "" {
		t.Fatalf("success without evidence record=%+v", byID["exec-no-output"])
	}
	if byID["exec-running"].Status != StatusRunning {
		t.Fatalf("running workflow status=%q, want unchanged", byID["exec-running"].Status)
	}

	callsAfterFirstSweep := len(reader.calls)
	report, err = service.ReconcileWorkflowExecutions(context.Background())
	if err != nil {
		t.Fatalf("second reconcile: %v", err)
	}
	if len(report.Reconciled) != 0 || len(reader.calls) != callsAfterFirstSweep+2 {
		t.Fatalf("second sweep report=%+v calls=%v, want terminal observation to be idempotent and running workflow inspected", report, reader.calls)
	}
}

func TestReconcileWorkflowExecutionsRunsCompletionProjection(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "chore", "recovered-completion", map[string]any{
		"name": "recovered-completion", "title": "Recovered completion", "status": "queued", "priority": 3, "tags": []string{},
	})
	service := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: testPlanRenderer(),
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
	})
	service.SetWorkflowStateReader(&workflowStateReaderStub{states: map[string]agentmanager.WorkflowExecutionState{
		"wf-completion-projection": {
			Status:           domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED,
			UpdatedAt:        "2026-08-22T12:00:00Z",
			TerminalEvidence: true,
		},
	}})
	if err := service.store.Save([]Record{{
		ExecutionID: "exec-completion-projection", RunID: "run-completion-projection", OpWorkflowID: "wf-completion-projection",
		BacklogKind: "chore", BacklogName: "recovered-completion", Status: StatusRunning, Mode: ModeManual,
	}}); err != nil {
		t.Fatalf("save seed: %v", err)
	}

	if _, err := service.ReconcileWorkflowExecutions(context.Background()); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	records, err := service.store.Load()
	if err != nil {
		t.Fatalf("load record: %v", err)
	}
	if len(records) != 1 || records[0].Status != StatusValidating || records[0].Finalization == nil {
		t.Fatalf("recovered record=%+v, want validating with finalization projection", records)
	}
}

// TestReconcileStrandedRecords sweeps run-id-less inspectable records to failed
// while leaving healthy records (with a run id, or already terminal) untouched.
func TestReconcileStrandedRecords(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "chore", "stranded-item", map[string]any{
		"name":     "stranded-item",
		"title":    "Stranded",
		"status":   "queued",
		"priority": 3,
		"tags":     []string{},
	})

	service := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: testPlanRenderer(),
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
	})

	seed := []Record{
		// Stranded: starting with no run id (the 542467 class).
		{ExecutionID: "stranded-1", BacklogKind: "chore", BacklogName: "stranded-item", Status: StatusStarting, Mode: ModeManual, PreviousStatus: "ready"},
		// Healthy running record with a run id — untouched.
		{ExecutionID: "healthy-1", BacklogKind: "chore", BacklogName: "stranded-item", Status: StatusRunning, RunID: "run-xyz", Mode: ModeManual},
		// Already terminal — untouched.
		{ExecutionID: "done-1", BacklogKind: "chore", BacklogName: "stranded-item", Status: StatusCompleted, RunID: "run-old", Mode: ModeManual},
		// Pending with no run id is NOT stranded (it has not started; the drain owns it).
		{ExecutionID: "pending-1", BacklogKind: "chore", BacklogName: "stranded-item", Status: StatusPending, Mode: ModeManual},
	}
	if err := service.store.Save(seed); err != nil {
		t.Fatalf("save seed: %v", err)
	}

	report, err := service.ReconcileStrandedRecords()
	if err != nil {
		t.Fatalf("ReconcileStrandedRecords error: %v", err)
	}
	if len(report.Stranded) != 1 || report.Stranded[0] != "stranded-1" {
		t.Fatalf("expected exactly stranded-1 swept, got %v", report.Stranded)
	}

	after, err := service.store.Load()
	if err != nil {
		t.Fatalf("load after reconcile: %v", err)
	}
	byID := map[string]Record{}
	for _, r := range after {
		byID[r.ExecutionID] = r
	}
	if byID["stranded-1"].Status != StatusFailed {
		t.Fatalf("stranded-1 should be failed, got %q", byID["stranded-1"].Status)
	}
	if byID["stranded-1"].FailureReason == "" {
		t.Fatalf("stranded-1 should carry a failure reason")
	}
	if byID["healthy-1"].Status != StatusRunning {
		t.Fatalf("healthy-1 (has run id) must be untouched, got %q", byID["healthy-1"].Status)
	}
	if byID["done-1"].Status != StatusCompleted {
		t.Fatalf("done-1 must be untouched, got %q", byID["done-1"].Status)
	}
	if byID["pending-1"].Status != StatusPending {
		t.Fatalf("pending-1 must be untouched, got %q", byID["pending-1"].Status)
	}

	// Idempotent: a second sweep finds nothing.
	report2, err := service.ReconcileStrandedRecords()
	if err != nil {
		t.Fatalf("second reconcile error: %v", err)
	}
	if len(report2.Stranded) != 0 {
		t.Fatalf("second sweep must find nothing, got %v", report2.Stranded)
	}
}

// TestReconcileStrandedRecordsReapsMissedOperations proves the sweep delivers a
// missed operation reap: a terminal (failed/canceled) record still correlated to
// a durable operation execution gets its operation offered a cancel through the
// OperationStarter seam (the 542467c6 divergence class — the stranded sweep
// marked the legacy record failed without reaping its agentops operation).
// Completed records and records without an operation correlation are untouched.
func TestReconcileStrandedRecordsReapsMissedOperations(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "chore", "diverged-item", map[string]any{
		"name":     "diverged-item",
		"title":    "Diverged",
		"status":   "backlog",
		"priority": 3,
		"tags":     []string{},
	})

	service := NewService(ServiceConfig{
		DataRoot:     root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: testPlanRenderer(),
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
	})
	starter := &stubOperationStarter{}
	service.SetOperationStarter(starter)

	seed := []Record{
		// Failed record with an op correlation whose reap was missed — must be reaped.
		{
			ExecutionID: "diverged-1", BacklogKind: "chore", BacklogName: "diverged-item", Status: StatusFailed,
			Mode: ModeManual, OpWorkflowID: "wf-plan-execution-x", OpExecutionID: "opx-stale",
		},
		// Completed record with an op correlation — must NOT be reaped (a running
		// op behind a completed record is a divergence for operator attention).
		{
			ExecutionID: "done-1", BacklogKind: "chore", BacklogName: "diverged-item", Status: StatusCompleted,
			Mode: ModeManual, OpWorkflowID: "wf-plan-execution-x", OpExecutionID: "opx-done",
		},
		// Failed record without any op correlation — nothing to reap.
		{ExecutionID: "legacy-1", BacklogKind: "chore", BacklogName: "diverged-item", Status: StatusFailed, Mode: ModeManual},
	}
	if err := service.store.Save(seed); err != nil {
		t.Fatalf("save seed: %v", err)
	}

	report, err := service.ReconcileStrandedRecords()
	if err != nil {
		t.Fatalf("ReconcileStrandedRecords error: %v", err)
	}
	if len(report.Stranded) != 0 {
		t.Fatalf("no stranded records expected, got %v", report.Stranded)
	}
	if len(report.OpReapsAttempted) != 1 || report.OpReapsAttempted[0] != "diverged-1" {
		t.Fatalf("expected exactly diverged-1 offered a reap, got %v", report.OpReapsAttempted)
	}
	if starter.cancelCalls != 1 {
		t.Fatalf("expected exactly one CancelOperation call, got %d", starter.cancelCalls)
	}
	if starter.cancelReq.ExecutionID != "opx-stale" {
		t.Fatalf("expected reap of opx-stale, got %q", starter.cancelReq.ExecutionID)
	}
	if starter.cancelReq.TargetKind != "plan-execution" || starter.cancelReq.TargetID != "test-plan-diverged-item" {
		t.Fatalf("unexpected reap target %s/%s", starter.cancelReq.TargetKind, starter.cancelReq.TargetID)
	}
}
