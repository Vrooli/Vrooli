package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
		_ = json.NewEncoder(w).Encode(AgentProfileListResponse{
			Profiles: []AgentProfile{
				{ID: "p1", Key: "default", Name: "Default Agent"},
				{ID: "p2", Key: "fast", Name: "Fast Agent"},
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &AgentManagerClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		resolver:   discovery.NewStaticResolver(server.URL),
	}

	result, err := client.ListProfiles(context.Background())
	if err != nil {
		t.Fatalf("ListProfiles returned error: %v", err)
	}
	if len(result.Profiles) != 2 {
		t.Errorf("expected 2 profiles, got %d", len(result.Profiles))
	}
	if result.Profiles[0].Key != "default" {
		t.Errorf("expected first profile key 'default', got %s", result.Profiles[0].Key)
	}
}

func TestAgentManagerClient_CreateTaskAndRun(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req agentTaskCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if req.ScopePath != "scenarios/my-app/" {
			t.Errorf("unexpected scope path: %s", req.ScopePath)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(agentTaskCreateResponse{ID: "task-001"})
	})
	mux.HandleFunc("/api/v1/runs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req agentRunCreateInternalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if req.TaskID != "task-001" {
			t.Errorf("expected taskId task-001, got %s", req.TaskID)
		}
		if req.RunMode != "sandboxed" {
			t.Errorf("expected runMode sandboxed, got %s", req.RunMode)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(agentRunCreateInternalResponse{ID: "run-001"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &AgentManagerClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		resolver:   discovery.NewStaticResolver(server.URL),
	}

	taskResp, err := client.CreateTask(context.Background(), agentTaskCreateRequest{
		Title:     "GCT review: my-app",
		ScopePath: "scenarios/my-app/",
	})
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}
	if taskResp.ID != "task-001" {
		t.Errorf("expected task ID task-001, got %s", taskResp.ID)
	}

	runResp, err := client.CreateRun(context.Background(), agentRunCreateInternalRequest{
		TaskID:  taskResp.ID,
		RunMode: "sandboxed",
	})
	if err != nil {
		t.Fatalf("CreateRun returned error: %v", err)
	}
	if runResp.ID != "run-001" {
		t.Errorf("expected run ID run-001, got %s", runResp.ID)
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
		_ = json.NewEncoder(w).Encode(AgentRun{
			ID:     "run-001",
			Status: "running",
			Phase:  "executing",
			Actions: &AgentRunActions{
				CanStop: true,
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &AgentManagerClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		resolver:   discovery.NewStaticResolver(server.URL),
	}

	result, err := client.GetRun(context.Background(), "run-001")
	if err != nil {
		t.Fatalf("GetRun returned error: %v", err)
	}
	if result.Status != "running" {
		t.Errorf("expected status running, got %s", result.Status)
	}
	if result.Actions == nil || !result.Actions.CanStop {
		t.Error("expected canStop=true")
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
		_ = json.NewEncoder(w).Encode(AgentRunEventsResponse{
			Events: []AgentRunEvent{
				{ID: "evt-6", RunID: "run-001", Sequence: 6, EventType: "message"},
				{ID: "evt-7", RunID: "run-001", Sequence: 7, EventType: "tool_call"},
			},
			Count: 2,
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &AgentManagerClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		resolver:   discovery.NewStaticResolver(server.URL),
	}

	result, err := client.GetRunEvents(context.Background(), "run-001", 5, 100)
	if err != nil {
		t.Fatalf("GetRunEvents returned error: %v", err)
	}
	if len(result.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(result.Events))
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
		_ = json.NewEncoder(w).Encode(AgentRunDiffResponse{
			RunID: "run-001",
			Files: []AgentRunDiffFile{
				{Path: "main.go", Status: "modified", Additions: 5, Deletions: 2},
			},
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &AgentManagerClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		resolver:   discovery.NewStaticResolver(server.URL),
	}

	result, err := client.GetRunDiff(context.Background(), "run-001")
	if err != nil {
		t.Fatalf("GetRunDiff returned error: %v", err)
	}
	if len(result.Files) != 1 {
		t.Errorf("expected 1 diff file, got %d", len(result.Files))
	}
	if result.Files[0].Path != "main.go" {
		t.Errorf("expected path main.go, got %s", result.Files[0].Path)
	}
}

func TestAgentManagerClient_ServerError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/profiles", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal failure"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &AgentManagerClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		resolver:   discovery.NewStaticResolver(server.URL),
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
		_ = json.NewEncoder(w).Encode(AgentRun{
			ID:     "run-001",
			Status: "running",
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &AgentManagerClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		resolver:   discovery.NewStaticResolver(server.URL),
	}

	result, err := client.ContinueRun(context.Background(), "run-001", AgentContinueRequest{Message: "try again"})
	if err != nil {
		t.Fatalf("ContinueRun returned error: %v", err)
	}
	if result.Status != "running" {
		t.Errorf("expected status running, got %s", result.Status)
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
		_ = json.NewEncoder(w).Encode(AgentRun{
			ID:     "run-001",
			Status: "cancelled",
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	client := &AgentManagerClient{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		resolver:   discovery.NewStaticResolver(server.URL),
	}

	result, err := client.StopRun(context.Background(), "run-001")
	if err != nil {
		t.Fatalf("StopRun returned error: %v", err)
	}
	if result.Status != "cancelled" {
		t.Errorf("expected status cancelled, got %s", result.Status)
	}
}
