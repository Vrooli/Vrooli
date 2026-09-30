// Tests for OperationalStatsHandler — focuses on the typed-Category
// 400-on-unknown contract (one of the four weakness fixes promised by
// the stats engine port).

package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"agent-manager/internal/eventlog"
	"agent-manager/internal/stats"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func newOperationalTestStack(t *testing.T) *mux.Router {
	t.Helper()
	dir := t.TempDir()
	db, err := sqlx.Connect("sqlite", "file:"+filepath.Join(dir, "h.db"))
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	schema := `
	CREATE TABLE IF NOT EXISTS run_events (
		id TEXT PRIMARY KEY,
		run_id TEXT NOT NULL,
		sequence INTEGER NOT NULL,
		event_type TEXT NOT NULL,
		timestamp TEXT NOT NULL,
		schema_version INTEGER NOT NULL DEFAULT 1,
		data TEXT NOT NULL,
		UNIQUE(run_id, sequence)
	);
	CREATE TABLE IF NOT EXISTS stats_checkpoint (
		name TEXT PRIMARY KEY,
		last_rowid INTEGER NOT NULL,
		updated_at TEXT NOT NULL DEFAULT (datetime('now'))
	);`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("schema: %v", err)
	}

	repo := eventlog.NewSQLiteRepository(db)
	cp := stats.NewSQLiteCheckpointStore(db)
	engine := stats.NewEngine(repo, cp, "test")

	r := mux.NewRouter()
	NewOperationalStatsHandler(engine).RegisterRoutes(r)
	return r
}

// TestOperationalHandler_BadCategory400 pins the contract that an
// unknown ?category=… returns HTTP 400 (with the known-categories list)
// instead of silently returning empty stats. This is the swarm-manager
// weakness #2 fix: a typo on the URL must not look like "no data".
func TestOperationalHandler_BadCategory400(t *testing.T) {
	router := newOperationalTestStack(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/operational?category=does_not_exist", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (silent zero would be 200)", w.Code)
	}
	var resp struct {
		Error           string   `json:"error"`
		KnownCategories []string `json:"known_categories"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Error == "" {
		t.Error("error message should explain the rejection")
	}
	if len(resp.KnownCategories) == 0 {
		t.Error("response should list the known categories so clients can correct their query")
	}
}

// TestOperationalHandler_DefaultCategoryIsSummary checks the empty
// ?category= routes to summary (no 400, structured 200 body).
func TestOperationalHandler_DefaultCategoryIsSummary(t *testing.T) {
	router := newOperationalTestStack(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/operational", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct == "" {
		t.Error("missing Content-Type")
	}
}

// TestOperationalHandler_FallbackEndpoint_Renders ensures the dedicated
// fallback URL succeeds even on an empty store and returns valid JSON.
func TestOperationalHandler_FallbackEndpoint_Renders(t *testing.T) {
	router := newOperationalTestStack(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/fallback", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("response is not valid JSON: %v", err)
	}
	if _, ok := resp["history"]; !ok {
		t.Error("response missing history field")
	}
}
