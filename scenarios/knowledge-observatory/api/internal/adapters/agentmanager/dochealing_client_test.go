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
	"knowledge-observatory/internal/services/dochealing"
)

func TestDocHealingClientCreateRun(t *testing.T) {
	t.Parallel()

	cfg := DefaultDocHealingProfileConfig()
	var reconcilePayload map[string]interface{}
	var taskPayload map[string]interface{}
	var runPayload map[string]interface{}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/profiles/reconcile-scenario", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&reconcilePayload); err != nil {
			t.Fatalf("decode reconcile payload: %v", err)
		}
		resp := &apipb.ReconcileScenarioProfilesResponse{
			Scenario: "knowledge-observatory",
			Results: []*apipb.ProfileReconcileResult{{
				ProfileKey: cfg.ProfileKey,
				ProfileId:  "profile-1",
			}},
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

	client := NewDocHealingClientWithBaseURL(5*time.Second, cfg, server.URL)
	runID, err := client.CreateRun(context.Background(), dochealing.AgentRunRequest{
		Title:       "Doc Heal",
		Description: "Fix docs",
		Prompt:      "Fix docs",
		ScopePath:   "/repo/scenarios/alpha",
		ProjectRoot: "/repo",
		Tag:         "doc-heal-1",
		Timeout:     45 * time.Second,
	})
	if err != nil {
		t.Fatalf("CreateRun failed: %v", err)
	}
	if runID != "run-1" {
		t.Fatalf("expected run id run-1, got %s", runID)
	}

	if reconcilePayload["scenario"] != "knowledge-observatory" {
		t.Fatalf("expected knowledge-observatory reconciliation, got %v", reconcilePayload["scenario"])
	}
	taskBody, ok := taskPayload["task"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected task payload")
	}
	if taskBody["title"] != "Doc Heal" {
		t.Fatalf("expected task title, got %v", taskBody["title"])
	}
	if runPayload["taskId"] != "task-1" {
		t.Fatalf("expected taskId task-1, got %v", runPayload["taskId"])
	}
	profileRef, ok := runPayload["profileRef"].(map[string]interface{})
	if !ok || profileRef["profileKey"] != cfg.ProfileKey {
		t.Fatalf("expected declared profile ref %q, got %v", cfg.ProfileKey, runPayload["profileRef"])
	}
	if _, hasDefaults := profileRef["defaults"]; hasDefaults {
		t.Fatalf("run profile ref must not send inline defaults: %v", profileRef)
	}
}

func TestDocHealingClientDiffAndReview(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/runs/run-2/diff", func(w http.ResponseWriter, r *http.Request) {
		resp := &apipb.GetRunDiffResponse{
			Diff: &domainpb.RunDiff{
				RunId:   "run-2",
				Content: "diff --git a/docs/README.md b/docs/README.md\n+line",
				Files: []*domainpb.FileDiff{
					{Path: "docs/README.md", ChangeType: "modified", Additions: 1, Deletions: 0},
				},
			},
		}
		writeProtoJSON(t, w, resp)
	})
	mux.HandleFunc("/api/v1/runs/run-2/approve", func(w http.ResponseWriter, r *http.Request) {
		resp := &apipb.ApproveRunResponse{
			Result: &domainpb.ApproveResult{Success: true, FilesApplied: 1},
		}
		writeProtoJSON(t, w, resp)
	})
	mux.HandleFunc("/api/v1/runs/run-2/reject", func(w http.ResponseWriter, r *http.Request) {
		resp := &apipb.RejectRunResponse{}
		writeProtoJSON(t, w, resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewDocHealingClientWithBaseURL(5*time.Second, DefaultDocHealingProfileConfig(), server.URL)
	diff, err := client.GetRunDiff(context.Background(), "run-2")
	if err != nil {
		t.Fatalf("GetRunDiff failed: %v", err)
	}
	if diff == nil || len(diff.Files) != 1 {
		t.Fatalf("expected diff files, got %+v", diff)
	}

	if _, err := client.ApproveRun(context.Background(), dochealing.ApprovalRequest{
		RunID: "run-2",
		Actor: "tester",
	}); err != nil {
		t.Fatalf("ApproveRun failed: %v", err)
	}
	if err := client.RejectRun(context.Background(), dochealing.RejectRequest{
		RunID:  "run-2",
		Actor:  "tester",
		Reason: "nope",
	}); err != nil {
		t.Fatalf("RejectRun failed: %v", err)
	}
}
