//go:build legacyproto
// +build legacyproto

package execution

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"test-genie/internal/basprobe/types"

	"connectrpc.com/connect"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api/apiconnect"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basexecution "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/encoding/protojson"
)

func timelineEntry(actionType basactions.ActionType, status basbase.StepStatus, success bool) *bastimeline.TimelineEntry {
	entry := &bastimeline.TimelineEntry{
		Action: &basactions.ActionDefinition{Type: actionType},
		Context: &basbase.EventContext{
			Success: boolPtr(success),
		},
	}
	if status != basbase.StepStatus_STEP_STATUS_UNSPECIFIED {
		entry.Aggregates = &bastimeline.TimelineEntryAggregates{Status: status}
	}
	if actionType == basactions.ActionType_ACTION_TYPE_ASSERT {
		entry.Context.Assertion = &basbase.AssertionResult{Success: success}
	}
	return entry
}

func connectWorkflowServer(handler func(context.Context, *connect.Request[basexecution.ExecuteAdhocRequest]) (*connect.Response[basexecution.ExecuteAdhocResponse], error)) *httptest.Server {
	mux := http.NewServeMux()
	mux.Handle(apiconnect.WorkflowsServiceExecuteAdhocWorkflowProcedure, connect.NewUnaryHandler(
		apiconnect.WorkflowsServiceExecuteAdhocWorkflowProcedure,
		handler,
	))
	return httptest.NewServer(mux)
}

func connectExecutionServer(
	statusHandler func(context.Context, *connect.Request[basapi.GetExecutionRequest]) (*connect.Response[basapi.GetExecutionResponse], error),
	timelineHandler func(context.Context, *connect.Request[basapi.GetExecutionTimelineRequest]) (*connect.Response[bastimeline.ExecutionTimeline], error),
) *httptest.Server {
	mux := http.NewServeMux()
	if statusHandler != nil {
		mux.Handle(apiconnect.ExecutionsServiceGetExecutionProcedure, connect.NewUnaryHandler(
			apiconnect.ExecutionsServiceGetExecutionProcedure,
			statusHandler,
		))
	}
	if timelineHandler != nil {
		mux.Handle(apiconnect.ExecutionsServiceGetExecutionTimelineProcedure, connect.NewUnaryHandler(
			apiconnect.ExecutionsServiceGetExecutionTimelineProcedure,
			timelineHandler,
		))
	}
	return httptest.NewServer(mux)
}

func TestClientHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("expected /health, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestClientHealthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.Health(context.Background())
	if err == nil {
		t.Fatal("expected error for unhealthy status")
	}
}

func TestClientExecuteWorkflow(t *testing.T) {
	expectedID := "exec-123"
	server := connectWorkflowServer(func(_ context.Context, req *connect.Request[basexecution.ExecuteAdhocRequest]) (*connect.Response[basexecution.ExecuteAdhocResponse], error) {
		if req.Msg.GetMetadata().GetName() != "test workflow" {
			t.Errorf("expected workflow metadata name, got %q", req.Msg.GetMetadata().GetName())
		}
		return connect.NewResponse(&basexecution.ExecuteAdhocResponse{ExecutionId: expectedID}), nil
	})
	defer server.Close()

	client := NewClient(server.URL)
	definition := map[string]any{"nodes": []any{}, "edges": []any{}}
	id, err := client.ExecuteWorkflow(context.Background(), definition, "test workflow", "test workflow")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if id != expectedID {
		t.Errorf("expected %s, got %s", expectedID, id)
	}
}

func TestClientExecuteWorkflowError(t *testing.T) {
	server := connectWorkflowServer(func(context.Context, *connect.Request[basexecution.ExecuteAdhocRequest]) (*connect.Response[basexecution.ExecuteAdhocResponse], error) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid workflow"))
	})
	defer server.Close()

	client := NewClient(server.URL)
	definition := map[string]any{"nodes": []any{}}
	_, err := client.ExecuteWorkflow(context.Background(), definition, "test", "test")
	if err == nil {
		t.Fatal("expected error for failed execution")
	}
	if !strings.Contains(err.Error(), "invalid workflow") {
		t.Errorf("expected error to contain response body, got: %v", err)
	}
}

