package execution

import (
	"context"
	"path/filepath"
	"testing"

	"swarm-manager/internal/transitionrun"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	executionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution"
	"google.golang.org/protobuf/types/known/structpb"
	"swarm-manager/internal/agentmanager"
)

type resumeCapableRenderer struct {
	*fakeMarkdownRenderer
	request        *executionv1.ResumeRequest
	statusComplete bool
	completeCalls  int
}

func (r *resumeCapableRenderer) Resume(_ context.Context, request *executionv1.ResumeRequest) (*executionv1.ResumeResponse, error) {
	r.request = request
	return &executionv1.ResumeResponse{Execution: &executionv1.Execution{Id: "plan-exec-1"}}, nil
}

func (r *resumeCapableRenderer) GetStatus(_ context.Context, _ *executionv1.GetStatusRequest) (*executionv1.GetStatusResponse, error) {
	return &executionv1.GetStatusResponse{Execution: &executionv1.Execution{Id: "plan-exec-1", Complete: r.statusComplete}}, nil
}

func (r *resumeCapableRenderer) Complete(_ context.Context, _ *executionv1.CompleteRequest) (*executionv1.CompleteResponse, error) {
	r.completeCalls++
	r.statusComplete = true
	return &executionv1.CompleteResponse{}, nil
}

func mustWorkflowOutput(t *testing.T, result map[string]any) *structpb.Value {
	t.Helper()
	output, err := structpb.NewValue(map[string]any{"result": result})
	if err != nil {
		t.Fatal(err)
	}
	return output
}

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
		PlanRenderer: testPlanRenderer(), PhasedPlanWorkflow: workflow, TransitionRegistry: testTransitionRegistry(t),
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

func workflowCorrelationFor(t *testing.T, service *Service, record Record) transitionrun.Correlation {
	t.Helper()
	correlation, err := service.transitionCorrelation(record)
	if err != nil {
		t.Fatalf("transition correlation for %s: %v", record.ExecutionID, err)
	}
	return correlation
}

func TestApplyPhasedPlanWorkflow_BlockedExactlyOnce(t *testing.T) { // [REQ:REQ-P0-011-IMMUTABLE-EXECUTION] [REQ:REQ-P0-011-ENVELOPE-PROVENANCE]
	service, workflow, started, root := setupPhasedPlanExecution(t, "blocked-plan")
	correlation := workflowCorrelationFor(t, service, started)
	workflow.completion = agentmanager.InvocationCompletion{
		ExecutionID: correlation.ExecutionID, DefinitionDigest: correlation.DefinitionDigest,
		Status:   domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED,
		Input:    workflow.invocation.Input,
		Output:   mustWorkflowOutput(t, map[string]any{"outcome": "needs_review", "handoff": "paused", "blocker": map[string]any{"code": "operator_input", "summary": "operator input required", "retryable": true}}),
		Attempts: []*domainpb.WorkflowNodeAttempt{{NodeId: "slice", Ordinal: 1, RunId: "run-1", ProfileIdentity: "swarm-manager/deep-work"}},
	}

	first, err := service.ApplyPhasedPlanWorkflow(context.Background(), started.ExecutionID)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if first.Idempotent || first.Record.Status != StatusNeedsReview || transitionApplyStateFor(t, service, correlation.ExecutionID) != transitionrun.ApplyStateComplete {
		t.Fatalf("unexpected first apply: %#v", first)
	}
	if applied := workflowCorrelationFor(t, service, first.Record); len(applied.Attempts) != 1 || applied.Attempts[0].RunID != "run-1" {
		t.Fatalf("attempt provenance not retained in correlation: %#v", applied.Attempts)
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
	correlation := workflowCorrelationFor(t, service, started)
	workflow.completion = agentmanager.InvocationCompletion{
		ExecutionID: correlation.ExecutionID, DefinitionDigest: correlation.DefinitionDigest,
		Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_BLOCKED, Input: workflow.invocation.Input,
		Output: mustWorkflowOutput(t, map[string]any{"outcome": "blocked", "reason": "operator input"}),
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
	if applied.Status != StatusStarting || transitionApplyStateFor(t, service, correlation.ExecutionID) == transitionrun.ApplyStateComplete || workflow.collectCalls != 0 {
		t.Fatalf("legacy housekeeping must not apply workflow: record=%#v collects=%d", applied, workflow.collectCalls)
	}
}

// [REQ:SWM-P0-006] phased slice execution: bounded slices and handoff plumbing into workflow input
func TestStartBindsIdempotentPlanManagerExecutionIntoWorkflowInput(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "execute", "plan-binding", map[string]any{
		"name": "plan-binding", "title": "Plan binding", "description": "bounded work",
		"status": "ready", "priority": 2, "tags": []string{},
		"acceptance_allow": []string{"scenarios/swarm-manager/**"},
	})
	renderer := &resumeCapableRenderer{fakeMarkdownRenderer: testPlanRenderer()}
	workflow := &stubPhasedPlanWorkflow{}
	service := NewService(ServiceConfig{
		DataRoot: root, StorePath: filepath.Join(root, ".vrooli", "execution-runs.json"),
		PlanRenderer: renderer, PhasedPlanWorkflow: workflow, TransitionRegistry: testTransitionRegistry(t),
	})
	queued, err := service.QueueBacklog(context.Background(), CreateRequest{BacklogKind: "execute", BacklogName: "plan-binding", Mode: ModeManual, MaxSlices: 2})
	if err != nil {
		t.Fatalf("queue: %v", err)
	}
	started, err := service.Start(context.Background(), queued.ExecutionID)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if renderer.request == nil || renderer.request.GetPlanOrExecution() == "" {
		t.Fatalf("expected plan-manager resume request, got %#v", renderer.request)
	}
	if started.PlanManagerExecutionID != "plan-exec-1" {
		t.Fatalf("plan-manager execution correlation = %q", started.PlanManagerExecutionID)
	}
	payload, ok := workflow.invocation.Input.AsInterface().(map[string]any)
	if !ok || payload["planExecutionId"] != "plan-exec-1" {
		t.Fatalf("workflow input omitted plan execution: %#v", payload)
	}
	constraints, _ := payload["constraints"].(map[string]any)
	if constraints["maxSlices"] != float64(2) {
		t.Fatalf("workflow constraints did not preserve maxSlices: %#v", constraints)
	}
}

// [REQ:SWM-P0-005] strategy selection validated against the declared registry
func TestQueueBacklogRejectsUnknownStrategy(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "execute", "unknown-strategy", map[string]any{
		"name": "unknown-strategy", "title": "Strategy validation", "description": "bounded work",
		"status": "ready", "priority": 2, "tags": []string{}, "acceptance_allow": []string{"scenarios/swarm-manager/**"},
	})
	service := NewService(ServiceConfig{DataRoot: root, StorePath: filepath.Join(root, ".vrooli", "execution-runs.json"), PlanRenderer: testPlanRenderer(), TransitionRegistry: testTransitionRegistry(t)})
	if _, err := service.QueueBacklog(context.Background(), CreateRequest{BacklogKind: "execute", BacklogName: "unknown-strategy", Mode: ModeManual, Strategy: "single-pass"}); err == nil {
		t.Fatal("expected unknown strategy rejection")
	}
}

