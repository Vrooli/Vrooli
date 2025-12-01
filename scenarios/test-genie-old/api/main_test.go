package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	pq "github.com/lib/pq"
)

func TestGetTestSuiteHandler_ReturnsPersistedSuite(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer func() {
		mockDB.Close()
		db = nil
	}()

	db = mockDB

	suiteID := uuid.New()
	generatedAt := time.Now().UTC().Truncate(time.Second)
	coverageJSON := []byte(`{"code_coverage": 92.5, "branch_coverage": 88.0, "function_coverage": 90.0}`)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, scenario_name, suite_type, coverage_metrics, generated_at, last_executed, status FROM test_suites WHERE id = $1`)).
		WithArgs(suiteID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "scenario_name", "suite_type", "coverage_metrics", "generated_at", "last_executed", "status"}).
			AddRow(suiteID.String(), "calendar", "unit,integration", coverageJSON, generatedAt, nil, "active"))

	dependenciesJSON := []byte(`[]`)
	tagsJSON := []byte(`[]`)
	createdAt := generatedAt.Add(1 * time.Minute)
	updatedAt := createdAt

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, description, test_type, test_code, expected_result, execution_timeout, dependencies, tags, priority, created_at, updated_at FROM test_cases WHERE suite_id = $1 ORDER BY created_at ASC`)).
		WithArgs(suiteID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "test_type", "test_code", "expected_result", "execution_timeout", "dependencies", "tags", "priority", "created_at", "updated_at"}).
			AddRow(uuid.New().String(), "health_check", "ensures service responds", "unit", "assert true", "pass", int64(120), dependenciesJSON, tagsJSON, "high", createdAt, updatedAt))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "suite_id", Value: suiteID.String()}}
	ctx.Request = httptest.NewRequest("GET", "/api/v1/test-suite/"+suiteID.String(), nil)

	getTestSuiteHandler(ctx)

	if recorder.Code != 200 {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var response TestSuite
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.ID != suiteID {
		t.Fatalf("expected suite ID %s, got %s", suiteID, response.ID)
	}
	if response.ScenarioName != "calendar" {
		t.Fatalf("expected scenario name 'calendar', got %s", response.ScenarioName)
	}
	if len(response.TestCases) != 1 {
		t.Fatalf("expected 1 test case, got %d", len(response.TestCases))
	}
	if response.CoverageMetrics.CodeCoverage != 92.5 {
		t.Fatalf("unexpected coverage metric: %+v", response.CoverageMetrics)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestExecuteTestSuiteHandler_PersistsRunningStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer func() {
		mockDB.Close()
		db = nil
	}()

	db = mockDB
	suiteID := uuid.New()

	originalRunner := runTestSuite
	defer func() { runTestSuite = originalRunner }()
	done := make(chan struct{})
	runTestSuite = func(uuid.UUID, uuid.UUID, string, string, int) {
		close(done)
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO test_executions (id, suite_id, execution_type, start_time, status, environment) VALUES ($1, $2, $3, $4, $5, $6)`)).
		WithArgs(sqlmock.AnyArg(), suiteID, "full", sqlmock.AnyArg(), "running", "local").
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM test_cases WHERE suite_id = $1`)).
		WithArgs(suiteID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "suite_id", Value: suiteID.String()}}
	body := bytes.NewBufferString(`{"execution_type":"full","environment":"local","parallel_execution":false}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/test-suite/"+suiteID.String()+"/execute", body)
	request.Header.Set("Content-Type", "application/json")
	ctx.Request = request

	executeTestSuiteHandler(ctx)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", recorder.Code)
	}

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected background runner to be invoked")
	}

	var response ExecuteTestSuiteResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Status != "running" {
		t.Fatalf("expected execution status 'running', got %s", response.Status)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestExecuteTestSuiteHandler_SyncsDiscoveredSuiteOnForeignKeyViolation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer func() {
		mockDB.Close()
		db = nil
		syncSuiteForExecution = syncDiscoveredSuiteToDatabase
	}()

	db = mockDB
	suiteID := uuid.New()

	originalRunner := runTestSuite
	defer func() { runTestSuite = originalRunner }()
	done := make(chan struct{})
	runTestSuite = func(uuid.UUID, uuid.UUID, string, string, int) {
		close(done)
	}

	called := false
	syncSuiteForExecution = func(contextCtx context.Context, id uuid.UUID) (*TestSuite, error) {
		called = true
		return &TestSuite{ID: id}, nil
	}

	insertExpectation := regexp.QuoteMeta(`INSERT INTO test_executions (id, suite_id, execution_type, start_time, status, environment) VALUES ($1, $2, $3, $4, $5, $6)`)

	mock.ExpectExec(insertExpectation).
		WithArgs(sqlmock.AnyArg(), suiteID, "full", sqlmock.AnyArg(), "running", "local").
		WillReturnError(&pq.Error{Code: "23503"})

	mock.ExpectExec(insertExpectation).
		WithArgs(sqlmock.AnyArg(), suiteID, "full", sqlmock.AnyArg(), "running", "local").
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM test_cases WHERE suite_id = $1`)).
		WithArgs(suiteID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "suite_id", Value: suiteID.String()}}
	body := bytes.NewBufferString(`{"execution_type":"full","environment":"local"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/test-suite/"+suiteID.String()+"/execute", body)
	request.Header.Set("Content-Type", "application/json")
	ctx.Request = request

	executeTestSuiteHandler(ctx)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", recorder.Code)
	}

	if !called {
		t.Fatalf("expected syncSuiteForExecution to be invoked")
	}

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected background runner to be invoked")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
