package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"git-control-tower/internal/testutil/fixtures"
	"git-control-tower/internal/testutil/httpx"

	"github.com/vrooli/api-core/discovery"
)

func TestAgentManagerClient_ListProfiles(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/profiles", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		// Proto-JSON format: snake_case fields, enum strings.
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"profiles": []map[string]interface{}{
				{"id": "p1", "profile_key": "default", "name": "Default Agent", "runner_type": "RUNNER_TYPE_CLAUDE_CODE"},
				{"id": "p2", "profile_key": "fast", "name": "Fast Agent", "runner_type": "RUNNER_TYPE_CUSTOM"},
			},
			"total": 2,
		})
	})
	server := httpx.NewHandlerServer(t, mux)

	client := &AgentManagerClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "agent-manager",
		},
		maxRetries:     0,
		retryBaseDelay: time.Millisecond,
	}

	result, err := client.ListProfiles(context.Background())
	if err != nil {
		t.Fatalf("ListProfiles returned error: %v", err)
	}
	if len(result.Profiles) != 2 {
		t.Errorf("expected 2 profiles, got %d", len(result.Profiles))
	}
	if result.Profiles[0].ProfileKey != "default" {
		t.Errorf("expected first profile key 'default', got %s", result.Profiles[0].ProfileKey)
	}
	if result.Total != 2 {
		t.Errorf("expected total 2, got %d", result.Total)
	}
}

func TestAgentManagerClient_CreateTaskAndRun(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/tasks", handleFakeTaskCreate(t))
	mux.HandleFunc("/api/v1/runs", handleFakeRunCreate(t))
	server := httpx.NewHandlerServer(t, mux)

	client := newTestAgentManagerClient(server.URL)

	t.Run("create task", func(t *testing.T) {
		taskResp, err := client.CreateTask(context.Background(), agentTaskCreateRequest{
			Task: agentTaskData{
				Title:     "GCT review: my-app",
				ScopePath: "scenarios/my-app/",
			},
		})
		if err != nil {
			t.Fatalf("CreateTask returned error: %v", err)
		}
		if taskResp.Task.ID != "task-001" {
			t.Errorf("expected task ID task-001, got %s", taskResp.Task.ID)
		}
	})

	t.Run("create run", func(t *testing.T) {
		runResp, err := client.CreateRun(context.Background(), agentRunCreateInternalRequest{
			TaskID:  "task-001",
			RunMode: 1,
			Tag:     "gct-my-app",
		})
		if err != nil {
			t.Fatalf("CreateRun returned error: %v", err)
		}
		if runResp.Run.ID != "run-001" {
			t.Errorf("expected run ID run-001, got %s", runResp.Run.ID)
		}
	})
}

func TestBuildAgentTaskDataUsesContractBackedScenarioScope(t *testing.T) {
	t.Parallel()

	repoDir := t.TempDir()
	fixtures.WriteRepoContract(t, repoDir)

	task := buildAgentTaskData(repoDir, AgentRunRequest{
		ScenarioSlug:  "my-app",
		Prompt:        "Review this scenario",
		AttachmentIDs: []string{"att-1"},
	})

	if task.ScopePath != "scenarios/my-app/" {
		t.Fatalf("ScopePath = %q, want %q", task.ScopePath, "scenarios/my-app/")
	}
	if task.Title != "GCT review: my-app" {
		t.Fatalf("Title = %q", task.Title)
	}
	if len(task.ContextAttachments) != 1 || task.ContextAttachments[0].AttachmentID != "att-1" {
		t.Fatalf("ContextAttachments = %#v", task.ContextAttachments)
	}
}

func handleFakeTaskCreate(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var raw map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Errorf("decode request: %v", err)
		}
		task, ok := raw["task"].(map[string]interface{})
		if !ok {
			t.Error("expected 'task' wrapper in request")
		}
		if sp, _ := task["scope_path"].(string); sp != "scenarios/my-app/" {
			t.Errorf("unexpected scope_path: %s", sp)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"task": map[string]string{"id": "task-001"},
		})
	}
}

func handleFakeRunCreate(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var raw map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if taskID, _ := raw["task_id"].(string); taskID != "task-001" {
			t.Errorf("expected task_id task-001, got %s", taskID)
		}
		if tag, _ := raw["tag"].(string); tag != "gct-my-app" {
			t.Errorf("expected tag gct-my-app, got %s", tag)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"run": map[string]string{"id": "run-001"},
		})
	}
}

