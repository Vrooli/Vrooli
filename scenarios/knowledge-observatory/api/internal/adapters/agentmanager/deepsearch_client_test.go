package agentmanager

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"knowledge-observatory/internal/services/deepsearch"
)

func TestDeepSearchClientCreateRun(t *testing.T) {
	t.Parallel()

	var ensurePayload map[string]interface{}
	var taskPayload map[string]interface{}
	var runPayload map[string]interface{}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/profiles/ensure", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&ensurePayload); err != nil {
			t.Fatalf("decode ensure payload: %v", err)
		}
		resp := &apipb.EnsureProfileResponse{
			Profile: &domainpb.AgentProfile{Id: "profile-1"},
		}
		writeProtoJSON(t, w, resp)
	})
	mux.HandleFunc("/api/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&taskPayload); err != nil {
			t.Fatalf("decode task payload: %v", err)
		}
		resp := &apipb.CreateTaskResponse{
			Task: &domainpb.Task{Id: "task-1"},
		}
		writeProtoJSON(t, w, resp)
	})
	mux.HandleFunc("/api/v1/runs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&runPayload); err != nil {
			t.Fatalf("decode run payload: %v", err)
		}
		resp := &apipb.CreateRunResponse{
			Run: &domainpb.Run{Id: "run-1"},
		}
		writeProtoJSON(t, w, resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	cfg := DefaultDeepSearchProfileConfig()
	client := NewDeepSearchClientWithBaseURL(5*time.Second, cfg, server.URL)
	runID, err := client.CreateRun(context.Background(), deepsearch.AgentRunRequest{
		Title:       "Deep Search",
		Description: "Integration test",
		Prompt:      "Find docs",
		ScopePath:   "/repo/scenarios",
		ProjectRoot: "/repo",
		Tag:         "doc-deep-search-1",
		Timeout:     45 * time.Second,
	})
	if err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}
	if runID != "run-1" {
		t.Fatalf("expected run id run-1, got %s", runID)
	}

	if ensurePayload["profileKey"] != cfg.ProfileKey {
		t.Fatalf("expected profileKey %q, got %v", cfg.ProfileKey, ensurePayload["profileKey"])
	}
	defaults, ok := ensurePayload["defaults"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected defaults in ensure payload")
	}
	if tools, ok := defaults["allowedTools"].([]interface{}); !ok || len(tools) == 0 {
		t.Fatalf("expected allowedTools in defaults, got %v", defaults["allowedTools"])
	}

	taskBody, ok := taskPayload["task"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected task payload")
	}
	if taskBody["title"] != "Deep Search" {
		t.Fatalf("expected task title, got %v", taskBody["title"])
	}
	if taskBody["scopePath"] != "/repo/scenarios" {
		t.Fatalf("expected scope path, got %v", taskBody["scopePath"])
	}

	if runPayload["taskId"] != "task-1" {
		t.Fatalf("expected taskId task-1, got %v", runPayload["taskId"])
	}
	if runPayload["agentProfileId"] != "profile-1" {
		t.Fatalf("expected agentProfileId profile-1, got %v", runPayload["agentProfileId"])
	}
	if runPayload["prompt"] != "Find docs" {
		t.Fatalf("expected prompt, got %v", runPayload["prompt"])
	}
}

func TestDeepSearchClientRunMapping(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/runs/run-2", func(w http.ResponseWriter, r *http.Request) {
		resp := &apipb.GetRunResponse{
			Run: &domainpb.Run{
				Id:       "run-2",
				Status:   domainpb.RunStatus_RUN_STATUS_FAILED,
				ErrorMsg: "boom",
			},
		}
		writeProtoJSON(t, w, resp)
	})
	mux.HandleFunc("/api/v1/runs/run-2/events", func(w http.ResponseWriter, r *http.Request) {
		resp := &apipb.GetRunEventsResponse{
			Events: []*domainpb.RunEvent{
				{
					Sequence:  1,
					EventType: domainpb.RunEventType_RUN_EVENT_TYPE_MESSAGE,
					Data: &domainpb.RunEvent_Message{
						Message: &domainpb.MessageEventData{
							Role:    "assistant",
							Content: "result",
						},
					},
				},
			},
		}
		writeProtoJSON(t, w, resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewDeepSearchClientWithBaseURL(5*time.Second, DefaultDeepSearchProfileConfig(), server.URL)
	run, err := client.GetRun(context.Background(), "run-2")
	if err != nil {
		t.Fatalf("GetRun failed: %v", err)
	}
	if run.Status != deepsearch.RunStatusFailed {
		t.Fatalf("expected failed status, got %s", run.Status)
	}
	if run.Error != "boom" {
		t.Fatalf("expected error boom, got %s", run.Error)
	}

	events, err := client.GetRunEvents(context.Background(), "run-2", 0)
	if err != nil {
		t.Fatalf("GetRunEvents failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != deepsearch.EventMessage || events[0].Content != "result" {
		t.Fatalf("unexpected event payload: %+v", events[0])
	}
}

func writeProtoJSON(t *testing.T, w http.ResponseWriter, msg proto.Message) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	body, err := protojson.MarshalOptions{UseProtoNames: false}.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal proto: %v", err)
	}
	_, _ = w.Write(body)
}
