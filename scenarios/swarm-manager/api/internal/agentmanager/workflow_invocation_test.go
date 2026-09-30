package agentmanager

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

func TestStartWorkflowRunsGuardBeforeAgentManagerTransport(t *testing.T) {
	service := NewWorkflowServiceWithClient(NewHTTPClientWithResolver(func(context.Context) (string, error) {
		return "", errors.New("transport must not be called")
	}, nil))
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
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("review evidence mutated owner: %s", r.Method)
					http.Error(w, "mutation", 500)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				var body string
				switch r.URL.Path {
				case "/api/v1/workflow-executions/main/result":
					consumer := "swarm-exec"
					if failure == "consumer" {
						consumer = "different-exec"
					}
					body = fmt.Sprintf(`{"execution":{"id":"main","status":"WORKFLOW_EXECUTION_STATUS_SUCCEEDED","approvalDigest":"approval","grantDigest":"grant","input":{"consumer":{"executionId":%q}},"budgetUsage":{"tokens":80,"accountingComplete":true},"output":{"result":{"handoff":"both protected tests passed"}}}}`, consumer)
				case "/api/v1/workflow-executions/main/trace":
					body = `{"attempts":[{"id":"slice-1","executionId":"main","nodeId":"slice","runId":"worker","conversationId":"fresh-conversation","idempotencyKey":"run-key"},{"id":"review-1","executionId":"main","nodeId":"review","childExecutionId":"child"}]}`
				case "/api/v1/runs/worker":
					key := "run-key"
					if failure == "run" {
						key = "foreign-run-key"
					}
					body = fmt.Sprintf(`{"run":{"id":"worker","idempotencyKey":%q,"status":"RUN_STATUS_COMPLETE","sandboxId":"sandbox","finalizationStatus":"RUN_FINALIZATION_STATUS_SUCCEEDED","summary":{"description":"observed protected tests 2/2","tokensUsed":50},"resolvedConfig":{"model":"must-not-copy-private-config"}}}`, key)
				case "/api/v1/workflow-executions/child/result":
					if failure == "unavailable" {
						http.Error(w, "owner unavailable", 503)
						return
					}
					parent := "main"
					if failure == "child" {
						parent = "foreign-parent"
					}
					body = fmt.Sprintf(`{"execution":{"id":"child","status":"WORKFLOW_EXECUTION_STATUS_SUCCEEDED","parentExecutionId":%q,"parentAttemptId":"review-1","approvalDigest":"approval","output":{"accepted":true}}}`, parent)
				case "/api/v1/workflow-executions/child/trace":
					body = `{"attempts":[]}`
				default:
					t.Errorf("unexpected path: %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				fmt.Fprint(w, body)
			}))
			defer server.Close()
			service := NewWorkflowServiceWithClient(NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client()))
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
			for _, expected := range []string{"fresh-conversation", "observed protected tests 2/2", "RUN_FINALIZATION_STATUS_SUCCEEDED", `"accepted":true`, `"tokens":80`} {
				if !strings.Contains(string(got), expected) {
					t.Errorf("missing %s in %s", expected, got)
				}
			}
			if strings.Contains(string(got), "must-not-copy-private-config") {
				t.Fatal("review copied private runner configuration")
			}
		})
	}
}
