package httpserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/scenarios"
	"test-genie/internal/shared"
	"test-genie/internal/suite"
)

// Config controls the HTTP transport settings.
type Config struct {
	Port        string
	ServiceName string
}

// Logger captures the logging surface the HTTP layer relies on.
type Logger interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
}

// Dependencies encapsulates the services the HTTP layer needs to operate.
type Dependencies struct {
	DB           *sql.DB
	SuiteQueue   suiteRequestQueue
	Executions   suite.ExecutionHistory
	ExecutionSvc suiteExecutor
	Scenarios    scenarioDirectory
	PhaseCatalog phaseCatalog
	Logger       Logger
}

type suiteRequestQueue interface {
	Queue(ctx context.Context, payload suite.QueueSuiteRequestInput) (*suite.SuiteRequest, error)
	List(ctx context.Context, limit int) ([]suite.SuiteRequest, error)
	Get(ctx context.Context, id uuid.UUID) (*suite.SuiteRequest, error)
	StatusSnapshot(ctx context.Context) (suite.SuiteRequestSnapshot, error)
}

type suiteExecutor interface {
	Execute(ctx context.Context, input suite.SuiteExecutionInput) (*orchestrator.SuiteExecutionResult, error)
}

type scenarioDirectory interface {
	ListSummaries(ctx context.Context) ([]scenarios.ScenarioSummary, error)
	GetSummary(ctx context.Context, name string) (*scenarios.ScenarioSummary, error)
	RunScenarioTests(ctx context.Context, name string, preferred string) (*scenarios.TestingCommand, *scenarios.TestingRunnerResult, error)
}

type phaseCatalog interface {
	DescribePhases() []phases.Descriptor
}

// Server wires the HTTP router, configuration, and service dependencies behind intentional seams.
type Server struct {
	config           Config
	db               *sql.DB
	router           *mux.Router
	suiteRequests    suiteRequestQueue
	executionHistory suite.ExecutionHistory
	executionSvc     suiteExecutor
	scenarios        scenarioDirectory
	phaseCatalog     phaseCatalog
	logger           Logger
}

type suiteExecutionPayload struct {
	ScenarioName   string   `json:"scenarioName"`
	SuiteRequestID string   `json:"suiteRequestId"`
	Preset         string   `json:"preset"`
	Phases         []string `json:"phases"`
	Skip           []string `json:"skip"`
	FailFast       bool     `json:"failFast"`
}

// New creates a configured HTTP server instance.
func New(config Config, deps Dependencies) (*Server, error) {
	if strings.TrimSpace(config.Port) == "" {
		return nil, fmt.Errorf("api port is required")
	}
	if deps.DB == nil {
		return nil, fmt.Errorf("database dependency is required")
	}
	if deps.SuiteQueue == nil {
		return nil, fmt.Errorf("suite request service is required")
	}
	if deps.Executions == nil {
		return nil, fmt.Errorf("execution history service is required")
	}
	if deps.ExecutionSvc == nil {
		return nil, fmt.Errorf("execution service is required")
	}
	if deps.Scenarios == nil {
		return nil, fmt.Errorf("scenario directory service is required")
	}
	if deps.PhaseCatalog == nil {
		return nil, fmt.Errorf("phase catalog dependency is required")
	}

	logger := deps.Logger
	if logger == nil {
		logger = log.Default()
	}

	srv := &Server{
		config:           config,
		db:               deps.DB,
		router:           mux.NewRouter(),
		suiteRequests:    deps.SuiteQueue,
		executionHistory: deps.Executions,
		executionSvc:     deps.ExecutionSvc,
		scenarios:        deps.Scenarios,
		phaseCatalog:     deps.PhaseCatalog,
		logger:           logger,
	}

	srv.setupRoutes()
	return srv, nil
}

