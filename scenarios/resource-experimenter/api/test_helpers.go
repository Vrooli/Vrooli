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
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput io.Writer
	cleanup        func()
}

// setupTestLogger initializes logging for testing
func setupTestLogger() func() {
	originalOutput := log.Writer()
	log.SetOutput(io.Discard) // Silence logs during tests unless debugging
	return func() { log.SetOutput(originalOutput) }
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	Server  *APIServer
	DB      *sql.DB
	Router  *mux.Router
	Cleanup func()
}

// setupTestDB creates an in-memory test database
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// Use environment variable or default to test database
	testDBURL := os.Getenv("TEST_POSTGRES_URL")
	if testDBURL == "" {
		// Use a test-specific database with vrooli postgres
		// Get password from environment or use default
		dbPassword := os.Getenv("POSTGRES_PASSWORD")
		if dbPassword == "" {
			dbPassword = "lUq9qvemypKpuEeXCV6Vnxak1"
		}
		testDBURL = fmt.Sprintf("postgres://vrooli:%s@localhost:5433/resource_experimenter_test?sslmode=disable", dbPassword)
	}

	db, err := sql.Open("postgres", testDBURL)
	if err != nil {
		t.Skipf("Skipping test - database not available: %v", err)
		return nil, func() {}
	}

	// Test database connectivity
	if err := db.Ping(); err != nil {
		t.Skipf("Skipping test - database not reachable: %v", err)
		return nil, func() {}
	}

	// Create test schema
	createTestSchema(t, db)

	cleanup := func() {
		// Clean up test data
		db.Exec("TRUNCATE TABLE experiment_logs CASCADE")
		db.Exec("TRUNCATE TABLE experiments CASCADE")
		db.Exec("TRUNCATE TABLE experiment_templates CASCADE")
		db.Exec("TRUNCATE TABLE available_scenarios CASCADE")
		db.Close()
	}

	return db, cleanup
}

// createTestSchema creates necessary database tables for testing
func createTestSchema(t *testing.T, db *sql.DB) {
	schema := `
		CREATE TABLE IF NOT EXISTS experiments (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			prompt TEXT,
			target_scenario VARCHAR(255),
			new_resource VARCHAR(255),
			existing_resources JSONB DEFAULT '[]',
			files_generated JSONB DEFAULT '{}',
			modifications_made JSONB DEFAULT '{}',
			status VARCHAR(50) DEFAULT 'requested',
			experiment_id UUID,
			claude_prompt TEXT,
			claude_response TEXT,
			generation_error TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			completed_at TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS experiment_templates (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			prompt_template TEXT,
			target_scenario_pattern VARCHAR(255),
			resource_category VARCHAR(100),
			usage_count INTEGER DEFAULT 0,
			success_rate DECIMAL(5,2),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			is_active BOOLEAN DEFAULT true
		);

		CREATE TABLE IF NOT EXISTS available_scenarios (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			display_name VARCHAR(255),
			description TEXT,
			path VARCHAR(500),
			current_resources JSONB DEFAULT '[]',
			resource_categories JSONB DEFAULT '[]',
			experimentation_friendly BOOLEAN DEFAULT true,
			complexity_level VARCHAR(50),
			last_experiment_date TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS experiment_logs (
			id UUID PRIMARY KEY,
			experiment_id UUID REFERENCES experiments(id) ON DELETE CASCADE,
			step VARCHAR(100),
			prompt TEXT,
			response TEXT,
			success BOOLEAN DEFAULT false,
			error_message TEXT,
			started_at TIMESTAMP DEFAULT NOW(),
			completed_at TIMESTAMP,
			duration_seconds INTEGER
		);
	`

	_, err := db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}
}

// setupTestServer creates a test API server with test database
func setupTestServer(t *testing.T) *TestEnvironment {
	db, dbCleanup := setupTestDB(t)
	if db == nil {
		t.Skip("Test database not available")
	}

	server := &APIServer{
		db: db,
	}
	server.InitRoutes()

	return &TestEnvironment{
		Server:  server,
		DB:      db,
		Router:  server.router,
		Cleanup: dbCleanup,
	}
}

// HTTPTestRequest represents an HTTP request for testing
type HTTPTestRequest struct {
	Method  string
	Path    string
	Body    interface{}
	Headers map[string]string
	URLVars map[string]string
}

