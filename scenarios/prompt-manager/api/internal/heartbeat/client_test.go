package heartbeat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
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

func TestGetEffortEnrollmentUsesExactCurrentOwnerBoardRow(t *testing.T) {
	want := &domainpb.EffortEnrollment{EffortRef: "effort:delivery", TargetRevision: "target-9"}
	body, err := protojson.Marshal(&domainpb.EffortBoard{Rows: []*domainpb.EffortBoardRow{{Enrollment: want}}})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != apiconnect.AgentManagerServiceGetEffortBoardProcedure {
			t.Errorf("unexpected owner request: %s %s", r.Method, r.URL.Path)
		}
		var request domainpb.GetEffortBoardRequest
		requestBody, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			t.Errorf("read request: %v", readErr)
		}
		if err := protojson.Unmarshal(requestBody, &request); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if request.GetEffortRef() != "effort:delivery" {
			t.Errorf("requested effort = %q", request.GetEffortRef())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer srv.Close()
	enrollment, err := newTestClient(t, srv).GetEffortEnrollment(context.Background(), "effort:delivery")
	if err != nil || enrollment.GetTargetRevision() != "target-9" {
		t.Fatalf("enrollment=%+v err=%v", enrollment, err)
	}
}

func TestGetEffortEnrollmentRejectsMissingExactRow(t *testing.T) {
	body, err := protojson.Marshal(&domainpb.EffortBoard{Rows: []*domainpb.EffortBoardRow{{Enrollment: &domainpb.EffortEnrollment{EffortRef: "effort:other", TargetRevision: "target-1"}}}})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer srv.Close()
	if _, err := newTestClient(t, srv).GetEffortEnrollment(context.Background(), "effort:delivery"); err == nil {
		t.Fatal("expected absence of exact requested enrollment to remain unknown")
	}
}

// --- CreateTask ---

func TestConversationGetTaskDistinguishesMissingFromOutage(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusNotFound, http.StatusBadGateway} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/api/v1/tasks/conversation-task" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"task":{"id":"conversation-task","title":"Conversation","description":"Context and message","scope_path":"/workspace"}}`))
			}))
			defer server.Close()
			client := NewAgentManagerClient(time.Second)
			client.testBaseURL = server.URL
			client.sleep = func(context.Context, time.Duration) error { return nil }
			task, err := client.GetTask(context.Background(), "conversation-task")
			if status == http.StatusOK && (err != nil || task == nil || task.ID != "conversation-task") {
				t.Fatalf("task read: %+v %v", task, err)
			}
			if status == http.StatusNotFound && (err != nil || task != nil) {
				t.Fatalf("missing task: %+v %v", task, err)
			}
			if status == http.StatusBadGateway && err == nil {
				t.Fatal("outage must not permit creating a new task")
			}
		})
	}
}

func TestGetTaskRejectsMalformedSuccessfulResponse(t *testing.T) {
	for _, body := range []string{`{}`, `{"task":null}`} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/api/v1/tasks/task-id" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(body))
			}))
			defer server.Close()

			client := newTestClient(t, server)
			task, err := client.GetTask(context.Background(), "task-id")
			if err == nil || task != nil {
				t.Fatalf("malformed success returned task=%+v err=%v; expected an error and no absence result", task, err)
			}
		})
	}
}

