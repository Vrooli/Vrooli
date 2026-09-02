package execution

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"swarm-manager/internal/transitionrun"

	"swarm-manager/internal/agentmanager"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	executionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/execution"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
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

func TestPhasedPlanFrontierPinsAuthoredContentNotRuntimeRendering(t *testing.T) {
	item := backlogItem{Kind: "execute", Name: "stable-frontier"}
	record := Record{ExecutionID: "swarm-exec-1"}
	base := &sharedv1.Plan{
		Id: "plan-1", Slug: "stable-frontier", Title: "Stable frontier", Purpose: "Prove the rail.",
		Status: sharedv1.PlanStatus_PLAN_STATUS_DRAFT, ContentHash: "runtime-hash-1", UpdatedAt: "2026-01-01T00:00:00Z",
		WorkPosture: sharedv1.WorkPosture_WORK_POSTURE_GREENFIELD, WorkPostureDetail: "first computed explanation",
		Phases: []*sharedv1.Phase{{Id: "phase-1", Order: 1, Title: "Proof", Status: sharedv1.PhaseStatus_PHASE_STATUS_TODO, Steps: []string{"Run the governed check."}, BaselineScope: []string{"computed command one"}}},
	}
	before, err := buildPhasedPlanSnapshot(item, record, "plan-1", "/repo", renderedPlanContent{Plan: base})
	if err != nil {
		t.Fatal(err)
	}
	runtimeChanged := proto.Clone(base).(*sharedv1.Plan)
	runtimeChanged.Status = sharedv1.PlanStatus_PLAN_STATUS_COMPLETE
	runtimeChanged.ContentHash = "runtime-hash-2"
	runtimeChanged.UpdatedAt = "2026-01-02T00:00:00Z"
	runtimeChanged.WorkPostureDetail = "different computed explanation"
	runtimeChanged.Phases[0].Status = sharedv1.PhaseStatus_PHASE_STATUS_DONE
	runtimeChanged.Phases[0].BaselineScope = []string{"computed command two"}
	afterStatusChange, err := buildPhasedPlanSnapshot(item, record, "plan-1", "/repo", renderedPlanContent{Plan: runtimeChanged})
	if err != nil {
		t.Fatal(err)
	}
	authoredChanged := proto.Clone(runtimeChanged).(*sharedv1.Plan)
	authoredChanged.Phases[0].Steps = []string{"Run a different check."}
	afterAuthoredChange, err := buildPhasedPlanSnapshot(item, record, "plan-1", "/repo", renderedPlanContent{Plan: authoredChanged})
	if err != nil {
		t.Fatal(err)
	}
	if before.FrontierDigest != afterStatusChange.FrontierDigest {
		t.Fatalf("runtime status changed frontier: before=%s after=%s", before.FrontierDigest, afterStatusChange.FrontierDigest)
	}
	if before.FrontierDigest == afterAuthoredChange.FrontierDigest {
		t.Fatal("authored plan change did not invalidate frontier")
	}
}

