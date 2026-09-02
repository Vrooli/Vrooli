package heartbeat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// newTestClient creates an AgentManagerClient pointed at the given test server.
// It bypasses service discovery by using a custom resolveBaseURL.
func newTestClient(t *testing.T, server *httptest.Server) *AgentManagerClient {
	t.Helper()
	c := &AgentManagerClient{
		httpClient: server.Client(),
		sleep:      func(context.Context, time.Duration) error { return nil },
	}
	// Override resolveBaseURL to return the test server URL.
	c.testBaseURL = server.URL
	return c
}

// --- Health ---

func TestHealth_ReturnsTrue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	ok, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected healthy")
	}
}

func TestHealth_ReturnsFalseFor503(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	// 503 is retryable, so Health will exhaust retries and return an error.
	_, err := client.Health(context.Background())
	if err == nil {
		t.Fatal("expected error after retries exhausted for 503")
	}
}

func TestHealth_ReturnsFalseFor403(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	// 403 is not retryable and not 200, so Health returns false without error.
	ok, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected unhealthy for 403")
	}
}

// --- CreateTask ---

func TestCreateTask_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/tasks" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Decode request and verify structure
		var req CreateTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Task == nil {
			t.Fatal("expected task in request")
		}
		if req.Task.Title != "test-title" {
			t.Errorf("expected title test-title, got %s", req.Task.Title)
		}

		resp := CreateTaskResponse{
			Task: &Task{
				ID:          "task-abc",
				Title:       req.Task.Title,
				Description: req.Task.Description,
				ScopePath:   req.Task.ScopePath,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	task, err := client.CreateTask(context.Background(), &Task{
		Title:       "test-title",
		Description: "test-desc",
		ScopePath:   "/tmp",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.ID != "task-abc" {
		t.Errorf("expected task ID task-abc, got %s", task.ID)
	}
}

func TestCreateTask_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	_, err := client.CreateTask(context.Background(), &Task{Title: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestCancelTask_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/tasks/task-abc/cancel" {
			t.Fatalf("unexpected cancel request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := newTestClient(t, srv).CancelTask(context.Background(), "task-abc"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- CreateRun ---

func TestCreateRun_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/runs" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req CreateRunRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.TaskID != "task-1" {
			t.Errorf("expected task_id task-1, got %s", req.TaskID)
		}
		if req.ProfileRef == nil || req.ProfileRef.ProfileKey != "my-profile" {
			t.Error("expected profile_ref with profile_key my-profile")
		}

		resp := CreateRunResponse{
			Run: &Run{
				ID:     "run-xyz",
				TaskID: req.TaskID,
				Status: "RUN_STATUS_RUNNING",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	run, err := client.CreateRun(context.Background(), &CreateRunRequest{
		TaskID:     "task-1",
		ProfileRef: &ProfileRef{ProfileKey: "my-profile"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run.ID != "run-xyz" {
		t.Errorf("expected run ID run-xyz, got %s", run.ID)
	}
}

// --- CreateRun request field validation ---

func TestCreateRunRequest_JSONFieldNames(t *testing.T) {
	tag := "heartbeat-team-1-agent-1-2025-01-01"
	req := CreateRunRequest{
		TaskID: "task-1",
		ProfileRef: &ProfileRef{
			ProfileKey: "my-profile",
		},
		Tag: &tag,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Verify snake_case field names match agent-manager proto schema
	expected := map[string]bool{
		"task_id":     true,
		"profile_ref": true,
		"tag":         true,
	}
	for key := range fields {
		if !expected[key] {
			t.Errorf("unexpected field %q in CreateRunRequest JSON — agent-manager proto may reject this", key)
		}
	}
	for key := range expected {
		if _, ok := fields[key]; !ok {
			t.Errorf("expected field %q missing from CreateRunRequest JSON", key)
		}
	}
}

// --- GetRun ---

func TestGetRun_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/runs/run-1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		resp := GetRunResponse{
			Run: &Run{ID: "run-1", Status: "RUN_STATUS_COMPLETE"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	run, err := client.GetRun(context.Background(), "run-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run.Status != "RUN_STATUS_COMPLETE" {
		t.Errorf("expected status RUN_STATUS_COMPLETE, got %s", run.Status)
	}
}

func TestGetRun_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	run, err := client.GetRun(context.Background(), "missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run != nil {
		t.Fatalf("expected nil run for 404, got %+v", run)
	}
}

// --- WaitForRun ---

func TestWaitForRun_ReturnsOnTerminal(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		status := "RUN_STATUS_RUNNING"
		if n >= 3 {
			status = "RUN_STATUS_COMPLETE"
		}
		resp := GetRunResponse{
			Run: &Run{ID: "run-1", Status: status},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	run, err := client.WaitForRun(context.Background(), "run-1", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if run.Status != "RUN_STATUS_COMPLETE" {
		t.Errorf("expected terminal status, got %s", run.Status)
	}
	if n := atomic.LoadInt32(&callCount); n < 3 {
		t.Errorf("expected at least 3 polls, got %d", n)
	}
}

func TestWaitForRun_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := GetRunResponse{
			Run: &Run{ID: "run-1", Status: "RUN_STATUS_RUNNING"},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.WaitForRun(ctx, "run-1", 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected context error")
	}
}

// --- StopRun ---

func TestStopRun_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/runs/run-1/stop" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	if err := client.StopRun(context.Background(), "run-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Retry Logic ---

func TestDoRequestWithRetry_RetriesOn5xx(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	ok, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected healthy after retries")
	}
	if n := atomic.LoadInt32(&callCount); n != 3 {
		t.Errorf("expected 3 attempts, got %d", n)
	}
}

func TestDoRequestWithRetry_ExhaustsRetries(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	_, err := client.Health(context.Background())
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	// Should attempt 4 times (initial + 3 retries)
	if n := atomic.LoadInt32(&callCount); n != 4 {
		t.Errorf("expected 4 attempts, got %d", n)
	}
}

func TestDoRequestWithRetry_NoRetryOn4xx(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad request"}`))
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	_, err := client.CreateTask(context.Background(), &Task{Title: "x", Description: "y", ScopePath: "/tmp"})
	if err == nil {
		t.Fatal("expected error for 400")
	}
	if n := atomic.LoadInt32(&callCount); n != 1 {
		t.Errorf("expected exactly 1 attempt (no retry on 4xx), got %d", n)
	}
}

// --- Error Parsing ---

func TestParseError_ErrorField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"validation failed: title required"}`))
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	_, err := client.CreateTask(context.Background(), &Task{Description: "x", ScopePath: "/tmp"})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "agent-manager error: validation failed: title required" {
		t.Errorf("unexpected error message: %s", got)
	}
}

func TestParseError_MessageField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"invalid JSON request body"}`))
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	_, err := client.CreateTask(context.Background(), &Task{Description: "x", ScopePath: "/tmp"})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "agent-manager error: invalid JSON request body" {
		t.Errorf("unexpected error message: %s", got)
	}
}

// --- Proto Error Response Parsing ---

func TestParseError_ProtoErrorResponse(t *testing.T) {
	// agent-manager may return proto-style error responses with "code" and "message"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"VALIDATION","message":"field 'timeout' has invalid Duration format","details":{}}`))
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	_, err := client.CreateTask(context.Background(), &Task{Title: "x", Description: "y", ScopePath: "/tmp"})
	if err == nil {
		t.Fatal("expected error")
	}
	// parseError should extract the "message" field
	got := err.Error()
	if got != "agent-manager error: field 'timeout' has invalid Duration format" {
		t.Errorf("unexpected error message: %s", got)
	}
}

func TestParseError_NoRecognizedFields(t *testing.T) {
	// agent-manager returns a response with neither "error" nor "message"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"status":"INTERNAL","detail":"something broke"}`))
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	// 500 is retryable, so it will exhaust retries
	_, err := client.Health(context.Background())
	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}
	// Should include the status code and body in the fallback message
	got := err.Error()
	if got == "" {
		t.Fatal("expected non-empty error message")
	}
}

// --- EnsureProfile ---

func TestEnsureProfile_Created(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/profiles/ensure" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req EnsureProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.ProfileKey != "test-profile" {
			t.Errorf("expected profile_key test-profile, got %s", req.ProfileKey)
		}

		resp := EnsureProfileResponse{
			Profile: &AgentProfile{ProfileKey: "test-profile"},
			Created: true,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	resp, err := client.EnsureProfile(context.Background(), &EnsureProfileRequest{
		ProfileKey: "test-profile",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Created {
		t.Error("expected created=true")
	}
}
