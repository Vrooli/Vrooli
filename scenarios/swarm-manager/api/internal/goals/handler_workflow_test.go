package goals

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/transitions"

	"github.com/gorilla/mux"
	"google.golang.org/protobuf/types/known/structpb"
)

type captureGoalWorkflow struct{ invocation WorkflowInvocation }

func (f *captureGoalWorkflow) StartWorkflow(_ context.Context, in WorkflowInvocation) (WorkflowStart, error) {
	f.invocation = in
	return WorkflowStart{ExecutionID: "wf-1", RunID: "run-1", DefinitionDigest: "sha256:test"}, nil
}

func (f *captureGoalWorkflow) CollectWorkflow(_ context.Context, _ string) (WorkflowCompletion, error) {
	return WorkflowCompletion{}, nil
}

type completedGoalWorkflow struct {
	captureGoalWorkflow
	completion WorkflowCompletion
}

func (f *completedGoalWorkflow) CollectWorkflow(_ context.Context, _ string) (WorkflowCompletion, error) {
	return f.completion, nil
}

type captureGoalProposalRecorder struct{ proposals []GoalWorkflowProposal }

func (r *captureGoalProposalRecorder) RecordGoalWorkflowProposals(_ context.Context, proposal GoalWorkflowProposal) (GoalWorkflowProposalReceipt, error) {
	r.proposals = append(r.proposals, proposal)
	return GoalWorkflowProposalReceipt{SessionID: "sess-workflow", ProposalIDs: []string{"prop-workflow"}}, nil
}

func goalWorkflowRegistry(t *testing.T) transitions.Registry {
	t.Helper()
	registry, err := transitions.LoadFS(fstest.MapFS{"registry.json": {Data: []byte(`[
{"schemaVersion":"swarm-transition/v1","key":"goal.plan","subject":"goal","kind":"workflow","workflow":{"owner":"swarm-manager","key":"swarm-manager/goal-plan"},"inputContract":"goal/v1","terminalOutcomes":["proposed"],"applyAction":"apply_goal_proposal"},
{"schemaVersion":"swarm-transition/v1","key":"goal.discover","subject":"goal","kind":"workflow","workflow":{"owner":"swarm-manager","key":"swarm-manager/goal-discover"},"inputContract":"goal/v1","terminalOutcomes":["proposed"],"applyAction":"apply_goal_proposal"},
{"schemaVersion":"swarm-transition/v1","key":"milestone.review","subject":"milestone","kind":"workflow","workflow":{"owner":"swarm-manager","key":"swarm-manager/milestone-review"},"inputContract":"milestone/v1","terminalOutcomes":["delivered"],"applyAction":"apply_milestone_review"}
]`)}}, ".")
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func TestGoalWorkflowLaunchPinsGoalSnapshot(t *testing.T) {
	svc := newTestService(t, []backlog.BacklogItem{item("execute", "a", "ready", nil)})
	if _, err := svc.Create(CreateRequest{Name: "delivery", Targets: []string{"execute/a"}}); err != nil {
		t.Fatal(err)
	}
	workflow := &captureGoalWorkflow{}
	handler := NewHandler(svc)
	handler.SetWorkflow(workflow, goalWorkflowRegistry(t))
	router := mux.NewRouter()
	handler.RegisterRoutes(router)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/goals/delivery/plan-run", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", response.Code, response.Body.String())
	}
	if workflow.invocation.WorkflowKey != "swarm-manager/goal-plan" || workflow.invocation.IdempotencyKey == "" {
		t.Fatalf("invocation = %#v", workflow.invocation)
	}
	payload := workflow.invocation.Input.AsInterface().(map[string]any)
	entity := payload["entity"].(map[string]any)
	if entity["kind"] != "goal" || entity["name"] != "delivery" || entity["version"] == "" {
		t.Fatalf("entity = %#v", entity)
	}
}

func TestMilestoneWorkflowLaunchRejectsUnknownMilestone(t *testing.T) {
	svc := newTestService(t, nil)
	if _, err := svc.Create(CreateRequest{Name: "delivery"}); err != nil {
		t.Fatal(err)
	}
	handler := NewHandler(svc)
	handler.SetWorkflow(&captureGoalWorkflow{}, goalWorkflowRegistry(t))
	router := mux.NewRouter()
	handler.RegisterRoutes(router)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/goals/delivery/milestones/missing/review-run", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d", response.Code)
	}
}

// [REQ:SWM-P0-012] goal planning loop files typed proposals into the decision inbox
func TestGoalWorkflowApplyFilesTypedProposalExactlyOnce(t *testing.T) {
	svc := newTestService(t, []backlog.BacklogItem{item("execute", "a", "ready", nil)})
	created, err := svc.Create(CreateRequest{Name: "delivery", Targets: []string{"execute/a"}})
	if err != nil {
		t.Fatal(err)
	}
	input, err := structpb.NewValue(map[string]any{"entity": map[string]any{"kind": "goal", "name": "delivery", "version": created.Goal.Updated}})
	if err != nil {
		t.Fatal(err)
	}
	output, err := structpb.NewValue(map[string]any{
		"result": map[string]any{
			"outcome": "proposed",
			"summary": "Create the delivery milestone",
			"proposals": []any{map[string]any{
				"form": "mutation_list", "base_version": created.Goal.Updated,
				"mutations": []any{map[string]any{"id": "m1", "op": "create_milestone", "goal_milestone": map[string]any{"name": "delivery", "title": "Delivery"}}},
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	workflow := &completedGoalWorkflow{completion: WorkflowCompletion{ExecutionID: "wf-1", DefinitionDigest: "sha256:test", Succeeded: true, Input: input, Output: output}}
	recorder := &captureGoalProposalRecorder{}
	handler := NewHandler(svc)
	handler.SetWorkflow(workflow, goalWorkflowRegistry(t))
	handler.SetWorkflowProposalRecorder(recorder)
	if err := handler.writeWorkflowPending("delivery", workflowPending{ExecutionID: "wf-1", DefinitionDigest: "sha256:test", Transition: "goal.plan", GoalVersion: created.Goal.Updated}); err != nil {
		t.Fatal(err)
	}
	result, err := handler.applyWorkflow(context.Background(), "delivery", "wf-1")
	if err != nil {
		t.Fatal(err)
	}
	if result["session_id"] != "sess-workflow" || len(recorder.proposals) != 1 || len(recorder.proposals[0].Payloads) != 1 {
		t.Fatalf("apply result = %#v, proposals = %#v", result, recorder.proposals)
	}
	result, err = handler.applyWorkflow(context.Background(), "delivery", "wf-1")
	if err != nil || result["already_applied"] != true || len(recorder.proposals) != 1 {
		t.Fatalf("idempotent result = %#v, err = %v, proposals = %#v", result, err, recorder.proposals)
	}
}