// makeHTTPRequest creates and executes an HTTP test request
func makeHTTPRequest(env *TestEnvironment, req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var bodyReader io.Reader
	if req.Body != nil {
		switch v := req.Body.(type) {
		case string:
			bodyReader = bytes.NewBufferString(v)
		case []byte:
			bodyReader = bytes.NewBuffer(v)
		default:
			jsonBody, err := json.Marshal(req.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal body: %v", err)
			}
			bodyReader = bytes.NewBuffer(jsonBody)
		}
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set URL vars if any
	if len(req.URLVars) > 0 {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	w := httptest.NewRecorder()
	env.Router.ServeHTTP(w, httpReq)

	return w, nil
}

// assertJSONResponse validates JSON response structure and status code
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, target interface{}) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	if target != nil {
		if err := json.NewDecoder(w.Body).Decode(target); err != nil {
			t.Errorf("Failed to decode JSON response: %v. Body: %s", err, w.Body.String())
		}
	}
}

// assertErrorResponse validates error response format
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedMessage string) {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	bodyStr := w.Body.String()
	if expectedMessage != "" && !contains(bodyStr, expectedMessage) {
		t.Errorf("Expected error message to contain '%s', got: %s", expectedMessage, bodyStr)
	}
}

// createTestExperiment creates a test experiment in the database
func createTestExperiment(t *testing.T, env *TestEnvironment, name string) *Experiment {
	exp := &Experiment{
		ID:             uuid.New().String(),
		Name:           name,
		Description:    fmt.Sprintf("Test experiment: %s", name),
		Prompt:         "Test prompt for integration",
		TargetScenario: "test-scenario",
		NewResource:    "test-resource",
		Status:         "requested",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	query := `INSERT INTO experiments (id, name, description, prompt, target_scenario, new_resource, status, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := env.DB.Exec(query, exp.ID, exp.Name, exp.Description, exp.Prompt,
		exp.TargetScenario, exp.NewResource, exp.Status, exp.CreatedAt, exp.UpdatedAt)
	if err != nil {
		t.Fatalf("Failed to create test experiment: %v", err)
	}

	return exp
}

// createTestTemplate creates a test experiment template in the database
func createTestTemplate(t *testing.T, env *TestEnvironment, name string) *ExperimentTemplate {
	tmpl := &ExperimentTemplate{
		ID:             uuid.New().String(),
		Name:           name,
		Description:    fmt.Sprintf("Test template: %s", name),
		PromptTemplate: "Test template: {{NAME}} - {{DESCRIPTION}}",
		IsActive:       true,
		UsageCount:     0,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	query := `INSERT INTO experiment_templates (id, name, description, prompt_template, is_active, usage_count, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := env.DB.Exec(query, tmpl.ID, tmpl.Name, tmpl.Description, tmpl.PromptTemplate,
		tmpl.IsActive, tmpl.UsageCount, tmpl.CreatedAt, tmpl.UpdatedAt)
	if err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	return tmpl
}

// createTestScenario creates a test available scenario in the database
func createTestScenario(t *testing.T, env *TestEnvironment, name string) *AvailableScenario {
	scenario := &AvailableScenario{
		ID:                      uuid.New().String(),
		Name:                    name,
		Path:                    fmt.Sprintf("/scenarios/%s", name),
		ExperimentationFriendly: true,
		ComplexityLevel:         "medium",
		CurrentResources:        []string{"postgres", "redis"},
		ResourceCategories:      []string{"storage", "cache"},
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}

	resourcesJSON, _ := json.Marshal(scenario.CurrentResources)
	categoriesJSON, _ := json.Marshal(scenario.ResourceCategories)

	query := `INSERT INTO available_scenarios (id, name, path, current_resources, resource_categories,
	          experimentation_friendly, complexity_level, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := env.DB.Exec(query, scenario.ID, scenario.Name, scenario.Path, resourcesJSON, categoriesJSON,
		scenario.ExperimentationFriendly, scenario.ComplexityLevel, scenario.CreatedAt, scenario.UpdatedAt)
	if err != nil {
		t.Fatalf("Failed to create test scenario: %v", err)
	}

	return scenario
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}

// assertResponseContains checks if response body contains expected text
func assertResponseContains(t *testing.T, w *httptest.ResponseRecorder, expected string) {
	if !contains(w.Body.String(), expected) {
		t.Errorf("Expected response to contain '%s', got: %s", expected, w.Body.String())
	}
}
