package execution

import (
	"context"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentmanager"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func setupPhasedPlanExecution(t *testing.T, name string) (*Service, *stubPhasedPlanWorkflow, Record, string) {
	t.Helper()
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "execute", name, map[string]any{
		"name": name, "title": "Phased plan", "description": "bounded work",
		"status": "ready", "priority": 2, "tags": []string{},
		"acceptance_allow": []string{"scenarios/swarm-manager/**"},
	})
	workflow := &stubPhasedPlanWorkflow{}
	service := NewService(ServiceConfig{
		DataRoot: root, StorePath: filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: testPlanRenderer(), PhasedPlanWorkflow: workflow,
	})
	queued, err := service.QueueBacklog(context.Background(), CreateRequest{BacklogKind: "execute", BacklogName: name, Mode: ModeManual})
	if err != nil {
		t.Fatalf("queue: %v", err)
	}
	started, err := service.Start(context.Background(), queued.ExecutionID)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	return service, workflow, started, root
}

func TestApplyPhasedPlanWorkflow_BlockedExactlyOnce(t *testing.T) { // [REQ:REQ-P0-011-IMMUTABLE-EXECUTION] [REQ:REQ-P0-011-ENVELOPE-PROVENANCE]
	service, workflow, started, root := setupPhasedPlanExecution(t, "blocked-plan")
	workflow.completion = agentmanager.PhasedPlanWorkflowCompletion{
		ExecutionID: started.AgentWorkflowExecutionID, DefinitionDigest: started.AgentWorkflowDefinition,
		Status:     domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BLOCKED,
		ConsumerID: started.ExecutionID, EntityKind: started.BacklogKind, EntityName: started.BacklogName,
		EntityVersion: started.AgentWorkflowEntityVersion, FrontierDigest: started.AgentWorkflowFrontier,
		Result:   []byte(`{"outcome":"blocked","handoff":"paused","completedSlice":"","blocker":{"code":"operator_input","summary":"operator input required","retryable":true}}`),
		Attempts: []agentmanager.WorkflowAttemptProvenance{{NodeID: "slice", Ordinal: 1, RunID: "run-1", ProfileIdentity: "swarm-manager/deep-work"}},
	}

	first, err := service.ApplyPhasedPlanWorkflow(context.Background(), started.ExecutionID)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if first.Idempotent || first.Record.Status != StatusNeedsReview || first.Record.AgentWorkflowApplyState != workflowApplyComplete {
		t.Fatalf("unexpected first apply: %#v", first)
	}
	if len(first.Record.AgentWorkflowAttempts) != 1 || first.Record.AgentWorkflowAttempts[0].RunID != "run-1" {
		t.Fatalf("attempt provenance not retained: %#v", first.Record.AgentWorkflowAttempts)
	}
	second, err := service.ApplyPhasedPlanWorkflow(context.Background(), started.ExecutionID)
	if err != nil {
		t.Fatalf("replay apply: %v", err)
	}
	if !second.Idempotent || workflow.collectCalls != 1 {
		t.Fatalf("replay must be local and idempotent: result=%#v collects=%d", second, workflow.collectCalls)
	}
	item := mustLoadBacklogItem(t, filepath.Join(root, "execute", "blocked-plan", "spec.json"))
	if item["status"] != backlogStatusInReview {
		t.Fatalf("backlog status = %#v", item["status"])
	}
}

