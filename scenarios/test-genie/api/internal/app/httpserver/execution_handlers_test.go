package httpserver

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"test-genie/internal/orchestrator"
)

func TestServer_handleListExecutions(t *testing.T) {
	history := &fakeExecutionHistory{
		listResults: []orchestrator.SuiteExecutionResult{
			{
				ExecutionID:  uuid.New(),
				ScenarioName: "demo",
				Success:      true,
			},
		},
	}
	srv := &Server{
		config:           Config{Port: "0", ServiceName: "Test Genie API"},
		router:           mux.NewRouter(),
		executionHistory: history,
		logger:           log.New(io.Discard, "", 0),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions?scenario=demo&limit=5", nil)
	w := httptest.NewRecorder()

	srv.handleListExecutions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if history.lastScenario != "demo" {
		t.Fatalf("expected scenario filter to propagate, got %s", history.lastScenario)
	}
}

func TestServer_handleGetExecution(t *testing.T) {
	executionID := uuid.MustParse("99999999-9999-9999-9999-999999999999")
	history := &fakeExecutionHistory{
		getResult: &orchestrator.SuiteExecutionResult{
			ExecutionID:  executionID,
			ScenarioName: "demo",
			StartedAt:    time.Now().Add(-time.Minute),
			CompletedAt:  time.Now(),
			Success:      false,
		},
	}
	srv := &Server{
		config:           Config{Port: "0", ServiceName: "Test Genie API"},
		router:           mux.NewRouter(),
		executionHistory: history,
		logger:           log.New(io.Discard, "", 0),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions/"+executionID.String(), nil)
	req = mux.SetURLVars(req, map[string]string{"id": executionID.String()})
	w := httptest.NewRecorder()

	srv.handleGetExecution(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if history.lastGet != executionID {
		t.Fatalf("expected handler to request execution %s, got %s", executionID, history.lastGet)
	}
}

type fakeExecutionHistory struct {
	listResults  []orchestrator.SuiteExecutionResult
	getResult    *orchestrator.SuiteExecutionResult
	latest       *orchestrator.SuiteExecutionResult
	lastScenario string
	lastGet      uuid.UUID
}

func (f *fakeExecutionHistory) List(ctx context.Context, scenario string, limit int, offset int) ([]orchestrator.SuiteExecutionResult, error) {
	f.lastScenario = scenario
	return f.listResults, nil
}

func (f *fakeExecutionHistory) Get(ctx context.Context, id uuid.UUID) (*orchestrator.SuiteExecutionResult, error) {
	f.lastGet = id
	return f.getResult, nil
}

func (f *fakeExecutionHistory) Latest(ctx context.Context) (*orchestrator.SuiteExecutionResult, error) {
	return f.latest, nil
}
