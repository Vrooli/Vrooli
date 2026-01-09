package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestEnv holds test environment resources
type TestEnv struct {
	Server  *Server
	DB      *sql.DB
	Cleanup func()
}

// setupTestLogger sets up a test logger that can be silenced
func setupTestLogger() func() {
	// Save original output
	originalOutput := log.Writer()

	// Redirect to discard during tests (comment out to see logs)
	log.SetOutput(io.Discard)

	// Return cleanup function
	return func() {
		log.SetOutput(originalOutput)
	}
}

// setupTestDirectory creates an isolated test environment
func setupTestDirectory(t *testing.T) *TestEnv {
	t.Helper()

	// Use test database URL
	testDBURL := getEnv("TEST_DATABASE_URL", "postgres://vrooli:lUq9qvemypKpuEeXCV6Vnxak1@localhost:5433/vrooli?sslmode=disable")

	// Connect to database
	db, err := sql.Open("postgres", testDBURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	// Create test server config
	config := &Config{
		Port:        "18000",
		DatabaseURL: testDBURL,
		N8NURL:      "http://localhost:5678",
		APIToken:    "test-token",
	}

	server := &Server{
		config: config,
		db:     db,
	}
	server.router = server.setupTestRoutes()

	cleanup := func() {
		// Clean up test data
		db.Exec("TRUNCATE TABLE datasets CASCADE")
		db.Exec("TRUNCATE TABLE data_transformations CASCADE")
		db.Exec("TRUNCATE TABLE data_quality_reports CASCADE")
		db.Exec("TRUNCATE TABLE streaming_sources CASCADE")
		db.Exec("TRUNCATE TABLE resources CASCADE")
		db.Exec("TRUNCATE TABLE executions CASCADE")
		db.Close()
	}

	return &TestEnv{
		Server:  server,
		DB:      db,
		Cleanup: cleanup,
	}
}

// setupTestRoutes creates a test router (needed to avoid panic)
func (s *Server) setupTestRoutes() *mux.Router {
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", s.handleHealth).Methods("GET", "OPTIONS")

	// Data processing endpoints
	router.HandleFunc("/api/v1/data/parse", s.wrapWithAuth(s.handleDataParse)).Methods("POST")
	router.HandleFunc("/api/v1/data/transform", s.wrapWithAuth(s.handleDataTransform)).Methods("POST")
	router.HandleFunc("/api/v1/data/validate", s.wrapWithAuth(s.handleDataValidate)).Methods("POST")
	router.HandleFunc("/api/v1/data/query", s.wrapWithAuth(s.handleDataQuery)).Methods("POST")
	router.HandleFunc("/api/v1/data/stream/create", s.wrapWithAuth(s.handleStreamCreate)).Methods("POST")
	router.HandleFunc("/api/v1/data/profile", s.wrapWithAuth(s.handleDataProfile)).Methods("POST")

	// Resource endpoints - match production REST patterns
	router.HandleFunc("/api/v1/resources", s.wrapWithAuth(s.handleListResources)).Methods("GET")
	router.HandleFunc("/api/v1/resources", s.wrapWithAuth(s.handleCreateResource)).Methods("POST")
	router.HandleFunc("/api/v1/resources/{id}", s.wrapWithAuth(s.handleGetResource)).Methods("GET")
	router.HandleFunc("/api/v1/resources/{id}", s.wrapWithAuth(s.handleUpdateResource)).Methods("PUT")
	router.HandleFunc("/api/v1/resources/{id}", s.wrapWithAuth(s.handleDeleteResource)).Methods("DELETE")

	// Workflow endpoints
	router.HandleFunc("/api/v1/execute", s.wrapWithAuth(s.handleExecuteWorkflow)).Methods("POST")
	router.HandleFunc("/api/v1/executions", s.wrapWithAuth(s.handleListExecutions)).Methods("GET")
	router.HandleFunc("/api/v1/executions/get", s.wrapWithAuth(s.handleGetExecutionTest)).Methods("GET")

	// Docs
	router.HandleFunc("/docs", s.handleDocs).Methods("GET")

	return router
}

// wrapWithAuth adds auth check to handlers for testing
func (s *Server) wrapWithAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check authorization header
		token := r.Header.Get("Authorization")
		if token != "" && token != "Bearer "+s.config.APIToken {
			s.sendError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		handler(w, r)
	}
}

// Test-specific handlers to avoid mux variable extraction issues
func (s *Server) handleListResourcesTest(w http.ResponseWriter, r *http.Request) {
	s.handleListResources(w, r)
}

func (s *Server) handleCreateResourceTest(w http.ResponseWriter, r *http.Request) {
	s.handleCreateResource(w, r)
}