func (s *Server) setupRoutes() {
	s.router.Use(s.loggingMiddleware)
	// Health endpoint at root for infrastructure agents
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

	apiRouter := s.router.PathPrefix("/api/v1").Subrouter()
	apiRouter.HandleFunc("/health", s.handleHealth).Methods("GET")
	apiRouter.HandleFunc("/suite-requests", s.handleCreateSuiteRequest).Methods("POST")
	apiRouter.HandleFunc("/suite-requests", s.handleListSuiteRequests).Methods("GET")
	apiRouter.HandleFunc("/suite-requests/{id}", s.handleGetSuiteRequest).Methods("GET")
	apiRouter.HandleFunc("/phases", s.handleListPhases).Methods("GET")
	apiRouter.HandleFunc("/executions", s.handleExecuteSuite).Methods("POST")
	apiRouter.HandleFunc("/executions", s.handleListExecutions).Methods("GET")
	apiRouter.HandleFunc("/executions/{id}", s.handleGetExecution).Methods("GET")
	apiRouter.HandleFunc("/scenarios", s.handleListScenarios).Methods("GET")
	apiRouter.HandleFunc("/scenarios/{name}", s.handleGetScenario).Methods("GET")
	apiRouter.HandleFunc("/scenarios/{name}/run-tests", s.handleRunScenarioTests).Methods("POST")
}

// Start launches the HTTP server with graceful shutdown
func (s *Server) Start() error {
	s.log("starting server", map[string]interface{}{
		"service": s.serviceName(),
		"port":    s.config.Port,
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", s.config.Port),
		Handler:      handlers.RecoveryHandler()(s.router),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
	case err := <-errCh:
		return fmt.Errorf("server startup failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	s.log("server stopped", nil)
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	dbStatus := "connected"

	if err := s.db.PingContext(r.Context()); err != nil {
		status = "unhealthy"
		dbStatus = "disconnected"
	}

	operations := map[string]interface{}{}
	if s.suiteRequests != nil {
		if snapshot, err := s.suiteRequests.StatusSnapshot(r.Context()); err == nil {
			operations["queue"] = queueSnapshotPayload(snapshot)
		} else if err != nil {
			s.log("queue snapshot failed", map[string]interface{}{"error": err.Error()})
		}
	}
	if s.executionHistory != nil {
		if latest, err := s.executionHistory.Latest(r.Context()); err == nil && latest != nil {
			operations["lastExecution"] = executionSummaryPayload(latest)
		} else if err != nil {
			s.log("latest execution lookup failed", map[string]interface{}{"error": err.Error()})
		}
	}

	response := map[string]interface{}{
		"status":    status,
		"service":   s.serviceName(),
		"version":   "1.0.0",
		"readiness": status == "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"dependencies": map[string]string{
			"database": dbStatus,
		},
		"operations": operations,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleCreateSuiteRequest(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload suite.QueueSuiteRequestInput
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	req, err := s.suiteRequests.Queue(r.Context(), payload)
	if err != nil {
		var vErr shared.ValidationError
		if errors.As(err, &vErr) {
			s.writeError(w, http.StatusBadRequest, vErr.Error())
			return
		}
		s.log("suite request insert failed", map[string]interface{}{"error": err.Error()})
		s.writeError(w, http.StatusInternalServerError, "failed to persist suite request")
		return
	}

	s.log("suite request created", map[string]interface{}{
		"id":       req.ID.String(),
		"scenario": req.ScenarioName,
		"types":    req.RequestedTypes,
	})
	s.writeJSON(w, http.StatusCreated, req)
}

func (s *Server) handleListSuiteRequests(w http.ResponseWriter, r *http.Request) {
	suites, err := s.suiteRequests.List(r.Context(), suite.MaxSuiteListPage)
	if err != nil {
		s.log("listing suite requests failed", map[string]interface{}{"error": err.Error()})
		s.writeError(w, http.StatusInternalServerError, "failed to load suite requests")
		return
	}

	payload := map[string]interface{}{
		"items": suites,
		"count": len(suites),
	}
	s.writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleGetSuiteRequest(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	rawID := strings.TrimSpace(params["id"])
	if rawID == "" {
		s.writeError(w, http.StatusBadRequest, "suite request id is required")
		return
	}

	requestID, err := uuid.Parse(rawID)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "suite request id must be a valid UUID")
		return
	}

	req, err := s.suiteRequests.Get(r.Context(), requestID)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusNotFound, "suite request not found")
			return
		}
		s.log("fetching suite request by id failed", map[string]interface{}{"error": err.Error()})
		s.writeError(w, http.StatusInternalServerError, "failed to load requested suite")
		return
	}

	s.writeJSON(w, http.StatusOK, req)
}

