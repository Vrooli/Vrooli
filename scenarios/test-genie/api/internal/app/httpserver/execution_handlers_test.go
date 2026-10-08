package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"test-genie/internal/orchestrator"
	"test-genie/internal/runmanager"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
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

func TestServer_handleAdmissionStatus(t *testing.T) {
	srv := &Server{runManager: runmanager.New(&stubSuiteExecutor{}, ""), logger: log.New(io.Discard, "", 0)}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admission", nil)
	res := httptest.NewRecorder()

	srv.handleAdmissionStatus(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", res.Code, res.Body.String())
	}
	var payload runmanager.AdmissionSnapshot
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("decode admission status: %v", err)
	}
	if payload.MaxConcurrentRuns < 1 || payload.MaxQueuedRuns < 1 || payload.MaxPreviewRuns < 1 {
		t.Fatalf("invalid admission limits: %#v", payload)
	}
}

type fakeExecutionHistory struct {
	listResults  []orchestrator.SuiteExecutionResult
	getResult    *orchestrator.SuiteExecutionResult
	latest       *orchestrator.SuiteExecutionResult
	lastScenario string
	lastLimit    int
	lastOffset   int
	lastGet      uuid.UUID
	listErr      error
}

func (f *fakeExecutionHistory) List(ctx context.Context, scenario string, limit int, offset int) ([]orchestrator.SuiteExecutionResult, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	f.lastScenario = scenario
	f.lastLimit = limit
	f.lastOffset = offset
	return f.listResults, nil
}

func (f *fakeExecutionHistory) Get(ctx context.Context, id uuid.UUID) (*orchestrator.SuiteExecutionResult, error) {
	f.lastGet = id
	return f.getResult, nil
}

func (f *fakeExecutionHistory) Latest(ctx context.Context) (*orchestrator.SuiteExecutionResult, error) {
	return f.latest, nil
}