func newTestAgentManagerClient(serverURL string) *AgentManagerClient {
	return &AgentManagerClient{
		BaseClient: BaseClient{
			httpClient:  httpx.TestClient(),
			resolver:    discovery.NewStaticResolver(serverURL),
			serviceName: "agent-manager",
		},
		maxRetries:     0,
		retryBaseDelay: time.Millisecond,
	}
}

func TestAgentManagerClient_GetRun(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/runs/run-001", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		// Proto-JSON: wrapped in "run", enum strings, snake_case.
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"run": map[string]interface{}{
				"id":             "run-001",
				"status":         "RUN_STATUS_RUNNING",
				"phase":          "RUN_PHASE_EXECUTING",
				"approval_state": "APPROVAL_STATE_NONE",
				"actions": map[string]interface{}{
					"can_stop":     true,
					"can_continue": false,
				},
				"summary": map[string]interface{}{
					"files_modified": []string{"main.go", "handler.go"},
					"cost_estimate":  1.23,
					"turns_used":     5,
				},
			},
		})
	})
	server := httpx.NewHandlerServer(t, mux)

	client := &AgentManagerClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "agent-manager",
		},
		maxRetries:     0,
		retryBaseDelay: time.Millisecond,
	}

	result, err := client.GetRun(context.Background(), "run-001")
	if err != nil {
		t.Fatalf("GetRun returned error: %v", err)
	}
	if result.Run.Status != "RUN_STATUS_RUNNING" {
		t.Errorf("expected wire status RUN_STATUS_RUNNING, got %s", result.Run.Status)
	}
	if result.Run.Actions == nil || !result.Run.Actions.CanStop {
		t.Error("expected can_stop=true")
	}
	if result.Run.Summary == nil || len(result.Run.Summary.FilesModified) != 2 {
		t.Error("expected 2 files_modified in summary")
	}
	if result.Run.Summary.CostEstimate != 1.23 {
		t.Errorf("expected cost_estimate 1.23, got %f", result.Run.Summary.CostEstimate)
	}
}

func TestAgentManagerClient_GetRunEvents(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/runs/run-001/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		afterSeq := r.URL.Query().Get("afterSequence")
		if afterSeq != "5" {
			t.Errorf("expected afterSequence=5, got %s", afterSeq)
		}
		w.Header().Set("Content-Type", "application/json")
		// Proto-JSON format: event_type enums, oneof data fields.
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"events": []map[string]interface{}{
				{
					"id":         "evt-6",
					"run_id":     "run-001",
					"sequence":   "6",
					"event_type": "RUN_EVENT_TYPE_MESSAGE",
					"message":    map[string]string{"role": "assistant", "content": "Hello world"},
				},
				{
					"id":         "evt-7",
					"run_id":     "run-001",
					"sequence":   "7",
					"event_type": "RUN_EVENT_TYPE_TOOL_CALL",
					"tool_call":  map[string]string{"tool_name": "read_file"},
				},
			},
		})
	})
	server := httpx.NewHandlerServer(t, mux)

	client := &AgentManagerClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "agent-manager",
		},
		maxRetries:     0,
		retryBaseDelay: time.Millisecond,
	}

	result, err := client.GetRunEvents(context.Background(), "run-001", 5, 100)
	if err != nil {
		t.Fatalf("GetRunEvents returned error: %v", err)
	}
	if len(result.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(result.Events))
	}
	if result.Events[0].EventType != "RUN_EVENT_TYPE_MESSAGE" {
		t.Errorf("expected wire event type RUN_EVENT_TYPE_MESSAGE, got %s", result.Events[0].EventType)
	}
}

func TestAgentManagerClient_GetRunDiff(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/runs/run-001/diff", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"diff": map[string]interface{}{
				"run_id":  "run-001",
				"content": "full diff content here",
				"files": []map[string]interface{}{
					{"path": "main.go", "change_type": "modified", "additions": 5, "deletions": 2},
				},
			},
		})
	})
	server := httpx.NewHandlerServer(t, mux)

	client := &AgentManagerClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "agent-manager",
		},
		maxRetries:     0,
		retryBaseDelay: time.Millisecond,
	}

	result, err := client.GetRunDiff(context.Background(), "run-001")
	if err != nil {
		t.Fatalf("GetRunDiff returned error: %v", err)
	}
	if result.Diff == nil {
		t.Fatal("expected non-nil diff")
	}
	if len(result.Diff.Files) != 1 {
		t.Errorf("expected 1 diff file, got %d", len(result.Diff.Files))
	}
	if result.Diff.Files[0].Path != "main.go" {
		t.Errorf("expected path main.go, got %s", result.Diff.Files[0].Path)
	}
	if result.Diff.Content != "full diff content here" {
		t.Errorf("unexpected diff content: %s", result.Diff.Content)
	}
}