func (s *Server) handleListScenarios(w http.ResponseWriter, r *http.Request) {
	if s.scenarios == nil {
		s.writeError(w, http.StatusInternalServerError, "scenario directory service unavailable")
		return
	}
	summaries, err := s.scenarios.ListSummaries(r.Context())
	if err != nil {
		s.log("listing scenarios failed", map[string]interface{}{"error": err.Error()})
		s.writeError(w, http.StatusInternalServerError, "failed to load scenarios")
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": summaries,
		"count": len(summaries),
	})
}

func (s *Server) handleGetScenario(w http.ResponseWriter, r *http.Request) {
	if s.scenarios == nil {
		s.writeError(w, http.StatusInternalServerError, "scenario directory service unavailable")
		return
	}
	params := mux.Vars(r)
	name := strings.TrimSpace(params["name"])
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "scenario name is required")
		return
	}

	summary, err := s.scenarios.GetSummary(r.Context(), name)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusNotFound, "scenario not found")
			return
		}
		s.log("fetching scenario failed", map[string]interface{}{"error": err.Error()})
		s.writeError(w, http.StatusInternalServerError, "failed to load scenario")
		return
	}

	s.writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleRunScenarioTests(w http.ResponseWriter, r *http.Request) {
	if s.scenarios == nil {
		s.writeError(w, http.StatusInternalServerError, "scenario directory service unavailable")
		return
	}
	params := mux.Vars(r)
	name := strings.TrimSpace(params["name"])
	if name == "" {
		s.writeError(w, http.StatusBadRequest, "scenario name is required")
		return
	}
	var payload struct {
		Type string `json:"type"`
	}
	if r.Body != nil {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
			s.writeError(w, http.StatusBadRequest, "invalid JSON payload")
			return
		}
	}

	cmd, result, err := s.scenarios.RunScenarioTests(r.Context(), name, payload.Type)
	if err != nil {
		switch {
		case errors.Is(err, os.ErrNotExist):
			s.writeError(w, http.StatusNotFound, "scenario not found")
		case shared.IsValidationError(err):
			s.writeError(w, http.StatusBadRequest, err.Error())
		default:
			s.log("scenario test execution failed", map[string]interface{}{
				"error":    err.Error(),
				"scenario": name,
			})
			s.writeError(w, http.StatusInternalServerError, "scenario tests failed")
		}
		return
	}

	response := map[string]interface{}{
		"status":  "completed",
		"command": cmd,
		"type":    cmd.Type,
	}
	if result != nil && strings.TrimSpace(result.LogPath) != "" {
		response["logPath"] = result.LogPath
	}
	s.writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleListPhases(w http.ResponseWriter, r *http.Request) {
	if s.phaseCatalog == nil {
		s.writeError(w, http.StatusInternalServerError, "phase catalog unavailable")
		return
	}
	descriptors := s.phaseCatalog.DescribePhases()
	payload := map[string]interface{}{
		"items": descriptors,
		"count": len(descriptors),
	}
	s.writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleListExecutions(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	scenario := strings.TrimSpace(params.Get("scenario"))
	limit := orchestrator.MaxExecutionHistory
	if raw := strings.TrimSpace(params.Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	offset := 0
	if raw := strings.TrimSpace(params.Get("offset")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	executions, err := s.executionHistory.List(r.Context(), scenario, limit, offset)
	if err != nil {
		s.log("listing executions failed", map[string]interface{}{"error": err.Error()})
		s.writeError(w, http.StatusInternalServerError, "failed to load execution history")
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": executions,
		"count": len(executions),
	})
}

func (s *Server) handleGetExecution(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	rawID := strings.TrimSpace(params["id"])
	if rawID == "" {
		s.writeError(w, http.StatusBadRequest, "execution id is required")
		return
	}
	executionID, err := uuid.Parse(rawID)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "execution id must be a valid UUID")
		return
	}

	result, err := s.executionHistory.Get(r.Context(), executionID)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusNotFound, "execution not found")
			return
		}
		s.log("fetching execution failed", map[string]interface{}{"error": err.Error()})
		s.writeError(w, http.StatusInternalServerError, "failed to load execution")
		return
	}

	s.writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleExecuteSuite(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload suiteExecutionPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	scenario := strings.TrimSpace(payload.ScenarioName)
	if scenario == "" {
		s.writeError(w, http.StatusBadRequest, "scenarioName is required")
		return
	}

	execRequest := orchestrator.SuiteExecutionRequest{
		ScenarioName: scenario,
		Preset:       strings.TrimSpace(payload.Preset),
		Phases:       payload.Phases,
		Skip:         payload.Skip,
		FailFast:     payload.FailFast,
	}

	var suiteRequestID *uuid.UUID
	if id := strings.TrimSpace(payload.SuiteRequestID); id != "" {
		parsed, err := uuid.Parse(id)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, "suiteRequestId must be a valid UUID")
			return
		}
		suiteRequestID = &parsed
	}

	if s.executionSvc == nil {
		s.writeError(w, http.StatusInternalServerError, "execution service unavailable")
		return
	}

	result, err := s.executionSvc.Execute(r.Context(), suite.SuiteExecutionInput{
		Request:        execRequest,
		SuiteRequestID: suiteRequestID,
	})
	if err != nil {
		if errors.Is(err, suite.ErrSuiteRequestNotFound) {
			s.writeError(w, http.StatusNotFound, "suite request not found")
			return
		}
		var vErr shared.ValidationError
		if errors.As(err, &vErr) {
			s.writeError(w, http.StatusBadRequest, vErr.Error())
			return
		}
		s.log("suite execution failed", map[string]interface{}{"error": err.Error()})
		s.writeError(w, http.StatusInternalServerError, "suite execution failed")
		return
	}

	s.writeJSON(w, http.StatusOK, result)
}

