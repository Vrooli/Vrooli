package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func TestServer_handleListExecutions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	srv := &Server{
		db:            db,
		suiteRequests: NewSuiteRequestService(NewPostgresSuiteRequestRepository(db)),
		executions:    NewSuiteExecutionRepository(db),
	}

	now := time.Now().UTC()
	rows := sqlmock.NewRows([]string{
		"id",
		"suite_request_id",
		"scenario_name",
		"preset_used",
		"success",
		"phases",
		"started_at",
		"completed_at",
	}).AddRow(
		"11111111-1111-1111-1111-111111111111",
		nil,
		"demo",
		nil,
		true,
		[]byte(`[{"name":"structure","status":"passed","durationSeconds":1}]`),
		now.Add(-time.Minute),
		now,
	)

	mock.ExpectQuery("SELECT\\s+id,\\s+suite_request_id").
		WithArgs("demo", 5, 0).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions?scenario=demo&limit=5", nil)
	w := httptest.NewRecorder()

	srv.handleListExecutions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestServer_handleGetExecution(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	srv := &Server{
		db:            db,
		suiteRequests: NewSuiteRequestService(NewPostgresSuiteRequestRepository(db)),
		executions:    NewSuiteExecutionRepository(db),
	}

	now := time.Now().UTC()
	executionID := uuid.MustParse("99999999-9999-9999-9999-999999999999")
	rows := sqlmock.NewRows([]string{
		"id",
		"suite_request_id",
		"scenario_name",
		"preset_used",
		"success",
		"phases",
		"started_at",
		"completed_at",
	}).AddRow(
		executionID.String(),
		nil,
		"demo",
		nil,
		false,
		[]byte(`[{"name":"structure","status":"failed","durationSeconds":2}]`),
		now.Add(-time.Minute),
		now,
	)

	mock.ExpectQuery("SELECT\\s+id,\\s+suite_request_id").
		WithArgs(executionID).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String(), nil)
	req = mux.SetURLVars(req, map[string]string{"id": executionID.String()})
	w := httptest.NewRecorder()

	srv.handleGetExecution(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
