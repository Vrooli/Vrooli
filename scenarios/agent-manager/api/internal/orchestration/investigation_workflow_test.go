package orchestration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// investigateTestDefinition mirrors the shape of the shipped
// scenarios/agent-manager/.vrooli/agent-manager/investigate.json declaration
// (investigate run -> await-approval wait -> decide branch -> apply run / end
// nodes) with inline prompt templates in place of promptRef so the workflow can
// be driven without a live prompt-manager. It exercises the same bindings, CEL
// approval edges, and structured-result plumbing the real definition uses.
func investigateTestDefinition() *domain.WorkflowRevision {
	resultSpec := &domain.ResultSpec{
		Version:        "result-spec/v1",
		Kind:           domain.ResultSpecKindJSONSchema,
		ExtractionMode: domain.StructuredExtractionDeterministic,
		Schema: json.RawMessage(`{
			"type":"object",
			"properties":{
				"summary":{"type":"string"},
				"categories":{"type":"array","items":{"type":"object",
					"properties":{"name":{"type":"string"},"recommendations":{"type":"array","items":{"type":"object","properties":{"text":{"type":"string"}},"required":["text"]}}},
					"required":["name","recommendations"]}}
			},
			"required":["summary","categories"]
		}`),
	}
	def := domain.WorkflowDefinition{
		SchemaVersion: domain.WorkflowSchemaVersionV1,
		Owner:         "agent-manager",
		Key:           "agent-manager/investigate",
		Version:       "1.0.0",
		InputSchema:   json.RawMessage(`{"type":"object","properties":{"context":{"type":"string"}},"required":["context"]}`),
		OutputSchema:  json.RawMessage(`{"type":"object","properties":{"findings":{"type":"object"}},"required":["findings"]}`),
		EntryNode:     "investigate",
		Nodes: []domain.WorkflowNode{
			{ID: "investigate", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{
				RoleRef:        "code.smart",
				Tag:            "agent-manager-investigation",
				PromptTemplate: "Investigate the following:\n{{.context}}",
				ResultSpec:     resultSpec,
				Bindings: []domain.WorkflowInputBinding{
					{Name: "context", Source: domain.WorkflowBindingInput, Selector: "$.context", Limit: 1, MaxBytes: 65536, RenderAs: "text", MissingPolicy: "error"},
				},
				MaxTurns: 75,
			}},
			{ID: "await-approval", Kind: domain.WorkflowNodeWait, Wait: &domain.WorkflowWaitNode{
				Signal:         "investigation.approval",
				TimeoutSeconds: 60,
				PayloadSchema:  json.RawMessage(`{"type":"object","properties":{"decision":{"type":"string","enum":["completed","rejected","abstained"]},"selected":{"type":"array","items":{"type":"string"}},"note":{"type":"string"}},"required":["decision"],"additionalProperties":false}`),
			}},
			{ID: "decide", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{}},
			{ID: "apply", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{
				RoleRef:        "code.smart",
				Tag:            "agent-manager-investigation-apply",
				PromptTemplate: "Apply findings:\n{{.findings}}\nApproval:\n{{.approval}}",
				Bindings: []domain.WorkflowInputBinding{
					{Name: "findings", Source: domain.WorkflowBindingStructured, Selector: "node=investigate;$.value", Limit: 1, MaxBytes: 32768, RenderAs: "text", MissingPolicy: "error"},
					{Name: "approval", Source: domain.WorkflowBindingSignal, Selector: "$.payload", Limit: 1, MaxBytes: 8192, RenderAs: "text", MissingPolicy: "error"},
				},
				MaxTurns: 100,
			}},
			{ID: "end-completed", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded", Bindings: []domain.WorkflowInputBinding{
				{Name: "findings", Source: domain.WorkflowBindingStructured, Selector: "node=investigate;$.value", Limit: 1, MaxBytes: 65536, RenderAs: "json", MissingPolicy: "error"},
			}}},
			{ID: "end-rejected", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "blocked", Bindings: []domain.WorkflowInputBinding{
				{Name: "findings", Source: domain.WorkflowBindingStructured, Selector: "node=investigate;$.value", Limit: 1, MaxBytes: 65536, RenderAs: "json", MissingPolicy: "error"},
			}}},
			{ID: "end-abstained", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "abstained", Bindings: []domain.WorkflowInputBinding{
				{Name: "findings", Source: domain.WorkflowBindingStructured, Selector: "node=investigate;$.value", Limit: 1, MaxBytes: 65536, RenderAs: "json", MissingPolicy: "error"},
			}}},
		},
		Edges: []domain.WorkflowEdge{
			{From: "investigate", To: "await-approval"},
			{From: "await-approval", To: "decide"},
			{From: "decide", To: "apply", Condition: "journal.exists(e, e.kind == 'signal' && e.payload.decision == 'completed')"},
			{From: "decide", To: "end-rejected", Condition: "journal.exists(e, e.kind == 'signal' && e.payload.decision == 'rejected')"},
			{From: "decide", To: "end-abstained", Condition: "journal.exists(e, e.kind == 'signal' && e.payload.decision == 'abstained')"},
			{From: "apply", To: "end-completed"},
		},
		Budgets: domain.WorkflowBudgets{
			WallTimeSeconds: 600, MaxTurns: 200, MaxTokens: 100000, MaxCostUSD: 20,
			MaxNodeAttempts: 2, MaxChildren: 2, MaxConcurrency: 1, MaxRecursion: 1, MaxRetries: 1, MaxWaitSeconds: 60,
		},
	}
	now := time.Now().UTC()
	return &domain.WorkflowRevision{ID: uuid.New(), Owner: "agent-manager", Key: "agent-manager/investigate", SemanticVersion: "1.0.0", Digest: "sha256:investigate-test", Definition: def, Active: true, SourcePath: ".vrooli/agent-manager/investigate.json", SourceHash: "sha256:investigate-test", SourceUpdatedAt: now, CreatedAt: now}
}

