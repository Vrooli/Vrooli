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

func TestWorkflowTimeoutHonorsLongFormSettingsWithBound(t *testing.T) {
	fallback := 5 * time.Minute
	if got := workflowTimeout(map[string]any{"settings": map[string]any{"timeout_ms": float64(3900000)}}, fallback); got != 65*time.Minute {
		t.Fatalf("workflowTimeout = %s, want 65m", got)
	}
	if got := workflowTimeout(map[string]any{"settings": map[string]any{"timeout_ms": float64(3 * 60 * 60 * 1000)}}, fallback); got != 2*time.Hour {
		t.Fatalf("workflowTimeout cap = %s, want 2h", got)
	}
}

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
			if got := req.Header().Get("X-Vrooli-Test-Mode"); got != "" {
				t.Fatalf("execution control-plane header = %q, want it omitted", got)
			}
			if got := req.Msg.GetParameters().GetBrowserProfile().GetExtraHeaders()["X-Vrooli-Test-Mode"]; got != "1" {
				t.Fatalf("browser profile test-mode header = %q, want 1", got)
			}
			return connect.NewResponse(&basexecution.ExecuteAdhocResponse{ExecutionId: "execution-1", Status: basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING}), nil
		},
	))
	mux.Handle(apiconnect.ExecutionsServiceGetExecutionProcedure, connect.NewUnaryHandler(
		apiconnect.ExecutionsServiceGetExecutionProcedure,
		func(_ context.Context, req *connect.Request[basapi.GetExecutionRequest]) (*connect.Response[basapi.GetExecutionResponse], error) {
			if got := req.Header().Get("X-Vrooli-Test-Mode"); got != "" {
				t.Fatalf("status control-plane header = %q, want it omitted", got)
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

func TestExecuteAdhocCarriesElectronTargetAndValidationContext(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(apiconnect.WorkflowsServiceExecuteAdhocWorkflowProcedure, connect.NewUnaryHandler(
		apiconnect.WorkflowsServiceExecuteAdhocWorkflowProcedure,
		func(_ context.Context, req *connect.Request[basexecution.ExecuteAdhocRequest]) (*connect.Response[basexecution.ExecuteAdhocResponse], error) {
			target := req.Msg.GetOptions().GetElectronTarget()
			if target.GetTargetId() != "target-1" || target.GetRendererId() != "renderer-1" || target.GetCdpTransport() != "loopback-authenticated" {
				t.Fatalf("electron target = %+v", target)
			}
			validation := req.Msg.GetOptions().GetValidationContext()
			if validation.GetIsolationLeaseId() != "lease-1" || validation.GetWorkflowId() != "workflow-1" {
				t.Fatalf("validation context = %+v", validation)
			}
			return connect.NewResponse(&basexecution.ExecuteAdhocResponse{ExecutionId: "execution-electron", Status: basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING}), nil
		},
	))
	mux.Handle(apiconnect.ExecutionsServiceGetExecutionProcedure, connect.NewUnaryHandler(
		apiconnect.ExecutionsServiceGetExecutionProcedure,
		func(_ context.Context, _ *connect.Request[basapi.GetExecutionRequest]) (*connect.Response[basapi.GetExecutionResponse], error) {
			return connect.NewResponse(&basapi.GetExecutionResponse{Execution: &basexecution.Execution{
				ExecutionId: "execution-electron", Status: basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED, CompletedAt: timestamppb.Now(),
			}}), nil
		},
	))
	server := httptest.NewServer(mux)
	defer server.Close()
	client := NewConnectClient(server.URL, server.Client())
	client.timeout = time.Second
	_, err := client.ExecuteAdhoc(context.Background(), ExecuteRequest{
		Definition: map[string]any{"metadata": map[string]any{"name": "electron", "execution_mode": "observer"}, "nodes": []any{}},
		Options: ExecuteOptions{
			ElectronTarget:    &ElectronTarget{TargetID: "target-1", CDPEndpoint: "http://127.0.0.1:9222", RendererID: "renderer-1", RendererURL: "file:///app", ScenarioName: "sample", ArtifactDigest: "sha256:app", ContextID: "ctx-1", CDPTransport: "loopback-authenticated"},
			ValidationContext: &ValidationContext{ContextID: "ctx-1", ScenarioName: "sample", ArtifactDigest: "sha256:app", TargetID: "target-1", WorkflowID: "workflow-1", ProfileID: "normal", IsolationLeaseID: "lease-1"},
		},
	})
	if err != nil {
		t.Fatalf("ExecuteAdhoc: %v", err)
	}
}

func TestTimelinePropagatesIsolationHeaders(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(apiconnect.ExecutionsServiceGetExecutionTimelineProcedure, connect.NewUnaryHandler(
		apiconnect.ExecutionsServiceGetExecutionTimelineProcedure,
		func(_ context.Context, req *connect.Request[basapi.GetExecutionTimelineRequest]) (*connect.Response[bastimeline.ExecutionTimeline], error) {
			if got := req.Header().Get("X-Vrooli-Test-Mode"); got != "" {
				t.Fatalf("timeline control-plane header = %q, want it omitted", got)
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
