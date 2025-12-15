//go:build legacyproto
// +build legacyproto

package execution

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"test-genie/internal/playbooks/types"

	browser_automation_studio_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workflows/execute-adhoc" {
			t.Errorf("expected /workflows/execute-adhoc, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content type")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"execution_id": expectedID})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	definition := map[string]any{"nodes": []any{}, "edges": []any{}}
	id, err := client.ExecuteWorkflow(context.Background(), definition, "test workflow")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if id != expectedID {
		t.Errorf("expected %s, got %s", expectedID, id)
	}
}

func TestClientExecuteWorkflowError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid workflow"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	definition := map[string]any{"nodes": []any{}}
	_, err := client.ExecuteWorkflow(context.Background(), definition, "test")
	if err == nil {
		t.Fatal("expected error for failed execution")
	}
	if !strings.Contains(err.Error(), "invalid workflow") {
		t.Errorf("expected error to contain response body, got: %v", err)
	}
}

func TestClientExecuteWorkflowMissingID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.ExecuteWorkflow(context.Background(), map[string]any{}, "test")
	if err == nil {
		t.Fatal("expected error for missing execution_id")
	}
}

func TestClientGetStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/executions/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		// Response matches BAS database.Execution model (protojson format):
		// - status uses enum name
		// - progress is int (0-100)
		// - current_step is string (step name/label)
		json.NewEncoder(w).Encode(map[string]any{
			"status":       "EXECUTION_STATUS_RUNNING",
			"progress":     50,
			"current_step": "Navigate to homepage",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	status, err := client.GetStatus(context.Background(), "exec-123")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if status == nil {
		t.Fatalf("expected status, got nil")
	}
	if status.GetStatus() != browser_automation_studio_v1.ExecutionStatus_EXECUTION_STATUS_RUNNING {
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("execution not found"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.GetStatus(context.Background(), "unknown")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestClientWaitForCompletionSuccess(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		status := "EXECUTION_STATUS_RUNNING"
		if callCount >= 3 {
			status = "EXECUTION_STATUS_COMPLETED"
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": status})
	}))
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "EXECUTION_STATUS_FAILED",
			"error":  "element not found",
		})
	}))
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "EXECUTION_STATUS_RUNNING"})
	}))
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
	expectedTimeline := &browser_automation_studio_v1.ExecutionTimeline{
		Frames: []*browser_automation_studio_v1.TimelineFrame{
			{StepType: browser_automation_studio_v1.StepType_STEP_TYPE_NAVIGATE},
		},
	}
	expectedData, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(expectedTimeline)
	if err != nil {
		t.Fatalf("failed to marshal expected timeline: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/timeline") {
			t.Errorf("expected timeline path, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedData))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	timeline, raw, err := client.GetTimeline(context.Background(), "exec-123")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if timeline == nil {
		t.Fatalf("expected parsed timeline, got nil")
	}
	if len(timeline.GetFrames()) != 1 || timeline.GetFrames()[0].GetStepType() != browser_automation_studio_v1.StepType_STEP_TYPE_NAVIGATE {
		t.Errorf("unexpected timeline contents: %+v", timeline.GetFrames())
	}
	if string(raw) != string(expectedData) {
		t.Errorf("expected %s, got %s", string(expectedData), string(raw))
	}
}

func TestClientGetTimelineError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, _, err := client.GetTimeline(context.Background(), "exec-123")
	if err == nil {
		t.Fatal("expected error for server error")
	}
}

func TestSummarizeTimeline(t *testing.T) {
	marshalTimeline := func(frames ...*browser_automation_studio_v1.TimelineFrame) []byte {
		tl := &browser_automation_studio_v1.ExecutionTimeline{Frames: frames}
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
			name:     "steps only",
			input:    marshalTimeline(&browser_automation_studio_v1.TimelineFrame{StepType: browser_automation_studio_v1.StepType_STEP_TYPE_NAVIGATE}, &browser_automation_studio_v1.TimelineFrame{StepType: browser_automation_studio_v1.StepType_STEP_TYPE_CLICK}),
			expected: " (2 steps)",
		},
		{
			name: "with assertions",
			input: marshalTimeline(
				&browser_automation_studio_v1.TimelineFrame{StepType: browser_automation_studio_v1.StepType_STEP_TYPE_NAVIGATE, Status: browser_automation_studio_v1.StepStatus_STEP_STATUS_COMPLETED, Success: true},
				&browser_automation_studio_v1.TimelineFrame{StepType: browser_automation_studio_v1.StepType_STEP_TYPE_ASSERT, Status: browser_automation_studio_v1.StepStatus_STEP_STATUS_COMPLETED, Success: true},
				&browser_automation_studio_v1.TimelineFrame{StepType: browser_automation_studio_v1.StepType_STEP_TYPE_ASSERT, Status: browser_automation_studio_v1.StepStatus_STEP_STATUS_COMPLETED, Success: true},
				&browser_automation_studio_v1.TimelineFrame{StepType: browser_automation_studio_v1.StepType_STEP_TYPE_ASSERT, Status: browser_automation_studio_v1.StepStatus_STEP_STATUS_FAILED, Success: false},
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
		status      string
		shouldError bool
		errorSubstr string
	}{
		{"completed status", "EXECUTION_STATUS_COMPLETED", false, ""},
		{"failed status", "EXECUTION_STATUS_FAILED", true, "failed"},
		{"cancelled status", "EXECUTION_STATUS_CANCELLED", true, "cancelled"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				resp := map[string]string{"status": tc.status}
				if tc.status == "EXECUTION_STATUS_FAILED" {
					resp["error"] = "workflow error message"
				}
				json.NewEncoder(w).Encode(resp)
			}))
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.WaitForCompletion(ctx, "exec-123")
	if err == nil {
		t.Fatal("expected error when GetStatus fails")
	}
}

func TestClientExecuteWorkflowInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.ExecuteWorkflow(context.Background(), map[string]any{}, "test")
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Errorf("expected decode error, got: %v", err)
	}
}

func TestClientGetStatusInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.GetStatus(context.Background(), "exec-123")
	if err == nil {
		t.Fatal("expected error for invalid JSON response")
	}
	if !strings.Contains(err.Error(), "decode") {
		t.Errorf("expected decode error, got: %v", err)
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "EXECUTION_STATUS_COMPLETED"})
	}))
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
