package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	pq "github.com/lib/pq"
)

func TestBuildSuiteRequestDefaults(t *testing.T) {
	req, err := buildSuiteRequest(suiteRequestPayload{
		ScenarioName: "document-manager",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if req.ScenarioName != "document-manager" {
		t.Fatalf("unexpected scenario name: %s", req.ScenarioName)
	}

	if req.CoverageTarget != 95 {
		t.Fatalf("expected coverage default to 95, got %d", req.CoverageTarget)
	}

	expectedTypes := []string{"unit", "integration"}
	if !slices.Equal(req.RequestedTypes, expectedTypes) {
		t.Fatalf("expected default types %v, got %v", expectedTypes, req.RequestedTypes)
	}

	if req.Priority != suitePriorityNormal {
		t.Fatalf("expected default priority %s, got %s", suitePriorityNormal, req.Priority)
	}

	if req.EstimatedQueueTime == 0 {
		t.Fatal("expected estimated queue time to be set")
	}
}

func TestBuildSuiteRequestInvalidType(t *testing.T) {
	_, err := buildSuiteRequest(suiteRequestPayload{
		ScenarioName:   "document-manager",
		RequestedTypes: stringList{"invalid"},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestServer_handleCreateSuiteRequest(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	srv := &Server{
		db: db,
	}

	mock.ExpectQuery("INSERT INTO suite_requests").
		WithArgs(
			sqlmock.AnyArg(), // id
			"demo-scenario",
			sqlmock.AnyArg(), // requested_types
			90,
			"high",
			suiteStatusQueued,
			sqlmock.AnyArg(), // notes
			sqlmock.AnyArg(), // delegation_issue_id
		).
		WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).
			AddRow(time.Now(), time.Now()))

	body := `{
		"scenarioName": "demo-scenario",
		"requestedTypes": ["unit","performance"],
		"coverageTarget": 90,
		"priority": "high",
		"notes": "seed from test"
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/suite-requests", strings.NewReader(body))
	w := httptest.NewRecorder()

	srv.handleCreateSuiteRequest(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.StatusCode)
	}

	var output SuiteRequest
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if output.ScenarioName != "demo-scenario" {
		t.Fatalf("unexpected scenario name %s", output.ScenarioName)
	}
	if output.Priority != "high" {
		t.Fatalf("unexpected priority %s", output.Priority)
	}
	if !slices.Contains(output.RequestedTypes, "unit") {
		t.Fatalf("expected requested types in payload: %v", output.RequestedTypes)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestServer_handleListSuiteRequests(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	srv := &Server{db: db}

	now := time.Now()
	rows := sqlmock.NewRows([]string{
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
	}).
		AddRow(
			"11111111-1111-1111-1111-111111111111",
			"demo",
			pq.StringArray{"unit"},
			95,
			"normal",
			"queued",
			"note",
			nil,
			now,
			now,
		)

	mock.ExpectQuery("SELECT\\s+id").
		WithArgs(maxSuiteListPage).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/suite-requests", nil)
	w := httptest.NewRecorder()
	srv.handleListSuiteRequests(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var payload struct {
		Items []SuiteRequest `json:"items"`
		Count int            `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}

	if payload.Count != 1 || len(payload.Items) != 1 {
		t.Fatalf("expected one suite, got %#v", payload)
	}
	if payload.Items[0].EstimatedQueueTime == 0 {
		t.Fatal("expected estimated queue time to be populated")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestServer_handleCreateSuiteRequestInvalidPayload(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	srv := &Server{db: db}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/suite-requests", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()
	srv.handleCreateSuiteRequest(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
