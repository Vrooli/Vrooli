package execution

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api/apiconnect"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestDefinitionToProtoPreservesNestedActionParams(t *testing.T) {
	def := map[string]any{
		"metadata": map[string]any{
			"name":           "dashboard-smoke",
			"execution_mode": "observer",
		},
		"nodes": []any{
			map[string]any{
				"id": "assert-page-loaded",
				"action": map[string]any{
					"type": "ACTION_TYPE_ASSERT",
					"assert": map[string]any{
						"selector": "@selector/dashboard.header",
						"mode":     "ASSERTION_MODE_TEXT_CONTAINS",
						"expected": map[string]any{"stringValue": "Test Genie"},
					},
				},
			},
		},
	}

	protoDef, err := definitionToProto(def)
	if err != nil {
		t.Fatalf("definitionToProto returned error: %v", err)
	}
	if len(protoDef.GetNodes()) != 1 {
		t.Fatalf("expected one node, got %d", len(protoDef.GetNodes()))
	}
	action := protoDef.GetNodes()[0].GetAction()
	if got := action.GetType(); got != basactions.ActionType_ACTION_TYPE_ASSERT {
		t.Fatalf("unexpected action type: %v", got)
	}
	assert := action.GetAssert()
	if assert == nil {
		t.Fatal("expected assert params to be preserved")
	}
	if got := assert.GetSelector(); got != "@selector/dashboard.header" {
		t.Fatalf("unexpected assert selector: %q", got)
	}
}

func TestDefinitionToProtoRejectsUnknownFields(t *testing.T) {
	def := map[string]any{
		"metadata": map[string]any{
			"name":           "dashboard-smoke",
			"execution_mode": "observer",
		},
		"unknown_future_field": "must not be discarded",
	}

	if _, err := definitionToProto(def); err == nil {
		t.Fatal("expected unknown workflow fields to be rejected")
	}
}

func TestExecuteAdhocStartsAsyncThenReturnsTerminalExecutionDetails(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(apiconnect.WorkflowsServiceExecuteAdhocWorkflowProcedure, connect.NewUnaryHandler(
		apiconnect.WorkflowsServiceExecuteAdhocWorkflowProcedure,
		func(_ context.Context, req *connect.Request[basexecution.ExecuteAdhocRequest]) (*connect.Response[basexecution.ExecuteAdhocResponse], error) {
			if req.Msg.GetWaitForCompletion() {
				t.Fatal("workflow-health must use BAS asynchronous execution")
			}
			if got := req.Header().Get("X-Vrooli-Test-Mode"); got != "1" {
				t.Fatalf("execution header = %q, want 1", got)
			}
			return connect.NewResponse(&basexecution.ExecuteAdhocResponse{ExecutionId: "execution-1", Status: basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING}), nil
		},
	))
	mux.Handle(apiconnect.ExecutionsServiceGetExecutionProcedure, connect.NewUnaryHandler(
		apiconnect.ExecutionsServiceGetExecutionProcedure,
		func(_ context.Context, req *connect.Request[basapi.GetExecutionRequest]) (*connect.Response[basapi.GetExecutionResponse], error) {
			if got := req.Header().Get("X-Vrooli-Test-Mode"); got != "1" {
				t.Fatalf("status header = %q, want 1", got)
			}
			if req.Msg.GetExecutionId() != "execution-1" {
				t.Fatalf("execution id = %q", req.Msg.GetExecutionId())
			}
			message := "selector [data-testid=save] remained disabled"
			return connect.NewResponse(&basapi.GetExecutionResponse{Execution: &basexecution.Execution{
				ExecutionId: "execution-1",
				Status:      basbase.ExecutionStatus_EXECUTION_STATUS_FAILED,
				CompletedAt: timestamppb.Now(),
				Error:       &message,
			}}), nil
		},
	))
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewConnectClient(server.URL, server.Client())
	client.timeout = time.Second
	result, err := client.ExecuteAdhoc(context.Background(), ExecuteRequest{Definition: map[string]any{
		"metadata": map[string]any{"name": "async-test", "execution_mode": "observer"},
		"nodes":    []any{},
	}, Parameters: Parameters{ExtraHeaders: map[string]string{"X-Vrooli-Test-Mode": "1"}}})
	if err != nil {
		t.Fatalf("ExecuteAdhoc: %v", err)
	}
	if result.ExecutionID != "execution-1" || result.Status != basbase.ExecutionStatus_EXECUTION_STATUS_FAILED {
		t.Fatalf("result = %+v", result)
	}
	if result.Error != "selector [data-testid=save] remained disabled" {
		t.Fatalf("error = %q", result.Error)
	}
}

func TestTimelinePropagatesIsolationHeaders(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(apiconnect.ExecutionsServiceGetExecutionTimelineProcedure, connect.NewUnaryHandler(
		apiconnect.ExecutionsServiceGetExecutionTimelineProcedure,
		func(_ context.Context, req *connect.Request[basapi.GetExecutionTimelineRequest]) (*connect.Response[bastimeline.ExecutionTimeline], error) {
			if got := req.Header().Get("X-Vrooli-Test-Mode"); got != "1" {
				t.Fatalf("timeline header = %q, want 1", got)
			}
			return connect.NewResponse(&bastimeline.ExecutionTimeline{}), nil
		},
	))
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewConnectClient(server.URL, server.Client())
	if _, err := client.Timeline(context.Background(), "execution-1", map[string]string{"X-Vrooli-Test-Mode": "1"}); err != nil {
		t.Fatalf("Timeline: %v", err)
	}
}
