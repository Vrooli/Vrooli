package store

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	if captured == nil || captured.GetEventType() != heartbeatAttemptEventType || captured.GetSource().GetScenario() != "prompt-manager" {
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
