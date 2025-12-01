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

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	pq "github.com/lib/pq"

	"test-genie/internal/suite"
)

func TestServer_handleCreateSuiteRequest_InvalidPayload(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	srv := &Server{
		config:        Config{Port: "0", ServiceName: "Test Genie API"},
		db:            db,
		router:        mux.NewRouter(),
		suiteRequests: suite.NewSuiteRequestService(suite.NewPostgresSuiteRequestRepository(db)),
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
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	srv := &Server{
		config:        Config{Port: "0", ServiceName: "Test Genie API"},
		db:            db,
		router:        mux.NewRouter(),
		suiteRequests: suite.NewSuiteRequestService(suite.NewPostgresSuiteRequestRepository(db)),
		logger:        log.New(io.Discard, "", 0),
	}

	now := time.Now().UTC()
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
	}).AddRow(
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
		WithArgs(suite.MaxSuiteListPage).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/suite-requests", nil)
	w := httptest.NewRecorder()

	srv.handleListSuiteRequests(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var payload struct {
		Items []suite.SuiteRequest `json:"items"`
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

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
