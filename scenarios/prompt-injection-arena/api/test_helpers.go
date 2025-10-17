// +build testing

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// TestLogger provides controlled logging during tests
type TestLogger struct {
	originalOutput *os.File
	cleanup        func()
}

// setupTestLogger initializes controlled logging for tests
func setupTestLogger() func() {
	// Suppress logs during tests unless -v is passed
	if !testing.Verbose() {
		gin.SetMode(gin.ReleaseMode)
		log.SetOutput(ioutil.Discard)
		return func() {
			log.SetOutput(os.Stdout)
		}
	}
	return func() {}
}

// TestEnvironment manages isolated test environment
type TestEnvironment struct {
	DB         *sql.DB
	TempDir    string
	OriginalWD string
	Cleanup    func()
}

// setupTestDB creates an isolated test database connection
func setupTestDB(t *testing.T) *TestEnvironment {
	// Set test environment variables
	os.Setenv("POSTGRES_HOST", "localhost")
	os.Setenv("POSTGRES_PORT", "5432")
	os.Setenv("POSTGRES_USER", "test_user")
	os.Setenv("POSTGRES_PASSWORD", "test_password")
	os.Setenv("POSTGRES_DB", "prompt_injection_arena_test")

	// Create test database connection
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"))

	testDB, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
		return nil
	}

	// Test connection
	if err := testDB.Ping(); err != nil {
		testDB.Close()
		t.Skipf("Skipping test: database not reachable: %v", err)
		return nil
	}

	// Save original db and replace with test db
	originalDB := db
	db = testDB

	return &TestEnvironment{
		DB: testDB,
		Cleanup: func() {
			db = originalDB
			testDB.Close()
		},
	}
}

// HTTPTestRequest represents a test HTTP request
type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        interface{}
	Headers     map[string]string
	QueryParams map[string]string
}

// makeHTTPRequest creates and executes a test HTTP request
func makeHTTPRequest(router *gin.Engine, req HTTPTestRequest) *httptest.ResponseRecorder {
	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, _ = json.Marshal(req.Body)
	}

	httpReq, _ := http.NewRequest(req.Method, req.Path, bytes.NewBuffer(bodyBytes))

	// Add headers
	httpReq.Header.Set("Content-Type", "application/json")
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// Add query parameters
	if len(req.QueryParams) > 0 {
		q := httpReq.URL.Query()
		for key, value := range req.QueryParams {
			q.Add(key, value)
		}
		httpReq.URL.RawQuery = q.Encode()
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)
	return w
}

// assertJSONResponse validates a JSON response
func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	if w.Code != expectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", expectedStatus, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
	}

	return response
}

// assertErrorResponse validates an error response
func assertErrorResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedErrorMsg string) {
	response := assertJSONResponse(t, w, expectedStatus)

	if errMsg, ok := response["error"].(string); !ok {
		t.Errorf("Expected 'error' field in response, got: %v", response)
	} else if expectedErrorMsg != "" && errMsg != expectedErrorMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrorMsg, errMsg)
	}
}

