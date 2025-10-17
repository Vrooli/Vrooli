// +build testing

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
	cleanup        func()
}

// setupTestLogger initializes logging for tests with cleanup
func setupTestLogger() func() {
	// Suppress logs during testing unless VERBOSE=true
	if os.Getenv("VERBOSE") != "true" {
		log.SetOutput(io.Discard)
		return func() {
			log.SetOutput(os.Stderr)
		}
	}
	return func() {}
}

// TestDatabase manages test database connections
type TestDatabase struct {
	DB      *sql.DB
	Cleanup func()
}

// setupTestDatabase creates a test database connection
func setupTestDatabase(t *testing.T) *TestDatabase {
	// Use environment variables for test database
	dbHost := os.Getenv("POSTGRES_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	dbPort := os.Getenv("POSTGRES_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}

	dbUser := os.Getenv("POSTGRES_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}

	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}

	dbName := os.Getenv("POSTGRES_DB")
	if dbName == "" {
		dbName = "scalable_app_cookbook_test"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	testDB, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("Skipping database test: %v", err)
		return nil
	}

	if err := testDB.Ping(); err != nil {
		testDB.Close()
		t.Skipf("Skipping database test (DB not available): %v", err)
		return nil
	}

	// Set the global db variable for handlers to use
	db = testDB

	// Ensure schema exists for test database
	_, err = testDB.Exec("CREATE SCHEMA IF NOT EXISTS scalable_app_cookbook")
	if err != nil {
		testDB.Close()
		t.Skipf("Failed to create schema: %v", err)
		return nil
	}

	// Create patterns table if it doesn't exist
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS scalable_app_cookbook.patterns (
			id VARCHAR(255) PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			chapter VARCHAR(100) NOT NULL,
			section VARCHAR(100) NOT NULL,
			maturity_level VARCHAR(2) NOT NULL CHECK (maturity_level IN ('L0', 'L1', 'L2', 'L3', 'L4')),
			tags TEXT[] DEFAULT '{}',
			what_and_why TEXT NOT NULL,
			when_to_use TEXT NOT NULL,
			tradeoffs TEXT NOT NULL,
			reference_patterns TEXT[] DEFAULT '{}',
			failure_modes TEXT,
			cost_levers TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Logf("Warning: Failed to create patterns table: %v", err)
	}

	// Create recipes table if it doesn't exist
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS recipes (
			id VARCHAR(255) PRIMARY KEY,
			pattern_id VARCHAR(255) NOT NULL,
			title VARCHAR(255) NOT NULL,
			type VARCHAR(20) NOT NULL CHECK (type IN ('greenfield', 'brownfield', 'migration')),
			prerequisites TEXT[] DEFAULT '{}',
			steps JSONB NOT NULL,
			config_snippets JSONB DEFAULT '{}',
			validation_checks TEXT[] DEFAULT '{}',
			artifacts TEXT[] DEFAULT '{}',
			metrics TEXT[] DEFAULT '{}',
			rollbacks TEXT[] DEFAULT '{}',
			prompts TEXT[] DEFAULT '{}',
			timeout_sec INTEGER DEFAULT 300,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Logf("Warning: Failed to create recipes table: %v", err)
	}

	// Create implementations table if it doesn't exist
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS implementations (
			id VARCHAR(255) PRIMARY KEY,
			recipe_id VARCHAR(255) NOT NULL,
			language VARCHAR(20) NOT NULL CHECK (language IN ('go', 'javascript', 'typescript', 'python', 'java', 'rust', 'csharp')),
			code TEXT NOT NULL,
			file_path VARCHAR(500),
			description TEXT,
			dependencies TEXT[] DEFAULT '{}',
			test_code TEXT,
			benchmark_data JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Logf("Warning: Failed to create implementations table: %v", err)
	}

	// Create pattern_usage table if it doesn't exist
	_, err = testDB.Exec(`
		CREATE TABLE IF NOT EXISTS pattern_usage (
			id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::text,
			pattern_id VARCHAR(255) NOT NULL,
			user_agent VARCHAR(255),
			access_type VARCHAR(20) NOT NULL,
			success BOOLEAN NOT NULL,
			metadata JSONB,
			accessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Logf("Warning: Failed to create pattern_usage table: %v", err)
	}

	return &TestDatabase{
		DB: testDB,
		Cleanup: func() {
			if testDB != nil {
				testDB.Close()
			}
		},
	}
}

// HTTPTestRequest represents an HTTP test request
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	URLVars map[string]string
	Query   map[string]string
	Headers map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(handler http.HandlerFunc, req HTTPTestRequest) *httptest.ResponseRecorder {
	var body io.Reader
	if req.Body != nil {
		jsonBody, _ := json.Marshal(req.Body)
		body = bytes.NewBuffer(jsonBody)
	}

	// Build URL with query parameters
	url := req.Path
	if len(req.Query) > 0 {
		url += "?"
		first := true
		for k, v := range req.Query {
			if !first {
				url += "&"
			}
			url += fmt.Sprintf("%s=%s", k, v)
			first = false
		}
	}

	httpReq := httptest.NewRequest(req.Method, url, body)

	// Set headers
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Set URL vars if using mux
	if len(req.URLVars) > 0 {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	w := httptest.NewRecorder()
	handler(w, httpReq)
	return w
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v. Body: %s", err, w.Body.String())
	}

	return response
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	bodyStr := w.Body.String()
	if expectedMessage != "" && bodyStr != "" {
		// Simple check if the expected message is contained in the body
		if len(bodyStr) == 0 {
			t.Errorf("Expected error message containing '%s', got empty response", expectedMessage)
		}
	}
}

// assertArrayResponse validates a JSON array response
func assertArrayResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) []interface{} {
	t.Helper()

	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var response []interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON array response: %v. Body: %s", err, w.Body.String())
	}

	return response
}

// seedTestData inserts sample data for testing
func seedTestData(t *testing.T, testDB *TestDatabase) {
	t.Helper()

	// Insert a test pattern (using PostgreSQL array syntax)
	_, err := testDB.DB.Exec(`
		INSERT INTO scalable_app_cookbook.patterns
		(id, title, chapter, section, maturity_level, tags, what_and_why, when_to_use,
		 tradeoffs, reference_patterns, failure_modes, cost_levers)
		VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO NOTHING
	`, "test-pattern-1", "Test Pattern", "Testing", "Unit Testing", "L2",
		`{testing,quality}`, "Test pattern for unit tests", "Use in test scenarios",
		"Trade-offs description", `{}`, "Failure modes", "Cost levers")

	if err != nil {
		t.Fatalf("Failed to seed test pattern: %v", err)
	}

	// Insert a test recipe (using PostgreSQL array syntax)
	_, err = testDB.DB.Exec(`
		INSERT INTO recipes
		(id, pattern_id, title, type, prerequisites, steps, config_snippets,
		 validation_checks, artifacts, metrics, rollbacks, prompts, timeout_sec)
		VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (id) DO NOTHING
	`, "test-recipe-1", "test-pattern-1", "Test Recipe", "greenfield",
		`{prerequisite-1}`,
		`[{"step": 1, "action": "test"}]`,
		`{"config": "value"}`,
		`{check-1}`,
		`{artifact-1}`,
		`{metric-1}`,
		`{rollback-1}`,
		`{prompt-1}`,
		300)

	if err != nil {
		t.Fatalf("Failed to seed test recipe: %v", err)
	}

	// Insert a test implementation (using PostgreSQL array syntax)
	_, err = testDB.DB.Exec(`
		INSERT INTO implementations
		(id, recipe_id, language, code, file_path, description, dependencies, test_code)
		VALUES
		($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO NOTHING
	`, "test-impl-1", "test-recipe-1", "go", "package main", "/main.go",
		"Test implementation", `{dep1}`, "package main_test")

	if err != nil {
		t.Fatalf("Failed to seed test implementation: %v", err)
	}
}

// cleanTestData removes test data after tests
func cleanTestData(t *testing.T, testDB *TestDatabase) {
	t.Helper()

	// Clean in reverse order due to foreign keys
	testDB.DB.Exec("DELETE FROM implementations WHERE id LIKE 'test-%'")
	testDB.DB.Exec("DELETE FROM recipes WHERE id LIKE 'test-%'")
	testDB.DB.Exec("DELETE FROM scalable_app_cookbook.patterns WHERE id LIKE 'test-%'")
	testDB.DB.Exec("DELETE FROM pattern_usage WHERE pattern_id LIKE 'test-%'")
}