func startInvestigateExecution(t *testing.T, o *Orchestrator, launcher *fakeRunLauncher) (*domain.WorkflowExecution, uuid.UUID, map[string]any) {
	t.Helper()
	ctx := context.Background()
	execution, err := o.StartWorkflowExecution(ctx, StartWorkflowExecutionRequest{
		Owner: "agent-manager", WorkflowKey: "agent-manager/investigate",
		Input: json.RawMessage(`{"context":"run overview + timeline for the failed run"}`), IdempotencyKey: "inv-exec-" + t.Name(),
	})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	investigateRun := waitForNodeRun(t, o.workflowExecutions, execution.ID, "investigate")

	// The investigate agent returns schema-shaped categorized recommendations.
	findings := map[string]any{
		"summary": "The run failed because the linter binary was missing.",
		"categories": []map[string]any{
			{"name": "Environment/Tooling", "recommendations": []map[string]any{{"text": "Install golangci-lint in the sandbox image."}}},
		},
	}
	launcher.complete(investigateRun, findings)
	// Completion nudge alone advances to the wait node — no consumer Advance call.
	o.nudgeWorkflowForRun(investigateRun)
	// The completion nudge alone must advance the execution from the investigate
	// run node to the operator-approval wait node (both surface as "waiting", so
	// gate on the node id, not the status).
	waitForWaitNode(t, o, execution.ID, "await-approval")
	return execution, investigateRun, findings
}

