package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	pq "github.com/lib/pq"
)

// Config holds minimal runtime configuration
type Config struct {
	Port        string
	DatabaseURL string
}

// Server wires the HTTP router and database connection
type Server struct {
	config *Config
	db     *sql.DB
	router *mux.Router
}

const (
	suiteStatusQueued    = "queued"
	suiteStatusDelegated = "delegated"
	suiteStatusCompleted = "completed"
	suiteStatusFailed    = "failed"

	suitePriorityLow    = "low"
	suitePriorityNormal = "normal"
	suitePriorityHigh   = "high"
	suitePriorityUrgent = "urgent"

	maxSuiteListPage = 50
)

var (
	allowedSuiteTypes = map[string]struct{}{
		"unit":        {},
		"integration": {},
		"performance": {},
		"vault":       {},
		"regression":  {},
	}
	defaultSuiteTypes = []string{"unit", "integration"}
	allowedPriorities = map[string]struct{}{
		suitePriorityLow:    {},
		suitePriorityNormal: {},
		suitePriorityHigh:   {},
		suitePriorityUrgent: {},
	}
)

// SuiteRequest represents a queued generation job that may be delegated to downstream agents.
type SuiteRequest struct {
	ID                 uuid.UUID `json:"id"`
	ScenarioName       string    `json:"scenarioName"`
	RequestedTypes     []string  `json:"requestedTypes"`
	CoverageTarget     int       `json:"coverageTarget"`
	Priority           string    `json:"priority"`
	Status             string    `json:"status"`
	Notes              string    `json:"notes,omitempty"`
	DelegationIssueID  *string   `json:"delegationIssueId,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
	EstimatedQueueTime int       `json:"estimatedQueueTimeSeconds,omitempty"`
}

type suiteRequestPayload struct {
	ScenarioName   string     `json:"scenarioName"`
	RequestedTypes stringList `json:"requestedTypes"`
	CoverageTarget int        `json:"coverageTarget"`
	Priority       string     `json:"priority"`
	Notes          string     `json:"notes"`
}

type stringList []string

func (sl *stringList) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		*sl = nil
		return nil
	}

	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) {
		*sl = nil
		return nil
	}

	var arr []string
	if err := json.Unmarshal(trimmed, &arr); err == nil {
		*sl = arr
		return nil
	}

	var single string
	if err := json.Unmarshal(trimmed, &single); err == nil {
		*sl = []string{single}
		return nil
	}

	return fmt.Errorf("expected string or array of strings for requestedTypes")
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

	srv := &Server{
		config: cfg,
		db:     db,
		router: mux.NewRouter(),
	}

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

	response := map[string]interface{}{
		"status":    status,
		"service":   "Test Genie API",
		"version":   "1.0.0",
		"readiness": status == "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"dependencies": map[string]string{
			"database": dbStatus,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleCreateSuiteRequest(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload suiteRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	req, err := buildSuiteRequest(payload)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := s.insertSuiteRequest(r.Context(), req); err != nil {
		s.log("suite request insert failed", map[string]interface{}{
			"error": err.Error(),
		})
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
	suites, err := s.fetchSuiteRequests(r.Context())
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

	req, err := s.fetchSuiteRequestByID(r.Context(), requestID)
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

func (s *Server) insertSuiteRequest(ctx context.Context, req *SuiteRequest) error {
	const q = `
INSERT INTO suite_requests (
	id, scenario_name, requested_types, coverage_target, priority, status, notes, delegation_issue_id
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING created_at, updated_at
`
	var note sql.NullString
	if req.Notes != "" {
		note = sql.NullString{String: req.Notes, Valid: true}
	}

	var delegation sql.NullString
	if req.DelegationIssueID != nil && *req.DelegationIssueID != "" {
		delegation = sql.NullString{String: *req.DelegationIssueID, Valid: true}
	}

	return s.db.QueryRowContext(
		ctx,
		q,
		req.ID,
		req.ScenarioName,
		pq.Array(req.RequestedTypes),
		req.CoverageTarget,
		req.Priority,
		req.Status,
		note,
		delegation,
	).Scan(&req.CreatedAt, &req.UpdatedAt)
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanSuiteRequest(scanner rowScanner) (SuiteRequest, error) {
	var req SuiteRequest
	var rawTypes pq.StringArray
	var note sql.NullString
	var delegation sql.NullString

	if err := scanner.Scan(
		&req.ID,
		&req.ScenarioName,
		&rawTypes,
		&req.CoverageTarget,
		&req.Priority,
		&req.Status,
		&note,
		&delegation,
		&req.CreatedAt,
		&req.UpdatedAt,
	); err != nil {
		return req, err
	}

	req.RequestedTypes = append([]string(nil), rawTypes...)
	if note.Valid {
		req.Notes = note.String
	}
	if delegation.Valid {
		req.DelegationIssueID = strPtr(delegation.String)
	}
	req.EstimatedQueueTime = estimateQueueSeconds(len(req.RequestedTypes), req.CoverageTarget)
	return req, nil
}

func (s *Server) fetchSuiteRequests(ctx context.Context) ([]SuiteRequest, error) {
	const q = `
SELECT
	id,
	scenario_name,
	requested_types,
	coverage_target,
	priority,
	status,
	notes,
	delegation_issue_id,
	created_at,
	updated_at
FROM suite_requests
ORDER BY created_at DESC
LIMIT $1
`
	rows, err := s.db.QueryContext(ctx, q, maxSuiteListPage)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suites []SuiteRequest
	for rows.Next() {
		req, err := scanSuiteRequest(rows)
		if err != nil {
			return nil, err
		}
		suites = append(suites, req)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return suites, nil
}

func (s *Server) fetchSuiteRequestByID(ctx context.Context, id uuid.UUID) (*SuiteRequest, error) {
	const q = `
SELECT
	id,
	scenario_name,
	requested_types,
	coverage_target,
	priority,
	status,
	notes,
	delegation_issue_id,
	created_at,
	updated_at
FROM suite_requests
WHERE id = $1
`
	row := s.db.QueryRowContext(ctx, q, id)
	req, err := scanSuiteRequest(row)
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func buildSuiteRequest(payload suiteRequestPayload) (*SuiteRequest, error) {
	scenario := strings.TrimSpace(payload.ScenarioName)
	if scenario == "" {
		return nil, fmt.Errorf("scenarioName is required")
	}

	types, err := normalizeSuiteTypes([]string(payload.RequestedTypes))
	if err != nil {
		return nil, err
	}

	coverage := payload.CoverageTarget
	if coverage == 0 {
		coverage = 95
	}
	if coverage < 1 || coverage > 100 {
		return nil, fmt.Errorf("coverageTarget must be between 1 and 100")
	}

	priority, err := normalizePriority(payload.Priority)
	if err != nil {
		return nil, err
	}

	notes := strings.TrimSpace(payload.Notes)

	req := &SuiteRequest{
		ID:             uuid.New(),
		ScenarioName:   scenario,
		RequestedTypes: types,
		CoverageTarget: coverage,
		Priority:       priority,
		Status:         suiteStatusQueued,
		Notes:          notes,
	}
	req.EstimatedQueueTime = estimateQueueSeconds(len(req.RequestedTypes), req.CoverageTarget)
	return req, nil
}

func normalizeSuiteTypes(values []string) ([]string, error) {
	if len(values) == 0 {
		return append([]string(nil), defaultSuiteTypes...), nil
	}

	seen := make(map[string]struct{}, len(values))
	var sanitized []string
	for _, value := range values {
		trimmed := strings.ToLower(strings.TrimSpace(value))
		if trimmed == "" {
			continue
		}
		if _, ok := allowedSuiteTypes[trimmed]; !ok {
			return nil, fmt.Errorf("requested type '%s' is not supported", value)
		}
		if _, dup := seen[trimmed]; dup {
			continue
		}
		seen[trimmed] = struct{}{}
		sanitized = append(sanitized, trimmed)
	}

	if len(sanitized) == 0 {
		return append([]string(nil), defaultSuiteTypes...), nil
	}

	return sanitized, nil
}

func normalizePriority(value string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return suitePriorityNormal, nil
	}
	if _, ok := allowedPriorities[trimmed]; !ok {
		return "", fmt.Errorf("priority '%s' is not supported", value)
	}
	return trimmed, nil
}

func estimateQueueSeconds(typeCount, coverage int) int {
	if typeCount <= 0 {
		typeCount = len(defaultSuiteTypes)
	}
	if coverage <= 0 {
		coverage = 95
	}
	// Deterministic heuristic: weight coverage target plus workload per type
	return (typeCount * 30) + coverage
}

func strPtr(value string) *string {
	v := value
	return &v
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
