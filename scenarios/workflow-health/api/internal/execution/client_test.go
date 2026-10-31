package execution

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
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
			if got := req.Msg.GetParameters().GetStartUrl(); got != "" {
				t.Fatalf("legacy execution start URL = %q, want empty", got)
			}
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

func TestExecuteAdhocCarriesAppTargetAndValidationContext(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(apiconnect.WorkflowsServiceExecuteAdhocWorkflowProcedure, connect.NewUnaryHandler(
		apiconnect.WorkflowsServiceExecuteAdhocWorkflowProcedure,
		func(_ context.Context, req *connect.Request[basexecution.ExecuteAdhocRequest]) (*connect.Response[basexecution.ExecuteAdhocResponse], error) {
			if got := req.Msg.GetParameters().GetStartUrl(); got != "file:///app" {
				t.Fatalf("bound Electron start URL = %q, want renderer URL", got)
			}
			target := req.Msg.GetOptions().GetAppTarget()
			if target.GetTargetKind() != "electron" || target.GetTargetId() != "target-1" || target.GetRendererId() != "renderer-1" || target.GetCdpTransport() != "loopback-authenticated" {
				t.Fatalf("app target = %+v", target)
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

func TestControlPlaneHeadersExcludeTargetAuthentication(t *testing.T) {
	got := controlPlaneHeaders(map[string]string{
		"Authorization":          "LocalSession target-token",
		"X-Vrooli-Test-Mode":     "1",
		"X-Vrooli-Test-Lease-ID": "lease-1",
		"X-Trace":                "trace-1",
	})
	if len(got) != 1 || got["X-Trace"] != "trace-1" {
		t.Fatalf("control-plane headers = %#v, want only X-Trace", got)
	}
}

func TestExecuteAdhocRetainsOneChildAcrossObservationWindowsAndTransientRead(t *testing.T) {
	var starts, reads, stops atomic.Int32
	mux := http.NewServeMux()
	mux.Handle(apiconnect.WorkflowsServiceExecuteAdhocWorkflowProcedure, connect.NewUnaryHandler(apiconnect.WorkflowsServiceExecuteAdhocWorkflowProcedure, func(context.Context, *connect.Request[basexecution.ExecuteAdhocRequest]) (*connect.Response[basexecution.ExecuteAdhocResponse], error) {
		starts.Add(1)
		return connect.NewResponse(&basexecution.ExecuteAdhocResponse{ExecutionId: "same-child"}), nil
	}))
	mux.Handle(apiconnect.ExecutionsServiceGetExecutionProcedure, connect.NewUnaryHandler(apiconnect.ExecutionsServiceGetExecutionProcedure, func(_ context.Context, req *connect.Request[basapi.GetExecutionRequest]) (*connect.Response[basapi.GetExecutionResponse], error) {
		if req.Msg.ExecutionId != "same-child" {
			t.Error("observation switched execution")
		}
		n := reads.Add(1)
		if n == 2 {
			return nil, connect.NewError(connect.CodeUnavailable, errors.New("temporary status outage"))
		}
		e := &basexecution.Execution{ExecutionId: "same-child", Status: basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING}
		if n >= 3 {
			e.Status = basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED
			e.CompletedAt = timestamppb.Now()
		}
		return connect.NewResponse(&basapi.GetExecutionResponse{Execution: e}), nil
	}))
	mux.Handle(apiconnect.ExecutionsServiceStopExecutionProcedure, connect.NewUnaryHandler(apiconnect.ExecutionsServiceStopExecutionProcedure, func(context.Context, *connect.Request[basapi.StopExecutionRequest]) (*connect.Response[basapi.StopExecutionResponse], error) {
		stops.Add(1)
		return connect.NewResponse(&basapi.StopExecutionResponse{Status: "stopped"}), nil
	}))
	server := httptest.NewServer(mux)
	defer server.Close()
	client := NewConnectClient(server.URL, server.Client())
	client.timeout = 40 * time.Millisecond
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	result, err := client.ExecuteAdhoc(ctx, ExecuteRequest{Definition: map[string]any{"metadata": map[string]any{"name": "long-owner", "execution_mode": "observer"}, "nodes": []any{}}})
	if err != nil || result.Status != basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED || starts.Load() != 1 || stops.Load() != 0 {
		t.Fatalf("observation window ended or replaced durable work: result=%v err=%v starts=%d stops=%d", result, err, starts.Load(), stops.Load())
	}
}

func TestExecuteAdhocOwnerAbortJoinsBrowserCleanupAndPreservesChildIdentity(t *testing.T) {
	seen, stopping, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var observed sync.Once
	var terminal atomic.Bool
	mux := http.NewServeMux()
	mux.Handle(apiconnect.WorkflowsServiceExecuteAdhocWorkflowProcedure, connect.NewUnaryHandler(apiconnect.WorkflowsServiceExecuteAdhocWorkflowProcedure, func(context.Context, *connect.Request[basexecution.ExecuteAdhocRequest]) (*connect.Response[basexecution.ExecuteAdhocResponse], error) {
		return connect.NewResponse(&basexecution.ExecuteAdhocResponse{ExecutionId: "abort-child"}), nil
	}))
	mux.Handle(apiconnect.ExecutionsServiceGetExecutionProcedure, connect.NewUnaryHandler(apiconnect.ExecutionsServiceGetExecutionProcedure, func(context.Context, *connect.Request[basapi.GetExecutionRequest]) (*connect.Response[basapi.GetExecutionResponse], error) {
		observed.Do(func() { close(seen) })
		e := &basexecution.Execution{ExecutionId: "abort-child", Status: basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING}
		if terminal.Load() {
			e.Status = basbase.ExecutionStatus_EXECUTION_STATUS_CANCELLED
			e.CompletedAt = timestamppb.Now()
		}
		return connect.NewResponse(&basapi.GetExecutionResponse{Execution: e}), nil
	}))
	mux.Handle(apiconnect.ExecutionsServiceStopExecutionProcedure, connect.NewUnaryHandler(apiconnect.ExecutionsServiceStopExecutionProcedure, func(ctx context.Context, req *connect.Request[basapi.StopExecutionRequest]) (*connect.Response[basapi.StopExecutionResponse], error) {
		if req.Msg.ExecutionId != "abort-child" {
			t.Error("aborted another execution")
		}
		close(stopping)
		select {
		case <-release:
			terminal.Store(true)
			return connect.NewResponse(&basapi.StopExecutionResponse{Status: "stopped"}), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}))
	server := httptest.NewServer(mux)
	defer server.Close()
	client := NewConnectClient(server.URL, server.Client())
	client.timeout = 10 * time.Second
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	type returned struct {
		result *ExecuteResult
		err    error
	}
	done := make(chan returned, 1)
	go func() {
		r, e := client.ExecuteAdhoc(ctx, ExecuteRequest{Definition: map[string]any{"metadata": map[string]any{"name": "abort-owner", "execution_mode": "observer"}, "nodes": []any{}}})
		done <- returned{r, e}
	}()
	<-seen
	cancel()
	<-stopping
	select {
	case r := <-done:
		t.Fatalf("returned before browser cleanup: %+v", r)
	default:
	}
	close(release)
	r := <-done
	if !errors.Is(r.err, context.Canceled) || r.result == nil || r.result.ExecutionID != "abort-child" || !terminal.Load() {
		t.Fatalf("unsettled or unidentified cancellation: %+v", r)
	}
}

// BAS records terminal status before its cancellation owner finishes cleanup.
// A timed-out Stop must be retried and joined even when Get reports CompletedAt.
func TestOwnerAbortDoesNotMistakeTerminalStatusForCleanupAfterStopTimeout(t *testing.T) {
	var stops atomic.Int32
	secondStop, release := make(chan struct{}), make(chan struct{})
	mux := http.NewServeMux()
	mux.Handle(apiconnect.ExecutionsServiceStopExecutionProcedure, connect.NewUnaryHandler(apiconnect.ExecutionsServiceStopExecutionProcedure, func(ctx context.Context, req *connect.Request[basapi.StopExecutionRequest]) (*connect.Response[basapi.StopExecutionResponse], error) {
		if req.Msg.ExecutionId != "cleanup-owner" {
			t.Error("switched cancellation owner")
		}
		if stops.Add(1) == 1 {
			<-ctx.Done()
			return nil, ctx.Err()
		}
		close(secondStop)
		select {
		case <-release:
			return connect.NewResponse(&basapi.StopExecutionResponse{Status: "stopped"}), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}))
	mux.Handle(apiconnect.ExecutionsServiceGetExecutionProcedure, connect.NewUnaryHandler(apiconnect.ExecutionsServiceGetExecutionProcedure, func(context.Context, *connect.Request[basapi.GetExecutionRequest]) (*connect.Response[basapi.GetExecutionResponse], error) {
		return connect.NewResponse(&basapi.GetExecutionResponse{Execution: &basexecution.Execution{ExecutionId: "cleanup-owner", Status: basbase.ExecutionStatus_EXECUTION_STATUS_CANCELLED, CompletedAt: timestamppb.Now()}}), nil
	}))
	server := httptest.NewServer(mux)
	defer server.Close()
	client := NewConnectClient(server.URL, server.Client())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	type outcome struct {
		result *ExecuteResult
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		r, e := client.settleCancelledExecution(ctx, "cleanup-owner", nil, 100*time.Millisecond)
		done <- outcome{r, e}
	}()
	select {
	case got := <-done:
		t.Fatalf("released fixture on terminal stamp without cleanup acknowledgement: %+v", got)
	case <-secondStop:
	case <-time.After(5 * time.Second):
		t.Fatal("did not retry the timed-out cleanup owner")
	}
	select {
	case got := <-done:
		t.Fatalf("released fixture before owner joined: %+v", got)
	default:
	}
	close(release)
	select {
	case got := <-done:
		if !errors.Is(got.err, context.Canceled) || got.result == nil || got.result.ExecutionID != "cleanup-owner" || stops.Load() != 2 {
			t.Fatalf("incorrect cancellation disposition: %+v stops=%d", got, stops.Load())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("did not return after cleanup acknowledgement")
	}
}
