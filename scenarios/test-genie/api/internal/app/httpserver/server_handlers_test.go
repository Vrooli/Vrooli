package httpserver

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/runmanager"
	"test-genie/internal/scenarios"
	"test-genie/internal/selfhealthsnapshots"
	"test-genie/internal/shared"
	"test-genie/internal/storage/sqlitedb"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	apihealth "github.com/vrooli/api-core/health"
	_ "modernc.org/sqlite"
)

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
						{Type: "phased", Command: []string{"./coverage/run-tests.sh"}},
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
		Command: []string{"./coverage/run-tests.sh"},
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
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/scenarios/demo/run-tests",
		strings.NewReader(`{"type":"phased","paths":["api/foo.go"],"playbooks":["bas/cases/login.json"],"filter":"UserTest"}`),
	)
	req = mux.SetURLVars(req, map[string]string{"name": "demo"})
	rec := httptest.NewRecorder()

	server.handleRunScenarioTests(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if scenarioSvc.runName != "demo" || scenarioSvc.runPreferred != "phased" {
		t.Fatalf("expected scenario run invocation, got name=%s type=%s", scenarioSvc.runName, scenarioSvc.runPreferred)
	}
	if len(scenarioSvc.runArgs) == 0 || scenarioSvc.runArgs[0] != "--path" {
		t.Fatalf("expected extra args to be forwarded, got %v", scenarioSvc.runArgs)
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
		planner    *fakeExecutionPlanner
		wantStatus int
		assert     func(t *testing.T, exec *stubSuiteExecutor)
	}{
		{
			name: "success",
			body: `{"scenarioName":"demo","phases":["unit"],"failFast":true,"uiUrl":"http://localhost:35771","apiUrl":"http://localhost:17551"}`,
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
				if exec.input.Request.UIURL != "http://localhost:35771" {
					t.Fatalf("expected uiUrl to pass through, got %s", exec.input.Request.UIURL)
				}
				if exec.input.Request.APIURL != "http://localhost:17551" {
					t.Fatalf("expected apiUrl to pass through, got %s", exec.input.Request.APIURL)
				}
			},
		},
		{
			name: "adaptive profile selection is expanded before run start",
			body: `{"scenarioName":"demo","preset":"quick"}`,
			executor: &stubSuiteExecutor{
				result: &orchestrator.SuiteExecutionResult{
					ExecutionID:  uuid.New(),
					ScenarioName: "demo",
					StartedAt:    time.Now(),
					CompletedAt:  time.Now(),
				},
			},
			planner: &fakeExecutionPlanner{
				result: &execution.ExecutionPlanPreview{
					ScenarioName: "demo",
					PresetUsed:   "quick",
					Profile: &execution.ProfilePlan{
						Name:          "quick",
						Strategy:      "budget_fast_feedback",
						BudgetSeconds: 180,
					},
					Phases: []execution.PlannedPhase{
						{Name: "structure", EstimatedDurationSeconds: 10},
						{Name: "unit", EstimatedDurationSeconds: 40},
					},
					OmittedPhases: []execution.PlannedPhase{
						{Name: "performance", OmissionReasons: []string{"omitted_budget_exceeded"}},
					},
					Summary: execution.ExecutionPlanSummary{
						PhaseCount:               2,
						EstimatedDurationSeconds: 50,
						BudgetSeconds:            180,
					},
				},
			},
			wantStatus: http.StatusOK,
			assert: func(t *testing.T, exec *stubSuiteExecutor) {
				// The budget-trimmed selection must reach the executor — it
				// cannot re-derive a profile fit — but it travels as a planner
				// resolution, not as explicit operator intent. Explicit phases
				// carry no preset, which is what erased preset_used on every
				// durable run and broke baseline reuse eligibility.
				if got := strings.Join(exec.input.Request.ResolvedPhases, ","); got != "structure,unit" {
					t.Fatalf("resolved phases = %s, want adaptive preview selection", got)
				}
				if len(exec.input.Request.Phases) != 0 {
					t.Fatalf("explicit phases = %v, want none for a preset request", exec.input.Request.Phases)
				}
				if exec.input.Request.Preset != "quick" {
					t.Fatalf("preset should remain quick for run metadata, got %q", exec.input.Request.Preset)
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
			name:       "retired suite request selector",
			body:       `{"scenarioName":"demo","suiteRequestId":"11111111-1111-1111-1111-111111111111"}`,
			executor:   &stubSuiteExecutor{},
			wantStatus: http.StatusBadRequest,
		},
		{
			// Errors raised while the suite executes surface as a durable failed
			// run. The blocking endpoint returns that terminal result with HTTP
			// 200; HTTP 500 is reserved for transport or service failures.
			name: "execution failure",
			body: `{"scenarioName":"demo"}`,
			executor: &stubSuiteExecutor{
				err: errors.New("execution failed"),
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{
				config:     Config{Port: "0"},
				router:     mux.NewRouter(),
				runManager: runmanager.New(tt.executor, ""),
				logger:     log.New(io.Discard, "", 0),
			}
			if tt.planner != nil {
				server.executionPlanner = tt.planner
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

func TestServer_handleExecuteSuiteIncludesFailureDetails(t *testing.T) {
	server := &Server{
		config: Config{Port: "0"},
		router: mux.NewRouter(),
		runManager: runmanager.New(&stubSuiteExecutor{
			err: errors.New("start target scenario demo: exit status 2"),
		}, ""),
		logger: log.New(io.Discard, "", 0),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/executions", strings.NewReader(`{
		"scenarioName": "demo",
		"scenarioPath": "/tmp/vrooli-template/scenarios/demo"
	}`))
	rec := httptest.NewRecorder()

	server.handleExecuteSuite(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var payload orchestrator.SuiteExecutionResult
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Success {
		t.Fatal("expected success=false")
	}
	if payload.Verdict != orchestrator.SuiteVerdictFail {
		t.Fatalf("verdict = %q, want %q", payload.Verdict, orchestrator.SuiteVerdictFail)
	}
	if payload.FailureReason != "start target scenario demo: exit status 2" {
		t.Fatalf("failureReason = %q", payload.FailureReason)
	}
	if payload.ScenarioName != "demo" {
		t.Fatalf("scenarioName = %q", payload.ScenarioName)
	}
	if payload.RunID == "" {
		t.Fatal("expected durable run id for terminal failure")
	}
}

func TestServer_handleHealthUsesCanonicalBoundedContract(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer db.Close()

	mock.ExpectPing().WillReturnError(nil)

	server := &Server{
		config: Config{Port: "0", ServiceName: "Test Genie API"},
		db:     database.NewFromPrimary(db),
		logger: log.New(io.Discard, "", 0),
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	server.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.Bytes()
	var payload map[string]interface{}
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&payload); err != nil {
		t.Fatalf("decode health payload: %v", err)
	}
	if payload["status"] != "healthy" {
		t.Fatalf("expected healthy status, got %v", payload["status"])
	}
	var standard apihealth.Response
	if err := json.Unmarshal(body, &standard); err != nil {
		t.Fatalf("health payload must decode as api-core health response: %v", err)
	}
	if dep, ok := standard.Dependencies["database"]; !ok || !dep.Connected {
		t.Fatalf("expected standard connected database dependency, got %#v", standard.Dependencies)
	}
	deps, ok := payload["dependencies"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected database dependency, got %#v", deps)
	}
	dbDep, ok := deps["database"].(map[string]interface{})
	if !ok || dbDep["connected"] != true {
		t.Fatalf("expected connected database dependency, got %#v", deps["database"])
	}
	if _, exists := payload["operations"]; exists {
		t.Fatalf("health must not include execution-history operations: %#v", payload["operations"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet mock expectations: %v", err)
	}
}

func TestServer_handleHealthDegradesForCachedSweepFailure(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectPing().WillReturnError(nil)
	status := &selfhealthsnapshots.StatusStore{}
	status.Record(selfhealthsnapshots.SweepStatus{Outcome: "timed_out", Error: "deadline exceeded"})
	server := &Server{config: Config{Port: "0"}, db: database.NewFromPrimary(db), sweepStatus: status, logger: log.New(io.Discard, "", 0)}
	rec := httptest.NewRecorder()
	server.handleHealth(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var payload apihealth.Response
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Status != apihealth.StatusDegraded || !payload.Readiness {
		t.Fatalf("payload = %#v", payload)
	}
	if dep := payload.Dependencies["self_health_sweep"]; dep.Connected {
		t.Fatalf("sweep dependency = %#v", dep)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestServer_handleHealthDoesNotWaitForHeldRuntimeSQLitePool(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test-genie.db")
	dsn, err := sqlitedb.BuildDSN(path)
	if err != nil {
		t.Fatal(err)
	}
	runtimeDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer runtimeDB.Close()
	runtimeDB.SetMaxOpenConns(1)
	healthDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer healthDB.Close()
	conn, err := runtimeDB.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	server := &Server{config: Config{Port: "0"}, db: database.NewFromPrimary(runtimeDB), healthDB: healthDB, logger: log.New(io.Discard, "", 0)}
	start := time.Now()
	rec := httptest.NewRecorder()
	server.handleHealth(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if elapsed := time.Since(start); elapsed > 250*time.Millisecond {
		t.Fatalf("health waited %s on held runtime pool", elapsed)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestServer_handleHealthDatabaseFailure(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer db.Close()

	mock.ExpectPing().WillReturnError(errors.New("offline"))
	history := &fakeExecutionHistory{}

	server := &Server{
		config:           Config{Port: "0"},
		db:               database.NewFromPrimary(db),
		executionHistory: history,
		logger:           log.New(io.Discard, "", 0),
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	server.handleHealth(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when unhealthy, got %d", rec.Code)
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode health payload: %v", err)
	}
	if payload["status"] != "unhealthy" {
		t.Fatalf("expected unhealthy status, got %v", payload["status"])
	}
	deps, ok := payload["dependencies"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected dependencies payload, got %#v", payload["dependencies"])
	}
	dbDep, ok := deps["database"].(map[string]interface{})
	if !ok || dbDep["connected"] != false {
		t.Fatalf("expected disconnected database dependency, got %#v", deps["database"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet mock expectations: %v", err)
	}
}

func TestCheckSQLiteSchemaRejectsUnavailableStore(t *testing.T) {
	db, err := sql.Open("sqlite", "file:/missing/test-genie-health.db?mode=ro")
	if err != nil {
		t.Fatalf("open test sqlite handle: %v", err)
	}
	defer db.Close()
	if err := checkSQLiteSchema(context.Background(), db); err == nil {
		t.Fatal("expected missing read-only database to fail health probe")
	}
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
	runArgs      []string

	scenarioRoot string
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

func (s *stubScenarioDirectory) RunScenarioTests(ctx context.Context, name string, preferred string, extraArgs []string, scenarioDirOverride string) (*scenarios.TestingCommand, *scenarios.TestingRunnerResult, error) {
	s.runName = name
	s.runPreferred = preferred
	if len(extraArgs) > 0 {
		s.runArgs = append([]string(nil), extraArgs...)
	}
	if s.runErr != nil {
		return nil, nil, s.runErr
	}
	return s.runResp, s.runResult, nil
}

func (s *stubScenarioDirectory) ListFiles(ctx context.Context, name string, opts scenarios.FileListOptions) ([]scenarios.FileNode, error) {
	return nil, nil
}

func (s *stubScenarioDirectory) ListFilesWithMeta(ctx context.Context, name string, opts scenarios.FileListOptions) (scenarios.FileListResult, error) {
	return scenarios.FileListResult{}, nil
}

func (s *stubScenarioDirectory) ScenarioRoot() string {
	return s.scenarioRoot
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

func (s *stubSuiteExecutor) ExecuteWithEvents(ctx context.Context, input execution.SuiteExecutionInput, emit orchestrator.ExecutionEventCallback) (*orchestrator.SuiteExecutionResult, error) {
	s.input = input
	return s.result, s.err
}
