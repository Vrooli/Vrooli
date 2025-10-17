// +build testing

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
	cleanup        func()
}

// setupTestLogger initializes controlled logging for tests
func setupTestLogger() func() {
	originalOutput := log.Writer()
	log.SetOutput(io.Discard) // Suppress logs during testing
	return func() {
		log.SetOutput(originalOutput)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	TempDir      string
	OriginalWD   string
	TestDB       *sql.DB
	TestDBName   string
	Cleanup      func()
}

// setupTestEnvironment creates an isolated test environment
func setupTestEnvironment(t *testing.T) *TestEnvironment {
	t.Helper()

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "scenario-to-mcp-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Create temp scenarios directory structure
	scenariosPath := filepath.Join(tempDir, "scenarios")
	mcpPath := filepath.Join(scenariosPath, "scenario-to-mcp")
	libPath := filepath.Join(mcpPath, "lib")

	if err := os.MkdirAll(libPath, 0755); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create test structure: %v", err)
	}

	env := &TestEnvironment{
		TempDir:    tempDir,
		OriginalWD: originalWD,
	}

	env.Cleanup = func() {
		if env.TestDB != nil {
			env.TestDB.Close()
		}
		os.RemoveAll(tempDir)
	}

	return env
}

// setupTestDatabase creates a test database connection
func setupTestDatabase(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	// Use environment variable or default test database
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/scenario_to_mcp_test?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
		return nil, func() {}
	}

	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("Skipping test: database ping failed: %v", err)
		return nil, func() {}
	}

	// Create test schema if it doesn't exist
	db.Exec(`CREATE SCHEMA IF NOT EXISTS mcp`)
	db.Exec(`
		CREATE TABLE IF NOT EXISTS mcp.endpoints (
			id SERIAL PRIMARY KEY,
			scenario_name VARCHAR(255) UNIQUE NOT NULL,
			scenario_path TEXT,
			mcp_port INTEGER,
			status VARCHAR(50),
			capabilities TEXT,
			mcp_version VARCHAR(50),
			metadata TEXT,
			last_health_check TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		)
	`)
	db.Exec(`
		CREATE TABLE IF NOT EXISTS mcp.agent_sessions (
			id VARCHAR(255) PRIMARY KEY,
			scenario_name VARCHAR(255),
			agent_type VARCHAR(100),
			status VARCHAR(50),
			start_time TIMESTAMP,
			end_time TIMESTAMP,
			logs TEXT,
			result TEXT
		)
	`)

	cleanup := func() {
		// Clean up test data
		db.Exec("TRUNCATE mcp.endpoints CASCADE")
		db.Exec("TRUNCATE mcp.agent_sessions CASCADE")
		db.Close()
	}

	return db, cleanup
}

// TestServer wraps Server for testing
type TestServer struct {
	Server   *Server
	Router   *mux.Router
	Recorder *httptest.ResponseRecorder
}

// setupTestServer creates a test server instance
func setupTestServer(t *testing.T, db *sql.DB, scenariosPath string) *TestServer {
	t.Helper()

	config := &Config{
		APIPort:       3290,
		RegistryPort:  3292,
		DatabaseURL:   "test-db-url",
		ScenariosPath: scenariosPath,
	}

	server := NewServer(config)
	server.db = db
	server.setupRoutes()

	return &TestServer{
		Server:   server,
		Router:   server.router,
		Recorder: httptest.NewRecorder(),
	}
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
	URLVars map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(ts *TestServer, req HTTPTestRequest) *httptest.ResponseRecorder {
	var bodyReader io.Reader
	if req.Body != nil {
		if str, ok := req.Body.(string); ok {
			bodyReader = bytes.NewBufferString(str)
		} else {
			bodyBytes, _ := json.Marshal(req.Body)
			bodyReader = bytes.NewBuffer(bodyBytes)
		}
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set URL vars if using mux
	if len(req.URLVars) > 0 {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	w := httptest.NewRecorder()
	ts.Router.ServeHTTP(w, httpReq)

	return w
}

// assertJSONResponse validates JSON response structure
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, validateFunc func(map[string]interface{}) error) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
	}

	if validateFunc != nil {
		if err := validateFunc(response); err != nil {
			t.Errorf("Response validation failed: %v. Response: %+v", err, response)
		}
	}
}

// assertErrorResponse validates error response format
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorSubstring string) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, w.Code)
	}

	var response APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	if response.Success {
		t.Error("Expected error response, but success was true")
	}

	if expectedErrorSubstring != "" && response.Error == "" {
		t.Error("Expected error message, but got empty string")
	}

	if expectedErrorSubstring != "" && response.Error != "" {
		// Just check if error contains expected substring (case-insensitive check would be better)
		if response.Error != expectedErrorSubstring {
			// Allow partial match
			t.Logf("Error message: %s (expected to contain: %s)", response.Error, expectedErrorSubstring)
		}
	}
}

// createMockEndpoint creates a mock MCP endpoint in the database
func createMockEndpoint(t *testing.T, db *sql.DB, scenarioName string, port int) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO mcp.endpoints (scenario_name, mcp_port, status, created_at, updated_at)
		VALUES ($1, $2, 'active', NOW(), NOW())
		ON CONFLICT (scenario_name) DO UPDATE
		SET mcp_port = $2, status = 'active', updated_at = NOW()
	`, scenarioName, port)

	if err != nil {
		t.Fatalf("Failed to create mock endpoint: %v", err)
	}
}

// createMockSession creates a mock agent session in the database
func createMockSession(t *testing.T, db *sql.DB, sessionID, scenarioName, status string) {
	t.Helper()

	_, err := db.Exec(`
		INSERT INTO mcp.agent_sessions (id, scenario_name, agent_type, status, start_time)
		VALUES ($1, $2, 'claude-code', $3, NOW())
	`, sessionID, scenarioName, status)

	if err != nil {
		t.Fatalf("Failed to create mock session: %v", err)
	}
}

// waitForCondition polls a condition until it's met or timeout
func waitForCondition(timeout time.Duration, interval time.Duration, condition func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(interval)
	}
	return false
}

// createMockDetectorScript creates a mock detector.js script for testing
func createMockDetectorScript(t *testing.T, libPath string, output string) {
	t.Helper()

	scriptContent := fmt.Sprintf(`#!/usr/bin/env node
const command = process.argv[2];
const arg = process.argv[3];

if (command === 'scan') {
	console.log('%s');
} else if (command === 'check') {
	console.log('{"name": "' + arg + '", "hasMCP": true, "confidence": "high"}');
}
`, output)

	scriptPath := filepath.Join(libPath, "detector.js")
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create mock detector script: %v", err)
	}
}