func TestAgentManagerClient_ServerError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/profiles", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal failure"})
	})
	server := httpx.NewHandlerServer(t, mux)

	client := &AgentManagerClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "agent-manager",
		},
		maxRetries:     0,
		retryBaseDelay: time.Millisecond,
	}

	_, err := client.ListProfiles(context.Background())
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestAgentManagerClient_ContinueRun(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/runs/run-001/continue", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req AgentContinueRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if req.Message != "try again" {
			t.Errorf("expected message 'try again', got %s", req.Message)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"run": map[string]interface{}{
				"id":     "run-001",
				"status": "RUN_STATUS_RUNNING",
			},
		})
	})
	server := httpx.NewHandlerServer(t, mux)

	client := &AgentManagerClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "agent-manager",
		},
		maxRetries:     0,
		retryBaseDelay: time.Millisecond,
	}

	result, err := client.ContinueRun(context.Background(), "run-001", AgentContinueRequest{Message: "try again"})
	if err != nil {
		t.Fatalf("ContinueRun returned error: %v", err)
	}
	if !result.Success {
		t.Error("expected success=true")
	}
	if result.Run == nil || result.Run.Status != "RUN_STATUS_RUNNING" {
		t.Error("expected run with status RUN_STATUS_RUNNING")
	}
}

func TestAgentManagerClient_StopRun(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/runs/run-001/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "stopped",
		})
	})
	server := httpx.NewHandlerServer(t, mux)

	client := &AgentManagerClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "agent-manager",
		},
		maxRetries:     0,
		retryBaseDelay: time.Millisecond,
	}

	result, err := client.StopRun(context.Background(), "run-001")
	if err != nil {
		t.Fatalf("StopRun returned error: %v", err)
	}
	if result.Status != "stopped" {
		t.Errorf("expected status stopped, got %s", result.Status)
	}
}

// ── Conversion tests ────────────────────────────────────────────────

func TestNormalizeEnum(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input  string
		prefix string
		want   string
	}{
		{"RUN_STATUS_RUNNING", "RUN_STATUS_", "running"},
		{"RUN_STATUS_NEEDS_REVIEW", "RUN_STATUS_", "needs_review"},
		{"RUN_PHASE_EXECUTING", "RUN_PHASE_", "executing"},
		{"APPROVAL_STATE_NONE", "APPROVAL_STATE_", "none"},
		{"", "RUN_STATUS_", ""},
		{"UNKNOWN_VALUE", "RUN_STATUS_", "unknown_value"},
		{"running", "RUN_STATUS_", "running"}, // already lowercase, no prefix
	}

	for _, tt := range tests {
		got := normalizeEnum(tt.input, tt.prefix)
		if got != tt.want {
			t.Errorf("normalizeEnum(%q, %q) = %q, want %q", tt.input, tt.prefix, got, tt.want)
		}
	}
}

