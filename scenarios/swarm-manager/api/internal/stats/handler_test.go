package stats_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/stats"
	"testing"

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
