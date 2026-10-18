package store

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"prompt-manager/internal/sourceledger"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestHeartbeatAttemptsUseVrooliEventsInProductionStore(t *testing.T) {
	var captured *domain.EventEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			defer r.Body.Close()
			body, _ := io.ReadAll(r.Body)
			captured = &domain.EventEnvelope{}
			if err := protojson.Unmarshal(body, captured); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusAccepted)
		case http.MethodGet:
			if captured == nil {
				_, _ = w.Write([]byte("[]"))
				return
			}
			body, err := protojson.Marshal(captured)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			_, _ = w.Write([]byte("["))
			_, _ = w.Write(body)
			_, _ = w.Write([]byte("]"))
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "teams", "infra-health"), 0o755); err != nil {
		t.Fatal(err)
	}
	teamStore := NewFileTeamStore(root, root, nil)
	teamStore.SetSourceLedger(sourceledger.NewAt(server.URL))
	teamStore.SetEventsEndpoint(server.URL)
	entry := &HeartbeatAttempt{ID: "attempt-1", TeamID: "infra-health", AgentID: "auditor", ProfileKey: "auditor", Status: "failed", Phase: "pre_run_failure", StartedAt: "2026-08-08T20:00:00Z", Error: "agent unavailable"}
	if err := teamStore.AppendHeartbeatAttempt(t.Context(), entry.TeamID, entry); err != nil {
		t.Fatalf("append heartbeat attempt: %v", err)
	}
	if captured == nil || captured.GetEventType() != heartbeatAttemptEventType || captured.GetSource().GetScenario() != "prompt-manager" || captured.GetEventId() != heartbeatAttemptFactEventID(entry.ID, entry.Phase) {
		t.Fatalf("unexpected event envelope: %#v", captured)
	}
	got, total, err := teamStore.ListHeartbeatAttempts(t.Context(), entry.TeamID, "", "", "", 10, 0)
	if err != nil {
		t.Fatalf("list heartbeat attempts: %v", err)
	}
	if total != 1 || len(got) != 1 || got[0].ID != entry.ID || got[0].Error != entry.Error {
		t.Fatalf("unexpected attempts: total=%d got=%+v", total, got)
	}
}

func TestHeartbeatAttemptLifecycleFactsHaveDistinctEventIDsAndProjectLatest(t *testing.T) {
	var mu sync.Mutex
	var captured []*domain.EventEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			defer r.Body.Close()
			body, _ := io.ReadAll(r.Body)
			env := &domain.EventEnvelope{}
			if err := protojson.Unmarshal(body, env); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			mu.Lock()
			for _, existing := range captured {
				if existing.GetEventId() == env.GetEventId() {
					mu.Unlock()
					w.WriteHeader(http.StatusConflict)
					return
				}
			}
			captured = append(captured, env)
			mu.Unlock()
			w.WriteHeader(http.StatusAccepted)
		case http.MethodGet:
			mu.Lock()
			events := append([]*domain.EventEnvelope(nil), captured...)
			mu.Unlock()
			w.Write([]byte("["))
			for i, env := range events {
				if i > 0 {
					w.Write([]byte(","))
				}
				body, err := protojson.Marshal(env)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Write(body)
			}
			w.Write([]byte("]"))
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "teams", "infra-health"), 0o755); err != nil {
		t.Fatal(err)
	}
	teamStore := NewFileTeamStore(root, root, nil)
	teamStore.SetSourceLedger(sourceledger.NewAt(server.URL))
	teamStore.SetEventsEndpoint(server.URL)
	start := "2026-10-17T01:00:00Z"
	logicalID := "heartbeat-logical-attempt-1"
	created := &HeartbeatAttempt{ID: logicalID, TeamID: "infra-health", AgentID: "auditor", Status: HeartbeatStatusRunning, Phase: "run_created", StartedAt: start, RunID: "run-1"}
	terminal := *created
	terminal.Status = HeartbeatStatusFailed
	terminal.Phase = "run_terminal"
	terminal.EndedAt = "2026-10-17T01:01:00Z"
	terminal.Error = "owner run failed"
	if err := teamStore.AppendHeartbeatAttempt(t.Context(), created.TeamID, created); err != nil {
		t.Fatalf("append run-created fact: %v", err)
	}
	if err := teamStore.AppendHeartbeatAttempt(t.Context(), terminal.TeamID, &terminal); err != nil {
		t.Fatalf("append terminal fact: %v", err)
	}
	mu.Lock()
	if len(captured) != 2 || captured[0].GetEventId() == captured[1].GetEventId() {
		mu.Unlock()
		t.Fatalf("lifecycle facts need distinct EventIds: %#v", captured)
	}
	mu.Unlock()

	latest, total, err := teamStore.ListHeartbeatAttempts(t.Context(), "infra-health", "auditor", "", "", 10, 0)
	if err != nil || total != 1 || len(latest) != 1 || latest[0].ID != logicalID || latest[0].Status != HeartbeatStatusFailed || latest[0].Phase != "run_terminal" {
		t.Fatalf("latest logical attempt = total %d %+v err=%v", total, latest, err)
	}
	running, total, err := teamStore.ListHeartbeatAttempts(t.Context(), "infra-health", "auditor", HeartbeatStatusRunning, "", 10, 0)
	if err != nil || total != 0 || len(running) != 0 {
		t.Fatalf("running filter returned a stale lifecycle fact: total %d %+v err=%v", total, running, err)
	}
}
