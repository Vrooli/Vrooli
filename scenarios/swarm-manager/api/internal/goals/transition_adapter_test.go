package goals

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/attempt"
	"swarm-manager/internal/attemptstore"
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
	attempts, err := attemptstore.LoadRounds(service.GoalDir(goal.Goal.Name), filepath.Join("attempts", "goal.plan", "goal-exec"), func(data []byte) (attempt.Attempt, error) {
		var value attempt.Attempt
		return value, json.Unmarshal(data, &value)
	})
	if err != nil || len(attempts) != 1 {
		t.Fatalf("workflow attempts = %#v, %v", attempts, err)
	}
	if got := attempts[0]; got.SubjectKind != "goal" || got.SubjectRef != goal.Goal.Name || got.TransitionKey != "goal.plan" || got.Assessment != "ready" || got.Status != "complete" {
		t.Fatalf("persisted attempt = %#v", got)
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

func TestDecodeMilestoneReviewRetainsCriterionEvidence(t *testing.T) {
	output, err := structpb.NewValue(map[string]any{"result": map[string]any{
		"verdict": "delivered", "assessment": "all criteria are supported", "proposals": []any{},
		"criterion_verdicts": []any{map[string]any{"criterion": "criterion-1", "verdict": "delivered", "evidence": []any{"e1"}}},
		"evidence": []any{map[string]any{
			"id": "e1", "criterion_id": "criterion-1", "settlement": "settled", "producer": "test-genie", "trust": "observed", "title": "Unit phase passed",
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeWorkflowResult(output, "milestone.review")
	if err != nil {
		t.Fatalf("decodeWorkflowResult: %v", err)
	}
	if len(decoded.Evidence) != 1 || decoded.Evidence[0].CriterionID != "criterion-1" || decoded.Evidence[0].Settlement != "settled" || decoded.Evidence[0].Producer != "test-genie" || decoded.Evidence[0].Trust != "observed" {
		t.Fatalf("decoded evidence = %#v", decoded.Evidence)
	}
}