func TestNormalizeEventType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"RUN_EVENT_TYPE_MESSAGE", "message"},
		{"RUN_EVENT_TYPE_TOOL_CALL", "tool_call"},
		{"RUN_EVENT_TYPE_TOOL_RESULT", "tool_result"},
		{"RUN_EVENT_TYPE_ERROR", "error"},
		{"RUN_EVENT_TYPE_STATUS", "status_change"}, // special case
		{"RUN_EVENT_TYPE_LOG", "log"},
		{"message", "message"}, // pass-through
		{"", ""},
	}

	for _, tt := range tests {
		got := normalizeEventType(tt.input)
		if got != tt.want {
			t.Errorf("normalizeEventType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// assertStringField checks a named string field on the API run.
func assertStringField(t *testing.T, label, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %s, want %s", label, got, want)
	}
}

func TestWireRunToAPI(t *testing.T) {
	t.Parallel()

	w := wireRun{
		ID:              "run-123",
		TaskID:          "task-456",
		Status:          "RUN_STATUS_NEEDS_REVIEW",
		Phase:           "RUN_PHASE_REVIEWING",
		ProgressPercent: 75,
		ErrorMsg:        "",
		SessionID:       "sess-789",
		ApprovalState:   "APPROVAL_STATE_PENDING",
		Summary: &wireRunSummary{
			FilesModified: []string{"a.go", "b.go"},
			FilesCreated:  []string{"c.go"},
			CostEstimate:  2.50,
			TurnsUsed:     10,
		},
		Actions: &wireRunActions{
			CanStop:    false,
			CanApprove: true,
			CanReject:  true,
			CanReview:  true,
		},
		CreatedAt: "2024-01-01T00:00:00Z",
		StartedAt: "2024-01-01T00:01:00Z",
	}

	got := wireRunToAPI(&w)

	assertStringField(t, "ID", got.ID, "run-123")
	assertStringField(t, "Status", got.Status, "needs_review")
	assertStringField(t, "Phase", got.Phase, "reviewing")
	assertStringField(t, "SessionID", got.SessionID, "sess-789")
	assertStringField(t, "ApprovalState", got.ApprovalState, "pending")

	if got.Summary == nil {
		t.Fatal("Summary is nil")
	}
	if len(got.Summary.FilesModified) != 2 {
		t.Errorf("FilesModified length: got %d, want 2", len(got.Summary.FilesModified))
	}
	if got.Summary.CostEstimate != 2.50 {
		t.Errorf("CostEstimate: got %f, want 2.50", got.Summary.CostEstimate)
	}
	if got.Actions == nil || !got.Actions.CanApprove {
		t.Error("expected CanApprove=true")
	}
	if got.Actions.CanStop {
		t.Error("expected CanStop=false")
	}
}

// assertEventData extracts the Data map from an API event and checks the expected key-value pairs.
func assertEventData(t *testing.T, got AgentRunEvent, wantType string, wantData map[string]interface{}) {
	t.Helper()
	if got.EventType != wantType {
		t.Errorf("EventType: got %s, want %s", got.EventType, wantType)
	}
	data, ok := got.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected Data to be map, got %T", got.Data)
	}
	for key, want := range wantData {
		if data[key] != want {
			t.Errorf("data[%s]: got %v, want %v", key, data[key], want)
		}
	}
}

func TestWireRunEventToAPI(t *testing.T) {
	t.Parallel()

	t.Run("message event", func(t *testing.T) {
		w := wireRunEvent{
			ID: "evt-1", RunID: "run-1", Sequence: 1,
			EventType: "RUN_EVENT_TYPE_MESSAGE", Timestamp: "2024-01-01T00:00:00Z",
			Message: &wireMessageData{Role: "assistant", Content: "hello"},
		}
		assertEventData(t, wireRunEventToAPI(&w), "message", map[string]interface{}{
			"role": "assistant", "content": "hello",
		})
	})

	t.Run("user message event", func(t *testing.T) {
		w := wireRunEvent{
			ID: "evt-u1", RunID: "run-1", Sequence: 10,
			EventType: "RUN_EVENT_TYPE_MESSAGE", Timestamp: "2024-01-01T00:00:05Z",
			Message: &wireMessageData{Role: "user", Content: "fix the bug"},
		}
		assertEventData(t, wireRunEventToAPI(&w), "message", map[string]interface{}{
			"role": "user", "content": "fix the bug",
		})
	})

	t.Run("tool_call event", func(t *testing.T) {
		w := wireRunEvent{
			ID: "evt-2", RunID: "run-1", Sequence: 2,
			EventType: "RUN_EVENT_TYPE_TOOL_CALL",
			ToolCall:  &wireToolCallData{ToolName: "read_file", Input: json.RawMessage(`"main.go"`)},
		}
		assertEventData(t, wireRunEventToAPI(&w), "tool_call", map[string]interface{}{
			"name": "read_file",
		})
	})

	t.Run("tool_call event with object input", func(t *testing.T) {
		w := wireRunEvent{
			ID: "evt-2b", RunID: "run-1", Sequence: 2,
			EventType: "RUN_EVENT_TYPE_TOOL_CALL",
			ToolCall:  &wireToolCallData{ToolName: "Bash", Input: json.RawMessage(`{"command":"ls","description":"list files"}`)},
		}
		got := wireRunEventToAPI(&w)
		assertEventData(t, got, "tool_call", map[string]interface{}{"name": "Bash"})
		data := got.Data.(map[string]interface{})
		input, ok := data["input"].(map[string]interface{})
		if !ok {
			t.Fatalf("input should be map, got %T", data["input"])
		}
		if input["command"] != "ls" {
			t.Errorf("input.command: got %v", input["command"])
		}
	})

	t.Run("status event", func(t *testing.T) {
		w := wireRunEvent{
			ID: "evt-3", RunID: "run-1", Sequence: 3,
			EventType: "RUN_EVENT_TYPE_STATUS",
			Status:    &wireStatusData{OldStatus: "RUN_STATUS_RUNNING", NewStatus: "RUN_STATUS_NEEDS_REVIEW"},
		}
		assertEventData(t, wireRunEventToAPI(&w), "status_change", map[string]interface{}{
			"newStatus": "needs_review", "oldStatus": "running",
		})
	})

	t.Run("error event", func(t *testing.T) {
		w := wireRunEvent{
			ID: "evt-4", RunID: "run-1", Sequence: 4,
			EventType: "RUN_EVENT_TYPE_ERROR",
			Error:     &wireErrorData{Message: "something broke", Code: "INTERNAL"},
		}
		assertEventData(t, wireRunEventToAPI(&w), "error", map[string]interface{}{
			"message": "something broke", "code": "INTERNAL",
		})
	})
}

func TestWireProfileToAPI(t *testing.T) {
	t.Parallel()

	w := wireAgentProfile{
		ID:          "p1",
		ProfileKey:  "default",
		Name:        "Default",
		RunnerType:  "RUNNER_TYPE_CLAUDE_CODE",
		Description: "desc",
		Model:       "claude-3",
	}
	got := wireProfileToAPI(&w)
	if got.Key != "default" {
		t.Errorf("Key: got %s", got.Key)
	}
	if got.RunnerType != "claude-code" {
		t.Errorf("RunnerType: got %s, want claude-code", got.RunnerType)
	}
}

// ── Retry tests ─────────────────────────────────────────────────────

func TestRetryOnTransportError(t *testing.T) {
	t.Parallel()

	var attempts int64
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/profiles", func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt64(&attempts, 1)
		if n == 1 {
			// Simulate a server error on first attempt.
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":"bad gateway"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"profiles": []map[string]interface{}{},
			"total":    0,
		})
	})
	server := httpx.NewHandlerServer(t, mux)

	client := &AgentManagerClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "agent-manager",
		},
		maxRetries:     2,
		retryBaseDelay: time.Millisecond,
	}

	result, err := client.ListProfiles(context.Background())
	if err != nil {
		t.Fatalf("expected success after retry, got error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if atomic.LoadInt64(&attempts) != 2 {
		t.Errorf("expected 2 attempts, got %d", atomic.LoadInt64(&attempts))
	}
}

