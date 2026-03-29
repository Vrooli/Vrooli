package httpserver

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"test-genie/internal/queue"
	"test-genie/internal/storage/sqliteutil"
	"test-genie/internal/testsqlite"
)

func newTestServer(t *testing.T, db *sql.DB) *Server {
	t.Helper()
	srv := &Server{
		config: Config{
			Port:        "0",
			ServiceName: "Test Genie API",
		},
		db:            db,
		router:        mux.NewRouter(),
		suiteRequests: queue.NewSuiteRequestService(queue.NewSQLiteSuiteRequestRepository(db)),
		logger:        log.New(io.Discard, "", 0),
	}
	srv.setupRoutes()
	return srv
}

func TestSuiteRequestLifecycleIntegration(t *testing.T) {
	t.Run("[REQ:TESTGENIE-SUITE-P0] API queues + fetches suite requests", func(t *testing.T) {
		db := testsqlite.Open(t)
		server := newTestServer(t, db)
		expectedTypes := []string{"unit", "integration"}

		body := bytes.NewBufferString(`{"scenarioName":"ecosystem-manager"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/suite-requests", body)
		rec := httptest.NewRecorder()
		server.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d", rec.Code)
		}

		var created queue.SuiteRequest
		if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if created.ScenarioName != "ecosystem-manager" {
			t.Fatalf("unexpected scenario saved: %s", created.ScenarioName)
		}
		if created.CoverageTarget != 95 {
			t.Fatalf("expected coverage default 95, got %d", created.CoverageTarget)
		}
		if len(created.RequestedTypes) != len(expectedTypes) {
			t.Fatalf("expected fallback types %v, got %v", expectedTypes, created.RequestedTypes)
		}
		for i, typ := range expectedTypes {
			if created.RequestedTypes[i] != typ {
				t.Fatalf("expected requestedTypes[%d]=%s, got %s", i, typ, created.RequestedTypes[i])
			}
		}
		if created.Priority != "normal" {
			t.Fatalf("expected default priority normal, got %s", created.Priority)
		}
		if created.DelegationIssueID != nil {
			t.Fatalf("expected no delegation issue id for deterministic fallback, got %s", *created.DelegationIssueID)
		}

		now := time.Now().UTC()
		if _, err := db.Exec(`
INSERT INTO suite_requests (
	id, scenario_name, requested_types, coverage_target, priority, status, notes, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET updated_at = excluded.updated_at
`,
			created.ID.String(),
			created.ScenarioName,
			`["unit","integration"]`,
			created.CoverageTarget,
			created.Priority,
			"queued",
			nil,
			sqliteutil.FormatTimestamp(now),
			sqliteutil.FormatTimestamp(now),
		); err != nil {
			t.Fatalf("upsert created request: %v", err)
		}

		getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/suite-requests/%s", created.ID), nil)
		getRec := httptest.NewRecorder()
		server.router.ServeHTTP(getRec, getReq)

		if getRec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", getRec.Code)
		}

		var fetched queue.SuiteRequest
		if err := json.NewDecoder(getRec.Body).Decode(&fetched); err != nil {
			t.Fatalf("failed to decode fetched suite: %v", err)
		}

		if fetched.ID != created.ID {
			t.Fatalf("expected id %s, got %s", created.ID, fetched.ID)
		}
		expectedEstimate := (len(expectedTypes) * 30) + 95
		if fetched.EstimatedQueueTime != expectedEstimate {
			t.Fatalf("unexpected queue estimate %d", fetched.EstimatedQueueTime)
		}
		if fetched.DelegationIssueID != nil {
			t.Fatalf("expected nil delegation id, got %s", *fetched.DelegationIssueID)
		}
	})
}