func TestGetTaskRetainsPhasePromptIDs(t *testing.T) {
	phaseID := "c2149690-93a4-4cf0-83d2-f21446ec3e58"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/tasks/task-id" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"task": map[string]any{
			"id": "task-id", "title": "task", "description": "request",
			"scope_path": "/workspace", "status": "queued", "phase_prompt_ids": []string{phaseID},
		}})
	}))
	defer server.Close()

	task, err := newTestClient(t, server).GetTask(context.Background(), "task-id")
	if err != nil {
		t.Fatal(err)
	}
	if task == nil || len(task.PhasePromptIDs) != 1 || task.PhasePromptIDs[0] != phaseID {
		t.Fatalf("phase prompt IDs were not retained: %+v", task)
	}
}

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
				ID:          "6a89415d-9f95-4ee0-a534-00619ce6fd5a",
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
	if task.ID != "6a89415d-9f95-4ee0-a534-00619ce6fd5a" {
		t.Errorf("expected task ID 6a89415d-9f95-4ee0-a534-00619ce6fd5a, got %s", task.ID)
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

func TestCreateTaskApprovalPendingIsTypedAndDefinitive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"approval_pending","detail":"operator approval requested"}`))
	}))
	defer server.Close()

	_, err := newTestClient(t, server).CreateTask(context.Background(), &Task{Title: "scheduled"})
	var pending *ApprovalPendingError
	if !errors.As(err, &pending) || pending.Stage != "creating_task" {
		t.Fatalf("error=%v; want typed task-stage approval pending", err)
	}
	if IsDispatchUncertain(err) {
		t.Fatalf("typed approval pending was classified as uncertain: %v", err)
	}
}

func TestCreateRunApprovalPendingIsTypedAndDefinitive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"approval_pending","detail":"operator approval requested"}`))
	}))
	defer server.Close()

	_, err := newTestClient(t, server).CreateRun(context.Background(), &CreateRunRequest{TaskID: "task-1", IdempotencyKey: "run-key"})
	var pending *ApprovalPendingError
	if !errors.As(err, &pending) || pending.Stage != "creating_run" {
		t.Fatalf("error=%v; want typed run-stage approval pending", err)
	}
	if IsDispatchUncertain(err) {
		t.Fatalf("typed approval pending was classified as uncertain: %v", err)
	}
}

func TestCreateRunReplayConflictIsUncertainButOrdinaryBadRequestIsRejected(t *testing.T) {
	for _, tc := range []struct {
		name          string
		status        int
		code          string
		wantUncertain bool
	}{
		{name: "accepted-key request conflict", status: http.StatusConflict, code: "STATE_IDEMPOTENCY_REPLAY_CONFLICT", wantUncertain: true},
		{name: "ordinary validation refusal", status: http.StatusBadRequest, code: "VALIDATION_FIELD", wantUncertain: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = fmt.Fprintf(w, `{"code":%q,"message":"request refused"}`, tc.code)
			}))
			defer server.Close()

			_, err := newTestClient(t, server).CreateRun(
				withCreateRunCaller(context.Background(), "Bearer fixture-owner", ""),
				&CreateRunRequest{TaskID: "task-1", IdempotencyKey: "run-key"},
			)
			if IsDispatchUncertain(err) != tc.wantUncertain {
				t.Fatalf("uncertain=%v err=%v; want %v", IsDispatchUncertain(err), err, tc.wantUncertain)
			}
			if IsDispatchRejected(err) == tc.wantUncertain {
				t.Fatalf("rejected=%v err=%v; want rejected=%v", IsDispatchRejected(err), err, !tc.wantUncertain)
			}
		})
	}
}

