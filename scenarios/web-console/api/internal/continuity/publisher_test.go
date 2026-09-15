package continuity

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestAgentManagerPublisherSendsStableProvenance(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/runs/import-transcript" || r.Method != http.MethodPost {
			t.Fatalf("request=%s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()
	err := (&AgentManagerPublisher{BaseURL: srv.URL}).Publish(context.Background(), PublicationRecord{
		SessionID: "web-session-1", LifecycleState: StateRecoverable, AgentType: "codex",
		CurrentTitle: "Recovered conversation", RolloutRef: "/private/native/rollout.jsonl", SourceFingerprint: "sha256:fingerprint",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got["sourceHarness"] != "web-console" || got["sourceSessionId"] != "web-session-1" || got["path"] != "/private/native/rollout.jsonl" {
		t.Fatalf("payload=%v", got)
	}
}

func TestAgentManagerPublisherMapsWebConsoleClaudeRunner(t *testing.T) {
	var payload struct {
		RunnerType string `json:"runnerType"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	err := (&AgentManagerPublisher{BaseURL: srv.URL}).Publish(context.Background(), PublicationRecord{
		SessionID: "web-session-claude", AgentType: "claude", RolloutRef: "/tmp/rollout.jsonl",
	})
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if payload.RunnerType != "claude-code" {
		t.Fatalf("runnerType = %q, want claude-code", payload.RunnerType)
	}
}

func TestAgentManagerPublisherRejectsMissingTranscript(t *testing.T) {
	err := (&AgentManagerPublisher{BaseURL: "http://example.invalid"}).Publish(context.Background(), PublicationRecord{SessionID: "s1"})
	if err == nil {
		t.Fatal("expected missing transcript error")
	}
}

func TestAgentManagerPublisherTreatsMissingTombstoneAsConverged(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/runs/external/tombstone" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	if err := (&AgentManagerPublisher{BaseURL: srv.URL}).Tombstone(context.Background(), PublicationRecord{SessionID: "never-imported"}); err != nil {
		t.Fatalf("Tombstone() error = %v, want nil for an already-absent source", err)
	}
}

func TestPublicRecordDoesNotExposeNativeLocators(t *testing.T) {
	public := PublicRecord(CatalogRecord{SessionID: "s1", AgentHomeRef: "/private/home", RolloutRef: "/private/rollout", Aliases: []Alias{{Kind: "agent_home", Value: "/private/home"}, {Kind: "agent_session", Value: "native-1"}, {Kind: "rollout", Value: "/private/rollout"}}})
	if public.AgentHomeRef != "" || public.RolloutRef != "" || len(public.Aliases) != 1 || public.Aliases[0].Value != "native-1" {
		t.Fatalf("public record=%+v", public)
	}
}

type recordingPublisher struct {
	seen  []PublicationRecord
	fails bool
}

func (p *recordingPublisher) Publish(_ context.Context, record PublicationRecord) error {
	p.seen = append(p.seen, record)
	if p.fails {
		return context.DeadlineExceeded
	}
	return nil
}

func (p *recordingPublisher) Tombstone(_ context.Context, record PublicationRecord) error {
	p.seen = append(p.seen, record)
	if p.fails {
		return context.DeadlineExceeded
	}
	return nil
}

func TestPublishPendingIsBoundedAndRetryable(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, statement := range []string{
		`CREATE TABLE conversation_catalog(session_id TEXT PRIMARY KEY, lifecycle_state TEXT, lifecycle_version INTEGER, backend TEXT, agent_type TEXT, agent_session_id TEXT, agent_home_ref TEXT, rollout_ref TEXT, original_title TEXT, current_title TEXT, topic_summary TEXT, cwd TEXT, created_at TEXT, last_activity_at TEXT, source_fingerprint TEXT)`,
		`CREATE TABLE conversation_aliases(session_id TEXT, alias_kind TEXT, alias_value TEXT, observed_at TEXT, PRIMARY KEY(alias_kind, alias_value))`,
		`CREATE TABLE continuity_publication_queue(session_id TEXT PRIMARY KEY, lifecycle_state TEXT, source_fingerprint TEXT, attempts INTEGER, last_error TEXT, published_fingerprint TEXT, updated_at TEXT)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	store := NewSQLCatalogStore(db)
	record := CatalogRecord{SessionID: "s1", LifecycleState: StateRecoverable, AgentType: "codex", CurrentTitle: "title", RolloutRef: "/tmp/rollout", CreatedAt: time.Unix(1, 0), LastActivityAt: time.Unix(1, 0), SourceFingerprint: "sha256:one"}
	if err := store.Upsert(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	failed := &recordingPublisher{fails: true}
	if report, err := store.PublishPending(context.Background(), failed, 1); err != nil || report.Failed != 1 || report.Next != 1 {
		t.Fatalf("failed report=%+v err=%v", report, err)
	}
	succeeded := &recordingPublisher{}
	if report, err := store.PublishPending(context.Background(), succeeded, 1); err != nil || report.Published != 1 || report.Next != 0 {
		t.Fatalf("success report=%+v err=%v", report, err)
	}
	if len(succeeded.seen) != 1 || succeeded.seen[0].SessionID != "s1" {
		t.Fatalf("published=%+v", succeeded.seen)
	}
}
