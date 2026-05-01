package stats_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/stats"

	"github.com/gorilla/mux"
	_ "modernc.org/sqlite"
)

func TestGetStatsEmpty(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	repo := eventlog.NewSQLiteRepository(db)
	if err := repo.InitSchema(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}

	engine := stats.NewEngine(repo)
	if err := engine.Rebuild(context.Background()); err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	handler := stats.NewHandler(engine)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/api/v1/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}

	var resp stats.StatsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.EventCount != 0 {
		t.Errorf("event count: got %d, want 0", resp.EventCount)
	}
}

func TestGetStatsIncludesAgentSessionMetrics(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	repo := eventlog.NewSQLiteRepository(db)
	if err := repo.InitSchema(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}

	started := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	appendEvent := func(eventType eventlog.EventType, metadata string, ts time.Time) {
		t.Helper()
		_, err := repo.Append(context.Background(), eventlog.Event{
			Timestamp:  ts,
			EntityType: eventlog.EntityAgentSession,
			EntityID:   "sess_stats",
			EventType:  eventType,
			Metadata:   json.RawMessage(metadata),
		})
		if err != nil {
			t.Fatalf("append %s: %v", eventType, err)
		}
	}

	appendEvent(eventlog.EventAgentSessionCreated, `{"session_kind":"meta_orchestration","status":"starting"}`, started)
	appendEvent(eventlog.EventAgentSessionStarted, `{"session_kind":"meta_orchestration","status":"running"}`, started.Add(time.Minute))
	appendEvent(eventlog.EventAgentSessionContinued, `{"session_kind":"meta_orchestration","status":"running"}`, started.Add(2*time.Minute))
	appendEvent(eventlog.EventAgentSessionProposalCreated, `{"session_kind":"meta_orchestration","proposal_kind":"backlog_batch_import"}`, started.Add(5*time.Minute))
	appendEvent(eventlog.EventAgentSessionProposalApplied, `{"session_kind":"meta_orchestration","proposal_kind":"backlog_batch_import","artifact_count":2}`, started.Add(6*time.Minute))
	appendEvent(eventlog.EventAgentSessionArtifactLinked, `{"session_kind":"meta_orchestration","artifact_type":"backlog_item","action":"created"}`, started.Add(7*time.Minute))
	appendEvent(eventlog.EventAgentSessionArtifactLinked, `{"session_kind":"meta_orchestration","artifact_type":"initiative","action":"created"}`, started.Add(8*time.Minute))
	appendEvent(eventlog.EventAgentSessionCompleted, `{"session_kind":"meta_orchestration","status":"complete"}`, started.Add(9*time.Minute))

	engine := stats.NewEngine(repo)
	if err := engine.Rebuild(context.Background()); err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	resp := engine.GetStats()
	if resp.Session.TotalSessions != 1 {
		t.Fatalf("total sessions = %d, want 1", resp.Session.TotalSessions)
	}
	if got := resp.Session.SessionsByKind["meta_orchestration"]; got != 1 {
		t.Fatalf("sessions by kind = %d, want 1", got)
	}
	if got := resp.Session.ProposalApplyRateByKind["meta_orchestration"]; got.Rate != 1 || got.SampleSize != 1 {
		t.Fatalf("apply rate = %+v, want rate=1 sample=1", got)
	}
	if resp.Session.SessionCreatedBacklogItems != 1 || resp.Session.SessionCreatedInitiatives != 1 {
		t.Fatalf("created counts = backlog:%d initiatives:%d, want 1/1", resp.Session.SessionCreatedBacklogItems, resp.Session.SessionCreatedInitiatives)
	}
	if resp.Session.AverageMessagesPerSession != 2 {
		t.Fatalf("avg messages = %v, want 2", resp.Session.AverageMessagesPerSession)
	}
	if resp.Session.FirstProposalSampleSize != 1 || resp.Session.AverageTimeToFirstProposalSeconds != 300 {
		t.Fatalf("first proposal = n:%d avg:%v, want n=1 avg=300", resp.Session.FirstProposalSampleSize, resp.Session.AverageTimeToFirstProposalSeconds)
	}
}

func TestGetStatsCategoryFilter(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	repo := eventlog.NewSQLiteRepository(db)
	if err := repo.InitSchema(context.Background()); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Add an event so throughput has data.
	_, err = repo.Append(context.Background(), eventlog.Event{
		EntityType: eventlog.EntityBacklogItem,
		EntityID:   "execute/a",
		EventType:  eventlog.EventBacklogCreated,
		Metadata:   json.RawMessage(`{"kind":"execute","status":"backlog","priority":5}`),
	})
	if err != nil {
		t.Fatalf("append: %v", err)
	}

	engine := stats.NewEngine(repo)
	if err := engine.Rebuild(context.Background()); err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	handler := stats.NewHandler(engine)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Request only throughput category.
	req := httptest.NewRequest("GET", "/api/v1/stats?category=throughput", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", w.Code)
	}

	var resp stats.StatsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Throughput should have data.
	if resp.EventCount != 1 {
		t.Errorf("event count: got %d, want 1", resp.EventCount)
	}
	// Non-requested categories should be zeroed.
	if resp.Agent.TotalExecutions != 0 {
		t.Errorf("agent should be zeroed, got %+v", resp.Agent)
	}
}
