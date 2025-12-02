package httpserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/queue"
	"test-genie/internal/scenarios"
	"test-genie/internal/shared"
)

func TestServer_handleGetSuiteRequestSuccess(t *testing.T) {
	id := uuid.New()
	q := &stubSuiteQueue{
		getResp: &queue.SuiteRequest{
			ID:           id,
			ScenarioName: "demo",
			Status:       "queued",
		},
	}
	server := &Server{
		config:        Config{Port: "0"},
		router:        mux.NewRouter(),
		suiteRequests: q,
		logger:        log.New(io.Discard, "", 0),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/suite-requests/"+id.String(), nil)
	req = mux.SetURLVars(req, map[string]string{"id": id.String()})
	rec := httptest.NewRecorder()

	server.handleGetSuiteRequest(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload queue.SuiteRequest
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.ID != id {
		t.Fatalf("expected suite request %s, got %s", id, payload.ID)
	}
	if q.lastGet != id {
		t.Fatalf("expected queue Get to receive %s, got %s", id, q.lastGet)
	}
}

func TestServer_handleGetSuiteRequestNotFound(t *testing.T) {
	q := &stubSuiteQueue{
		getErr: sql.ErrNoRows,
	}
	server := &Server{
		config:        Config{Port: "0"},
		router:        mux.NewRouter(),
		suiteRequests: q,
		logger:        log.New(io.Discard, "", 0),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/suite-requests/123", nil)
	req = mux.SetURLVars(req, map[string]string{"id": uuid.NewString()})
	rec := httptest.NewRecorder()

	server.handleGetSuiteRequest(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestServer_handleListScenarios(t *testing.T) {
	scenarioSvc := &stubScenarioDirectory{
		summaries: []scenarios.ScenarioSummary{
			{
				ScenarioName: "demo",
				Testing: &scenarios.TestingCapabilities{
					Phased:    true,
					HasTests:  true,
					Preferred: "phased",
					Commands: []scenarios.TestingCommand{
						{Type: "phased", Command: []string{"./test/run-tests.sh"}},
					},
				},
			},
			{ScenarioName: "other"},
		},
	}
	server := &Server{
		config:    Config{Port: "0"},
		router:    mux.NewRouter(),
		scenarios: scenarioSvc,
		logger:    log.New(io.Discard, "", 0),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios", nil)
	rec := httptest.NewRecorder()

	server.handleListScenarios(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var payload struct {
		Items []scenarios.ScenarioSummary `json:"items"`
		Count int                         `json:"count"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Count != 2 || len(payload.Items) != 2 {
		t.Fatalf("expected 2 scenarios, got %#v", payload)
	}
	if payload.Items[0].Testing == nil || payload.Items[0].Testing.Preferred != "phased" {
		t.Fatalf("expected testing capabilities in payload: %#v", payload.Items[0])
	}
}

func TestServer_handleListExecutionsAppliesQueryParams(t *testing.T) {
	history := &fakeExecutionHistory{
		listResults: []orchestrator.SuiteExecutionResult{
			{ScenarioName: "demo"},
		},
	}
	server := &Server{
		config:           Config{Port: "0"},
		router:           mux.NewRouter(),
		executionHistory: history,
		logger:           log.New(io.Discard, "", 0),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions?scenario=demo&limit=5&offset=2", nil)
	rec := httptest.NewRecorder()

	server.handleListExecutions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if history.lastScenario != "demo" || history.lastLimit != 5 || history.lastOffset != 2 {
		t.Fatalf("expected filters to propagate, got scenario=%s limit=%d offset=%d", history.lastScenario, history.lastLimit, history.lastOffset)
	}
}

func TestServer_handleListExecutionsError(t *testing.T) {
	history := &fakeExecutionHistory{
		listErr: errors.New("boom"),
	}
	server := &Server{
		config:           Config{Port: "0"},
		router:           mux.NewRouter(),
		executionHistory: history,
		logger:           log.New(io.Discard, "", 0),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/executions", nil)
	rec := httptest.NewRecorder()

	server.handleListExecutions(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestServer_handleGetScenarioErrors(t *testing.T) {
	tests := []struct {
		name       string
		param      string
		scenarioFn func() *Server
		wantStatus int
	}{
		{
			name:  "missing name",
			param: "   ",
			scenarioFn: func() *Server {
				return &Server{
					config:    Config{Port: "0"},
					router:    mux.NewRouter(),
					scenarios: &stubScenarioDirectory{},
					logger:    log.New(io.Discard, "", 0),
				}
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:  "not found",
			param: "missing",
			scenarioFn: func() *Server {
				return &Server{
					config: Config{Port: "0"},
					router: mux.NewRouter(),
					scenarios: &stubScenarioDirectory{
						getErr: sql.ErrNoRows,
					},
					logger: log.New(io.Discard, "", 0),
				}
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:  "lookup error",
			param: "demo",
			scenarioFn: func() *Server {
				return &Server{
					config: Config{Port: "0"},
					router: mux.NewRouter(),
					scenarios: &stubScenarioDirectory{
						getErr: context.DeadlineExceeded,
					},
					logger: log.New(io.Discard, "", 0),
				}
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.scenarioFn()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/scenarios/test", nil)
			req = mux.SetURLVars(req, map[string]string{"name": tt.param})
			rec := httptest.NewRecorder()

			server.handleGetScenario(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestServer_handleRunScenarioTests(t *testing.T) {
	command := &scenarios.TestingCommand{
		Type:    "phased",
		Command: []string{"./test/run-tests.sh"},
	}
	scenarioSvc := &stubScenarioDirectory{
		runResp:   command,
		runResult: &scenarios.TestingRunnerResult{LogPath: "/tmp/log"},
	}
	server := &Server{
		config:    Config{Port: "0"},
		router:    mux.NewRouter(),
		scenarios: scenarioSvc,
		logger:    log.New(io.Discard, "", 0),
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/demo/run-tests", strings.NewReader(`{"type":"phased"}`))
	req = mux.SetURLVars(req, map[string]string{"name": "demo"})
	rec := httptest.NewRecorder()

	server.handleRunScenarioTests(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if scenarioSvc.runName != "demo" || scenarioSvc.runPreferred != "phased" {
		t.Fatalf("expected scenario run invocation, got name=%s type=%s", scenarioSvc.runName, scenarioSvc.runPreferred)
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["type"] != "phased" {
		t.Fatalf("expected phased type in response, got %#v", payload)
	}
	if payload["logPath"] != "/tmp/log" {
		t.Fatalf("expected log path in response, got %#v", payload)
	}
}

func TestServer_handleRunScenarioTestsErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"validation", shared.NewValidationError("bad"), http.StatusBadRequest},
		{"not_found", os.ErrNotExist, http.StatusNotFound},
		{"internal", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scenarioSvc := &stubScenarioDirectory{
				runErr: tt.err,
			}
			server := &Server{
				config:    Config{Port: "0"},
				router:    mux.NewRouter(),
				scenarios: scenarioSvc,
				logger:    log.New(io.Discard, "", 0),
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/scenarios/demo/run-tests", nil)
			req = mux.SetURLVars(req, map[string]string{"name": "demo"})
			rec := httptest.NewRecorder()

			server.handleRunScenarioTests(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d (%s)", tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestServer_handleExecuteSuite(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		executor   *stubSuiteExecutor
		wantStatus int
		assert     func(t *testing.T, exec *stubSuiteExecutor)
	}{
		{
			name: "success",
			body: `{"scenarioName":"demo","suiteRequestId":"11111111-1111-1111-1111-111111111111","phases":["unit"],"failFast":true}`,
			executor: &stubSuiteExecutor{
				result: &orchestrator.SuiteExecutionResult{
					ExecutionID:  uuid.New(),
					ScenarioName: "demo",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				},
			},
			wantStatus: http.StatusOK,
			assert: func(t *testing.T, exec *stubSuiteExecutor) {
				if exec.input.Request.ScenarioName != "demo" {
					t.Fatalf("expected scenario demo, got %s", exec.input.Request.ScenarioName)
				}
				if exec.input.SuiteRequestID == nil {
					t.Fatal("expected suite request ID to be set")
				}
			},
		},
		{
			name:       "missing scenario",
			body:       `{}`,
			executor:   &stubSuiteExecutor{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid suite request id",
			body:       `{"scenarioName":"demo","suiteRequestId":"not-a-uuid"}`,
			executor:   &stubSuiteExecutor{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "suite request not found",
			body: `{"scenarioName":"demo","suiteRequestId":"11111111-1111-1111-1111-111111111111"}`,
			executor: &stubSuiteExecutor{
				err: execution.ErrSuiteRequestNotFound,
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "validation error",
			body: `{"scenarioName":"demo"}`,
			executor: &stubSuiteExecutor{
				err: shared.NewValidationError("bad request"),
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{
				config:       Config{Port: "0"},
				router:       mux.NewRouter(),
				executionSvc: tt.executor,
				logger:       log.New(io.Discard, "", 0),
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/executions", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			server.handleExecuteSuite(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d (%s)", tt.wantStatus, rec.Code, rec.Body.String())
			}
			if tt.assert != nil {
				tt.assert(t, tt.executor)
			}
		})
	}
}

func TestServer_handleHealthReportsOperations(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer db.Close()

	mock.ExpectPing().WillReturnError(nil)

	oldest := time.Now().Add(-2 * time.Minute)
	q := &stubSuiteQueue{
		snapshot: queue.SuiteRequestSnapshot{
			Total:          5,
			Queued:         3,
			Delegated:      1,
			Running:        1,
			Completed:      0,
			Failed:         0,
			OldestQueuedAt: &oldest,
		},
	}
	history := &fakeExecutionHistory{
		latest: &orchestrator.SuiteExecutionResult{
			ExecutionID:  uuid.New(),
			ScenarioName: "demo",
			StartedAt:    time.Now().Add(-time.Minute),
			CompletedAt:  time.Now(),
			Success:      true,
			PhaseSummary: orchestrator.PhaseSummary{Total: 2, Passed: 2},
		},
	}

	server := &Server{
		config:           Config{Port: "0", ServiceName: "Test Genie API"},
		db:               db,
		suiteRequests:    q,
		executionHistory: history,
		logger:           log.New(io.Discard, "", 0),
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	server.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode health payload: %v", err)
	}
	if payload["status"] != "healthy" {
		t.Fatalf("expected healthy status, got %v", payload["status"])
	}
	deps, ok := payload["dependencies"].(map[string]interface{})
	if !ok || deps["database"] != "connected" {
		t.Fatalf("expected database dependency, got %#v", deps)
	}
	operations, ok := payload["operations"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected operations payload, got %#v", payload["operations"])
	}
	queuePayload, ok := operations["queue"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected queue payload, got %#v", operations)
	}
	if queuePayload["pending"].(float64) != 4 {
		t.Fatalf("expected pending count 4, got %v", queuePayload["pending"])
	}
	executionPayload, ok := operations["lastExecution"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected last execution payload, got %#v", operations)
	}
	if executionPayload["scenario"] != "demo" {
		t.Fatalf("expected scenario demo, got %#v", executionPayload)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet mock expectations: %v", err)
	}
}

func TestServer_handleHealthDatabaseFailure(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer db.Close()

	mock.ExpectPing().WillReturnError(errors.New("offline"))
	q := &stubSuiteQueue{}
	history := &fakeExecutionHistory{}

	server := &Server{
		config:           Config{Port: "0"},
		db:               db,
		suiteRequests:    q,
		executionHistory: history,
		logger:           log.New(io.Discard, "", 0),
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	server.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 even when unhealthy, got %d", rec.Code)
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode health payload: %v", err)
	}
	if payload["status"] != "unhealthy" {
		t.Fatalf("expected unhealthy status, got %v", payload["status"])
	}
	deps := payload["dependencies"].(map[string]interface{})
	if deps["database"] != "disconnected" {
		t.Fatalf("expected disconnected dependency, got %#v", deps)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet mock expectations: %v", err)
	}
}

type stubSuiteQueue struct {
	getResp     *queue.SuiteRequest
	getErr      error
	lastGet     uuid.UUID
	snapshot    queue.SuiteRequestSnapshot
	snapshotErr error
}

func (s *stubSuiteQueue) Queue(ctx context.Context, payload queue.QueueSuiteRequestInput) (*queue.SuiteRequest, error) {
	return nil, nil
}

func (s *stubSuiteQueue) List(ctx context.Context, limit int) ([]queue.SuiteRequest, error) {
	return nil, nil
}

func (s *stubSuiteQueue) Get(ctx context.Context, id uuid.UUID) (*queue.SuiteRequest, error) {
	s.lastGet = id
	return s.getResp, s.getErr
}

func (s *stubSuiteQueue) StatusSnapshot(ctx context.Context) (queue.SuiteRequestSnapshot, error) {
	if s.snapshotErr != nil {
		return queue.SuiteRequestSnapshot{}, s.snapshotErr
	}
	return s.snapshot, nil
}

type stubScenarioDirectory struct {
	summaries    []scenarios.ScenarioSummary
	err          error
	getResp      *scenarios.ScenarioSummary
	getErr       error
	runResp      *scenarios.TestingCommand
	runResult    *scenarios.TestingRunnerResult
	runErr       error
	runName      string
	runPreferred string
}

func (s *stubScenarioDirectory) ListSummaries(ctx context.Context) ([]scenarios.ScenarioSummary, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.summaries, nil
}

func (s *stubScenarioDirectory) GetSummary(ctx context.Context, name string) (*scenarios.ScenarioSummary, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.getResp, nil
}

func (s *stubScenarioDirectory) RunScenarioTests(ctx context.Context, name string, preferred string) (*scenarios.TestingCommand, *scenarios.TestingRunnerResult, error) {
	s.runName = name
	s.runPreferred = preferred
	if s.runErr != nil {
		return nil, nil, s.runErr
	}
	return s.runResp, s.runResult, nil
}

type stubSuiteExecutor struct {
	input  execution.SuiteExecutionInput
	result *orchestrator.SuiteExecutionResult
	err    error
}

func (s *stubSuiteExecutor) Execute(ctx context.Context, input execution.SuiteExecutionInput) (*orchestrator.SuiteExecutionResult, error) {
	s.input = input
	return s.result, s.err
}
