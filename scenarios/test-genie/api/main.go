package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// Config holds minimal runtime configuration
type Config struct {
	Port        string
	DatabaseURL string
}

// Server wires the HTTP router and database connection
type Server struct {
	config        *Config
	db            *sql.DB
	router        *mux.Router
	suiteRequests *SuiteRequestService
	executions    *SuiteExecutionRepository
	orchestrator  *SuiteOrchestrator
	executionSvc  *SuiteExecutionService
}

type suiteExecutionPayload struct {
	ScenarioName   string   `json:"scenarioName"`
	SuiteRequestID string   `json:"suiteRequestId"`
	Preset         string   `json:"preset"`
	Phases         []string `json:"phases"`
	Skip           []string `json:"skip"`
	FailFast       bool     `json:"failFast"`
}

// NewServer initializes configuration, database, and routes
func NewServer() (*Server, error) {
	dbURL, err := resolveDatabaseURL()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Port:        requireEnv("API_PORT"),
		DatabaseURL: dbURL,
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	if err := ensureDatabaseSchema(db); err != nil {
		return nil, fmt.Errorf("failed to apply database schema: %w", err)
	}

	scenariosRoot, err := resolveScenariosRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve scenarios root: %w", err)
	}
	orchestrator, err := NewSuiteOrchestrator(scenariosRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize orchestrator: %w", err)
	}

	srv := &Server{
		config:        cfg,
		db:            db,
		router:        mux.NewRouter(),
		suiteRequests: NewSuiteRequestService(NewPostgresSuiteRequestRepository(db)),
		executions:    NewSuiteExecutionRepository(db),
		orchestrator:  orchestrator,
	}
	srv.executionSvc = NewSuiteExecutionService(srv.orchestrator, srv.executions, srv.suiteRequests)

	srv.setupRoutes()
	return srv, nil
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	// Health endpoint at root for infrastructure agents
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

	apiRouter := s.router.PathPrefix("/api/v1").Subrouter()
	apiRouter.HandleFunc("/health", s.handleHealth).Methods("GET")
	apiRouter.HandleFunc("/suite-requests", s.handleCreateSuiteRequest).Methods("POST")
	apiRouter.HandleFunc("/suite-requests", s.handleListSuiteRequests).Methods("GET")
	apiRouter.HandleFunc("/suite-requests/{id}", s.handleGetSuiteRequest).Methods("GET")
	apiRouter.HandleFunc("/executions", s.handleExecuteSuite).Methods("POST")
	apiRouter.HandleFunc("/executions", s.handleListExecutions).Methods("GET")
	apiRouter.HandleFunc("/executions/{id}", s.handleGetExecution).Methods("GET")
}