func TestAuthoredPlanFrontierPreservesLiteralHTMLCharacters(t *testing.T) {
	frontier, err := authoredPlanFrontierJSON(&sharedv1.Plan{
		Id:    "plan-1",
		Title: "Execute <execution> & verify > output",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(frontier, `\\u003c`) || strings.Contains(frontier, `\\u003e`) || strings.Contains(frontier, `\\u0026`) {
		t.Fatalf("frontier used HTML escaping: %s", frontier)
	}
	if !strings.Contains(frontier, `"title":"Execute <execution> & verify > output"`) {
		t.Fatalf("frontier lost literal HTML characters: %s", frontier)
	}
}

func TestRebasePhasedPlanWorkflowRequiresStableFrontier(t *testing.T) {
	service, workflow, started, _ := setupPhasedPlanExecution(t, "rebase-plan")
	correlation := workflowCorrelationFor(t, service, started)
	legacyVersion := correlation.EntityVersion
	if _, err := service.transitionRunner.UpdateCorrelation(correlation.ExecutionID, func(value *transitionrun.Correlation) error {
		value.EntityVersion = "sha256:legacy-lifecycle-inclusive-version"
		return nil
	}); err != nil {
		t.Fatalf("seed legacy correlation: %v", err)
	}
	rebased, err := service.RebasePhasedPlanWorkflow(context.Background(), started.ExecutionID, "operator:codex", "Migrate legacy lifecycle-inclusive subject version after canonicalization fix.")
	if err != nil {
		t.Fatalf("rebase: %v", err)
	}
	if rebased.Status != started.Status {
		t.Fatalf("rebase changed lifecycle status: before=%s after=%s", started.Status, rebased.Status)
	}
	updated := workflowCorrelationFor(t, service, rebased)
	if updated.EntityVersionRebasedBy != "operator:codex" || updated.EntityVersionRebasedAt == "" || updated.EntityVersionRebasedReason == "" {
		t.Fatalf("rebase provenance missing: %#v", updated)
	}
	if !slices.Contains(updated.DeclaredOutcomes, "complete") {
		t.Fatalf("rebase did not migrate declared outcomes: %#v", updated.DeclaredOutcomes)
	}
	if updated.FrontierDigest != correlation.FrontierDigest || updated.EntityVersion == "sha256:legacy-lifecycle-inclusive-version" || updated.EntityVersion != legacyVersion {
		t.Fatalf("unexpected rebase correlation: before=%#v after=%#v", correlation, updated)
	}
	if workflow.collectCalls != 0 {
		t.Fatalf("rebase should not collect or apply workflow result: %d", workflow.collectCalls)
	}
}

func TestPhasedPlanEntityVersionIgnoresBacklogLifecycleMetadata(t *testing.T) {
	item := backlogItem{Name: "stable", Kind: "fix", Title: "Stable", Status: "queued", Updated: "2026-01-01T00:00:00Z"}
	record := Record{ExecutionID: "execution-1"}
	plan := &sharedv1.Plan{Id: "plan-1", Title: "Plan"}
	queued, err := buildPhasedPlanSnapshot(item, record, "plan-1", "/repo", renderedPlanContent{Plan: plan})
	if err != nil {
		t.Fatal(err)
	}
	lifecycleChanged := item
	lifecycleChanged.Status = "in_progress"
	lifecycleChanged.Updated = "2026-01-01T00:01:00Z"
	lifecycleChanged.PlanAcceptance = &planAcceptance{Actor: "operator", AcceptedAt: "2026-01-01T00:00:00Z", PlanContentHash: "sha256:plan", SubjectVersion: "sha256:acceptance"}
	started, err := buildPhasedPlanSnapshot(lifecycleChanged, record, "plan-1", "/repo", renderedPlanContent{Plan: plan})
	if err != nil {
		t.Fatal(err)
	}
	if queued.EntityVersion != started.EntityVersion {
		t.Fatalf("lifecycle metadata changed entity version: queued=%s started=%s", queued.EntityVersion, started.EntityVersion)
	}
	authoredChanged := lifecycleChanged
	authoredChanged.Title = "Edited"
	edited, err := buildPhasedPlanSnapshot(authoredChanged, record, "plan-1", "/repo", renderedPlanContent{Plan: plan})
	if err != nil {
		t.Fatal(err)
	}
	if queued.EntityVersion == edited.EntityVersion {
		t.Fatal("authored backlog edit did not invalidate entity version")
	}
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

func TestApplyPhasedPlanWorkflow_BudgetExhaustedIsResumable(t *testing.T) {
	service, workflow, started, root := setupPhasedPlanExecution(t, "budget-plan")
	correlation := workflowCorrelationFor(t, service, started)
	workflow.completion = agentmanager.InvocationCompletion{
		ExecutionID: correlation.ExecutionID, DefinitionDigest: correlation.DefinitionDigest,
		Status:       domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED,
		TerminalCode: "budget_exhausted", BudgetName: "tokens", Input: workflow.invocation.Input,
		Output: mustWorkflowOutput(t, map[string]any{"outcome": "budget_exhausted", "reason": "token ceiling reached"}),
	}
	result, err := service.ApplyPhasedPlanWorkflow(context.Background(), started.ExecutionID)
	if err != nil {
		t.Fatalf("apply budget exhaustion: %v", err)
	}
	if result.Record.Status != StatusBudgetExhausted {
		t.Fatalf("status=%s, want budget_exhausted", result.Record.Status)
	}
	if result.Record.FailureReason != "budget_exhausted" {
		t.Fatalf("failure reason=%q, want terminal code", result.Record.FailureReason)
	}
	item := mustLoadBacklogItem(t, filepath.Join(root, "execute", "budget-plan", "spec.json"))
	if item["status"] != backlogStatusInReview {
		t.Fatalf("backlog status=%v, want in_review", item["status"])
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
