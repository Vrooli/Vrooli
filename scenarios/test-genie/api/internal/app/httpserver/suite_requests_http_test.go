package httpserver

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"test-genie/internal/queue"
	"test-genie/internal/storage/sqliteutil"
	"test-genie/internal/testsqlite"

	"github.com/gorilla/mux"
)

func TestServer_handleCreateSuiteRequest_InvalidPayload(t *testing.T) {
	db := testsqlite.Open(t)

	srv := &Server{
		config:        Config{Port: "0", ServiceName: "Test Genie API"},
		db:            db,
		router:        mux.NewRouter(),
		suiteRequests: queue.NewSuiteRequestService(queue.NewSQLiteSuiteRequestRepository(db)),
		logger:        log.New(io.Discard, "", 0),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/suite-requests", strings.NewReader(`{}`))
	w := httptest.NewRecorder()

	srv.handleCreateSuiteRequest(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestServer_handleListSuiteRequests(t *testing.T) {
	db := testsqlite.Open(t)

	srv := &Server{
		config:        Config{Port: "0", ServiceName: "Test Genie API"},
		db:            db,
		router:        mux.NewRouter(),
		suiteRequests: queue.NewSuiteRequestService(queue.NewSQLiteSuiteRequestRepository(db)),
		logger:        log.New(io.Discard, "", 0),
	}

	now := time.Now().UTC()
	if _, err := db.Exec(`
INSERT INTO suite_requests (
	id, scenario_name, requested_types, coverage_target, priority, status, notes, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
`,
		"11111111-1111-1111-1111-111111111111",
		"demo",
		`["unit"]`,
		95,
		"normal",
		"queued",
		"note",
		sqliteutil.FormatTimestamp(now),
		sqliteutil.FormatTimestamp(now),
	); err != nil {
		t.Fatalf("seed suite request: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/suite-requests", nil)
	w := httptest.NewRecorder()

	srv.handleListSuiteRequests(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var payload struct {
		Items []queue.SuiteRequest `json:"items"`
		Count int                  `json:"count"`
	}
	if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	if payload.Count != 1 || len(payload.Items) != 1 {
		t.Fatalf("expected single suite request, got %#v", payload)
	}
	if payload.Items[0].EstimatedQueueTime == 0 {
		t.Fatalf("expected queue time to be populated: %#v", payload.Items[0])
	}
}