func (s *Server) writeJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		s.log("failed to encode JSON response", map[string]interface{}{
			"error": err.Error(),
		})
	}
}

func (s *Server) writeError(w http.ResponseWriter, statusCode int, message string) {
	s.writeJSON(w, statusCode, map[string]string{"error": message})
}

// loggingMiddleware prints simple request logs.
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

func (s *Server) log(msg string, fields map[string]interface{}) {
	if s.logger == nil {
		return
	}
	if len(fields) == 0 {
		s.logger.Print(msg)
		return
	}
	s.logger.Printf("%s | %v", msg, fields)
}

func queueSnapshotPayload(snapshot suite.SuiteRequestSnapshot) map[string]interface{} {
	payload := map[string]interface{}{
		"total":     snapshot.Total,
		"queued":    snapshot.Queued,
		"delegated": snapshot.Delegated,
		"running":   snapshot.Running,
		"completed": snapshot.Completed,
		"failed":    snapshot.Failed,
		"pending":   snapshot.Queued + snapshot.Delegated,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	if snapshot.OldestQueuedAt != nil {
		payload["oldestQueuedAt"] = snapshot.OldestQueuedAt.Format(time.RFC3339)
		age := time.Since(*snapshot.OldestQueuedAt).Seconds()
		if age < 0 {
			age = 0
		}
		payload["oldestQueuedAgeSeconds"] = int(age)
	}
	return payload
}

func executionSummaryPayload(result *orchestrator.SuiteExecutionResult) map[string]interface{} {
	if result == nil {
		return nil
	}
	return map[string]interface{}{
		"executionId":  result.ExecutionID,
		"scenario":     result.ScenarioName,
		"success":      result.Success,
		"completedAt":  result.CompletedAt.Format(time.RFC3339),
		"startedAt":    result.StartedAt.Format(time.RFC3339),
		"phaseSummary": result.PhaseSummary,
		"preset":       result.PresetUsed,
	}
}

func (s *Server) serviceName() string {
	name := strings.TrimSpace(s.config.ServiceName)
	if name == "" {
		return "Test Genie API"
	}
	return name
}