func TestUnrecognizedTask202RemainsUncertain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"accepted"}`))
	}))
	defer server.Close()

	_, err := newTestClient(t, server).CreateTask(context.Background(), &Task{Title: "scheduled"})
	if !IsDispatchUncertain(err) || IsApprovalPending(err) {
		t.Fatalf("unrecognized 202 outcome must remain uncertain, got %v", err)
	}
}

func TestApprovalPendingCauseCannotPromoteTypedOutcomeToUncertain(t *testing.T) {
	err := NewApprovalPendingError("creating_task", errors.New("approval timed out awaiting operator"))
	if IsDispatchUncertain(err) {
		t.Fatalf("typed approval pending cause text changed dispatch classification: %v", err)
	}
}

func TestCancelTask_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/tasks/6a89415d-9f95-4ee0-a534-00619ce6fd5a/cancel" {
			t.Fatalf("unexpected cancel request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	if err := newTestClient(t, srv).CancelTask(context.Background(), "6a89415d-9f95-4ee0-a534-00619ce6fd5a"); err != nil {
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
	run, err := client.CreateRun(withCreateRunCaller(context.Background(), "Bearer fixture-owner", ""), &CreateRunRequest{
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

func TestCreateRunDelegatedKeepsAuthorityRequestLocal(t *testing.T) {
	enforceAuth01(t)
	const secret = "private-owner-bearer"
	var auths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auths = append(auths, r.Header.Get("Authorization"))
		var req CreateRunRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.TaskID == "delegated" && (req.RequestedScopes == nil || req.RequestedScopes.ExpectedOwnerSubject != "owner-a" || len(req.RequestedScopes.Scopes) != 1) {
			t.Fatal("delegated authority constraints lost")
		}
		body, _ := json.Marshal(req)
		if strings.Contains(string(body), secret) {
			t.Fatal("credential leaked into request body")
		}
		_ = json.NewEncoder(w).Encode(CreateRunResponse{Run: &Run{ID: "run-1"}})
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	request := &CreateRunRequest{TaskID: "delegated", IdempotencyKey: "wake-1", RequestedScopes: &RunIdentityScopeRequest{Scopes: []string{"agent-manager:supervise"}, ExpectedOwnerSubject: "owner-a"}}
	if _, err := c.CreateRunDelegated(context.Background(), request, secret); err != nil {
		t.Fatal(err)
	}
	if _, err := c.CreateRun(context.Background(), &CreateRunRequest{TaskID: "ordinary"}); err == nil {
		t.Fatal("default caller inherited delegated authority")
	}
	if len(auths) != 1 || auths[0] != "Bearer "+secret {
		t.Fatal("delegation changed default caller authority")
	}
}

func TestCreateRunDelegatedRefusalDoesNotLeakOrRedirect(t *testing.T) {
	const secret = "private-owner-bearer"
	for _, status := range []int{http.StatusForbidden, http.StatusInternalServerError, http.StatusTemporaryRedirect} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Location", "/other-operation")
				w.WriteHeader(status)
				_, _ = w.Write([]byte(secret))
			}))
			defer srv.Close()
			c := newTestClient(t, srv)
			req := &CreateRunRequest{TaskID: "task", IdempotencyKey: "wake", RequestedScopes: &RunIdentityScopeRequest{Scopes: []string{}, ExpectedOwnerSubject: "owner-a"}}
			if run, err := c.CreateRunDelegated(context.Background(), req, secret); run != nil || err == nil || strings.Contains(err.Error(), secret) {
				t.Fatal("refusal leaked credential or accepted run")
			}
			if calls != 1 {
				t.Fatal("delegated request was retried or redirected")
			}
		})
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

// Error envelope parsing is a pure mapping claim; transport refusal typing is
// protected separately by the task dispatch tests.
func TestParseErrorOwnerEnvelopes(t *testing.T) {
	cases := []struct{ body, want string }{
		{`{"error":"validation failed: title required"}`, "agent-manager error: status 400: validation failed: title required"},
		{`{"message":"invalid JSON request body"}`, "agent-manager error: status 400: invalid JSON request body"},
		{`{"code":"VALIDATION","message":"field 'timeout' has invalid Duration format","details":{}}`, "agent-manager error: status 400: field 'timeout' has invalid Duration format"},
		{`{"status":"INTERNAL","detail":"something broke"}`, `agent-manager error: status 400: body: {"status":"INTERNAL","detail":"something broke"}`},
	}
	for _, tc := range cases {
		t.Run(tc.body, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			recorder.WriteHeader(http.StatusBadRequest)
			recorder.WriteString(tc.body)
			response := recorder.Result()
			defer response.Body.Close()
			if err := (&AgentManagerClient{}).parseError(response); err == nil || err.Error() != tc.want {
				t.Fatalf("err=%v want=%s", err, tc.want)
			}
		})
	}
}

func TestParseErrorPreservesAuthorizationStatus(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			recorder := httptest.NewRecorder()
			recorder.WriteHeader(status)
			recorder.WriteString(`{"error":"authorization refused"}`)
			response := recorder.Result()
			defer response.Body.Close()
			err := (&AgentManagerClient{}).parseError(response)
			got, ok := AgentManagerAuthorizationStatus(err)
			if !ok || got != status {
				t.Fatalf("authorization status = %d, %v; want %d", got, ok, status)
			}
			if err.Error() != "agent-manager error: status "+fmt.Sprint(status)+": authorization refused" {
				t.Fatalf("user-facing error = %q", err)
			}
		})
	}
}

func TestParseErrorPreservesOnlyAllowlistedAdmissionReason(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		reason     string
		wantReason string
		wantError  string
	}{
		{name: "known reason", status: http.StatusForbidden, reason: "grant_limit_reached", wantReason: "grant_limit_reached", wantError: "agent-manager error: status 403: service authority does not include this operation"},
		{name: "rate-limited reason", status: http.StatusTooManyRequests, reason: "approval_raise_limited", wantReason: "approval_raise_limited", wantError: "agent-manager error: status 429: service authority does not include this operation"},
		{name: "unknown reason is dropped", status: http.StatusForbidden, reason: "internal-secret-detail", wantReason: "", wantError: "agent-manager error: status 403: service authority does not include this operation"},
		{name: "reason on unrelated status is dropped", status: http.StatusBadRequest, reason: "grant_limit_reached", wantReason: "", wantError: "agent-manager error: status 400: service authority does not include this operation"},
		{name: "legacy response has no reason", status: http.StatusForbidden, wantReason: "", wantError: "agent-manager error: status 403: service authority does not include this operation"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"error":"service authority does not include this operation"}`
			if tc.reason != "" {
				body = `{"error":"service authority does not include this operation","admission_reason":"` + tc.reason + `"}`
			}
			recorder := httptest.NewRecorder()
			recorder.WriteHeader(tc.status)
			recorder.WriteString(body)
			response := recorder.Result()
			defer response.Body.Close()

			err := (&AgentManagerClient{}).parseError(response)
			var ownerErr *AgentManagerHTTPError
			if !errors.As(err, &ownerErr) {
				t.Fatalf("error type = %T; want *AgentManagerHTTPError", err)
			}
			if ownerErr.AdmissionReason != tc.wantReason {
				t.Fatalf("AdmissionReason=%q, want %q", ownerErr.AdmissionReason, tc.wantReason)
			}
			if err.Error() != tc.wantError {
				t.Fatalf("error prose changed: %q, want %q", err, tc.wantError)
			}
			if strings.Contains(err.Error(), "internal-secret-detail") {
				t.Fatal("unrecognized admission metadata leaked into error prose")
			}
		})
	}
}