// createTestInjectionTechnique creates a test injection technique in the database
func createTestInjectionTechnique(t *testing.T, db *sql.DB, name string) *InjectionTechnique {
	technique := &InjectionTechnique{
		ID:                uuid.New().String(),
		Name:              name,
		Category:          "test-category",
		Description:       "Test injection technique",
		ExamplePrompt:     "Test prompt",
		DifficultyScore:   0.5,
		SuccessRate:       0.7,
		SourceAttribution: "test",
		IsActive:          true,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		CreatedBy:         "test-user",
	}

	_, err := db.Exec(`
		INSERT INTO injection_techniques
		(id, name, category, description, example_prompt, difficulty_score,
		 success_rate, source_attribution, is_active, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		technique.ID, technique.Name, technique.Category, technique.Description,
		technique.ExamplePrompt, technique.DifficultyScore, technique.SuccessRate,
		technique.SourceAttribution, technique.IsActive, technique.CreatedAt,
		technique.UpdatedAt, technique.CreatedBy)

	if err != nil {
		t.Fatalf("Failed to create test injection technique: %v", err)
	}

	return technique
}

// createTestAgentConfig creates a test agent configuration in the database
func createTestAgentConfig(t *testing.T, db *sql.DB, name string) *AgentConfiguration {
	safetyConstraints := map[string]interface{}{
		"max_retries": 3,
		"timeout":     30,
	}
	constraintsJSON, _ := json.Marshal(safetyConstraints)

	config := &AgentConfiguration{
		ID:                uuid.New().String(),
		Name:              name,
		SystemPrompt:      "You are a helpful assistant",
		ModelName:         "test-model",
		Temperature:       0.7,
		MaxTokens:         100,
		SafetyConstraints: safetyConstraints,
		RobustnessScore:   0.0,
		TestsRun:          0,
		TestsPassed:       0,
		IsActive:          true,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		CreatedBy:         "test-user",
	}

	_, err := db.Exec(`
		INSERT INTO agent_configurations
		(id, name, system_prompt, model_name, temperature, max_tokens,
		 safety_constraints, robustness_score, tests_run, tests_passed,
		 is_active, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		config.ID, config.Name, config.SystemPrompt, config.ModelName,
		config.Temperature, config.MaxTokens, constraintsJSON,
		config.RobustnessScore, config.TestsRun, config.TestsPassed,
		config.IsActive, config.CreatedAt, config.UpdatedAt, config.CreatedBy)

	if err != nil {
		t.Fatalf("Failed to create test agent configuration: %v", err)
	}

	return config
}

// createTestResult creates a test result in the database
func createTestResult(t *testing.T, db *sql.DB, injectionID, agentID string) *TestResult {
	safetyViolations := []map[string]interface{}{}
	violationsJSON, _ := json.Marshal(safetyViolations)

	metadata := map[string]interface{}{"test": true}
	metadataJSON, _ := json.Marshal(metadata)

	result := &TestResult{
		ID:               uuid.New().String(),
		InjectionID:      injectionID,
		AgentID:          agentID,
		Success:          true,
		ResponseText:     "Test response",
		ExecutionTimeMS:  100,
		SafetyViolations: safetyViolations,
		ConfidenceScore:  0.9,
		ErrorMessage:     "",
		ExecutedAt:       time.Now(),
		TestSessionID:    uuid.New().String(),
		Metadata:         metadata,
	}

	_, err := db.Exec(`
		INSERT INTO test_results
		(id, injection_id, agent_id, success, response_text, execution_time_ms,
		 safety_violations, confidence_score, error_message, executed_at,
		 test_session_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		result.ID, result.InjectionID, result.AgentID, result.Success,
		result.ResponseText, result.ExecutionTimeMS, violationsJSON,
		result.ConfidenceScore, result.ErrorMessage, result.ExecutedAt,
		result.TestSessionID, metadataJSON)

	if err != nil {
		t.Fatalf("Failed to create test result: %v", err)
	}

	return result
}

// cleanupTestData removes test data from the database
func cleanupTestData(db *sql.DB, tables ...string) error {
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DELETE FROM %s WHERE created_by = 'test-user'", table))
		if err != nil {
			return fmt.Errorf("failed to cleanup %s: %v", table, err)
		}
	}
	return nil
}

// setupTestRouter creates a test Gin router with all routes
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Health check
	router.GET("/health", healthCheck)

	// API routes
	api := router.Group("/api/v1")
	{
		// Injection library
		api.GET("/injections/library", getInjectionLibrary)
		api.POST("/injections", addInjectionTechnique)
		api.GET("/injections/similar", getSimilarInjections)

		// Leaderboards
		api.GET("/leaderboards/agents", getAgentLeaderboard)
		api.GET("/leaderboards/injections", getInjectionLeaderboard)

		// Agent testing
		api.POST("/security/test-agent", testAgent)

		// Statistics
		api.GET("/statistics", getStatistics)

		// Vector search
		api.POST("/vector/search", vectorSearch)
		api.POST("/vector/index", indexInjection)

		// Tournament system
		api.GET("/tournaments", getTournaments)
		api.POST("/tournaments", createTournamentHandler)
		api.POST("/tournaments/:id/run", runTournamentHandler)
		api.GET("/tournaments/:id/results", getTournamentResults)

		// Research export
		api.POST("/export/research", exportResearch)
		api.GET("/export/formats", getExportFormats)
	}

	return router
}