// Start launches the HTTP server with graceful shutdown
func (s *Server) Start() error {
	s.log("starting server", map[string]interface{}{
		"service": "test-genie-api",
		"port":    s.config.Port,
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", s.config.Port),
		Handler:      handlers.RecoveryHandler()(s.router),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log("server startup failed", map[string]interface{}{"error": err.Error()})
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

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
	if s.executions != nil {
		if latest, err := s.executions.Latest(r.Context()); err == nil && latest != nil {
			operations["lastExecution"] = executionSummaryPayload(*latest)
		} else if err != nil && err != sql.ErrNoRows {
			s.log("latest execution lookup failed", map[string]interface{}{"error": err.Error()})
		}
	}

	response := map[string]interface{}{
		"status":    status,
		"service":   "Test Genie API",
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
	var payload QueueSuiteRequestInput
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	req, err := s.suiteRequests.Queue(r.Context(), payload)
	if err != nil {
		var vErr validationError
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
	suites, err := s.suiteRequests.List(r.Context(), maxSuiteListPage)
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

func (s *Server) handleListExecutions(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	scenario := strings.TrimSpace(params.Get("scenario"))
	limit := maxExecutionHistory
	if raw := strings.TrimSpace(params.Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	records, err := s.executions.ListRecent(r.Context(), scenario, limit)
	if err != nil {
		s.log("listing executions failed", map[string]interface{}{"error": err.Error()})
		s.writeError(w, http.StatusInternalServerError, "failed to load execution history")
		return
	}

	items := make([]SuiteExecutionResult, 0, len(records))
	for i := range records {
		result := executionRecordToResult(&records[i])
		items = append(items, *result)
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
		"count": len(items),
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

	record, err := s.executions.GetByID(r.Context(), executionID)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeError(w, http.StatusNotFound, "execution not found")
			return
		}
		s.log("fetching execution failed", map[string]interface{}{"error": err.Error()})
		s.writeError(w, http.StatusInternalServerError, "failed to load execution")
		return
	}

	s.writeJSON(w, http.StatusOK, executionRecordToResult(record))
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

	execRequest := SuiteExecutionRequest{
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

	result, err := s.executionSvc.Execute(r.Context(), SuiteExecutionInput{
		Request:        execRequest,
		SuiteRequestID: suiteRequestID,
	})
	if err != nil {
		if errors.Is(err, errSuiteRequestNotFound) {
			s.writeError(w, http.StatusNotFound, "suite request not found")
			return
		}
		var vErr validationError
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

// loggingMiddleware prints simple request logs
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

func (s *Server) log(msg string, fields map[string]interface{}) {
	if len(fields) == 0 {
		log.Println(msg)
		return
	}
	log.Printf("%s | %v", msg, fields)
}

func queueSnapshotPayload(snapshot SuiteRequestSnapshot) map[string]interface{} {
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

func executionSummaryPayload(record SuiteExecutionRecord) map[string]interface{} {
	result := executionRecordToResult(&record)
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

func executionRecordToResult(record *SuiteExecutionRecord) *SuiteExecutionResult {
	if record == nil {
		return nil
	}
	var phases []PhaseExecutionResult
	if len(record.Phases) > 0 {
		phases = append(phases, record.Phases...)
	}
	res := &SuiteExecutionResult{
		ExecutionID:    record.ID,
		SuiteRequestID: record.SuiteRequestID,
		ScenarioName:   record.ScenarioName,
		StartedAt:      record.StartedAt,
		CompletedAt:    record.CompletedAt,
		Success:        record.Success,
		PresetUsed:     record.PresetUsed,
		Phases:         phases,
	}
	res.PhaseSummary = summarizePhases(res.Phases)
	return res
}

func ensureDatabaseSchema(db *sql.DB) error {
	schemaPath, err := resolveInitializationFile("schema.sql")
	if err != nil {
		return fmt.Errorf("initialization schema lookup failed: %w", err)
	}
	if err := execSQLFile(db, schemaPath); err != nil {
		return err
	}
	if seedPath, err := resolveInitializationFile("seed.sql"); err == nil {
		if err := execSQLFile(db, seedPath); err != nil {
			return err
		}
	}
	return nil
}

func resolveInitializationFile(name string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to determine working directory: %w", err)
	}
	scenarioDir := filepath.Dir(wd)
	target := filepath.Join(scenarioDir, "initialization", "postgres", name)
	if _, err := os.Stat(target); err != nil {
		return "", fmt.Errorf("initialization file not accessible (%s): %w", target, err)
	}
	return target, nil
}

func execSQLFile(db *sql.DB, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	var builder strings.Builder
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to scan SQL file %s: %w", path, err)
	}

	statements := strings.Split(builder.String(), ";")
	for _, stmt := range statements {
		trimmed := strings.TrimSpace(stmt)
		if trimmed == "" {
			continue
		}
		if _, err := db.Exec(trimmed); err != nil {
			return fmt.Errorf("failed to execute SQL statement from %s: %w", path, err)
		}
	}
	return nil
}

func requireEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		log.Fatalf("environment variable %s is required. Run the scenario via 'vrooli scenario run <name>' so lifecycle exports it.", key)
	}
	return value
}

func resolveDatabaseURL() (string, error) {
	if raw := strings.TrimSpace(os.Getenv("DATABASE_URL")); raw != "" {
		return raw, nil
	}

	host := strings.TrimSpace(os.Getenv("POSTGRES_HOST"))
	port := strings.TrimSpace(os.Getenv("POSTGRES_PORT"))
	user := strings.TrimSpace(os.Getenv("POSTGRES_USER"))
	password := strings.TrimSpace(os.Getenv("POSTGRES_PASSWORD"))
	name := strings.TrimSpace(os.Getenv("POSTGRES_DB"))

	if host == "" || port == "" || user == "" || password == "" || name == "" {
		return "", fmt.Errorf("DATABASE_URL or POSTGRES_HOST/PORT/USER/PASSWORD/DB must be set by the lifecycle system")
	}

	pgURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   name,
	}
	values := pgURL.Query()
	values.Set("sslmode", "disable")
	pgURL.RawQuery = values.Encode()

	return pgURL.String(), nil
}

func resolveScenariosRoot() (string, error) {
	if raw := strings.TrimSpace(os.Getenv("SCENARIOS_ROOT")); raw != "" {
		return filepath.Abs(raw)
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to determine working directory: %w", err)
	}
	scenarioDir := filepath.Dir(wd)
	root := filepath.Dir(scenarioDir)
	return root, nil
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start test-genie

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	server, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
