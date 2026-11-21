package agentmanager

import (
	"context"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestStartWorkflowRunsGuardBeforeAgentManagerTransport(t *testing.T) {
	service := NewWorkflowServiceWithClient(NewHTTPClientWithResolver(func(context.Context) (string, error) { return "", errors.New("transport must not be called") }, nil))
	service.SetStartGuard(func(_ context.Context, workflowKey string) error {
		if workflowKey != "swarm-manager/plan-workshop-review" {
			t.Fatalf("key=%q", workflowKey)
		}
		return errors.New("plan-manager is stale")
	})
	input, _ := structpb.NewValue(map[string]any{})
	_, err := service.StartWorkflow(context.Background(), Invocation{Owner: "swarm-manager", WorkflowKey: "swarm-manager/plan-workshop-review", Input: input})
	if err == nil || !strings.Contains(err.Error(), "plan-manager is stale") {
		t.Fatalf("error=%v", err)
	}
}

func TestReviewWorkflowSnapshotPreservesBoundWorkerAndChildReviewEvidence(t *testing.T) {
	for _, failure := range []string{"", "consumer", "child", "run", "unavailable"} {
		t.Run(failure, func(t *testing.T) {
			input, _ := structpb.NewValue(map[string]any{"consumer": map[string]any{"executionId": "swarm-exec"}})
			output, _ := structpb.NewValue(map[string]any{"result": map[string]any{"handoff": "both protected tests passed"}})
			if failure == "consumer" {
				input, _ = structpb.NewValue(map[string]any{"consumer": map[string]any{"executionId": "different-exec"}})
			}
			root := &domainpb.WorkflowExecution{Id: "main", Owner: "swarm-manager", WorkflowKey: "swarm-manager/review", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, ApprovalDigest: "approval", GrantDigest: "grant", Input: input, Output: output, BudgetUsage: &domainpb.WorkflowBudgetUsage{Tokens: 80, AccountingComplete: true}}
			childParent := "main"
			if failure == "child" {
				childParent = "foreign-parent"
			}
			child := &domainpb.WorkflowExecution{Id: "child", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, ParentExecutionId: childParent, ParentAttemptId: "review-1", ApprovalDigest: "approval", Output: output}
			workerKey := "run-key"
			if failure == "run" {
				workerKey = "foreign-run-key"
			}
			sandbox := "sandbox"
			run := &domainpb.Run{Id: "worker", IdempotencyKey: workerKey, Status: domainpb.RunStatus_RUN_STATUS_COMPLETE, SandboxId: &sandbox, FinalizationStatus: domainpb.RunFinalizationStatus_RUN_FINALIZATION_STATUS_SUCCEEDED, Summary: &domainpb.RunSummary{Description: "observed protected tests 2/2", TokensUsed: 50}}
			workerAttempt := &domainpb.WorkflowNodeAttempt{Id: "slice-1", ExecutionId: "main", NodeId: "slice", RunId: "worker", ConversationId: "fresh-conversation", IdempotencyKey: "run-key"}
			childAttempt := &domainpb.WorkflowNodeAttempt{Id: "review-1", ExecutionId: "main", NodeId: "review", ChildExecutionId: "child"}
			fixture := &workflowRPCFixture{
				executions: map[string]*domainpb.WorkflowExecution{"main": root, "child": child},
				traces:     map[string]*apipb.GetWorkflowExecutionTraceResponse{"main": {Execution: root, Attempts: []*domainpb.WorkflowNodeAttempt{workerAttempt, childAttempt}}, "child": {Execution: child}},
				runs:       map[string]*domainpb.Run{"worker": run},
			}
			if failure == "unavailable" {
				fixture.resultErrors = map[string]error{"child": connect.NewError(connect.CodeUnavailable, errors.New("owner unavailable"))}
			}
			service := NewWorkflowServiceWithClient(workflowTestClient(t, fixture))
			got, err := service.ReviewWorkflowSnapshot(context.Background(), "main", "swarm-exec", "approval", "grant")
			if failure != "" {
				if err == nil || len(got) != 0 {
					t.Fatalf("foreign or unavailable evidence admitted: %s err=%v", got, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			for _, expected := range []string{"fresh-conversation", "observed protected tests 2/2", "RUN_FINALIZATION_STATUS_SUCCEEDED", `"handoff":"both protected tests passed"`, `"tokens":80`} {
				if !strings.Contains(string(got), expected) {
					t.Errorf("missing %s in %s", expected, got)
				}
			}
			if strings.Contains(string(got), "resolvedConfig") {
				t.Fatal("review copied private runner configuration")
			}
		})
	}
}