func TestReconcilePlanManagerCompletionCompletesBoundExecution(t *testing.T) {
	renderer := &resumeCapableRenderer{fakeMarkdownRenderer: testPlanRenderer()}
	service := NewService(ServiceConfig{DataRoot: t.TempDir(), PlanRenderer: renderer})
	record := Record{PlanManagerExecutionID: "plan-exec-1"}
	if err := service.reconcilePlanManagerCompletion(context.Background(), &record, "complete"); err != nil {
		t.Fatalf("reconcile plan-manager completion: %v", err)
	}
	if renderer.completeCalls != 1 || record.PlanManagerReconciledAt == "" {
		t.Fatalf("completion reconciliation = calls:%d record:%#v", renderer.completeCalls, record)
	}
	if err := service.reconcilePlanManagerCompletion(context.Background(), &record, "complete"); err != nil {
		t.Fatalf("idempotent reconciliation: %v", err)
	}
	if renderer.completeCalls != 1 {
		t.Fatalf("reconciliation retried after persistence: %d calls", renderer.completeCalls)
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
	_, err := service.ApprovePhasedPlanWorkflow(context.Background(), started.ExecutionID, "alice")
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	second, err := service.ApprovePhasedPlanWorkflow(context.Background(), started.ExecutionID, "bob")
	if err != nil {
		t.Fatalf("approve replay: %v", err)
	}
	correlation := workflowCorrelationFor(t, service, second)
	if correlation.ApprovalTime == "" || correlation.ApprovalActor != "alice" {
		t.Fatalf("approval decision was not stable: %#v", correlation)
	}
	if workflow.approveCalls != 2 {
		t.Fatalf("expected idempotent signal redelivery, got %d", workflow.approveCalls)
	}
}

func TestApplyPhasedPlanWorkflow_RejectsChangedFrontier(t *testing.T) { // [REQ:REQ-P0-011-IMMUTABLE-EXECUTION]
	service, workflow, started, _ := setupPhasedPlanExecution(t, "stale-plan")
	correlation := workflowCorrelationFor(t, service, started)
	workflow.completion = agentmanager.InvocationCompletion{
		ExecutionID: correlation.ExecutionID, DefinitionDigest: correlation.DefinitionDigest,
		Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED,
		Input: func() *structpb.Value {
			payload, _ := workflow.invocation.Input.AsInterface().(map[string]any)
			plan, _ := payload["plan"].(map[string]any)
			plan["frontierDigest"] = "sha256:stale"
			value, err := structpb.NewValue(payload)
			if err != nil {
				t.Fatal(err)
			}
			return value
		}(),
		Output: mustWorkflowOutput(t, map[string]any{"outcome": "complete"}),
	}
	if _, err := service.ApplyPhasedPlanWorkflow(context.Background(), started.ExecutionID); err == nil {
		t.Fatal("expected stale workflow frontier rejection")
	}
	records, _, _ := service.loadRecordLocked(started.ExecutionID)
	for _, record := range records {
		if record.ExecutionID == started.ExecutionID && transitionApplyStateFor(t, service, correlation.ExecutionID) == transitionrun.ApplyStateComplete {
			t.Fatalf("stale result was claimed: %#v", record)
		}
	}
}

// transitionApplyStateFor reads apply state from the shared correlation, which
// is now its only home. Tests assert through this rather than a record field so
// they cannot pass against a stale second copy.
func transitionApplyStateFor(t *testing.T, service *Service, workflowExecutionID string) string {
	t.Helper()
	if service.transitionRunner == nil || workflowExecutionID == "" {
		return ""
	}
	correlation, err := service.transitionRunner.GetCorrelation(workflowExecutionID)
	if err != nil {
		return ""
	}
	return correlation.ApplyState
}
