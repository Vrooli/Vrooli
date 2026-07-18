package agentmanager

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestPlanRepairWorkflowUsesGenericInvocationBoundary(t *testing.T) {
	input, _ := structpb.NewValue(map[string]any{"entity": map[string]any{"kind": "fix", "name": "repair", "version": "v1"}})
	output, _ := structpb.NewValue(map[string]any{"result": map[string]any{"outcome": "ready", "candidatePlan": "# repaired", "summary": "fixed findings"}})
	execution := &domainpb.WorkflowExecution{Id: "repair-1", DefinitionDigest: "sha256:repair", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, Input: input, Output: output}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/workflow-executions":
			body, _ := io.ReadAll(r.Body)
			for _, want := range []string{`"workflowKey":"swarm-manager/plan-repair"`, `"frontierDigest":"frontier"`, `"maxRepairAttempts":2`} {
				if !strings.Contains(string(body), want) {
					t.Fatalf("start body missing %s: %s", want, body)
				}
			}
			writeWorkflowProto(t, w, &apipb.WorkflowExecutionResponse{Execution: execution}, http.StatusAccepted)
		case strings.HasSuffix(r.URL.Path, "/wait"):
			writeWorkflowProto(t, w, &apipb.WaitWorkflowExecutionResponse{Execution: execution}, http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/result"):
			writeWorkflowProto(t, w, &apipb.WorkflowExecutionResponse{Execution: execution}, http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/trace"):
			writeWorkflowProto(t, w, &apipb.GetWorkflowExecutionTraceResponse{Execution: execution, Attempts: []*domainpb.WorkflowNodeAttempt{{NodeId: "repair", RunId: "run-repair"}}}, http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	service := NewWorkflowServiceWithClient(NewHTTPClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client()))
	start, err := service.StartPlanRepair(context.Background(), PlanRepairSnapshot{EntityKind: "fix", EntityName: "repair", EntityVersion: "v1", PlanReference: "plan-1", PlanContent: "# bad", FrontierDigest: "frontier", ValidationFindings: []any{map[string]any{"code": "missing"}}, CheckedAt: "2026-07-17T12:00:00Z", MaxRepairAttempts: 2}, "repair-1")
	if err != nil || start.RunID != "run-repair" {
		t.Fatalf("start=%#v err=%v", start, err)
	}
	completion, err := service.CollectPlanRepair(context.Background(), start.ExecutionID)
	if err != nil || !strings.Contains(string(completion.Result), `"candidatePlan":"# repaired"`) {
		t.Fatalf("completion=%#v err=%v", completion, err)
	}
}