func TestCreateTaskSingleAttemptPreservesUncertainOutcome(t *testing.T) {
	cases := []struct {
		name         string
		status       int
		body         string
		loseResponse bool
	}{
		{"accepted response lost", 0, "", true},
		{"accepted but server error", http.StatusInternalServerError, `{"error":"owner response lost"}`, false},
		{"malformed success", http.StatusCreated, `{`, false},
		{"missing task", http.StatusCreated, `{"task":null}`, false},
		{"empty identity", http.StatusOK, `{"task":{"id":""}}`, false},
		{"blank identity", http.StatusCreated, `{"task":{"id":"  "}}`, false},
		{"malformed identity", http.StatusCreated, `{"task":{"id":"not-uuid"}}`, false},
		{"nil UUID", http.StatusCreated, `{"task":{"id":"00000000-0000-0000-0000-000000000000"}}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var accepted, calls int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&calls, 1)
				if r.Method != http.MethodPost || r.URL.Path != "/api/v1/tasks" {
					t.Errorf("request=%s %s", r.Method, r.URL.Path)
				}
				var req CreateTaskRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Task == nil || req.Task.Title != "original task" {
					t.Errorf("request=%+v err=%v", req, err)
				}
				// Acceptance precedes response loss; every repeat would create another task.
				atomic.AddInt32(&accepted, 1)
				if tc.loseResponse {
					conn, _, err := w.(http.Hijacker).Hijack()
					if err != nil {
						t.Errorf("hijack: %v", err)
						return
					}
					_ = conn.Close()
					return
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			client := newTestClient(t, server)
			task, err := client.CreateTask(context.Background(), &Task{Title: "original task"})
			var uncertain *DispatchUncertainError
			if task != nil || !errors.As(err, &uncertain) || uncertain.Cause == nil {
				t.Fatalf("task=%+v err=%v, want typed uncertainty with original cause", task, err)
			}
			if n := atomic.LoadInt32(&calls); n != 1 {
				t.Fatalf("task POSTs=%d, want one", n)
			}
			if n := atomic.LoadInt32(&accepted); n != 1 {
				t.Fatalf("accepted tasks=%d, want one", n)
			}
		})
	}
}

func TestCreateTaskRefusesBodyResendingRedirectWithoutChangingOtherCalls(t *testing.T) {
	for _, status := range []int{http.StatusTemporaryRedirect, http.StatusPermanentRedirect} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var original, redirected, health int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/v1/tasks":
					atomic.AddInt32(&original, 1)
					w.Header().Set("Location", "/redirected-task")
					w.WriteHeader(status)
				case "/redirected-task":
					atomic.AddInt32(&redirected, 1)
					_, _ = w.Write([]byte(`{"task":{"id":"second-task"}}`))
				case "/health":
					http.Redirect(w, r, "/health-target", http.StatusFound)
				case "/health-target":
					atomic.AddInt32(&health, 1)
					w.WriteHeader(http.StatusOK)
				default:
					t.Errorf("unexpected request=%s %s", r.Method, r.URL.Path)
				}
			}))
			defer server.Close()
			client := newTestClient(t, server)
			task, err := client.CreateTask(context.Background(), &Task{Title: "original task"})
			if task != nil || !IsDispatchUncertain(err) {
				t.Fatalf("redirect task=%+v err=%v, want uncertainty without replay", task, err)
			}
			if atomic.LoadInt32(&original) != 1 || atomic.LoadInt32(&redirected) != 0 {
				t.Fatalf("original=%d redirected=%d", original, redirected)
			}
			ok, err := client.Health(context.Background())
			if err != nil || !ok || atomic.LoadInt32(&health) != 1 || client.httpClient.CheckRedirect != nil {
				t.Fatalf("unrelated client redirect changed: healthy=%v err=%v calls=%d", ok, err, health)
			}
		})
	}
}

func TestCreateTaskKeepsCanceledContextAndDoesNotSend(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { atomic.AddInt32(&calls, 1) }))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := newTestClient(t, server).CreateTask(ctx, &Task{Title: "original task"})
	if !IsDispatchRejected(err) || IsDispatchUncertain(err) || !errors.Is(err, context.Canceled) || atomic.LoadInt32(&calls) != 0 {
		t.Fatalf("err=%v calls=%d, want preserved cancellation before sending", err, calls)
	}
}

func TestCreateRunCanceledContextIsRejectedBeforeSend(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { atomic.AddInt32(&calls, 1) }))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := newTestClient(t, server).CreateRun(ctx, &CreateRunRequest{TaskID: "task-1", IdempotencyKey: "run-key"})
	if !IsDispatchRejected(err) || IsDispatchUncertain(err) || !errors.Is(err, context.Canceled) || atomic.LoadInt32(&calls) != 0 {
		t.Fatalf("err=%v calls=%d, want proven pre-send cancellation", err, calls)
	}
}

func TestCreateRunCancellationAfterRequestBeginsRemainsUncertain(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
	}))
	defer server.Close()
	defer close(release)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client := newTestClient(t, server)
	result := make(chan error, 1)
	go func() {
		_, err := client.CreateRun(ctx, &CreateRunRequest{TaskID: "task-1", IdempotencyKey: "run-key"})
		result <- err
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("CreateRun never reached the owner fixture")
	}
	cancel()
	select {
	case err := <-result:
		if !IsDispatchUncertain(err) || IsDispatchRejected(err) || !errors.Is(err, context.Canceled) {
			t.Fatalf("err=%v; after request reached owner, cancellation must remain uncertain", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("CreateRun did not return after cancellation")
	}
}

func TestCreateTaskDefinitiveRefusalNeverBecomesTransportUncertainty(t *testing.T) {
	for _, status := range []int{400, 401, 403, 409, 422, 429} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var calls int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.WriteHeader(status)
				fmt.Fprint(w, `{"error":"timeout: owner unreachable"}`)
			}))
			defer server.Close()
			_, err := newTestClient(t, server).CreateTask(context.Background(), &Task{Title: "original task"})
			if !IsDispatchRejected(err) || IsDispatchUncertain(err) || calls != 1 {
				t.Fatalf("err=%v calls=%d", err, calls)
			}
		})
	}
}
func TestCreateTaskUnsentFailuresAreRejected(t *testing.T) {
	client := NewAgentManagerClient(time.Second)
	client.testBaseURL = "://bad"
	for _, task := range []*Task{{Title: "original"}, {ContextAttachments: json.RawMessage(`{`)}} {
		_, err := client.CreateTask(context.Background(), task)
		if !IsDispatchRejected(err) || IsDispatchUncertain(err) {
			t.Fatalf("err=%v", err)
		}
	}
}

func TestResolveProfileExactReadContract(t *testing.T) {
	key := "prompt-manager/heartbeat-judgment"
	const identity = "6a89415d-9f95-4ee0-a534-00619ce6fd5a"
	cases := []struct {
		name, body string
		wantErr    bool
	}{
		{"owner snake case and preserved budget", `{"profiles":[{"id":"` + identity + `","profile_key":"` + key + `","max_weighted_tokens":"10000","timeout":"600s","max_turns":50,"owner_scenario":"prompt-manager","source_hash":"original","sandbox_config":{"mode":"SANDBOX_MODE_PROTECTED"}}],"total":1}`, false},
		{"owner camel case", `{"profiles":[{"id":"` + identity + `","profileKey":"` + key + `"}],"total":1}`, false},
		{"positive partial catalog", `{"profiles":[{"id":"` + identity + `","profile_key":"` + key + `"}],"total":12}`, false},
		{"missing exact namespace", `{"profiles":[{"id":"` + identity + `","profile_key":"other/heartbeat-judgment"}]}`, true},
		{"missing", `{"profiles":[]}`, true},
		{"duplicates", `{"profiles":[{"id":"` + identity + `","profile_key":"` + key + `"},{"id":"` + identity + `","profile_key":"` + key + `"}]}`, true},
		{"invalid identity", `{"profiles":[{"id":"not-uuid","profile_key":"` + key + `"}]}`, true},
		{"null identity", `{"profiles":[{"profile_key":"` + key + `"}]}`, true},
		{"unsupported schema", `{"profiles":[{"id":"` + identity + `","profile_key":"` + key + `","unknown_control":true}]}`, true},
		{"truncated", `{"profiles":[`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodGet || r.URL.Path != "/api/v1/profiles" || r.URL.RawQuery != "" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			profile, err := newTestClient(t, server).ResolveProfile(context.Background(), key)
			if (err != nil) != tc.wantErr || calls != 1 {
				t.Fatalf("profile=%v err=%v calls=%d", profile, err, calls)
			}
			if !tc.wantErr && (profile.Id != identity || profile.ProfileKey != key) {
				t.Fatalf("identity lost: %v", profile)
			}
			if tc.name == "owner snake case and preserved budget" && (profile.MaxWeightedTokens != 10000 || profile.MaxTurns != 50 || profile.Timeout.AsDuration() != 600*time.Second || profile.OwnerScenario != "prompt-manager" || profile.SandboxConfig == nil) {
				t.Fatalf("owner controls lost: %v", profile)
			}
		})
	}
}