func TestNoRetryOn4xx(t *testing.T) {
	t.Parallel()

	var attempts int64
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/profiles", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&attempts, 1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad request"}`))
	})
	server := httpx.NewHandlerServer(t, mux)

	client := &AgentManagerClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 5 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "agent-manager",
		},
		maxRetries:     2,
		retryBaseDelay: time.Millisecond,
	}

	_, err := client.ListProfiles(context.Background())
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if atomic.LoadInt64(&attempts) != 1 {
		t.Errorf("expected 1 attempt (no retry on 4xx), got %d", atomic.LoadInt64(&attempts))
	}
}

func TestReResolveOnTransportFailure(t *testing.T) {
	t.Parallel()

	// Pre-cache a bad URL, then verify it gets cleared on failure.
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/profiles", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"profiles": []map[string]interface{}{},
			"total":    0,
		})
	})
	server := httpx.NewHandlerServer(t, mux)

	client := &AgentManagerClient{
		BaseClient: BaseClient{
			httpClient:  &http.Client{Timeout: 2 * time.Second},
			resolver:    discovery.NewStaticResolver(server.URL),
			serviceName: "agent-manager",
		},
		maxRetries:     2,
		retryBaseDelay: time.Millisecond,
	}

	// Pre-cache a bad URL.
	client.mu.Lock()
	client.cachedBaseURL = "http://127.0.0.1:1" // Non-connectable port.
	client.mu.Unlock()

	result, err := client.ListProfiles(context.Background())
	if err != nil {
		t.Fatalf("expected success after re-resolve, got error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Verify cache was updated to the real URL.
	client.mu.Lock()
	cached := client.cachedBaseURL
	client.mu.Unlock()
	if cached != server.URL {
		t.Errorf("expected cached URL to be %s, got %s", server.URL, cached)
	}
}