func TestClientExecuteWorkflowMissingID(t *testing.T) {
	server := connectWorkflowServer(func(context.Context, *connect.Request[basexecution.ExecuteAdhocRequest]) (*connect.Response[basexecution.ExecuteAdhocResponse], error) {
		return connect.NewResponse(&basexecution.ExecuteAdhocResponse{}), nil
	})
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.ExecuteWorkflow(context.Background(), map[string]any{}, "test", "test")
	if err == nil {
		t.Fatal("expected error for missing execution_id")
	}
}

func TestClientGetStatus(t *testing.T) {
	server := connectExecutionServer(func(_ context.Context, req *connect.Request[basapi.GetExecutionRequest]) (*connect.Response[basapi.GetExecutionResponse], error) {
		if req.Msg.GetExecutionId() != "exec-123" {
			t.Errorf("expected execution id exec-123, got %q", req.Msg.GetExecutionId())
		}
		currentStep := "Navigate to homepage"
		return connect.NewResponse(&basapi.GetExecutionResponse{
			Execution: &basexecution.Execution{
				Status:      basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING,
				Progress:    50,
				CurrentStep: &currentStep,
			},
		}), nil
	}, nil)
	defer server.Close()

	client := NewClient(server.URL)
	status, err := client.GetStatus(context.Background(), "exec-123")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if status == nil {
		t.Fatalf("expected status, got nil")
	}
	if status.GetStatus() != basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING {
		t.Errorf("expected running, got %s", types.ExecutionStatusToString(status.GetStatus()))
	}
	if status.GetProgress() != 50 {
		t.Errorf("expected 50 progress, got %d", status.GetProgress())
	}
	if status.GetCurrentStep() != "Navigate to homepage" {
		t.Errorf("expected current_step 'Navigate to homepage', got %s", status.GetCurrentStep())
	}
}

