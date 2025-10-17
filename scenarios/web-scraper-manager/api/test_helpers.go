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
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalFlags  int
	originalPrefix string
	originalOutput io.Writer
}

// setupTestLogger initializes a test logger with controlled output
func setupTestLogger() func() {
	originalFlags := log.Flags()
	originalPrefix := log.Prefix()
	originalOutput := log.Writer()

	// Set test logger with minimal output
	log.SetFlags(log.LstdFlags)
	log.SetPrefix("[test] ")

	return func() {
		log.SetFlags(originalFlags)
		log.SetPrefix(originalPrefix)
		log.SetOutput(originalOutput)
	}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	DB         *sql.DB
	TempDir    string
	OriginalDB *sql.DB
	Cleanup    func()
}

// setupTestDatabase creates an isolated test database connection
func setupTestDatabase(t *testing.T) *TestEnvironment {
	// Store original DB
	originalDB := db

	// Create in-memory SQLite DB for testing
	testDB, err := sql.Open("postgres", os.Getenv("POSTGRES_URL"))
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Set as global db
	db = testDB

	return &TestEnvironment{
		DB:         testDB,
		OriginalDB: originalDB,
		Cleanup: func() {
			testDB.Close()
			db = originalDB
		},
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

// makeHTTPRequest creates and executes an HTTP request for testing
func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, error) {
	var body io.Reader
	if req.Body != nil {
		jsonData, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	httpReq, err := http.NewRequest(req.Method, req.Path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set default headers
	httpReq.Header.Set("Content-Type", "application/json")

	// Set custom headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Set URL variables (for mux)
	if req.URLVars != nil {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	// Create response recorder
	w := httptest.NewRecorder()

	return w, nil
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) APIResponse {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var response APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode JSON response: %v. Body: %s", err, w.Body.String())
	}

	return response
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedError string) {
	response := assertJSONResponse(t, w, expectedStatus)

	if response.Success {
		t.Error("Expected success=false in error response")
	}

	if response.Error == "" {
		t.Error("Expected error message in response")
	}

	if expectedError != "" && response.Error != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, response.Error)
	}
}

// assertSuccessResponse validates a success response
func assertSuccessResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) APIResponse {
	response := assertJSONResponse(t, w, expectedStatus)

	if !response.Success {
		t.Errorf("Expected success=true, got false. Error: %s", response.Error)
	}

	return response
}

// TestAgent provides a pre-configured agent for testing
type TestAgent struct {
	Agent   *ScrapingAgent
	Cleanup func()
}

// createTestAgent creates a test agent in the database
func createTestAgent(t *testing.T, name, platform string) *TestAgent {
	agent := &ScrapingAgent{
		ID:            uuid.New().String(),
		Name:          name,
		Platform:      platform,
		AgentType:     "WebsiteAgent",
		Configuration: map[string]interface{}{"url": "https://example.com"},
		Enabled:       true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Tags:          []string{"test"},
	}

	configJSON, _ := json.Marshal(agent.Configuration)
	tagsJSON, _ := json.Marshal(agent.Tags)

	query := `
		INSERT INTO scraping_agents (id, name, platform, agent_type, configuration,
		                           enabled, created_at, updated_at, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := db.Exec(query, agent.ID, agent.Name, agent.Platform, agent.AgentType,
		configJSON, agent.Enabled, agent.CreatedAt, agent.UpdatedAt, tagsJSON)

	if err != nil {
		t.Fatalf("Failed to create test agent: %v", err)
	}

	return &TestAgent{
		Agent: agent,
		Cleanup: func() {
			db.Exec("DELETE FROM scraping_agents WHERE id = $1", agent.ID)
		},
	}
}

// TestTarget provides a pre-configured target for testing
type TestTarget struct {
	Target  *ScrapingTarget
	Cleanup func()
}

// createTestTarget creates a test target in the database
func createTestTarget(t *testing.T, agentID, url string) *TestTarget {
	target := &ScrapingTarget{
		ID:             uuid.New().String(),
		AgentID:        agentID,
		URL:            url,
		SelectorConfig: map[string]interface{}{"title": "h1"},
		RateLimitMs:    1000,
		MaxRetries:     3,
		CreatedAt:      time.Now(),
	}

	selectorJSON, _ := json.Marshal(target.SelectorConfig)

	query := `
		INSERT INTO scraping_targets (id, agent_id, url, selector_config,
		                            rate_limit_ms, max_retries, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := db.Exec(query, target.ID, target.AgentID, target.URL,
		selectorJSON, target.RateLimitMs, target.MaxRetries, target.CreatedAt)

	if err != nil {
		t.Fatalf("Failed to create test target: %v", err)
	}

	return &TestTarget{
		Target: target,
		Cleanup: func() {
			db.Exec("DELETE FROM scraping_targets WHERE id = $1", target.ID)
		},
	}
}

// TestResult provides a pre-configured result for testing
type TestResult struct {
	Result  *ScrapingResult
	Cleanup func()
}

// createTestResult creates a test result in the database
func createTestResult(t *testing.T, agentID, targetID, status string) *TestResult {
	extractedCount := 5
	execTime := 1234
	completedAt := time.Now()

	result := &ScrapingResult{
		ID:              uuid.New().String(),
		AgentID:         agentID,
		TargetID:        targetID,
		RunID:           fmt.Sprintf("run_%d", time.Now().Unix()),
		Status:          status,
		Data:            map[string]interface{}{"title": "Test Page"},
		ExtractedCount:  &extractedCount,
		StartedAt:       time.Now().Add(-5 * time.Second),
		CompletedAt:     &completedAt,
		ExecutionTimeMs: &execTime,
	}

	dataJSON, _ := json.Marshal(result.Data)

	query := `
		INSERT INTO scraping_results (id, agent_id, target_id, run_id, status,
		                            data, extracted_count, started_at, completed_at,
		                            execution_time_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := db.Exec(query, result.ID, result.AgentID, result.TargetID, result.RunID,
		result.Status, dataJSON, result.ExtractedCount, result.StartedAt,
		result.CompletedAt, result.ExecutionTimeMs)

	if err != nil {
		t.Fatalf("Failed to create test result: %v", err)
	}

	return &TestResult{
		Result: result,
		Cleanup: func() {
			db.Exec("DELETE FROM scraping_results WHERE id = $1", result.ID)
		},
	}
}

// cleanupTestData removes all test data from the database
func cleanupTestData(t *testing.T) {
	tables := []string{"scraping_results", "scraping_targets", "scraping_agents"}
	for _, table := range tables {
		if _, err := db.Exec(fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			t.Logf("Warning: Failed to cleanup %s: %v", table, err)
		}
	}
}