func waitForWaitNode(t *testing.T, o *Orchestrator, executionID uuid.UUID, nodeID string) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		x, err := o.GetWorkflowExecution(context.Background(), executionID)
		if err != nil {
			t.Fatalf("get execution: %v", err)
		}
		if x.CurrentNodeID == nodeID && x.Status == domain.WorkflowExecutionWaiting {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("execution never reached wait node %q (last node=%q status=%s)", nodeID, x.CurrentNodeID, x.Status)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func newInvestigateOrchestrator(t *testing.T, launcher *fakeRunLauncher) *Orchestrator {
	t.Helper()
	ctx := context.Background()
	o, repos := newRelayOrchestrator(t, launcher)
	if err := repos.Workflows.ActivateBatch(ctx, []*domain.WorkflowRevision{investigateTestDefinition()}); err != nil {
		t.Fatalf("activate: %v", err)
	}
	o.SetWorkflowNudger(NewWorkflowNudger(o.NudgeDrive, 2, 5*time.Second))
	o.workflowNudger.Start()
	t.Cleanup(o.workflowNudger.Stop)
	return o
}

// TestInvestigationWorkflowCompletedPathBindsStructuredResult is the Phase 6
// doctrine proof: an investigation reaches the operator-approval wait node via
// the completion nudge alone (zero consumer Advance), and after a "completed"
// approval signal the apply run launches bound to the investigate node's
// structured result and the operator's selection — with no transcript re-finding
// and no polling worker.
func TestInvestigationWorkflowCompletedPathBindsStructuredResult(t *testing.T) {
	ctx := context.Background()
	launcher := newFakeRunLauncher()
	o := newInvestigateOrchestrator(t, launcher)

	execution, _, _ := startInvestigateExecution(t, o, launcher)

	// Operator approves with an explicit recommendation selection.
	payload, _ := json.Marshal(map[string]any{
		"decision": "completed",
		"selected": []string{"Install golangci-lint in the sandbox image."},
	})
	if _, err := o.SignalWorkflowExecution(ctx, WorkflowExecutionSignalRequest{
		ExecutionID: execution.ID, Signal: "investigation.approval", Payload: payload, IdempotencyKey: "approve-" + execution.ID.String(),
	}); err != nil {
		t.Fatalf("signal: %v", err)
	}

	// The signal drives the workflow through the branch to the apply run.
	applyRun := waitForNodeRun(t, o.workflowExecutions, execution.ID, "apply")
	prompt := launcher.promptFor(applyRun)
	if !strings.Contains(prompt, "golangci-lint in the sandbox image") {
		t.Fatalf("apply prompt missing structured findings; got: %q", prompt)
	}
	if !strings.Contains(prompt, "\"decision\":\"completed\"") {
		t.Fatalf("apply prompt missing approval payload; got: %q", prompt)
	}

	// Complete the apply run; the completion nudge drives to the terminal state.
	launcher.complete(applyRun, map[string]any{"applied": true})
	o.nudgeWorkflowForRun(applyRun)

	res, err := o.WaitWorkflowExecution(ctx, execution.ID, 5*time.Second)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if res.TimedOut || res.Execution.Status != domain.WorkflowExecutionSucceeded {
		t.Fatalf("execution did not reach succeeded: %+v", res.Execution)
	}
	if !strings.Contains(string(res.Execution.Output), "golangci-lint") {
		t.Fatalf("output findings not bound: %s", res.Execution.Output)
	}
}

func TestInvestigationWorkflowRejectedPathSkipsApply(t *testing.T) {
	ctx := context.Background()
	launcher := newFakeRunLauncher()
	o := newInvestigateOrchestrator(t, launcher)
	execution, _, _ := startInvestigateExecution(t, o, launcher)

	payload, _ := json.Marshal(map[string]any{"decision": "rejected"})
	if _, err := o.SignalWorkflowExecution(ctx, WorkflowExecutionSignalRequest{
		ExecutionID: execution.ID, Signal: "investigation.approval", Payload: payload, IdempotencyKey: "reject-" + execution.ID.String(),
	}); err != nil {
		t.Fatalf("signal: %v", err)
	}

	res, err := o.WaitWorkflowExecution(ctx, execution.ID, 5*time.Second)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if res.Execution.Status != domain.WorkflowExecutionBlocked {
		t.Fatalf("rejected investigation should end blocked, got %s", res.Execution.Status)
	}
	if id := runIDForNode(t, o.workflowExecutions, execution.ID, "apply"); id != uuid.Nil {
		t.Fatalf("rejected investigation must not dispatch an apply run")
	}
}

func TestInvestigationWorkflowAbstainedPathSkipsApply(t *testing.T) {
	ctx := context.Background()
	launcher := newFakeRunLauncher()
	o := newInvestigateOrchestrator(t, launcher)
	execution, _, _ := startInvestigateExecution(t, o, launcher)

	payload, _ := json.Marshal(map[string]any{"decision": "abstained"})
	if _, err := o.SignalWorkflowExecution(ctx, WorkflowExecutionSignalRequest{
		ExecutionID: execution.ID, Signal: "investigation.approval", Payload: payload, IdempotencyKey: "abstain-" + execution.ID.String(),
	}); err != nil {
		t.Fatalf("signal: %v", err)
	}

	res, err := o.WaitWorkflowExecution(ctx, execution.ID, 5*time.Second)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if res.Execution.Status != domain.WorkflowExecutionAbstained {
		t.Fatalf("abstained investigation should end abstained, got %s", res.Execution.Status)
	}
	if id := runIDForNode(t, o.workflowExecutions, execution.ID, "apply"); id != uuid.Nil {
		t.Fatalf("abstained investigation must not dispatch an apply run")
	}
}