func TestClientGetStatusError(t *testing.T) {
	server := connectExecutionServer(func(context.Context, *connect.Request[basapi.GetExecutionRequest]) (*connect.Response[basapi.GetExecutionResponse], error) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("execution not found"))
	}, nil)
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.GetStatus(context.Background(), "unknown")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestClientWaitForCompletionSuccess(t *testing.T) {
	callCount := 0
	server := connectExecutionServer(func(context.Context, *connect.Request[basapi.GetExecutionRequest]) (*connect.Response[basapi.GetExecutionResponse], error) {
		callCount++
		status := basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING
		if callCount >= 3 {
			status = basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED
		}
		return connect.NewResponse(&basapi.GetExecutionResponse{
			Execution: &basexecution.Execution{Status: status},
		}), nil
	}, nil)
	defer server.Close()

	client := NewClient(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := client.WaitForCompletion(ctx, "exec-123")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if callCount < 3 {
		t.Errorf("expected at least 3 calls, got %d", callCount)
	}
}

func TestClientWaitForCompletionFailed(t *testing.T) {
	server := connectExecutionServer(func(context.Context, *connect.Request[basapi.GetExecutionRequest]) (*connect.Response[basapi.GetExecutionResponse], error) {
		failure := "element not found"
		return connect.NewResponse(&basapi.GetExecutionResponse{
			Execution: &basexecution.Execution{
				Status: basbase.ExecutionStatus_EXECUTION_STATUS_FAILED,
				Error:  &failure,
			},
		}), nil
	}, nil)
	defer server.Close()

	client := NewClient(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.WaitForCompletion(ctx, "exec-123")
	if err == nil {
		t.Fatal("expected error for failed workflow")
	}
	if !strings.Contains(err.Error(), "element not found") {
		t.Errorf("expected failure reason in error, got: %v", err)
	}
}

func TestClientWaitForCompletionCanceled(t *testing.T) {
	server := connectExecutionServer(func(context.Context, *connect.Request[basapi.GetExecutionRequest]) (*connect.Response[basapi.GetExecutionResponse], error) {
		return connect.NewResponse(&basapi.GetExecutionResponse{
			Execution: &basexecution.Execution{Status: basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING},
		}), nil
	}, nil)
	defer server.Close()

	client := NewClient(server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := client.WaitForCompletion(ctx, "exec-123")
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

func TestClientGetTimeline(t *testing.T) {
	expectedTimeline := &bastimeline.ExecutionTimeline{
		Entries: []*bastimeline.TimelineEntry{
			{Action: &basactions.ActionDefinition{Type: basactions.ActionType_ACTION_TYPE_NAVIGATE}},
		},
	}
	expectedData, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(expectedTimeline)
	if err != nil {
		t.Fatalf("failed to marshal expected timeline: %v", err)
	}
	server := connectExecutionServer(nil, func(_ context.Context, req *connect.Request[basapi.GetExecutionTimelineRequest]) (*connect.Response[bastimeline.ExecutionTimeline], error) {
		if req.Msg.GetExecutionId() != "exec-123" {
			t.Errorf("expected execution id exec-123, got %q", req.Msg.GetExecutionId())
		}
		return connect.NewResponse(expectedTimeline), nil
	})
	defer server.Close()

	client := NewClient(server.URL)
	timeline, raw, err := client.GetTimeline(context.Background(), "exec-123")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if timeline == nil {
		t.Fatalf("expected parsed timeline, got nil")
	}
	if len(timeline.GetEntries()) != 1 || timeline.GetEntries()[0].GetAction().GetType() != basactions.ActionType_ACTION_TYPE_NAVIGATE {
		t.Errorf("unexpected timeline contents: %+v", timeline.GetEntries())
	}
	if string(raw) != string(expectedData) {
		t.Errorf("expected %s, got %s", string(expectedData), string(raw))
	}
}

func TestClientGetTimelineError(t *testing.T) {
	server := connectExecutionServer(nil, func(context.Context, *connect.Request[basapi.GetExecutionTimelineRequest]) (*connect.Response[bastimeline.ExecutionTimeline], error) {
		return nil, connect.NewError(connect.CodeInternal, errors.New("timeline unavailable"))
	})
	defer server.Close()

	client := NewClient(server.URL)
	_, _, err := client.GetTimeline(context.Background(), "exec-123")
	if err == nil {
		t.Fatal("expected error for server error")
	}
}

func TestSummarizeTimeline(t *testing.T) {
	marshalTimeline := func(entries ...*bastimeline.TimelineEntry) []byte {
		tl := &bastimeline.ExecutionTimeline{Entries: entries}
		data, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(tl)
		if err != nil {
			t.Fatalf("failed to marshal timeline: %v", err)
		}
		return data
	}

	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "empty data",
			input:    nil,
			expected: "",
		},
		{
			name:     "no frames",
			input:    marshalTimeline(),
			expected: "",
		},
		{
			name: "steps only",
			input: marshalTimeline(
				timelineEntry(basactions.ActionType_ACTION_TYPE_NAVIGATE, basbase.StepStatus_STEP_STATUS_UNSPECIFIED, false),
				timelineEntry(basactions.ActionType_ACTION_TYPE_CLICK, basbase.StepStatus_STEP_STATUS_UNSPECIFIED, false),
			),
			expected: " (2 steps)",
		},
		{
			name: "with assertions",
			input: marshalTimeline(
				timelineEntry(basactions.ActionType_ACTION_TYPE_NAVIGATE, basbase.StepStatus_STEP_STATUS_COMPLETED, true),
				timelineEntry(basactions.ActionType_ACTION_TYPE_ASSERT, basbase.StepStatus_STEP_STATUS_COMPLETED, true),
				timelineEntry(basactions.ActionType_ACTION_TYPE_ASSERT, basbase.StepStatus_STEP_STATUS_COMPLETED, true),
				timelineEntry(basactions.ActionType_ACTION_TYPE_ASSERT, basbase.StepStatus_STEP_STATUS_FAILED, false),
			),
			expected: " (4 steps, 2/3 assertions passed)",
		},
		{
			name:     "invalid json",
			input:    []byte(`{invalid}`),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SummarizeTimeline(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestClientWithHTTPClient(t *testing.T) {
	customClient := &http.Client{Timeout: 30 * time.Second}
	client := NewClient("http://localhost").WithHTTPClient(customClient)
	if client.httpClient != customClient {
		t.Error("expected custom HTTP client to be set")
	}
}

func TestClientWaitForHealth(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := client.WaitForHealth(ctx)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if callCount < 3 {
		t.Errorf("expected at least 3 calls, got %d", callCount)
	}
}

func TestClientWaitForHealthCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := client.WaitForHealth(ctx)
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
}

func TestClientHealthConnectionError(t *testing.T) {
	// Use a server that's closed to simulate connection error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	client := NewClient(server.URL)
	err := client.Health(context.Background())
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

func TestClientWaitForCompletionStatusVariations(t *testing.T) {
	tests := []struct {
		name        string
		status      basbase.ExecutionStatus
		shouldError bool
		errorSubstr string
	}{
		{"completed status", basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED, false, ""},
		{"failed status", basbase.ExecutionStatus_EXECUTION_STATUS_FAILED, true, "failed"},
		{"cancelled status", basbase.ExecutionStatus_EXECUTION_STATUS_CANCELLED, true, "cancelled"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := connectExecutionServer(func(context.Context, *connect.Request[basapi.GetExecutionRequest]) (*connect.Response[basapi.GetExecutionResponse], error) {
				exec := &basexecution.Execution{Status: tc.status}
				if tc.status == basbase.ExecutionStatus_EXECUTION_STATUS_FAILED {
					msg := "workflow error message"
					exec.Error = &msg
				}
				return connect.NewResponse(&basapi.GetExecutionResponse{Execution: exec}), nil
			}, nil)
			defer server.Close()

			client := NewClient(server.URL)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := client.WaitForCompletion(ctx, "exec-123")
			if tc.shouldError {
				if err == nil {
					t.Fatal("expected error")
				}
				if tc.errorSubstr != "" && !strings.Contains(err.Error(), tc.errorSubstr) {
					t.Errorf("expected error to contain %q, got: %v", tc.errorSubstr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected success, got error: %v", err)
				}
			}
		})
	}
}

func TestClientWaitForCompletionGetStatusError(t *testing.T) {
	server := connectExecutionServer(func(context.Context, *connect.Request[basapi.GetExecutionRequest]) (*connect.Response[basapi.GetExecutionResponse], error) {
		return nil, connect.NewError(connect.CodeInternal, errors.New("server error"))
	}, nil)
	defer server.Close()

	client := NewClient(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.WaitForCompletion(ctx, "exec-123")
	if err == nil {
		t.Fatal("expected error when GetStatus fails")
	}
}

func TestClientExecuteWorkflowInvalidConnectResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(apiconnect.WorkflowsServiceExecuteAdhocWorkflowProcedure, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/proto")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not proto"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.ExecuteWorkflow(context.Background(), map[string]any{}, "test", "test")
	if err == nil {
		t.Fatal("expected error for invalid Connect response")
	}
}

func TestClientGetStatusInvalidConnectResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(apiconnect.ExecutionsServiceGetExecutionProcedure, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/proto")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not proto"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.GetStatus(context.Background(), "exec-123")
	if err == nil {
		t.Fatal("expected error for invalid Connect response")
	}
}

func TestClientGetTimelineConnectionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	client := NewClient(server.URL)
	_, _, err := client.GetTimeline(context.Background(), "exec-123")
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:8080/api/v1")
	if client.baseURL != "http://localhost:8080/api/v1" {
		t.Errorf("expected baseURL to be set, got %s", client.baseURL)
	}
	if client.httpClient == nil {
		t.Error("expected httpClient to be initialized")
	}
	if client.httpClient.Timeout != DefaultTimeout {
		t.Errorf("expected timeout %v, got %v", DefaultTimeout, client.httpClient.Timeout)
	}
}

func TestClientWaitForHealthImmediateSuccess(t *testing.T) {
	// Test that when BAS is already healthy, WaitForHealth returns immediately
	// without waiting for the first ticker tick
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	start := time.Now()
	err := client.WaitForHealth(context.Background())
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected exactly 1 call (immediate check), got %d", callCount)
	}
	// Should complete in well under 1 second (the old bug would have waited 2 seconds)
	if elapsed > 500*time.Millisecond {
		t.Errorf("expected immediate return, but took %v", elapsed)
	}
}

func TestClientWaitForCompletionImmediateSuccess(t *testing.T) {
	// Test that when workflow is already completed, WaitForCompletion returns immediately
	callCount := 0
	server := connectExecutionServer(func(context.Context, *connect.Request[basapi.GetExecutionRequest]) (*connect.Response[basapi.GetExecutionResponse], error) {
		callCount++
		return connect.NewResponse(&basapi.GetExecutionResponse{
			Execution: &basexecution.Execution{Status: basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED},
		}), nil
	}, nil)
	defer server.Close()

	client := NewClient(server.URL)
	start := time.Now()
	err := client.WaitForCompletion(context.Background(), "exec-123")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected exactly 1 call (immediate check), got %d", callCount)
	}
	// Should complete in well under 1 second (the old bug would have waited 1 second)
	if elapsed > 500*time.Millisecond {
		t.Errorf("expected immediate return, but took %v", elapsed)
	}
}