func (s *Server) handleGetResourceTest(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		s.sendError(w, http.StatusBadRequest, "id parameter required")
		return
	}

	query := `SELECT id, name, description, config, created_at FROM resources WHERE id = $1`
	row := s.db.QueryRow(query, id)

	var name, description string
	var config json.RawMessage
	var createdAt time.Time

	err := row.Scan(&id, &name, &description, &config, &createdAt)
	if err == sql.ErrNoRows {
		s.sendError(w, http.StatusNotFound, "resource not found")
		return
	}
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to query resource")
		return
	}

	resource := map[string]interface{}{
		"id":          id,
		"name":        name,
		"description": description,
		"config":      config,
		"created_at":  createdAt,
	}

	s.sendJSON(w, http.StatusOK, resource)
}

func (s *Server) handleUpdateResourceTest(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		s.sendError(w, http.StatusBadRequest, "id parameter required")
		return
	}

	var input map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.sendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	query := `UPDATE resources SET name = $2, description = $3, config = $4, updated_at = $5 WHERE id = $1`
	result, err := s.db.Exec(query, id, input["name"], input["description"], input["config"], time.Now())

	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to update resource")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		s.sendError(w, http.StatusNotFound, "resource not found")
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{"id": id, "updated_at": time.Now()})
}

func (s *Server) handleDeleteResourceTest(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		s.sendError(w, http.StatusBadRequest, "id parameter required")
		return
	}

	query := `DELETE FROM resources WHERE id = $1`
	result, err := s.db.Exec(query, id)

	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to delete resource")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		s.sendError(w, http.StatusNotFound, "resource not found")
		return
	}

	s.sendJSON(w, http.StatusOK, map[string]interface{}{"deleted": true, "id": id})
}

func (s *Server) handleGetExecutionTest(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		s.sendError(w, http.StatusBadRequest, "id parameter required")
		return
	}

	query := `SELECT id, workflow_id, status, input_data, output_data, error_message, started_at, completed_at
	          FROM executions WHERE id = $1`

	row := s.db.QueryRow(query, id)

	var workflowID, status string
	var inputData, outputData json.RawMessage
	var errorMessage sql.NullString
	var startedAt time.Time
	var completedAt sql.NullTime

	err := row.Scan(&id, &workflowID, &status, &inputData, &outputData, &errorMessage, &startedAt, &completedAt)

	if err == sql.ErrNoRows {
		s.sendError(w, http.StatusNotFound, "execution not found")
		return
	}
	if err != nil {
		s.sendError(w, http.StatusInternalServerError, "failed to query execution")
		return
	}

	execution := map[string]interface{}{
		"id":          id,
		"workflow_id": workflowID,
		"status":      status,
		"input_data":  inputData,
		"output_data": outputData,
		"started_at":  startedAt,
	}

	if errorMessage.Valid {
		execution["error_message"] = errorMessage.String
	}
	if completedAt.Valid {
		execution["completed_at"] = completedAt.Time
	}

	s.sendJSON(w, http.StatusOK, execution)
}

// makeHTTPRequest creates an HTTP request with proper headers
func makeHTTPRequest(method, path string, body interface{}, token string) *http.Request {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return req
}

// assertJSONResponse validates JSON response structure
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
	}

	return response
}

// assertErrorResponse validates error response structure
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedError string) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	var response Response
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	if response.Success {
		t.Error("Expected success=false in error response")
	}

	if expectedError != "" && response.Error != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, response.Error)
	}
}

// createTestResource creates a test resource in the database
func createTestResource(t *testing.T, db *sql.DB) string {
	t.Helper()

	id := "test-resource-" + time.Now().Format("20060102150405")
	query := `INSERT INTO resources (id, name, description, config, created_at) VALUES ($1, $2, $3, $4, $5)`

	_, err := db.Exec(query, id, "Test Resource", "Test Description", `{"key": "value"}`, time.Now())
	if err != nil {
		t.Fatalf("Failed to create test resource: %v", err)
	}

	return id
}

// createTestExecution creates a test execution in the database
func createTestExecution(t *testing.T, db *sql.DB) string {
	t.Helper()

	id := "test-execution-" + time.Now().Format("20060102150405")
	query := `INSERT INTO executions (id, workflow_id, status, input_data, output_data, started_at)
	          VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := db.Exec(query, id, "workflow-1", "completed", `{"input": "test"}`, `{"output": "result"}`, time.Now())
	if err != nil {
		t.Fatalf("Failed to create test execution: %v", err)
	}

	return id
}

// Set environment variable for tests
func init() {
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
}
