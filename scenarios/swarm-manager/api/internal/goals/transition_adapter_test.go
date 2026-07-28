package goals

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/transitionrun"
	"swarm-manager/internal/transitionrunner"
	"swarm-manager/internal/transitions"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

type goalTransitionWorkflow struct {
	start      agentmanager.WorkflowStart
	completion agentmanager.InvocationCompletion
	invocation agentmanager.Invocation
}

func (f *goalTransitionWorkflow) StartWorkflow(_ context.Context, in agentmanager.Invocation) (agentmanager.WorkflowStart, error) {
	f.invocation = in
	return f.start, nil
}

func (f *goalTransitionWorkflow) CollectWorkflow(context.Context, string) (agentmanager.InvocationCompletion, error) {
	return f.completion, nil
}

type goalTransitionRecorder struct{ calls int }

func (r *goalTransitionRecorder) RecordGoalWorkflowProposals(context.Context, GoalWorkflowProposal) (GoalWorkflowProposalReceipt, error) {
	r.calls++
	return GoalWorkflowProposalReceipt{SessionID: "session-1", ProposalIDs: []string{"proposal-1"}}, nil
}

func goalTransitionRegistry(t *testing.T) transitions.Registry {
	t.Helper()
	dir := t.TempDir()
	contents := []byte(`{"schemaVersion":"swarm-transition/v1","key":"goal.plan","subject":"goal","kind":"workflow","workflow":{"owner":"swarm-manager","key":"goal-plan"},"inputContract":"goal-plan-input/v1","terminalOutcomes":["proposed"],"applyAction":"apply_goal_proposal"}`)
	if err := os.WriteFile(filepath.Join(dir, "goal-plan.json"), contents, 0o600); err != nil {
		t.Fatal(err)
	}
	registry, err := transitions.LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

// [REQ:SWM-P0-012] goal planning projects typed proposals exactly once.
func TestGoalTransitionAdapterProjectsSharedJournalAndRecordsReceipt(t *testing.T) {
	service := newTestService(t, nil)
	goal, err := service.Create(CreateRequest{Name: "release", Title: "Release"})
	if err != nil {
		t.Fatal(err)
	}
	output, err := structpb.NewValue(map[string]any{"result": map[string]any{"outcome": "proposed", "summary": "ready", "proposals": []any{}}})
	if err != nil {
		t.Fatal(err)
	}
	workflow := &goalTransitionWorkflow{start: agentmanager.WorkflowStart{ExecutionID: "goal-exec", DefinitionDigest: "sha256:goal"}, completion: agentmanager.InvocationCompletion{ExecutionID: "goal-exec", DefinitionDigest: "sha256:goal", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, Output: output}}
	runner := transitionrunner.New(goalTransitionRegistry(t), workflow, transitionrun.NewFileStore(t.TempDir()), nil)
	handler := NewHandler(service)
	recorder := &goalTransitionRecorder{}
	handler.SetWorkflowProposalRecorder(recorder)
	handler.RegisterTransitionAdapter(runner)
	handler.SetTransitionRunner(runner)

	if _, err := runner.Start(context.Background(), "goal.plan", goal.Goal.Name); err != nil {
		t.Fatal(err)
	}
	pending, err := handler.ListPendingWorkflows()
	if err != nil || len(pending) != 1 || pending[0].ExecutionID != "goal-exec" || pending[0].Stale {
		t.Fatalf("pending = %#v, %v", pending, err)
	}
	applied, err := handler.ApplyWorkflowResult(context.Background(), goal.Goal.Name, "goal-exec")
	if err != nil {
		t.Fatal(err)
	}
	if applied.SessionID != "session-1" || recorder.calls != 1 || applied.AlreadyApplied {
		t.Fatalf("applied = %#v calls=%d", applied, recorder.calls)
	}
	replay, err := handler.ApplyWorkflowResult(context.Background(), goal.Goal.Name, "goal-exec")
	if err != nil || !replay.AlreadyApplied || recorder.calls != 1 {
		t.Fatalf("replay = %#v, %v calls=%d", replay, err, recorder.calls)
	}
	pending, err = handler.ListPendingWorkflows()
	if err != nil || len(pending) != 0 {
		t.Fatalf("pending after apply = %#v, %v", pending, err)
	}
}