func TestProcessActiveExecutionsDoesNotApplyPhasedPlanWorkflow(t *testing.T) {
	service, workflow, started, _ := setupPhasedPlanExecution(t, "auto-plan")
	workflow.completion = agentmanager.PhasedPlanWorkflowCompletion{
		ExecutionID: started.AgentWorkflowExecutionID, DefinitionDigest: started.AgentWorkflowDefinition,
		Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BLOCKED, ConsumerID: started.ExecutionID,
		EntityKind: started.BacklogKind, EntityName: started.BacklogName, EntityVersion: started.AgentWorkflowEntityVersion,
		FrontierDigest: started.AgentWorkflowFrontier, Result: []byte(`{"outcome":"blocked","reason":"operator input"}`),
	}
	if err := service.ProcessActiveExecutions(context.Background()); err != nil {
		t.Fatalf("process active: %v", err)
	}
	records, _, err := service.loadRecordLocked(started.ExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	var applied Record
	for _, record := range records {
		if record.ExecutionID == started.ExecutionID {
			applied = record
		}
	}
	if applied.Status != StatusStarting || applied.AgentWorkflowApplyState != "" || workflow.collectCalls != 0 {
		t.Fatalf("legacy housekeeping must not apply workflow: record=%#v collects=%d", applied, workflow.collectCalls)
	}
}

func TestApplyPhasedPlanWorkflow_ResumesPersistedClaim(t *testing.T) { // [REQ:REQ-P0-011-IMMUTABLE-EXECUTION]
	service, workflow, started, _ := setupPhasedPlanExecution(t, "claimed-plan")
	records, idx, err := service.loadRecordLocked(started.ExecutionID)
	if err != nil {
		t.Fatal(err)
	}
	records[idx].AgentWorkflowApplyState = workflowApplyClaimed
	records[idx].AgentWorkflowOutcome = "abstained"
	records[idx].AgentWorkflowResult = []byte(`{"outcome":"abstained","summary":"insufficient evidence"}`)
	if err := service.store.Save(records); err != nil {
		t.Fatal(err)
	}

	result, err := service.ApplyPhasedPlanWorkflow(context.Background(), started.ExecutionID)
	if err != nil {
		t.Fatalf("resume claim: %v", err)
	}
	if result.Record.Status != StatusNeedsReview || result.Record.AgentWorkflowApplyState != workflowApplyComplete {
		t.Fatalf("claim not completed: %#v", result.Record)
	}
	if workflow.collectCalls != 0 {
		t.Fatalf("claimed recovery recollected external state %d times", workflow.collectCalls)
	}
}

func TestParsePhasedPlanOutcomeUsesTypedWorkflowTerminalStatus(t *testing.T) {
	blocked, err := parsePhasedPlanOutcome(domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BLOCKED, []byte(`{"outcome":"continue","handoff":"last slice"}`))
	if err != nil || blocked.Outcome != "blocked" {
		t.Fatalf("blocked result=%+v err=%v", blocked, err)
	}
	abstained, err := parsePhasedPlanOutcome(domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_ABSTAINED, []byte(`{"outcome":"abstained","reason":"insufficient evidence"}`))
	if err != nil || abstained.Outcome != "abstained" || abstained.Reason != "insufficient evidence" {
		t.Fatalf("abstained result=%+v err=%v", abstained, err)
	}
}

func TestApprovePhasedPlanWorkflow_PersistsDecisionBeforeIdempotentSignal(t *testing.T) {
	service, workflow, started, _ := setupPhasedPlanExecution(t, "approval-plan")
	first, err := service.ApprovePhasedPlanWorkflow(context.Background(), started.ExecutionID, "alice")
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	second, err := service.ApprovePhasedPlanWorkflow(context.Background(), started.ExecutionID, "bob")
	if err != nil {
		t.Fatalf("approve replay: %v", err)
	}
	if first.AgentWorkflowApprovalAt == "" || second.AgentWorkflowApprovalBy != "alice" {
		t.Fatalf("approval decision was not stable: %#v", second)
	}
	if workflow.approveCalls != 2 {
		t.Fatalf("expected idempotent signal redelivery, got %d", workflow.approveCalls)
	}
}

func TestApplyPhasedPlanWorkflow_RejectsChangedFrontier(t *testing.T) { // [REQ:REQ-P0-011-IMMUTABLE-EXECUTION]
	service, workflow, started, _ := setupPhasedPlanExecution(t, "stale-plan")
	workflow.completion = agentmanager.PhasedPlanWorkflowCompletion{
		ExecutionID: started.AgentWorkflowExecutionID, DefinitionDigest: started.AgentWorkflowDefinition,
		Status:     domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED,
		ConsumerID: started.ExecutionID, EntityKind: started.BacklogKind, EntityName: started.BacklogName,
		EntityVersion: started.AgentWorkflowEntityVersion, FrontierDigest: "sha256:stale",
		Result: []byte(`{"outcome":"complete"}`),
	}
	if _, err := service.ApplyPhasedPlanWorkflow(context.Background(), started.ExecutionID); err == nil {
		t.Fatal("expected stale workflow frontier rejection")
	}
	records, _, _ := service.loadRecordLocked(started.ExecutionID)
	for _, record := range records {
		if record.ExecutionID == started.ExecutionID && record.AgentWorkflowApplyState != "" {
			t.Fatalf("stale result was claimed: %#v", record)
		}
	}
}
