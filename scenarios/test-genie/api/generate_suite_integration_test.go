package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	pq "github.com/lib/pq"
)

func newTestServer(t *testing.T, db *sql.DB) *Server {
	t.Helper()
	srv := &Server{
		config: &Config{
			Port:        "0",
			DatabaseURL: "postgres://test-genie",
		},
		db:     db,
		router: mux.NewRouter(),
	}
	srv.setupRoutes()
	return srv
}

func TestSuiteRequestLifecycleIntegration(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	server := newTestServer(t, db)

	now := time.Now()
	mock.ExpectQuery("INSERT INTO suite_requests").
		WithArgs(
			sqlmock.AnyArg(),    // id
			"ecosystem-manager", // scenario_name
			sqlmock.AnyArg(),    // requested_types array
			95,                  // coverage_target default
			suitePriorityNormal, // priority fallback
			suiteStatusQueued,   // status
			sqlmock.AnyArg(),    // notes (NULL)
			sqlmock.AnyArg(),    // delegation_issue_id (NULL)
		).
		WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(now, now))

	body := bytes.NewBufferString(`{"scenarioName":"ecosystem-manager"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/suite-requests", body)
	rec := httptest.NewRecorder()
	server.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	var created SuiteRequest
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if created.ScenarioName != "ecosystem-manager" {
		t.Fatalf("unexpected scenario saved: %s", created.ScenarioName)
	}
	if created.CoverageTarget != 95 {
		t.Fatalf("expected coverage default 95, got %d", created.CoverageTarget)
	}
	if len(created.RequestedTypes) != len(defaultSuiteTypes) {
		t.Fatalf("expected fallback types %v, got %v", defaultSuiteTypes, created.RequestedTypes)
	}
	if created.Priority != suitePriorityNormal {
		t.Fatalf("expected default priority %s, got %s", suitePriorityNormal, created.Priority)
	}
	if created.DelegationIssueID != nil {
		t.Fatalf("expected no delegation issue id for deterministic fallback, got %s", *created.DelegationIssueID)
	}

	expectedID := created.ID
	row := sqlmock.NewRows([]string{
		"id",
		"scenario_name",
		"requested_types",
		"coverage_target",
		"priority",
		"status",
		"notes",
		"delegation_issue_id",
		"created_at",
		"updated_at",
	}).AddRow(
		expectedID,
		created.ScenarioName,
		pq.StringArray(defaultSuiteTypes),
		created.CoverageTarget,
		created.Priority,
		suiteStatusQueued,
		nil,
		nil,
		now,
		now,
	)

	mock.ExpectQuery("SELECT\\s+id").
		WithArgs(expectedID).
		WillReturnRows(row)

	getReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/suite-requests/%s", expectedID), nil)
	getRec := httptest.NewRecorder()
	server.router.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", getRec.Code)
	}

	var fetched SuiteRequest
	if err := json.NewDecoder(getRec.Body).Decode(&fetched); err != nil {
		t.Fatalf("failed to decode fetched suite: %v", err)
	}

	if fetched.ID != expectedID {
		t.Fatalf("expected id %s, got %s", expectedID, fetched.ID)
	}
	if fetched.EstimatedQueueTime != estimateQueueSeconds(len(defaultSuiteTypes), 95) {
		t.Fatalf("unexpected queue estimate %d", fetched.EstimatedQueueTime)
	}
	if fetched.DelegationIssueID != nil {
		t.Fatalf("expected nil delegation id, got %s", *fetched.DelegationIssueID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
