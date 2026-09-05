package orchestration

import (
	"context"
	"encoding/json"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func waitWorkflowDefinition() *domain.WorkflowRevision {
	revision := relayDefinition()
	revision.ID = uuid.New()
	revision.Key = "owner/wait-for-signal"
	revision.Digest = "sha256:wait-for-signal"
	revision.Definition.Key = revision.Key
	revision.Definition.EntryNode = "wait"
	revision.Definition.Nodes = []domain.WorkflowNode{
		{ID: "wait", Kind: domain.WorkflowNodeWait, Wait: &domain.WorkflowWaitNode{Signal: "continue", TimeoutSeconds: 60}},
		{ID: "end", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	revision.Definition.Edges = []domain.WorkflowEdge{{From: "wait", To: "end"}}
	return revision
}

func startWaitWorkflow(t *testing.T, o *Orchestrator) *domain.WorkflowExecution {
	t.Helper()
	execution, err := o.StartWorkflowExecution(context.Background(), StartWorkflowExecutionRequest{
		Owner: "owner", WorkflowKey: "owner/wait-for-signal", Input: json.RawMessage(`{}`), IdempotencyKey: t.Name() + "/" + uuid.NewString(),
	})
	if err != nil {
		t.Fatalf("start workflow: %v", err)
	}
	if execution.Status != domain.WorkflowExecutionWaiting {
		t.Fatalf("status=%s, want waiting", execution.Status)
	}
	return execution
}

func TestWorkflowExecutionControlAndTriageProjections(t *testing.T) {
	ctx := context.Background()
	launcher := newFakeRunLauncher()
	o, repos := newRelayOrchestrator(t, launcher)
	revision := waitWorkflowDefinition()
	if err := repos.Workflows.ActivateBatch(ctx, []*domain.WorkflowRevision{revision}); err != nil {
		t.Fatalf("activate: %v", err)
	}

	execution := startWaitWorkflow(t, o)
	listed, err := o.ListWorkflowExecutions(ctx, ListWorkflowExecutionsRequest{Owner: "owner", WorkflowKey: revision.Key, Limit: 10})
	if err != nil || len(listed) != 1 || listed[0].ID != execution.ID {
		t.Fatalf("list=%+v err=%v", listed, err)
	}
	trace, err := o.GetWorkflowExecutionTrace(ctx, execution.ID, 0, 0)
	if err != nil || trace.Execution.ID != execution.ID || len(trace.Journal) == 0 {
		t.Fatalf("trace=%+v err=%v", trace, err)
	}
	if attempts, err := o.ListWorkflowExecutionRuns(ctx, execution.ID); err != nil || len(attempts) != 0 {
		t.Fatalf("attempts=%+v err=%v", attempts, err)
	}

	resumed, err := o.ResumeWorkflowExecution(ctx, WorkflowExecutionOperationRequest{ExecutionID: execution.ID, IdempotencyKey: "resume"})
	if err != nil || resumed.Idempotent || resumed.Execution.Status != domain.WorkflowExecutionWaiting {
		t.Fatalf("resume=%+v err=%v", resumed, err)
	}
	repeatedResume, err := o.ResumeWorkflowExecution(ctx, WorkflowExecutionOperationRequest{ExecutionID: execution.ID, IdempotencyKey: "resume"})
	if err != nil || !repeatedResume.Idempotent {
		t.Fatalf("repeat resume=%+v err=%v", repeatedResume, err)
	}

	cancelled, err := o.CancelWorkflowExecution(ctx, WorkflowExecutionOperationRequest{ExecutionID: execution.ID, IdempotencyKey: "cancel", Reason: "operator stopped the drill"})
	if err != nil || cancelled.Execution.Status != domain.WorkflowExecutionCancelled || cancelled.Idempotent {
		t.Fatalf("cancel=%+v err=%v", cancelled, err)
	}
	repeatedCancel, err := o.CancelWorkflowExecution(ctx, WorkflowExecutionOperationRequest{ExecutionID: execution.ID, IdempotencyKey: "cancel"})
	if err != nil || !repeatedCancel.Idempotent || repeatedCancel.Execution.Status != domain.WorkflowExecutionCancelled {
		t.Fatalf("repeat cancel=%+v err=%v", repeatedCancel, err)
	}

	failed := startWaitWorkflow(t, o)
	if _, err := o.FailWorkflowExecution(ctx, failed.ID, "injected_failure", "triage fixture"); err != nil {
		t.Fatalf("fail workflow: %v", err)
	}
	retried, err := o.RetryWorkflowExecution(ctx, WorkflowExecutionOperationRequest{ExecutionID: failed.ID, IdempotencyKey: "retry"})
	if err != nil || retried.Idempotent || retried.Execution.Status != domain.WorkflowExecutionWaiting || retried.Execution.BudgetUsage.Retries != 1 {
		t.Fatalf("retry=%+v err=%v", retried, err)
	}
	repeatedRetry, err := o.RetryWorkflowExecution(ctx, WorkflowExecutionOperationRequest{ExecutionID: failed.ID, IdempotencyKey: "retry"})
	if err != nil || !repeatedRetry.Idempotent {
		t.Fatalf("repeat retry=%+v err=%v", repeatedRetry, err)
	}

	simulation, err := o.SimulateWorkflow(ctx, SimulateWorkflowRequest{DefinitionDigest: revision.Digest, Input: json.RawMessage(`{}`)})
	if err != nil || !simulation.Valid || len(simulation.Nodes) != 2 || simulation.Nodes[0].WaitSignal != "continue" || len(simulation.PossibleTerminalNodes) != 1 {
		t.Fatalf("simulation=%+v err=%v", simulation, err)
	}
	if _, err := o.SimulateWorkflow(ctx, SimulateWorkflowRequest{DefinitionDigest: "sha256:missing", Input: json.RawMessage(`{}`)}); err == nil {
		t.Fatal("missing revision simulation should fail")
	}
}

func TestSimulateWorkflowProjectsEachSupportedNodeStrategy(t *testing.T) {
	ctx := context.Background()
	o, repos := newRelayOrchestrator(t, newFakeRunLauncher())
	revision := relayDefinition()
	revision.ID = uuid.New()
	revision.Key = "owner/simulation-strategies"
	revision.Digest = "sha256:simulation-strategies"
	revision.Definition.Key = revision.Key
	revision.Definition.EntryNode = "run"
	revision.Definition.Nodes = []domain.WorkflowNode{
		{ID: "run", Kind: domain.WorkflowNodeRun, Run: &domain.WorkflowRunNode{ProfileKey: "reviewer", RoleRef: "analysis"}},
		{ID: "continue", Kind: domain.WorkflowNodeContinue, Continue: &domain.WorkflowContinueNode{ConversationFromNode: "run"}},
		{ID: "child", Kind: domain.WorkflowNodeChild, Child: &domain.WorkflowChildNode{WorkflowKey: "owner/child", Version: "2.0.0"}},
		{ID: "wait", Kind: domain.WorkflowNodeWait, Wait: &domain.WorkflowWaitNode{Signal: "approval", TimeoutSeconds: 45}},
		{ID: "join", Kind: domain.WorkflowNodeJoin, Join: &domain.WorkflowJoinNode{Strategy: "quorum", Quorum: 2}},
		{ID: "branch", Kind: domain.WorkflowNodeBranch, Branch: &domain.WorkflowBranchNode{Parallel: true}},
		{ID: "end", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{Status: "succeeded"}},
	}
	if err := repos.Workflows.ActivateBatch(ctx, []*domain.WorkflowRevision{revision}); err != nil {
		t.Fatalf("activate: %v", err)
	}

	simulation, err := o.SimulateWorkflow(ctx, SimulateWorkflowRequest{Owner: " owner ", WorkflowKey: " owner/simulation-strategies ", Input: json.RawMessage(`{}`)})
	if err != nil || !simulation.Valid || len(simulation.Nodes) != len(revision.Definition.Nodes) {
		t.Fatalf("simulate = %+v, err=%v", simulation, err)
	}
	nodes := map[string]WorkflowNodePlan{}
	for _, node := range simulation.Nodes {
		nodes[node.NodeID] = node
	}
	if got := nodes["run"]; got.ExecutionStrategy != "fresh_run" || got.ProfileKey != "reviewer" || got.RoleRef != "analysis" {
		t.Fatalf("run projection = %+v", got)
	}
	if got := nodes["continue"]; got.ExecutionStrategy != "continue" || got.ContinuationSource != "run" {
		t.Fatalf("continue projection = %+v", got)
	}
	if got := nodes["child"]; got.ExecutionStrategy != "child_workflow" || got.ChildWorkflowKey != "owner/child" || got.ChildWorkflowVersion != "2.0.0" {
		t.Fatalf("child projection = %+v", got)
	}
	if got := nodes["wait"]; got.WaitSignal != "approval" || got.WaitTimeoutSeconds != 45 {
		t.Fatalf("wait projection = %+v", got)
	}
	if got := nodes["join"]; got.JoinStrategy != "quorum" || got.JoinQuorum != 2 {
		t.Fatalf("join projection = %+v", got)
	}
	if got := nodes["branch"]; !got.Parallel {
		t.Fatalf("branch projection = %+v", got)
	}
	if len(simulation.PossibleTerminalNodes) != 1 || simulation.PossibleTerminalNodes[0] != "end" {
		t.Fatalf("terminal nodes = %+v", simulation.PossibleTerminalNodes)
	}
}
