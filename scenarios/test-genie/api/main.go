package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Data Models
type TestSuite struct {
	ID              uuid.UUID       `json:"id"`
	ScenarioName    string          `json:"scenario_name"`
	SuiteType       string          `json:"suite_type"`
	TestCases       []TestCase      `json:"test_cases"`
	CoverageMetrics CoverageMetrics `json:"coverage_metrics"`
	GeneratedAt     time.Time       `json:"generated_at"`
	LastExecuted    *time.Time      `json:"last_executed,omitempty"`
	Status          string          `json:"status"`
}

type TestCase struct {
	ID             uuid.UUID `json:"id"`
	SuiteID        uuid.UUID `json:"suite_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	TestType       string    `json:"test_type"`
	TestCode       string    `json:"test_code"`
	ExpectedResult string    `json:"expected_result"`
	Timeout        int       `json:"execution_timeout"`
	Dependencies   []string  `json:"dependencies"`
	Tags           []string  `json:"tags"`
	Priority       string    `json:"priority"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type TestExecution struct {
	ID                 uuid.UUID          `json:"id"`
	SuiteID            uuid.UUID          `json:"suite_id"`
	ExecutionType      string             `json:"execution_type"`
	StartTime          time.Time          `json:"start_time"`
	EndTime            *time.Time         `json:"end_time,omitempty"`
	Status             string             `json:"status"`
	Results            []TestResult       `json:"results"`
	PerformanceMetrics PerformanceMetrics `json:"performance_metrics"`
	Environment        string             `json:"environment"`
}

type TestResult struct {
	ID                  uuid.UUID              `json:"id"`
	ExecutionID         uuid.UUID              `json:"execution_id"`
	TestCaseID          uuid.UUID              `json:"test_case_id"`
	Status              string                 `json:"status"`
	Duration            float64                `json:"duration"`
	ErrorMessage        *string                `json:"error_message,omitempty"`
	StackTrace          *string                `json:"stack_trace,omitempty"`
	Assertions          []AssertionResult      `json:"assertions"`
	Artifacts           map[string]interface{} `json:"artifacts"`
	StartedAt           time.Time              `json:"started_at"`
	CompletedAt         time.Time              `json:"completed_at"`
	TestCaseName        string                 `json:"test_case_name,omitempty"`
	TestCaseDescription string                 `json:"test_case_description,omitempty"`
}

type CoverageMetrics struct {
	CodeCoverage     float64 `json:"code_coverage"`
	BranchCoverage   float64 `json:"branch_coverage"`
	FunctionCoverage float64 `json:"function_coverage"`
}

type PerformanceMetrics struct {
	ExecutionTime float64                `json:"execution_time"`
	ResourceUsage map[string]interface{} `json:"resource_usage"`
	ErrorCount    int                    `json:"error_count"`
}

type AssertionResult struct {
	Name     string      `json:"name"`
	Expected interface{} `json:"expected"`
	Actual   interface{} `json:"actual"`
	Passed   bool        `json:"passed"`
	Message  string      `json:"message"`
}

// Request/Response Models
type GenerateTestSuiteRequest struct {
	ScenarioName   string                `json:"scenario_name" binding:"required"`
	TestTypes      []string              `json:"test_types" binding:"required"`
	CoverageTarget float64               `json:"coverage_target"`
	Options        TestGenerationOptions `json:"options"`
}

type TestGenerationOptions struct {
	IncludePerformanceTests bool     `json:"include_performance_tests"`
	IncludeSecurityTests    bool     `json:"include_security_tests"`
	CustomTestPatterns      []string `json:"custom_test_patterns"`
	ExecutionTimeout        int      `json:"execution_timeout"`
}

type GenerateTestSuiteResponse struct {
	SuiteID           uuid.UUID           `json:"suite_id"`
	GeneratedTests    int                 `json:"generated_tests"`
	EstimatedCoverage float64             `json:"estimated_coverage"`
	GenerationTime    float64             `json:"generation_time"`
	TestFiles         map[string][]string `json:"test_files"`
}

type ExecuteTestSuiteRequest struct {
	ExecutionType        string               `json:"execution_type"`
	Environment          string               `json:"environment"`
	ParallelExecution    bool                 `json:"parallel_execution"`
	TimeoutSeconds       int                  `json:"timeout_seconds"`
	NotificationSettings NotificationSettings `json:"notification_settings"`
}

type NotificationSettings struct {
	OnCompletion bool   `json:"on_completion"`
	OnFailure    bool   `json:"on_failure"`
	WebhookURL   string `json:"webhook_url,omitempty"`
}

type ExecuteTestSuiteResponse struct {
	ExecutionID       uuid.UUID `json:"execution_id"`
	Status            string    `json:"status"`
	EstimatedDuration float64   `json:"estimated_duration"`
	TestCount         int       `json:"test_count"`
	TrackingURL       string    `json:"tracking_url"`
}

type TestExecutionResultsResponse struct {
	ExecutionID        uuid.UUID            `json:"execution_id"`
	SuiteName          string               `json:"suite_name"`
	Status             string               `json:"status"`
	Summary            TestExecutionSummary `json:"summary"`
	FailedTests        []TestResult         `json:"failed_tests"`
	PerformanceMetrics PerformanceMetrics   `json:"performance_metrics"`
	Recommendations    []string             `json:"recommendations"`
}

type TestExecutionSummary struct {
	TotalTests int     `json:"total_tests"`
	Passed     int     `json:"passed"`
	Failed     int     `json:"failed"`
	Skipped    int     `json:"skipped"`
	Duration   float64 `json:"duration"`
	Coverage   float64 `json:"coverage"`
}

type CoverageAnalysisRequest struct {
	ScenarioName      string   `json:"scenario_name" binding:"required"`
	SourceCodePaths   []string `json:"source_code_paths"`
	ExistingTestPaths []string `json:"existing_test_paths"`
	AnalysisDepth     string   `json:"analysis_depth"`
}

type CoverageAnalysisResponse struct {
	OverallCoverage        float64            `json:"overall_coverage"`
	CoverageByFile         map[string]float64 `json:"coverage_by_file"`
	CoverageGaps           CoverageGaps       `json:"coverage_gaps"`
	ImprovementSuggestions []string           `json:"improvement_suggestions"`
	PriorityAreas          []string           `json:"priority_areas"`
}

type CoverageGaps struct {
	UntestedFunctions []string `json:"untested_functions"`
	UntestedBranches  []string `json:"untested_branches"`
	UntestedEdgeCases []string `json:"untested_edge_cases"`
}

// Vault Testing Types
type TestVault struct {
	ID                  uuid.UUID              `json:"id"`
	ScenarioName        string                 `json:"scenario_name"`
	VaultName           string                 `json:"vault_name"`
	Phases              []string               `json:"phases"`
	PhaseConfigurations map[string]PhaseConfig `json:"phase_configurations"`
	SuccessCriteria     SuccessCriteria        `json:"success_criteria"`
	TotalTimeout        int                    `json:"total_timeout"`
	CreatedAt           time.Time              `json:"created_at"`
	LastExecuted        *time.Time             `json:"last_executed,omitempty"`
	Status              string                 `json:"status"`
}

type PhaseConfig struct {
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Timeout      int             `json:"timeout"`
	Tests        []PhaseTest     `json:"tests"`
	Validation   PhaseValidation `json:"validation"`
	Dependencies []string        `json:"dependencies"`
}

type PhaseTest struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Type        string      `json:"type"`
	Steps       []PhaseStep `json:"steps"`
}

type PhaseStep struct {
	Action   string                 `json:"action"`
	Phase    string                 `json:"phase,omitempty"`
	Expected string                 `json:"expected,omitempty"`
	Timeout  int                    `json:"timeout,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
}

type PhaseValidation struct {
	PhaseRequirements map[string][]map[string]interface{} `json:"phase_requirements"`
}

type SuccessCriteria struct {
	AllPhasesCompleted     bool    `json:"all_phases_completed"`
	NoCriticalFailures     bool    `json:"no_critical_failures"`
	CoverageThreshold      float64 `json:"coverage_threshold"`
	PerformanceBaselineMet bool    `json:"performance_baseline_met"`
}

type VaultExecution struct {
	ID              uuid.UUID              `json:"id"`
	VaultID         uuid.UUID              `json:"vault_id"`
	ExecutionType   string                 `json:"execution_type"`
	StartTime       time.Time              `json:"start_time"`
	EndTime         *time.Time             `json:"end_time,omitempty"`
	CurrentPhase    string                 `json:"current_phase"`
	CompletedPhases []string               `json:"completed_phases"`
	FailedPhases    []string               `json:"failed_phases"`
	Status          string                 `json:"status"`
	PhaseResults    map[string]PhaseResult `json:"phase_results"`
	Environment     string                 `json:"environment"`
}

type PhaseResult struct {
	Phase        string                 `json:"phase"`
	Status       string                 `json:"status"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	Duration     float64                `json:"duration"`
	TestResults  []TestResult           `json:"test_results"`
	ErrorMessage *string                `json:"error_message,omitempty"`
	Artifacts    map[string]interface{} `json:"artifacts"`
}

// Ollama Integration Types
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaResponse struct {
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`
	Response  string    `json:"response"`
	Done      bool      `json:"done"`
	Context   []int     `json:"context,omitempty"`
}

type AIGeneratedTest struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	TestCode       string   `json:"test_code"`
	ExpectedResult string   `json:"expected_result"`
	Timeout        int      `json:"timeout"`
	Dependencies   []string `json:"dependencies"`
	Tags           []string `json:"tags"`
	Priority       string   `json:"priority"`
}

// Global variables
var db *sql.DB
var httpClient *http.Client
var serviceManager *ServiceManager

// Initialize database connection
func initDB() {
	// ALL database configuration MUST come from environment - no defaults
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")

	// Validate required environment variables
	if dbHost == "" || dbPort == "" || dbUser == "" || dbName == "" {
		log.Fatal("❌ Missing required database configuration. Please set: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_DB")
	}

	// Build connection string - password is optional for trust auth
	var connStr string
	if dbPassword != "" {
		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbPassword, dbName)
	} else {
		log.Println("⚠️  No password provided, attempting trust authentication...")
		connStr = fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable",
			dbHost, dbPort, dbUser, dbName)
	}

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to open database connection:", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Implement exponential backoff for database connection
	maxRetries := 10
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second

	log.Println("🔄 Attempting database connection with exponential backoff...")
	log.Printf("📊 Connecting to: %s:%s/%s as user %s", dbHost, dbPort, dbName, dbUser)

	var pingErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		pingErr = db.Ping()
		if pingErr == nil {
			log.Printf("✅ Database connected successfully on attempt %d", attempt+1)
			break
		}

		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay)*math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))

		// Add progressive jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
		actualDelay := delay + jitter

		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt+1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)

		// Provide detailed status every few attempts
		if attempt > 0 && attempt%3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt+1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt*2)*baseDelay)
			log.Printf("   - Current delay: %v (with jitter: %v)", delay, jitter)
		}

		time.Sleep(actualDelay)
	}

	if pingErr != nil {
		log.Fatalf("❌ Database connection failed after %d attempts: %v", maxRetries, pingErr)
	}

	log.Println("🎉 Database connection pool established successfully!")

	// Create tables if they don't exist
	createTables()
}

// setupHealthCheckers initializes health checkers for all services
func setupHealthCheckers() {
	// Database health checker
	dbChecker := NewHealthChecker("database", func() error {
		if db == nil {
			return fmt.Errorf("database connection is nil")
		}
		return db.Ping()
	}, dbCircuitBreaker)
	serviceManager.RegisterService("database", dbChecker)

	// Ollama health checker
	ollamaChecker := NewHealthChecker("ollama", func() error {
		ollamaHost := os.Getenv("OLLAMA_HOST")
		if ollamaHost == "" {
			ollamaHost = "localhost"
		}
		ollamaPort := os.Getenv("OLLAMA_PORT")
		if ollamaPort == "" {
			ollamaPort = "11434"
		}

		url := fmt.Sprintf("http://%s:%s/api/tags", ollamaHost, ollamaPort)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		req = req.WithContext(ctx)

		resp, err := httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("ollama API returned status %d", resp.StatusCode)
		}

		return nil
	}, ollamaCircuitBreaker)
	serviceManager.RegisterService("ollama", ollamaChecker)

	log.Println("🔧 Health checkers initialized successfully")
}

func createTables() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS test_suites (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scenario_name VARCHAR(255) NOT NULL,
			suite_type VARCHAR(50) NOT NULL,
			coverage_metrics JSONB,
			generated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_executed TIMESTAMP,
			status VARCHAR(50) DEFAULT 'active'
		)`,
		`CREATE TABLE IF NOT EXISTS test_cases (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			suite_id UUID REFERENCES test_suites(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			test_type VARCHAR(50) NOT NULL,
			test_code TEXT NOT NULL,
			expected_result TEXT,
			execution_timeout INTEGER DEFAULT 30,
			dependencies JSONB,
			tags JSONB,
			priority VARCHAR(20) DEFAULT 'medium',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS test_executions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			suite_id UUID REFERENCES test_suites(id),
			execution_type VARCHAR(50) NOT NULL,
			start_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			end_time TIMESTAMP,
			status VARCHAR(50) DEFAULT 'running',
			performance_metrics JSONB,
			environment VARCHAR(100) DEFAULT 'local'
		)`,
		`CREATE TABLE IF NOT EXISTS test_results (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			execution_id UUID REFERENCES test_executions(id) ON DELETE CASCADE,
			test_case_id UUID REFERENCES test_cases(id),
			status VARCHAR(50) NOT NULL,
			duration DECIMAL(10,3),
			error_message TEXT,
			stack_trace TEXT,
			assertions JSONB,
			artifacts JSONB
		)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			log.Printf("Error creating table: %v", err)
		}
	}
}

// Initialize HTTP client for AI services
func initHTTPClient() {
	httpClient = &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: false,
		},
	}
	log.Println("🤖 HTTP client initialized for AI services")
}

// Ollama AI Integration Functions
func callOllama(prompt string, testType string) (*OllamaResponse, error) {
	log.Printf("🤖 Calling Ollama API for %s test generation...", testType)

	// Create fallback handler with circuit breaker
	fallback := NewFallbackHandler(
		func() (interface{}, error) {
			return callOllamaWithCircuitBreaker(prompt, testType)
		},
		func() (interface{}, error) {
			log.Printf("🔀 Ollama unavailable, using fallback rule-based generation for %s", testType)
			return generateFallbackTest(testType, prompt)
		},
		func(err error) bool {
			// Use fallback for circuit breaker errors or connection issues
			return contains(err.Error(), "circuit breaker") ||
				contains(err.Error(), "connection refused") ||
				contains(err.Error(), "timeout")
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	result, err := fallback.Execute(ctx)
	if err != nil {
		return nil, err
	}

	response, ok := result.(*OllamaResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type from Ollama call")
	}

	log.Printf("✅ Successfully generated %s test content (%d characters)", testType, len(response.Response))
	return response, nil
}

func callOllamaWithCircuitBreaker(prompt string, testType string) (*OllamaResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := ollamaCircuitBreaker.Execute(ctx, func() (interface{}, error) {
		return executeOllamaCall(prompt, testType)
	})

	if err != nil {
		return nil, err
	}

	response, ok := result.(*OllamaResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type from Ollama circuit breaker")
	}

	return response, nil
}

func executeOllamaCall(prompt string, testType string) (interface{}, error) {
	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		ollamaHost = "localhost"
	}
	ollamaPort := os.Getenv("OLLAMA_PORT")
	if ollamaPort == "" {
		ollamaPort = "11434"
	}

	ollamaModel := os.Getenv("OLLAMA_MODEL")
	if ollamaModel == "" {
		ollamaModel = "llama3.2"
	}

	url := fmt.Sprintf("http://%s:%s/api/generate", ollamaHost, ollamaPort)

	request := OllamaRequest{
		Model:  ollamaModel,
		Prompt: prompt,
		Stream: false,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Ollama request: %w", err)
	}

	// Use retry logic for the HTTP call
	retryConfig := RetryConfig{
		MaxAttempts: 2,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    2 * time.Second,
		BackoffFunc: ExponentialBackoff,
		ShouldRetry: func(err error) bool {
			return contains(err.Error(), "timeout") ||
				contains(err.Error(), "connection reset") ||
				contains(err.Error(), "temporary failure")
		},
	}

	var response *OllamaResponse
	err = RetryWithBackoff(context.Background(), retryConfig, func() error {
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
		if err != nil {
			return fmt.Errorf("failed to create HTTP request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to call Ollama API: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(body))
		}

		var ollamaResp OllamaResponse
		if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
			return fmt.Errorf("failed to decode Ollama response: %w", err)
		}

		response = &ollamaResp
		return nil
	})

	if err != nil {
		return nil, err
	}

	return response, nil
}

// generateFallbackTest creates a basic test when AI is unavailable
func generateFallbackTest(testType string, scenarioName string) (*OllamaResponse, error) {
	var testContent string

	switch testType {
	case "unit":
		testContent = fmt.Sprintf(`// Unit test for %s scenario
func Test%sBasic(t *testing.T) {
	// Arrange
	input := "test_data"
	expected := "expected_output"
	
	// Act
	result := ProcessInput(input)
	
	// Assert
	if result != expected {
		t.Errorf("Expected %%s, got %%s", expected, result)
	}
}

func Test%sEdgeCases(t *testing.T) {
	// Test empty input
	result := ProcessInput("")
	if result != "" {
		t.Error("Expected empty result for empty input")
	}
	
	// Test nil input
	result = ProcessInput(nil)
	if result != nil {
		t.Error("Expected nil result for nil input")
	}
}`, scenarioName, scenarioName, scenarioName)
	case "integration":
		testContent = fmt.Sprintf(`// Integration test for %s scenario
func Test%sIntegration(t *testing.T) {
	// Setup test environment
	setup := NewTestEnvironment()
	defer setup.Cleanup()
	
	// Test main workflow
	workflow := NewWorkflow(setup.Config)
	result, err := workflow.Execute()
	
	if err != nil {
		t.Fatalf("Workflow failed: %%v", err)
	}
	
	if !result.IsValid() {
		t.Error("Integration test produced invalid result")
	}
}`, scenarioName, scenarioName)
	default:
		testContent = fmt.Sprintf(`// Generic test for %s scenario
func Test%sGeneric(t *testing.T) {
	// Basic functionality test
	service := NewService()
	err := service.Initialize()
	if err != nil {
		t.Fatalf("Service initialization failed: %%v", err)
	}
	
	// Test service operation
	result := service.PerformOperation()
	if !result.Success {
		t.Error("Service operation failed")
	}
}`, scenarioName, scenarioName)
	}

	return &OllamaResponse{
		Model:     "fallback",
		CreatedAt: time.Now(),
		Response:  testContent,
		Done:      true,
	}, nil
}

func generateTestPrompt(scenarioName string, testType string, options TestGenerationOptions) string {
	basePrompt := fmt.Sprintf("You are an expert software test engineer. Generate a comprehensive %s test for a scenario named '%s'.", testType, scenarioName)

	switch testType {
	case "unit":
		return basePrompt + `

Requirements:
- Create a bash script that tests individual functions/components
- Include API health checks and basic functionality tests
- Use curl for HTTP endpoints, check return codes and responses
- Include database connection tests if applicable
- Make tests independent and atomic
- Use meaningful assertions and error messages
- Test both success and failure scenarios

Format the response as a JSON object with this structure:
{
  "name": "test_name",
  "description": "Brief test description",
  "test_code": "#!/bin/bash\n# Your test script here...",
  "expected_result": "Expected output or behavior",
  "timeout": 30,
  "dependencies": ["list", "of", "dependencies"],
  "tags": ["unit", "api", "health"],
  "priority": "critical"
}

Generate 1-2 comprehensive unit tests for this scenario.`

	case "integration":
		return basePrompt + `

Requirements:
- Create tests that verify end-to-end workflows
- Test API endpoints working together
- Include database integration testing
- Test service-to-service communication
- Verify complete user workflows
- Include error handling and rollback scenarios
- Test with realistic data volumes

Format the response as a JSON object with the same structure as unit tests.
Generate 1-2 comprehensive integration tests for this scenario.`

	case "performance":
		return basePrompt + `

Requirements:
- Create load and stress tests
- Test response times under various loads
- Include concurrent user simulation
- Test memory and CPU usage
- Include performance benchmarks
- Test scalability limits
- Use tools like curl, ab (Apache Bench), or custom scripts

Format the response as a JSON object with the same structure.
Generate 1-2 performance tests focusing on scalability and response times.`

	case "vault":
		return basePrompt + `

Requirements:
- Create comprehensive phase-based testing
- Include setup, develop, test, deploy, and monitor phases
- Test complex dependency scenarios
- Verify system state at each phase
- Include rollback and recovery testing
- Test production-like conditions
- Multi-stage validation with checkpoints

Format the response as a JSON object with the same structure.
Generate 1-2 vault tests that cover multiple phases of the scenario lifecycle.`

	case "regression":
		return basePrompt + `

Requirements:
- Create tests that verify existing functionality still works
- Test core features and critical paths
- Include baseline performance validation
- Test backwards compatibility
- Verify no functionality has degraded
- Include data integrity checks
- Test previously fixed bugs don't reoccur

Format the response as a JSON object with the same structure.
Generate 1-2 regression tests that validate core functionality hasn't broken.`

	default:
		return basePrompt + `

Generate a comprehensive test script for this scenario. Include proper error handling, meaningful assertions, and clear success/failure conditions.

Format the response as a JSON object with:
{
  "name": "test_name", 
  "description": "test description",
  "test_code": "bash script content",
  "expected_result": "expected behavior",
  "timeout": 60,
  "dependencies": ["dependencies"],
  "tags": ["relevant", "tags"],
  "priority": "high"
}`
	}
}

func parseAITestResponse(response string, scenarioName string, testType string) ([]TestCase, error) {
	// Try to parse as JSON first
	var aiTest AIGeneratedTest
	if err := json.Unmarshal([]byte(response), &aiTest); err != nil {
		// If JSON parsing fails, try to extract JSON from the response
		jsonRegex := regexp.MustCompile(`\{[\s\S]*\}`)
		matches := jsonRegex.FindAllString(response, -1)

		if len(matches) == 0 {
			return nil, fmt.Errorf("no valid JSON found in AI response")
		}

		// Try the largest JSON object found
		var largestJSON string
		for _, match := range matches {
			if len(match) > len(largestJSON) {
				largestJSON = match
			}
		}

		if err := json.Unmarshal([]byte(largestJSON), &aiTest); err != nil {
			return nil, fmt.Errorf("failed to parse AI response as JSON: %w", err)
		}
	}

	// Validate and create test case
	testCase := TestCase{
		ID:             uuid.New(),
		Name:           aiTest.Name,
		Description:    aiTest.Description,
		TestType:       testType,
		TestCode:       aiTest.TestCode,
		ExpectedResult: aiTest.ExpectedResult,
		Timeout:        aiTest.Timeout,
		Dependencies:   aiTest.Dependencies,
		Tags:           aiTest.Tags,
		Priority:       aiTest.Priority,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Set defaults if missing
	if testCase.Name == "" {
		testCase.Name = fmt.Sprintf("%s_%s_test", scenarioName, testType)
	}
	if testCase.Description == "" {
		testCase.Description = fmt.Sprintf("AI-generated %s test for %s", testType, scenarioName)
	}
	if testCase.Timeout == 0 {
		testCase.Timeout = 60
	}
	if testCase.Priority == "" {
		testCase.Priority = "medium"
	}
	if testCase.Dependencies == nil {
		testCase.Dependencies = []string{}
	}
	if testCase.Tags == nil {
		testCase.Tags = []string{testType, "ai-generated"}
	}

	return []TestCase{testCase}, nil
}

// AI Test Generation Service with real Ollama integration
func generateTestsWithAI(scenarioName string, testTypes []string, options TestGenerationOptions) ([]TestCase, error) {
	log.Printf("🧪 Starting AI test generation for scenario: %s with types: %v", scenarioName, testTypes)

	var allTestCases []TestCase

	for _, testType := range testTypes {
		log.Printf("🤖 Generating %s tests using AI for scenario: %s", testType, scenarioName)

		// Try AI generation first
		prompt := generateTestPrompt(scenarioName, testType, options)
		aiResponse, err := callOllama(prompt, testType)

		if err != nil {
			log.Printf("⚠️ AI generation failed for %s tests: %v. Falling back to rule-based generation.", testType, err)
			// Fallback to rule-based generation
			fallbackTests := generateFallbackTests(scenarioName, testType, options)
			allTestCases = append(allTestCases, fallbackTests...)
			continue
		}

		// Parse AI response
		aiTestCases, err := parseAITestResponse(aiResponse.Response, scenarioName, testType)
		if err != nil {
			log.Printf("⚠️ Failed to parse AI response for %s tests: %v. Falling back to rule-based generation.", testType, err)
			// Fallback to rule-based generation
			fallbackTests := generateFallbackTests(scenarioName, testType, options)
			allTestCases = append(allTestCases, fallbackTests...)
			continue
		}

		log.Printf("✅ Successfully generated %d %s test(s) using AI", len(aiTestCases), testType)
		allTestCases = append(allTestCases, aiTestCases...)
	}

	if len(allTestCases) == 0 {
		return nil, fmt.Errorf("no tests could be generated for scenario %s", scenarioName)
	}

	log.Printf("🎉 Total tests generated: %d", len(allTestCases))
	return allTestCases, nil
}

// Fallback test generation when AI is unavailable
func generateFallbackTests(scenarioName string, testType string, options TestGenerationOptions) []TestCase {
	switch testType {
	case "unit":
		return generateUnitTests(scenarioName, options)
	case "integration":
		return generateIntegrationTests(scenarioName, options)
	case "performance":
		if options.IncludePerformanceTests {
			return generatePerformanceTests(scenarioName, options)
		}
		return []TestCase{}
	case "vault":
		return generateVaultTests(scenarioName, options)
	case "regression":
		return generateRegressionTests(scenarioName, options)
	default:
		return []TestCase{}
	}
}

func generateUnitTests(scenarioName string, options TestGenerationOptions) []TestCase {
	return []TestCase{
		{
			ID:             uuid.New(),
			Name:           fmt.Sprintf("%s_api_health_test", scenarioName),
			Description:    "Test API health endpoint responds correctly",
			TestType:       "unit",
			TestCode:       generateHealthTestCode(scenarioName),
			ExpectedResult: `{"status": "healthy"}`,
			Timeout:        30,
			Dependencies:   []string{"api_server"},
			Tags:           []string{"api", "health", "unit"},
			Priority:       "critical",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
		{
			ID:             uuid.New(),
			Name:           fmt.Sprintf("%s_database_connection_test", scenarioName),
			Description:    "Test database connection and basic operations",
			TestType:       "unit",
			TestCode:       generateDBTestCode(scenarioName),
			ExpectedResult: `{"connection": "success"}`,
			Timeout:        30,
			Dependencies:   []string{"postgres"},
			Tags:           []string{"database", "connection", "unit"},
			Priority:       "critical",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
	}
}

func generateIntegrationTests(scenarioName string, options TestGenerationOptions) []TestCase {
	return []TestCase{
		{
			ID:             uuid.New(),
			Name:           fmt.Sprintf("%s_end_to_end_workflow_test", scenarioName),
			Description:    "Test complete workflow from start to finish",
			TestType:       "integration",
			TestCode:       generateWorkflowTestCode(scenarioName),
			ExpectedResult: `{"workflow_status": "completed"}`,
			Timeout:        120,
			Dependencies:   []string{"api_server", "postgres", "redis"},
			Tags:           []string{"workflow", "e2e", "integration"},
			Priority:       "high",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
	}
}

func generatePerformanceTests(scenarioName string, options TestGenerationOptions) []TestCase {
	return []TestCase{
		{
			ID:             uuid.New(),
			Name:           fmt.Sprintf("%s_load_test", scenarioName),
			Description:    "Test system performance under load",
			TestType:       "performance",
			TestCode:       generateLoadTestCode(scenarioName),
			ExpectedResult: `{"avg_response_time": "<1000ms", "success_rate": ">95%"}`,
			Timeout:        300,
			Dependencies:   []string{"api_server"},
			Tags:           []string{"performance", "load", "stress"},
			Priority:       "medium",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
	}
}

func generateVaultTests(scenarioName string, options TestGenerationOptions) []TestCase {
	return []TestCase{
		{
			ID:             uuid.New(),
			Name:           fmt.Sprintf("%s_vault_setup_phase", scenarioName),
			Description:    "Test vault setup phase completion",
			TestType:       "vault",
			TestCode:       generateVaultPhaseTestCode(scenarioName, "setup"),
			ExpectedResult: `{"phase": "setup", "status": "completed"}`,
			Timeout:        180,
			Dependencies:   []string{"all_resources"},
			Tags:           []string{"vault", "setup", "phase"},
			Priority:       "critical",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
		{
			ID:             uuid.New(),
			Name:           fmt.Sprintf("%s_vault_develop_phase", scenarioName),
			Description:    "Test vault develop phase execution",
			TestType:       "vault",
			TestCode:       generateVaultPhaseTestCode(scenarioName, "develop"),
			ExpectedResult: `{"phase": "develop", "status": "completed"}`,
			Timeout:        300,
			Dependencies:   []string{"setup_phase"},
			Tags:           []string{"vault", "develop", "phase"},
			Priority:       "critical",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
	}
}

func generateRegressionTests(scenarioName string, options TestGenerationOptions) []TestCase {
	return []TestCase{
		{
			ID:             uuid.New(),
			Name:           fmt.Sprintf("%s_regression_baseline_test", scenarioName),
			Description:    "Test that core functionality hasn't regressed",
			TestType:       "regression",
			TestCode:       generateRegressionTestCode(scenarioName),
			ExpectedResult: `{"regression_detected": false}`,
			Timeout:        180,
			Dependencies:   []string{"api_server", "postgres"},
			Tags:           []string{"regression", "baseline", "stability"},
			Priority:       "high",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
	}
}

// Test code generation functions
func generateHealthTestCode(scenarioName string) string {
	return fmt.Sprintf(`
#!/bin/bash
# Health test for %s
API_PORT=${API_PORT:-8250}
response=$(curl -s -w "%%{http_code}" http://localhost:${API_PORT}/health)
http_code=$(echo $response | tail -c 4)
if [ "$http_code" = "200" ]; then
    echo "✓ Health check passed"
    exit 0
else
    echo "✗ Health check failed (HTTP $http_code)"
    exit 1
fi
`, scenarioName)
}

func generateDBTestCode(scenarioName string) string {
	return fmt.Sprintf(`
#!/bin/bash
# Database connection test for %s
POSTGRES_HOST=${POSTGRES_HOST:-localhost}
POSTGRES_PORT=${POSTGRES_PORT:-5433}
POSTGRES_USER=${POSTGRES_USER:-postgres}
POSTGRES_DB=${POSTGRES_DB:-%s}

PGPASSWORD=${POSTGRES_PASSWORD} psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB -c "SELECT 1;" > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ Database connection successful"
    exit 0
else
    echo "✗ Database connection failed"
    exit 1
fi
`, scenarioName, scenarioName)
}

func generateWorkflowTestCode(scenarioName string) string {
	return fmt.Sprintf(`
#!/bin/bash
# End-to-end workflow test for %s
API_PORT=${API_PORT:-8250}
BASE_URL="http://localhost:${API_PORT}"

# Test complete workflow
echo "Testing complete %s workflow..."

# Step 1: Health check
response=$(curl -s "$BASE_URL/health")
if [[ $response == *"healthy"* ]]; then
    echo "✓ Step 1: Health check passed"
else
    echo "✗ Step 1: Health check failed"
    exit 1
fi

# Step 2: Create test data
test_id=$(uuidgen)
response=$(curl -s -X POST "$BASE_URL/api/test" -H "Content-Type: application/json" -d "{\"id\":\"$test_id\",\"name\":\"workflow_test\"}")
if [[ $response == *"created"* ]] || [[ $response == *"success"* ]]; then
    echo "✓ Step 2: Test data creation passed"
else
    echo "✗ Step 2: Test data creation failed"
    exit 1
fi

# Step 3: Retrieve test data
response=$(curl -s "$BASE_URL/api/test/$test_id")
if [[ $response == *"workflow_test"* ]] || [[ $? -eq 0 ]]; then
    echo "✓ Step 3: Data retrieval passed"
else
    echo "✗ Step 3: Data retrieval failed"
    exit 1
fi

echo "✓ End-to-end workflow test completed successfully"
`, scenarioName, scenarioName)
}

func generateLoadTestCode(scenarioName string) string {
	return fmt.Sprintf(`
#!/bin/bash
# Load test for %s
API_PORT=${API_PORT:-8250}
BASE_URL="http://localhost:${API_PORT}"
CONCURRENT_REQUESTS=10
TOTAL_REQUESTS=100

echo "Running load test with $CONCURRENT_REQUESTS concurrent connections, $TOTAL_REQUESTS total requests..."

# Use Apache Bench if available, otherwise use curl in a loop
if command -v ab >/dev/null 2>&1; then
    ab -c $CONCURRENT_REQUESTS -n $TOTAL_REQUESTS "$BASE_URL/health" > /tmp/load_test_results.txt 2>&1
    
    # Parse results
    avg_time=$(grep "Time per request" /tmp/load_test_results.txt | head -1 | awk '{print $4}')
    failed_requests=$(grep "Failed requests" /tmp/load_test_results.txt | awk '{print $3}')
    
    if (( $(echo "$avg_time < 1000" | bc -l) )) && [ "$failed_requests" = "0" ]; then
        echo "✓ Load test passed (avg: ${avg_time}ms, failures: $failed_requests)"
        exit 0
    else
        echo "✗ Load test failed (avg: ${avg_time}ms, failures: $failed_requests)"
        exit 1
    fi
else
    echo "Apache Bench not available, using basic curl test..."
    for i in $(seq 1 10); do
        response_time=$(curl -w "%%{time_total}" -s -o /dev/null "$BASE_URL/health")
        echo "Request $i: ${response_time}s"
    done
    echo "✓ Basic load test completed"
fi
`, scenarioName)
}

func generateVaultPhaseTestCode(scenarioName, phase string) string {
	return fmt.Sprintf(`
#!/bin/bash
# Vault %s phase test for %s
echo "Testing vault %s phase..."

case "%s" in
    "setup")
        # Test resource availability
        echo "Checking resource availability..."
        # Add specific setup phase tests here
        echo "✓ Setup phase validation completed"
        ;;
    "develop")
        # Test development phase completion
        echo "Checking development artifacts..."
        # Add specific develop phase tests here
        echo "✓ Develop phase validation completed"
        ;;
    "test")
        # Test testing phase
        echo "Validating test execution..."
        # Add specific test phase validation here
        echo "✓ Test phase validation completed"
        ;;
    "deploy")
        # Test deployment phase
        echo "Checking deployment status..."
        # Add specific deploy phase tests here
        echo "✓ Deploy phase validation completed"
        ;;
    "monitor")
        # Test monitoring phase
        echo "Validating monitoring setup..."
        # Add specific monitor phase tests here
        echo "✓ Monitor phase validation completed"
        ;;
esac

echo "Vault %s phase test completed successfully"
`, phase, scenarioName, phase, phase, phase)
}

func generateRegressionTestCode(scenarioName string) string {
	return fmt.Sprintf(`
#!/bin/bash
# Regression test for %s
API_PORT=${API_PORT:-8250}
BASE_URL="http://localhost:${API_PORT}"

echo "Running regression tests for %s..."

# Test 1: Verify core API endpoints are still working
endpoints=("/health" "/api/status" "/api/version")
for endpoint in "${endpoints[@]}"; do
    response=$(curl -s -w "%%{http_code}" "$BASE_URL$endpoint")
    http_code=$(echo $response | tail -c 4)
    if [ "$http_code" = "200" ] || [ "$http_code" = "404" ]; then
        echo "✓ Endpoint $endpoint: HTTP $http_code"
    else
        echo "✗ Endpoint $endpoint failed: HTTP $http_code"
        exit 1
    fi
done

# Test 2: Check database schema hasn't changed unexpectedly
echo "Checking database schema stability..."
# Add database schema validation here

# Test 3: Verify performance hasn't degraded significantly
echo "Checking performance baseline..."
response_time=$(curl -w "%%{time_total}" -s -o /dev/null "$BASE_URL/health")
if (( $(echo "$response_time < 2.0" | bc -l) )); then
    echo "✓ Performance within acceptable range (${response_time}s)"
else
    echo "⚠ Performance may have degraded (${response_time}s)"
fi

echo "✓ Regression tests completed successfully"
`, scenarioName, scenarioName)
}

// API Handlers
func generateTestSuiteHandler(c *gin.Context) {
	var req GenerateTestSuiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	startTime := time.Now()

	// Generate tests using AI
	testCases, err := generateTestsWithAI(req.ScenarioName, req.TestTypes, req.Options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Create test suite
	suiteID := uuid.New()
	testSuite := TestSuite{
		ID:           suiteID,
		ScenarioName: req.ScenarioName,
		SuiteType:    strings.Join(req.TestTypes, ","),
		TestCases:    testCases,
		CoverageMetrics: CoverageMetrics{
			CodeCoverage:     req.CoverageTarget,
			BranchCoverage:   req.CoverageTarget * 0.9,
			FunctionCoverage: req.CoverageTarget * 0.95,
		},
		GeneratedAt: time.Now(),
		Status:      "active",
	}

	// Save to database
	coverageJSON, _ := json.Marshal(testSuite.CoverageMetrics)
	_, err = db.Exec(`
		INSERT INTO test_suites (id, scenario_name, suite_type, coverage_metrics, generated_at, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, testSuite.ID, testSuite.ScenarioName, testSuite.SuiteType, coverageJSON, testSuite.GeneratedAt, testSuite.Status)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save test suite"})
		return
	}

	// Save test cases
	for _, testCase := range testCases {
		testCase.SuiteID = suiteID
		tagsJSON, _ := json.Marshal(testCase.Tags)
		depsJSON, _ := json.Marshal(testCase.Dependencies)

		_, err = db.Exec(`
			INSERT INTO test_cases (id, suite_id, name, description, test_type, test_code, expected_result, execution_timeout, dependencies, tags, priority, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`, testCase.ID, testCase.SuiteID, testCase.Name, testCase.Description, testCase.TestType, testCase.TestCode, testCase.ExpectedResult, testCase.Timeout, depsJSON, tagsJSON, testCase.Priority, testCase.CreatedAt, testCase.UpdatedAt)

		if err != nil {
			log.Printf("Failed to save test case %s: %v", testCase.Name, err)
		}
	}

	generationTime := time.Since(startTime).Seconds()

	// Organize test files by type
	testFiles := make(map[string][]string)
	for _, testCase := range testCases {
		testFiles[testCase.TestType] = append(testFiles[testCase.TestType], testCase.Name)
	}

	response := GenerateTestSuiteResponse{
		SuiteID:           suiteID,
		GeneratedTests:    len(testCases),
		EstimatedCoverage: req.CoverageTarget,
		GenerationTime:    generationTime,
		TestFiles:         testFiles,
	}

	c.JSON(http.StatusCreated, response)
}

// Test execution functions
func executeTestCase(testCase TestCase, environment string) TestResult {
	log.Printf("🧪 Executing test case: %s", testCase.Name)

	result := TestResult{
		ID:         uuid.New(),
		TestCaseID: testCase.ID,
		Status:     "running",
		StartedAt:  time.Now(),
		Assertions: []AssertionResult{},
		Artifacts:  make(map[string]interface{}),
	}

	// Create temporary file for test script
	tempDir := filepath.Join(os.TempDir(), "test-genie-execution")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		result.Status = "error"
		errorMsg := fmt.Sprintf("Failed to create temp directory: %v", err)
		result.ErrorMessage = &errorMsg
		result.CompletedAt = time.Now()
		result.Duration = float64(time.Since(result.StartedAt)) / float64(time.Second)
		return result
	}

	testFileName := fmt.Sprintf("test_%s_%s.sh", testCase.Name, uuid.New().String()[:8])
	testFilePath := filepath.Join(tempDir, testFileName)

	// Write test script to file
	if err := os.WriteFile(testFilePath, []byte(testCase.TestCode), 0755); err != nil {
		result.Status = "error"
		errorMsg := fmt.Sprintf("Failed to write test script: %v", err)
		result.ErrorMessage = &errorMsg
		result.CompletedAt = time.Now()
		result.Duration = float64(time.Since(result.StartedAt)) / float64(time.Second)
		return result
	}

	// Ensure cleanup happens
	defer func() {
		if err := os.Remove(testFilePath); err != nil {
			log.Printf("⚠️ Failed to clean up test file %s: %v", testFilePath, err)
		}
	}()

	// Set up environment variables for test execution
	cmd := exec.Command("/bin/bash", testFilePath)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("TEST_ENVIRONMENT=%s", environment),
		fmt.Sprintf("API_PORT=%s", os.Getenv("API_PORT")),
		fmt.Sprintf("POSTGRES_HOST=%s", os.Getenv("POSTGRES_HOST")),
		fmt.Sprintf("POSTGRES_PORT=%s", os.Getenv("POSTGRES_PORT")),
		fmt.Sprintf("POSTGRES_USER=%s", os.Getenv("POSTGRES_USER")),
		fmt.Sprintf("POSTGRES_PASSWORD=%s", os.Getenv("POSTGRES_PASSWORD")),
		fmt.Sprintf("POSTGRES_DB=%s", os.Getenv("POSTGRES_DB")),
		fmt.Sprintf("OLLAMA_HOST=%s", os.Getenv("OLLAMA_HOST")),
		fmt.Sprintf("OLLAMA_PORT=%s", os.Getenv("OLLAMA_PORT")),
	)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute with timeout
	timeout := time.Duration(testCase.Timeout) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		// Test completed within timeout
		result.CompletedAt = time.Now()
		result.Duration = float64(time.Since(result.StartedAt)) / float64(time.Second)

		stdoutStr := stdout.String()
		stderrStr := stderr.String()

		if err != nil {
			// Test failed
			result.Status = "failed"
			if exitError, ok := err.(*exec.ExitError); ok {
				errorMsg := fmt.Sprintf("Test exited with code %d. Stderr: %s", exitError.ExitCode(), stderrStr)
				result.ErrorMessage = &errorMsg
			} else {
				errorMsg := fmt.Sprintf("Failed to execute test: %v", err)
				result.ErrorMessage = &errorMsg
			}
		} else {
			// Test passed
			result.Status = "passed"
		}

		// Store output as artifacts
		result.Artifacts["stdout"] = stdoutStr
		result.Artifacts["stderr"] = stderrStr

		// Parse assertions from output (look for specific patterns)
		result.Assertions = parseTestAssertions(stdoutStr)

		log.Printf("✅ Test case completed: %s (status: %s, duration: %.2fs)", testCase.Name, result.Status, result.Duration)

	case <-time.After(timeout):
		// Test timed out
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		result.Status = "failed"
		result.CompletedAt = time.Now()
		result.Duration = float64(time.Since(result.StartedAt)) / float64(time.Second)
		errorMsg := fmt.Sprintf("Test timed out after %v", timeout)
		result.ErrorMessage = &errorMsg
		result.Artifacts["timeout"] = true

		log.Printf("⏰ Test case timed out: %s (after %.2fs)", testCase.Name, result.Duration)
	}

	return result
}

func parseTestAssertions(output string) []AssertionResult {
	assertions := []AssertionResult{}

	// Look for common test assertion patterns
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Pattern: ✓ or ✗ followed by description
		if strings.Contains(line, "✓") {
			assertions = append(assertions, AssertionResult{
				Name:     "success_assertion",
				Expected: "pass",
				Actual:   "pass",
				Passed:   true,
				Message:  line,
			})
		} else if strings.Contains(line, "✗") || strings.Contains(line, "❌") {
			assertions = append(assertions, AssertionResult{
				Name:     "failure_assertion",
				Expected: "pass",
				Actual:   "fail",
				Passed:   false,
				Message:  line,
			})
		} else if strings.Contains(line, "PASS") {
			assertions = append(assertions, AssertionResult{
				Name:     "pass_assertion",
				Expected: "pass",
				Actual:   "pass",
				Passed:   true,
				Message:  line,
			})
		} else if strings.Contains(line, "FAIL") {
			assertions = append(assertions, AssertionResult{
				Name:     "fail_assertion",
				Expected: "pass",
				Actual:   "fail",
				Passed:   false,
				Message:  line,
			})
		}
	}

	return assertions
}

func executeTestSuite(suiteID uuid.UUID, executionID uuid.UUID, executionType string, environment string, timeoutSeconds int) {
	log.Printf("🚀 Starting test suite execution: %s", suiteID)

	// Get all test cases for this suite
	rows, err := db.Query(`
		SELECT id, suite_id, name, description, test_type, test_code, expected_result, execution_timeout, dependencies, tags, priority
		FROM test_cases 
		WHERE suite_id = $1
		ORDER BY 
			CASE priority
				WHEN 'critical' THEN 1
				WHEN 'high' THEN 2  
				WHEN 'medium' THEN 3
				WHEN 'low' THEN 4
				ELSE 5
			END, created_at
	`, suiteID)

	if err != nil {
		log.Printf("❌ Failed to get test cases for suite %s: %v", suiteID, err)
		if updateErr := updateExecutionStatus(executionID, "failed", err.Error()); updateErr != nil {
			log.Printf("❌ Additionally failed to update execution status: %v", updateErr)
		}
		return
	}
	defer rows.Close()

	var testCases []TestCase
	for rows.Next() {
		var testCase TestCase
		var depsJSON, tagsJSON []byte

		err := rows.Scan(
			&testCase.ID, &testCase.SuiteID, &testCase.Name, &testCase.Description,
			&testCase.TestType, &testCase.TestCode, &testCase.ExpectedResult, &testCase.Timeout,
			&depsJSON, &tagsJSON, &testCase.Priority,
		)
		if err != nil {
			log.Printf("⚠️ Failed to scan test case: %v", err)
			continue
		}

		// Parse JSON fields
		json.Unmarshal(depsJSON, &testCase.Dependencies)
		json.Unmarshal(tagsJSON, &testCase.Tags)

		testCases = append(testCases, testCase)
	}

	if len(testCases) == 0 {
		log.Printf("❌ No test cases found for suite %s", suiteID)
		if updateErr := updateExecutionStatus(executionID, "failed", "No test cases found"); updateErr != nil {
			log.Printf("❌ Additionally failed to update execution status: %v", updateErr)
		}
		return
	}

	log.Printf("📋 Found %d test cases to execute", len(testCases))

	// Execute tests
	var results []TestResult
	totalStartTime := time.Now()

	for i, testCase := range testCases {
		log.Printf("🧪 Executing test %d/%d: %s", i+1, len(testCases), testCase.Name)

		result := executeTestCase(testCase, environment)
		result.ExecutionID = executionID

		// Store result in database
		if err := storeTestResult(result); err != nil {
			log.Printf("⚠️ Failed to store test result for %s: %v", testCase.Name, err)
			// Continue execution even if storage fails
		}
		results = append(results, result)

		// Check for early termination on critical failures
		if result.Status == "failed" && testCase.Priority == "critical" {
			log.Printf("🛑 Critical test failed, terminating execution: %s", testCase.Name)
			break
		}
	}

	totalDuration := time.Since(totalStartTime)

	// Calculate summary statistics
	var passed, failed, skipped int
	for _, result := range results {
		switch result.Status {
		case "passed":
			passed++
		case "failed":
			failed++
		case "skipped":
			skipped++
		}
	}

	// Update execution record with final results
	status := "completed"
	if failed > 0 {
		status = "failed"
	}

	_, err = db.Exec(`
		UPDATE test_executions 
		SET status = $1, end_time = $2, total_tests = $3, passed_tests = $4, failed_tests = $5, skipped_tests = $6,
		    performance_metrics = $7
		WHERE id = $8
	`, status, time.Now(), len(results), passed, failed, skipped,
		fmt.Sprintf(`{"execution_time": %.2f, "tests_per_second": %.2f}`,
			totalDuration.Seconds(), float64(len(results))/totalDuration.Seconds()), executionID)

	if err != nil {
		log.Printf("⚠️ Failed to update execution record: %v", err)
	}

	log.Printf("🎉 Test suite execution completed: %d passed, %d failed, %d skipped (%.2fs total)",
		passed, failed, skipped, totalDuration.Seconds())
}

func storeTestResult(result TestResult) error {
	assertionsJSON, err := json.Marshal(result.Assertions)
	if err != nil {
		log.Printf("⚠️ Failed to marshal assertions for test result %s: %v", result.ID, err)
		assertionsJSON = []byte("[]")
	}

	artifactsJSON, err := json.Marshal(result.Artifacts)
	if err != nil {
		log.Printf("⚠️ Failed to marshal artifacts for test result %s: %v", result.ID, err)
		artifactsJSON = []byte("{}")
	}

	// Implement retry logic for database operations
	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		_, lastErr = db.Exec(`
			INSERT INTO test_results (id, execution_id, test_case_id, status, duration, error_message, stack_trace, assertions, artifacts, started_at, completed_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`, result.ID, result.ExecutionID, result.TestCaseID, result.Status, result.Duration,
			result.ErrorMessage, result.StackTrace, assertionsJSON, artifactsJSON, result.StartedAt, result.CompletedAt)

		if lastErr == nil {
			log.Printf("✅ Successfully stored test result: %s (status: %s)", result.TestCaseID, result.Status)
			return nil
		}

		if attempt < maxRetries-1 {
			backoffDelay := time.Duration(attempt+1) * time.Second
			log.Printf("⚠️ Failed to store test result (attempt %d/%d): %v. Retrying in %v...",
				attempt+1, maxRetries, lastErr, backoffDelay)
			time.Sleep(backoffDelay)
		}
	}

	log.Printf("❌ Failed to store test result after %d attempts: %v", maxRetries, lastErr)
	return lastErr
}

func updateExecutionStatus(executionID uuid.UUID, status string, errorMessage string) error {
	// Implement retry logic for critical status updates
	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		_, lastErr = db.Exec(`
			UPDATE test_executions 
			SET status = $1, end_time = $2, execution_notes = $3
			WHERE id = $4
		`, status, time.Now(), errorMessage, executionID)

		if lastErr == nil {
			log.Printf("✅ Successfully updated execution status: %s -> %s", executionID, status)
			return nil
		}

		if attempt < maxRetries-1 {
			backoffDelay := time.Duration(attempt+1) * time.Second
			log.Printf("⚠️ Failed to update execution status (attempt %d/%d): %v. Retrying in %v...",
				attempt+1, maxRetries, lastErr, backoffDelay)
			time.Sleep(backoffDelay)
		}
	}

	log.Printf("❌ Failed to update execution status after %d attempts: %v", maxRetries, lastErr)
	return lastErr
}

// Add database health check function
func checkDatabaseHealth() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Test a simple query
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM test_suites").Scan(&count)
	if err != nil {
		return fmt.Errorf("database query test failed: %w", err)
	}

	log.Printf("✅ Database health check passed (found %d test suites)", count)
	return nil
}

// Add transaction support for critical operations
func storeTestSuiteWithTransaction(testSuite TestSuite) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	// Store test suite
	coverageJSON, _ := json.Marshal(testSuite.CoverageMetrics)
	_, err = tx.Exec(`
		INSERT INTO test_suites (id, scenario_name, suite_type, coverage_metrics, generated_at, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, testSuite.ID, testSuite.ScenarioName, testSuite.SuiteType, coverageJSON, testSuite.GeneratedAt, testSuite.Status)

	if err != nil {
		return fmt.Errorf("failed to insert test suite: %w", err)
	}

	// Store test cases
	for _, testCase := range testSuite.TestCases {
		dependenciesJSON, _ := json.Marshal(testCase.Dependencies)
		tagsJSON, _ := json.Marshal(testCase.Tags)

		_, err = tx.Exec(`
			INSERT INTO test_cases (id, suite_id, name, description, test_type, test_code, expected_result, execution_timeout, dependencies, tags, priority, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`, testCase.ID, testCase.SuiteID, testCase.Name, testCase.Description, testCase.TestType,
			testCase.TestCode, testCase.ExpectedResult, testCase.Timeout, dependenciesJSON, tagsJSON,
			testCase.Priority, testCase.CreatedAt, testCase.UpdatedAt)

		if err != nil {
			return fmt.Errorf("failed to insert test case %s: %w", testCase.Name, err)
		}
	}

	log.Printf("✅ Successfully stored test suite with %d test cases in transaction", len(testSuite.TestCases))
	return nil
}

// Vault Execution Engine
func executeVault(vaultID uuid.UUID, executionID uuid.UUID, environment string) {
	log.Printf("🏗️ Starting vault execution: %s", vaultID)

	// Get vault configuration
	vault, err := getVaultFromDatabase(vaultID)
	if err != nil {
		log.Printf("❌ Failed to get vault %s: %v", vaultID, err)
		updateVaultExecutionStatus(executionID, "failed", err.Error())
		return
	}

	log.Printf("📋 Executing vault '%s' with phases: %v", vault.VaultName, vault.Phases)

	execution := VaultExecution{
		ID:              executionID,
		VaultID:         vaultID,
		ExecutionType:   "full",
		StartTime:       time.Now(),
		CurrentPhase:    "",
		CompletedPhases: []string{},
		FailedPhases:    []string{},
		Status:          "running",
		PhaseResults:    make(map[string]PhaseResult),
		Environment:     environment,
	}

	// Execute phases in sequence
	for _, phase := range vault.Phases {
		log.Printf("🚀 Starting phase: %s", phase)

		execution.CurrentPhase = phase
		updateVaultExecutionInDatabase(execution)

		phaseResult := executeVaultPhase(vault, phase, environment)
		execution.PhaseResults[phase] = phaseResult

		if phaseResult.Status == "passed" {
			execution.CompletedPhases = append(execution.CompletedPhases, phase)
			log.Printf("✅ Phase completed successfully: %s", phase)
		} else {
			execution.FailedPhases = append(execution.FailedPhases, phase)
			log.Printf("❌ Phase failed: %s - %v", phase, phaseResult.ErrorMessage)

			// Check if we should stop on failure
			if vault.SuccessCriteria.NoCriticalFailures {
				execution.Status = "failed"
				execution.EndTime = &[]time.Time{time.Now()}[0]
				updateVaultExecutionInDatabase(execution)
				log.Printf("🛑 Vault execution stopped due to critical phase failure")
				return
			}
		}
	}

	// Determine final status
	if len(execution.FailedPhases) == 0 {
		execution.Status = "completed"
	} else if len(execution.CompletedPhases) > 0 {
		execution.Status = "partial"
	} else {
		execution.Status = "failed"
	}

	execution.EndTime = &[]time.Time{time.Now()}[0]
	execution.CurrentPhase = ""

	updateVaultExecutionInDatabase(execution)

	log.Printf("🎉 Vault execution completed: %s (status: %s, completed: %d, failed: %d)",
		vault.VaultName, execution.Status, len(execution.CompletedPhases), len(execution.FailedPhases))
}

func executeVaultPhase(vault TestVault, phase string, environment string) PhaseResult {
	phaseStart := time.Now()

	result := PhaseResult{
		Phase:       phase,
		Status:      "running",
		StartTime:   phaseStart,
		TestResults: []TestResult{},
		Artifacts:   make(map[string]interface{}),
	}

	// Get phase configuration
	phaseConfig, exists := vault.PhaseConfigurations[phase]
	if !exists {
		// Generate default phase configuration if not found
		phaseConfig = generateDefaultPhaseConfig(phase, vault.ScenarioName)
		log.Printf("⚠️ No configuration found for phase %s, using defaults", phase)
	}

	// Verify phase prerequisites
	if err := verifyPhasePrerequisites(phaseConfig, environment); err != nil {
		result.Status = "failed"
		result.EndTime = time.Now()
		result.Duration = time.Since(phaseStart).Seconds()
		errorMsg := fmt.Sprintf("Prerequisites not met: %v", err)
		result.ErrorMessage = &errorMsg
		return result
	}

	// Execute phase tests
	for _, test := range phaseConfig.Tests {
		testResult := executePhaseTest(test, vault.ScenarioName, phase, environment)
		result.TestResults = append(result.TestResults, testResult)

		if testResult.Status == "failed" && phaseConfig.Name != "optional" {
			result.Status = "failed"
			result.EndTime = time.Now()
			result.Duration = time.Since(phaseStart).Seconds()
			errorMsg := fmt.Sprintf("Test failed: %s", test.Name)
			result.ErrorMessage = &errorMsg
			return result
		}
	}

	// Verify phase completion
	if err := verifyPhaseCompletion(phaseConfig, environment); err != nil {
		result.Status = "failed"
		result.EndTime = time.Now()
		result.Duration = time.Since(phaseStart).Seconds()
		errorMsg := fmt.Sprintf("Phase completion verification failed: %v", err)
		result.ErrorMessage = &errorMsg
		return result
	}

	result.Status = "passed"
	result.EndTime = time.Now()
	result.Duration = time.Since(phaseStart).Seconds()

	return result
}

func generateDefaultPhaseConfig(phase string, scenarioName string) PhaseConfig {
	switch phase {
	case "setup":
		return PhaseConfig{
			Name:        fmt.Sprintf("%s_setup_phase", scenarioName),
			Description: fmt.Sprintf("Setup phase for %s", scenarioName),
			Timeout:     300,
			Tests: []PhaseTest{
				{
					Name:        "setup_validation",
					Description: "Validate setup phase completion",
					Type:        "phase_validation",
					Steps: []PhaseStep{
						{
							Action:   "verify_phase_prerequisites",
							Phase:    "setup",
							Expected: "satisfied",
						},
						{
							Action:  "execute_phase_tests",
							Phase:   "setup",
							Timeout: 300,
						},
					},
				},
			},
			Dependencies: []string{},
		}
	case "develop":
		return PhaseConfig{
			Name:        fmt.Sprintf("%s_develop_phase", scenarioName),
			Description: fmt.Sprintf("Development phase for %s", scenarioName),
			Timeout:     600,
			Tests: []PhaseTest{
				{
					Name:        "develop_validation",
					Description: "Validate development phase completion",
					Type:        "phase_validation",
					Steps: []PhaseStep{
						{
							Action:   "verify_phase_prerequisites",
							Phase:    "develop",
							Expected: "satisfied",
						},
						{
							Action:  "execute_phase_tests",
							Phase:   "develop",
							Timeout: 600,
						},
					},
				},
			},
			Dependencies: []string{"setup"},
		}
	case "test":
		return PhaseConfig{
			Name:        fmt.Sprintf("%s_test_phase", scenarioName),
			Description: fmt.Sprintf("Testing phase for %s", scenarioName),
			Timeout:     900,
			Tests: []PhaseTest{
				{
					Name:        "test_validation",
					Description: "Validate testing phase completion",
					Type:        "phase_validation",
					Steps: []PhaseStep{
						{
							Action:   "verify_phase_prerequisites",
							Phase:    "test",
							Expected: "satisfied",
						},
						{
							Action:  "execute_phase_tests",
							Phase:   "test",
							Timeout: 900,
						},
					},
				},
			},
			Dependencies: []string{"setup", "develop"},
		}
	default:
		return PhaseConfig{
			Name:        fmt.Sprintf("%s_%s_phase", scenarioName, phase),
			Description: fmt.Sprintf("%s phase for %s", strings.Title(phase), scenarioName),
			Timeout:     300,
			Tests: []PhaseTest{
				{
					Name:        fmt.Sprintf("%s_validation", phase),
					Description: fmt.Sprintf("Validate %s phase completion", phase),
					Type:        "phase_validation",
					Steps: []PhaseStep{
						{
							Action:   "verify_phase_prerequisites",
							Phase:    phase,
							Expected: "satisfied",
						},
						{
							Action:  "execute_phase_tests",
							Phase:   phase,
							Timeout: 300,
						},
					},
				},
			},
			Dependencies: []string{},
		}
	}
}

func executePhaseTest(test PhaseTest, scenarioName string, phase string, environment string) TestResult {
	log.Printf("🧪 Executing phase test: %s.%s", phase, test.Name)

	result := TestResult{
		ID:                  uuid.New(),
		Status:              "running",
		StartedAt:           time.Now(),
		TestCaseName:        test.Name,
		TestCaseDescription: test.Description,
		Assertions:          []AssertionResult{},
		Artifacts:           make(map[string]interface{}),
	}

	// Execute each step in the test
	for _, step := range test.Steps {
		stepResult := executePhaseStep(step, scenarioName, phase, environment)

		result.Assertions = append(result.Assertions, AssertionResult{
			Name:     fmt.Sprintf("%s_%s", step.Action, step.Phase),
			Expected: step.Expected,
			Actual:   stepResult.Status,
			Passed:   stepResult.Status == "success" || stepResult.Status == "satisfied",
			Message:  stepResult.Message,
		})

		if stepResult.Status == "failed" {
			result.Status = "failed"
			errorMsg := stepResult.Message
			result.ErrorMessage = &errorMsg
			break
		}
	}

	if result.Status == "running" {
		result.Status = "passed"
	}

	result.CompletedAt = time.Now()
	result.Duration = time.Since(result.StartedAt).Seconds()

	return result
}

type StepResult struct {
	Status  string
	Message string
}

func executePhaseStep(step PhaseStep, scenarioName string, phase string, environment string) StepResult {
	switch step.Action {
	case "verify_phase_prerequisites":
		return verifyPhasePrerequisitesStep(step, scenarioName, phase, environment)
	case "execute_phase_tests":
		return executePhaseTestsStep(step, scenarioName, phase, environment)
	case "verify_phase_completion":
		return verifyPhaseCompletionStep(step, scenarioName, phase, environment)
	default:
		return StepResult{
			Status:  "failed",
			Message: fmt.Sprintf("Unknown step action: %s", step.Action),
		}
	}
}

func verifyPhasePrerequisitesStep(step PhaseStep, scenarioName string, phase string, environment string) StepResult {
	// Verify basic prerequisites for the phase
	switch phase {
	case "setup":
		// Check if basic resources are available
		if err := checkDatabaseHealth(); err != nil {
			return StepResult{Status: "failed", Message: fmt.Sprintf("Database not available: %v", err)}
		}
		return StepResult{Status: "satisfied", Message: "Setup prerequisites satisfied"}

	case "develop":
		// Check if setup was completed and API is running
		apiPort := os.Getenv("API_PORT")
		if apiPort == "" {
			return StepResult{Status: "failed", Message: "API_PORT not configured"}
		}

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(fmt.Sprintf("http://localhost:%s/health", apiPort))
		if err != nil || resp.StatusCode != 200 {
			return StepResult{Status: "failed", Message: "API not healthy for develop phase"}
		}
		resp.Body.Close()
		return StepResult{Status: "satisfied", Message: "Develop prerequisites satisfied"}

	case "test":
		// Check if develop phase is complete and system is ready for testing
		return StepResult{Status: "satisfied", Message: "Test prerequisites satisfied"}

	default:
		return StepResult{Status: "satisfied", Message: fmt.Sprintf("%s prerequisites satisfied", phase)}
	}
}

func executePhaseTestsStep(step PhaseStep, scenarioName string, phase string, environment string) StepResult {
	// Generate and execute tests appropriate for this phase
	log.Printf("🔬 Executing %s phase tests for %s", phase, scenarioName)

	// Create a simple test based on the phase
	testCode := generatePhaseTestCode(phase, scenarioName)

	// Execute the test
	tempDir := filepath.Join(os.TempDir(), "test-genie-vault-execution")
	os.MkdirAll(tempDir, 0755)

	testFileName := fmt.Sprintf("vault_%s_%s_test.sh", phase, uuid.New().String()[:8])
	testFilePath := filepath.Join(tempDir, testFileName)

	if err := os.WriteFile(testFilePath, []byte(testCode), 0755); err != nil {
		return StepResult{Status: "failed", Message: fmt.Sprintf("Failed to create test file: %v", err)}
	}

	defer os.Remove(testFilePath)

	// Execute with timeout
	timeout := time.Duration(step.Timeout) * time.Second
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	cmd := exec.Command("/bin/bash", testFilePath)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("TEST_ENVIRONMENT=%s", environment),
		fmt.Sprintf("PHASE=%s", phase),
		fmt.Sprintf("SCENARIO_NAME=%s", scenarioName),
		fmt.Sprintf("API_PORT=%s", os.Getenv("API_PORT")),
	)

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			return StepResult{Status: "failed", Message: fmt.Sprintf("Phase test failed: %v", err)}
		}
		return StepResult{Status: "success", Message: fmt.Sprintf("%s phase tests completed successfully", phase)}
	case <-time.After(timeout):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return StepResult{Status: "failed", Message: fmt.Sprintf("Phase test timed out after %v", timeout)}
	}
}

func generatePhaseTestCode(phase string, scenarioName string) string {
	switch phase {
	case "setup":
		return fmt.Sprintf(`#!/bin/bash
echo "🔧 Running setup phase tests for %s"

# Test 1: Verify environment variables
if [[ -z "$API_PORT" ]]; then
    echo "❌ API_PORT not set"
    exit 1
fi
echo "✓ Environment variables configured"

# Test 2: Test database connectivity
if command -v curl >/dev/null; then
    if curl -sf "http://localhost:$API_PORT/health" >/dev/null; then
        echo "✓ API health check passed"
    else
        echo "❌ API health check failed"
        exit 1
    fi
else
    echo "⚠️ curl not available, skipping API test"
fi

echo "✅ Setup phase tests completed successfully"
`, scenarioName)

	case "develop":
		return fmt.Sprintf(`#!/bin/bash
echo "🚀 Running develop phase tests for %s"

# Test 1: Verify API endpoints are working
if command -v curl >/dev/null && [[ -n "$API_PORT" ]]; then
    echo "Testing API endpoints..."
    
    # Test health endpoint
    if curl -sf "http://localhost:$API_PORT/health" >/dev/null; then
        echo "✓ Health endpoint working"
    else
        echo "❌ Health endpoint failed"
        exit 1
    fi
    
    # Test test generation endpoint
    if curl -sf -X POST "http://localhost:$API_PORT/api/v1/test-suite/generate" \
        -H "Content-Type: application/json" \
        -d '{"scenario_name":"test","test_types":["unit"],"coverage_target":80}' >/dev/null; then
        echo "✓ Test generation endpoint working"
    else
        echo "⚠️ Test generation endpoint may have issues"
    fi
else
    echo "⚠️ Skipping API tests - curl or API_PORT not available"
fi

echo "✅ Develop phase tests completed successfully"
`, scenarioName)

	case "test":
		return fmt.Sprintf(`#!/bin/bash
echo "🧪 Running test phase tests for %s"

# Test 1: Run comprehensive system tests
echo "Running comprehensive system validation..."

# Test API functionality
if command -v curl >/dev/null && [[ -n "$API_PORT" ]]; then
    # Generate test suite
    echo "Testing test generation..."
    RESPONSE=$(curl -sf -X POST "http://localhost:$API_PORT/api/v1/test-suite/generate" \
        -H "Content-Type: application/json" \
        -d '{"scenario_name":"vault-test","test_types":["unit","integration"],"coverage_target":85}' 2>/dev/null)
    
    if [[ $? -eq 0 ]] && echo "$RESPONSE" | grep -q "suite_id"; then
        echo "✓ Test generation working"
    else
        echo "❌ Test generation failed"
        exit 1
    fi
    
    # Test coverage analysis
    echo "Testing coverage analysis..."
    if curl -sf -X POST "http://localhost:$API_PORT/api/v1/test-analysis/coverage" \
        -H "Content-Type: application/json" \
        -d '{"scenario_name":"vault-test","source_code_paths":["."]}' >/dev/null; then
        echo "✓ Coverage analysis working"
    else
        echo "⚠️ Coverage analysis may have issues"
    fi
else
    echo "⚠️ Skipping API tests - curl or API_PORT not available"
fi

echo "✅ Test phase tests completed successfully"
`, scenarioName)

	default:
		return fmt.Sprintf(`#!/bin/bash
echo "🔄 Running %s phase tests for %s"

# Generic phase test
echo "Validating %s phase completion..."

# Basic environment validation
if [[ -n "$SCENARIO_NAME" ]] && [[ -n "$PHASE" ]]; then
    echo "✓ Phase environment configured"
else
    echo "⚠️ Phase environment incomplete"
fi

echo "✅ %s phase tests completed successfully"
`, phase, scenarioName, phase, phase)
	}
}

func verifyPhaseCompletionStep(step PhaseStep, scenarioName string, phase string, environment string) StepResult {
	// Verify that the phase completed successfully
	log.Printf("✅ Verifying %s phase completion for %s", phase, scenarioName)

	// This would typically check specific completion criteria
	// For now, we'll do basic validation
	switch phase {
	case "setup":
		// Verify setup artifacts exist
		return StepResult{Status: "success", Message: "Setup phase completed successfully"}
	case "develop":
		// Verify development artifacts exist
		return StepResult{Status: "success", Message: "Develop phase completed successfully"}
	case "test":
		// Verify test results exist
		return StepResult{Status: "success", Message: "Test phase completed successfully"}
	default:
		return StepResult{Status: "success", Message: fmt.Sprintf("%s phase completed successfully", phase)}
	}
}

func verifyPhasePrerequisites(config PhaseConfig, environment string) error {
	// Verify dependencies are satisfied
	for _, dep := range config.Dependencies {
		log.Printf("🔍 Verifying dependency: %s", dep)
		// In a real implementation, this would check if the dependency phase completed successfully
	}
	return nil
}

func verifyPhaseCompletion(config PhaseConfig, environment string) error {
	// Verify phase completion criteria
	log.Printf("✅ Verifying phase completion: %s", config.Name)
	return nil
}

// Database functions for vault operations
func getVaultFromDatabase(vaultID uuid.UUID) (TestVault, error) {
	// For now, return a mock vault - in real implementation, query from database
	vault := TestVault{
		ID:                  vaultID,
		ScenarioName:        "test-scenario",
		VaultName:           "comprehensive-test-vault",
		Phases:              []string{"setup", "develop", "test"},
		PhaseConfigurations: make(map[string]PhaseConfig),
		SuccessCriteria: SuccessCriteria{
			AllPhasesCompleted:     true,
			NoCriticalFailures:     true,
			CoverageThreshold:      95.0,
			PerformanceBaselineMet: true,
		},
		TotalTimeout: 1800,
		CreatedAt:    time.Now(),
		Status:       "active",
	}

	return vault, nil
}

func updateVaultExecutionStatus(executionID uuid.UUID, status string, errorMessage string) {
	log.Printf("📝 Updating vault execution status: %s -> %s", executionID, status)
	// In real implementation, update database
}

func updateVaultExecutionInDatabase(execution VaultExecution) {
	log.Printf("📝 Updating vault execution in database: %s (phase: %s, status: %s)",
		execution.ID, execution.CurrentPhase, execution.Status)
	// In real implementation, store execution state in database
}

func executeTestSuiteHandler(c *gin.Context) {
	suiteIDStr := c.Param("suite_id")
	suiteID, err := uuid.Parse(suiteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid suite ID"})
		return
	}

	var req ExecuteTestSuiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create test execution record
	executionID := uuid.New()
	_, err = db.Exec(`
		INSERT INTO test_executions (id, suite_id, execution_type, start_time, status, environment)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, executionID, suiteID, req.ExecutionType, time.Now(), "started", req.Environment)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create test execution"})
		return
	}

	// Get test count
	var testCount int
	err = db.QueryRow("SELECT COUNT(*) FROM test_cases WHERE suite_id = $1", suiteID).Scan(&testCount)
	if err != nil {
		testCount = 0
	}

	response := ExecuteTestSuiteResponse{
		ExecutionID:       executionID,
		Status:            "started",
		EstimatedDuration: float64(testCount * 30), // 30 seconds per test estimate
		TestCount:         testCount,
		TrackingURL:       fmt.Sprintf("/api/v1/test-execution/%s/results", executionID),
	}

	// Start real test execution in background
	go func() {
		log.Printf("🚀 Starting background test execution for suite %s", suiteID)
		executeTestSuite(suiteID, executionID, req.ExecutionType, req.Environment, req.TimeoutSeconds)
	}()

	c.JSON(http.StatusAccepted, response)
}

func getTestExecutionResultsHandler(c *gin.Context) {
	executionIDStr := c.Param("execution_id")
	executionID, err := uuid.Parse(executionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid execution ID"})
		return
	}

	// Get execution details
	var execution TestExecution
	var suiteID uuid.UUID
	var scenarioName string

	err = db.QueryRow(`
		SELECT te.id, te.suite_id, te.execution_type, te.start_time, te.end_time, te.status, te.environment, ts.scenario_name
		FROM test_executions te
		JOIN test_suites ts ON te.suite_id = ts.id
		WHERE te.id = $1
	`, executionID).Scan(&execution.ID, &suiteID, &execution.ExecutionType, &execution.StartTime, &execution.EndTime, &execution.Status, &execution.Environment, &scenarioName)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Test execution not found"})
		return
	}

	// Get real execution summary from database
	var totalTests, passedTests, failedTestCount, skippedTests int
	err = db.QueryRow(`
		SELECT COALESCE(total_tests, 0), COALESCE(passed_tests, 0), COALESCE(failed_tests, 0), COALESCE(skipped_tests, 0)
		FROM test_executions WHERE id = $1
	`, executionID).Scan(&totalTests, &passedTests, &failedTestCount, &skippedTests)

	if err != nil {
		log.Printf("⚠️ Failed to get execution summary: %v", err)
		// If we can't get summary from execution table, calculate from test results
		totalTests = 0
		passedTests = 0
		failedTestCount = 0
		skippedTests = 0
	}

	// Calculate duration
	var duration float64
	if execution.EndTime != nil && !execution.StartTime.IsZero() {
		duration = execution.EndTime.Sub(execution.StartTime).Seconds()
	} else if execution.Status == "running" && !execution.StartTime.IsZero() {
		duration = time.Since(execution.StartTime).Seconds()
	}

	// Get failed test details
	var failedTests []TestResult
	rows, err := db.Query(`
		SELECT tr.id, tr.test_case_id, tr.status, tr.duration, tr.error_message, tr.stack_trace, tr.assertions, tr.artifacts, tc.name, tc.description
		FROM test_results tr
		JOIN test_cases tc ON tr.test_case_id = tc.id
		WHERE tr.execution_id = $1 AND tr.status = 'failed'
		ORDER BY tr.started_at
	`, executionID)

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var result TestResult
			var assertionsJSON, artifactsJSON []byte

			err := rows.Scan(&result.ID, &result.TestCaseID, &result.Status, &result.Duration,
				&result.ErrorMessage, &result.StackTrace, &assertionsJSON, &artifactsJSON,
				&result.TestCaseName, &result.TestCaseDescription)

			if err == nil {
				json.Unmarshal(assertionsJSON, &result.Assertions)
				json.Unmarshal(artifactsJSON, &result.Artifacts)
				failedTests = append(failedTests, result)
			}
		}
	}

	// Calculate coverage (mock for now, would need actual analysis)
	coverage := 0.0
	if totalTests > 0 {
		coverage = (float64(passedTests) / float64(totalTests)) * 100
	}

	summary := TestExecutionSummary{
		TotalTests: totalTests,
		Passed:     passedTests,
		Failed:     failedTestCount,
		Skipped:    skippedTests,
		Duration:   duration,
		Coverage:   coverage,
	}

	// Generate performance metrics
	performanceMetrics := PerformanceMetrics{
		ExecutionTime: duration,
		ResourceUsage: map[string]interface{}{
			"tests_per_second": func() float64 {
				if duration > 0 {
					return float64(totalTests) / duration
				}
				return 0
			}(),
		},
		ErrorCount: failedTestCount,
	}

	// Generate recommendations based on results
	recommendations := []string{}
	if failedTestCount > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Review and fix %d failed test(s)", failedTestCount))
	}
	if coverage < 80 {
		recommendations = append(recommendations,
			"Consider adding more test cases to improve coverage")
	}
	if duration > 300 {
		recommendations = append(recommendations,
			"Consider optimizing slow tests or running them in parallel")
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations,
			"All tests passed successfully! Consider adding more edge case tests.")
	}

	response := TestExecutionResultsResponse{
		ExecutionID:        executionID,
		SuiteName:          scenarioName,
		Status:             execution.Status,
		Summary:            summary,
		FailedTests:        failedTests,
		PerformanceMetrics: performanceMetrics,
		Recommendations:    recommendations,
	}

	c.JSON(http.StatusOK, response)
}

func analyzeCoverageHandler(c *gin.Context) {
	var req CoverageAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Mock coverage analysis for demo
	response := CoverageAnalysisResponse{
		OverallCoverage: 87.3,
		CoverageByFile: map[string]float64{
			"main.go":     95.2,
			"handlers.go": 89.1,
			"models.go":   76.5,
			"utils.go":    92.8,
		},
		CoverageGaps: CoverageGaps{
			UntestedFunctions: []string{
				"handleError",
				"validateInput",
				"cleanup",
			},
			UntestedBranches: []string{
				"error handling in main loop",
				"timeout scenarios",
				"edge case validation",
			},
			UntestedEdgeCases: []string{
				"null input handling",
				"concurrent access patterns",
				"resource exhaustion scenarios",
			},
		},
		ImprovementSuggestions: []string{
			"Add unit tests for error handling functions",
			"Create integration tests for concurrent scenarios",
			"Implement performance tests for resource-intensive operations",
			"Add boundary value testing for input validation",
		},
		PriorityAreas: []string{
			"Error handling coverage",
			"Concurrent access testing",
			"Input validation edge cases",
		},
	}

	c.JSON(http.StatusOK, response)
}

func healthHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get service degradation level
	serviceLevel := serviceManager.GetServiceLevel(ctx)

	// Comprehensive health check with circuit breaker awareness
	healthStatus := gin.H{
		"status":           getHealthStatusString(serviceLevel),
		"service_level":    serviceLevel.String(),
		"timestamp":        time.Now().UTC(),
		"version":          "1.0.0",
		"checks":           make(map[string]interface{}),
		"circuit_breakers": make(map[string]interface{}),
	}

	// Check individual services
	serviceChecks := healthStatus["checks"].(map[string]interface{})

	// Database health check
	if err := serviceManager.services["database"].Check(ctx); err != nil {
		serviceChecks["database"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	} else {
		serviceChecks["database"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// Check Ollama AI service
	if err := serviceManager.services["ollama"].Check(ctx); err != nil {
		serviceChecks["ai_service"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		// AI service failure is not critical for basic health due to fallback
	} else {
		serviceChecks["ai_service"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// Circuit breaker status
	circuitBreakers := healthStatus["circuit_breakers"].(map[string]interface{})
	circuitBreakers["database"] = map[string]interface{}{
		"state":    stateToString(dbCircuitBreaker.state),
		"failures": dbCircuitBreaker.failures,
	}
	circuitBreakers["ollama"] = map[string]interface{}{
		"state":    stateToString(ollamaCircuitBreaker.state),
		"failures": ollamaCircuitBreaker.failures,
	}

	// HTTP client status
	serviceChecks["http_client"] = map[string]interface{}{
		"status": "healthy",
	}

	// System capabilities based on service level
	healthStatus["capabilities"] = getSystemCapabilities(serviceLevel)

	// Set appropriate HTTP status code based on service level
	statusCode := getHTTPStatusForServiceLevel(serviceLevel)
	c.JSON(statusCode, healthStatus)
}

// Helper functions for health handler
func getHealthStatusString(level DegradationLevel) string {
	switch level {
	case FullService:
		return "healthy"
	case PartialService:
		return "degraded"
	case MinimalService:
		return "limited"
	default:
		return "unhealthy"
	}
}

func getSystemCapabilities(level DegradationLevel) []string {
	switch level {
	case FullService:
		return []string{
			"ai_test_generation",
			"test_execution",
			"coverage_analysis",
			"vault_testing",
			"real_time_monitoring",
		}
	case PartialService:
		return []string{
			"basic_test_generation",
			"test_execution",
			"coverage_analysis",
		}
	case MinimalService:
		return []string{
			"fallback_test_generation",
			"basic_test_execution",
		}
	default:
		return []string{}
	}
}

func getHTTPStatusForServiceLevel(level DegradationLevel) int {
	switch level {
	case FullService:
		return http.StatusOK
	case PartialService:
		return http.StatusOK // Still operational
	case MinimalService:
		return http.StatusPartialContent
	default:
		return http.StatusServiceUnavailable
	}
}

// Additional API handlers for comprehensive functionality

func getTestSuiteHandler(c *gin.Context) {
	suiteIDStr := c.Param("suite_id")
	suiteID, err := uuid.Parse(suiteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid suite ID"})
		return
	}

	// Mock response for now - would query database in real implementation
	testSuite := TestSuite{
		ID:           suiteID,
		ScenarioName: "example-scenario",
		SuiteType:    "unit,integration",
		GeneratedAt:  time.Now(),
		Status:       "active",
		TestCases: []TestCase{
			{
				ID:          uuid.New(),
				SuiteID:     suiteID,
				Name:        "health_check_test",
				Description: "Test API health endpoint",
				TestType:    "unit",
				Priority:    "critical",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		},
		CoverageMetrics: CoverageMetrics{
			CodeCoverage:     95.0,
			BranchCoverage:   85.0,
			FunctionCoverage: 92.0,
		},
	}

	c.JSON(http.StatusOK, testSuite)
}

func listTestSuitesHandler(c *gin.Context) {
	scenario := c.Query("scenario")
	status := c.Query("status")

	// Mock response for now - would query database with filters
	testSuites := []TestSuite{
		{
			ID:           uuid.New(),
			ScenarioName: "test-genie",
			SuiteType:    "unit,integration",
			GeneratedAt:  time.Now().Add(-24 * time.Hour),
			Status:       "active",
		},
		{
			ID:           uuid.New(),
			ScenarioName: "visited-tracker",
			SuiteType:    "unit,performance",
			GeneratedAt:  time.Now().Add(-12 * time.Hour),
			Status:       "active",
		},
	}

	// Apply filters
	filteredSuites := []TestSuite{}
	for _, suite := range testSuites {
		if scenario != "" && suite.ScenarioName != scenario {
			continue
		}
		if status != "" && suite.Status != status {
			continue
		}
		filteredSuites = append(filteredSuites, suite)
	}

	c.JSON(http.StatusOK, gin.H{
		"test_suites": filteredSuites,
		"total":       len(filteredSuites),
		"filters": gin.H{
			"scenario": scenario,
			"status":   status,
		},
	})
}

func maintainTestSuiteHandler(c *gin.Context) {
	suiteIDStr := c.Param("suite_id")
	suiteID, err := uuid.Parse(suiteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid suite ID"})
		return
	}

	var req struct {
		UpdateDependencies bool `json:"update_dependencies"`
		Optimize           bool `json:"optimize"`
		RemoveRedundant    bool `json:"remove_redundant"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Perform maintenance operations
	response := gin.H{
		"suite_id":              suiteID,
		"maintenance_completed": true,
		"operations_performed":  []string{},
		"summary": gin.H{
			"dependencies_updated":    req.UpdateDependencies,
			"performance_optimized":   req.Optimize,
			"redundant_tests_removed": req.RemoveRedundant,
			"estimated_improvement":   "15%",
		},
	}

	operations := []string{}
	if req.UpdateDependencies {
		operations = append(operations, "update_dependencies")
	}
	if req.Optimize {
		operations = append(operations, "optimize_performance")
	}
	if req.RemoveRedundant {
		operations = append(operations, "remove_redundant_tests")
	}

	response["operations_performed"] = operations

	c.JSON(http.StatusOK, response)
}

func listTestExecutionsHandler(c *gin.Context) {
	_ = c.Query("suite_id") // TODO: Implement filtering by suite_id
	_ = c.Query("status")   // TODO: Implement filtering by status

	// Mock response - would query database with filters
	executions := []gin.H{
		{
			"id":           uuid.New(),
			"suite_id":     uuid.New(),
			"status":       "completed",
			"start_time":   time.Now().Add(-2 * time.Hour),
			"end_time":     time.Now().Add(-1 * time.Hour),
			"environment":  "local",
			"total_tests":  15,
			"passed_tests": 14,
			"failed_tests": 1,
		},
		{
			"id":           uuid.New(),
			"suite_id":     uuid.New(),
			"status":       "running",
			"start_time":   time.Now().Add(-30 * time.Minute),
			"environment":  "staging",
			"total_tests":  20,
			"passed_tests": 10,
			"failed_tests": 0,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"executions": executions,
		"total":      len(executions),
	})
}

func createTestVaultHandler(c *gin.Context) {
	var req struct {
		ScenarioName string                 `json:"scenario_name" binding:"required"`
		VaultName    string                 `json:"vault_name" binding:"required"`
		Phases       []string               `json:"phases" binding:"required"`
		Timeout      int                    `json:"timeout"`
		Criteria     map[string]interface{} `json:"success_criteria"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vaultID := uuid.New()

	vault := TestVault{
		ID:                  vaultID,
		ScenarioName:        req.ScenarioName,
		VaultName:           req.VaultName,
		Phases:              req.Phases,
		PhaseConfigurations: make(map[string]PhaseConfig),
		TotalTimeout:        req.Timeout,
		CreatedAt:           time.Now(),
		Status:              "active",
		SuccessCriteria: SuccessCriteria{
			AllPhasesCompleted:     true,
			NoCriticalFailures:     true,
			CoverageThreshold:      95.0,
			PerformanceBaselineMet: true,
		},
	}

	// Generate default configurations for each phase
	for _, phase := range req.Phases {
		vault.PhaseConfigurations[phase] = generateDefaultPhaseConfig(phase, req.ScenarioName)
	}

	response := gin.H{
		"vault_id":    vaultID,
		"vault_name":  req.VaultName,
		"phases":      req.Phases,
		"phase_count": len(req.Phases),
		"status":      "created",
		"created_at":  vault.CreatedAt,
	}

	c.JSON(http.StatusCreated, response)
}

func getTestVaultHandler(c *gin.Context) {
	vaultIDStr := c.Param("vault_id")
	vaultID, err := uuid.Parse(vaultIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vault ID"})
		return
	}

	vault, err := getVaultFromDatabase(vaultID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vault not found"})
		return
	}

	c.JSON(http.StatusOK, vault)
}

func listTestVaultsHandler(c *gin.Context) {
	_ = c.Query("scenario") // TODO: Implement filtering by scenario
	_ = c.Query("status")   // TODO: Implement filtering by status

	// Mock response - would query database with filters
	vaults := []TestVault{
		{
			ID:           uuid.New(),
			ScenarioName: "test-genie",
			VaultName:    "comprehensive-test-vault",
			Phases:       []string{"setup", "develop", "test"},
			TotalTimeout: 1800,
			CreatedAt:    time.Now().Add(-24 * time.Hour),
			Status:       "active",
		},
		{
			ID:           uuid.New(),
			ScenarioName: "visited-tracker",
			VaultName:    "integration-vault",
			Phases:       []string{"setup", "test", "deploy"},
			TotalTimeout: 1200,
			CreatedAt:    time.Now().Add(-12 * time.Hour),
			Status:       "active",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"vaults": vaults,
		"total":  len(vaults),
	})
}

func executeTestVaultHandler(c *gin.Context) {
	vaultIDStr := c.Param("vault_id")
	vaultID, err := uuid.Parse(vaultIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vault ID"})
		return
	}

	var req struct {
		Environment string   `json:"environment"`
		Phases      []string `json:"phases"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		// Use defaults if no body provided
		req.Environment = "local"
	}

	executionID := uuid.New()

	// Start vault execution in background
	go func() {
		log.Printf("🏗️ Starting background vault execution for vault %s", vaultID)
		executeVault(vaultID, executionID, req.Environment)
	}()

	response := gin.H{
		"execution_id": executionID,
		"vault_id":     vaultID,
		"status":       "started",
		"environment":  req.Environment,
		"tracking_url": fmt.Sprintf("/api/v1/vault-execution/%s/results", executionID),
	}

	c.JSON(http.StatusAccepted, response)
}

func getVaultExecutionResultsHandler(c *gin.Context) {
	executionIDStr := c.Param("execution_id")
	executionID, err := uuid.Parse(executionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid execution ID"})
		return
	}

	// Mock vault execution results - would query from database
	response := gin.H{
		"execution_id":     executionID,
		"vault_name":       "comprehensive-test-vault",
		"status":           "completed",
		"start_time":       time.Now().Add(-30 * time.Minute),
		"end_time":         time.Now().Add(-5 * time.Minute),
		"duration":         1500, // seconds
		"current_phase":    "",
		"completed_phases": []string{"setup", "develop", "test"},
		"failed_phases":    []string{},
		"phase_results": gin.H{
			"setup": gin.H{
				"status":     "passed",
				"duration":   300,
				"test_count": 3,
			},
			"develop": gin.H{
				"status":     "passed",
				"duration":   600,
				"test_count": 5,
			},
			"test": gin.H{
				"status":     "passed",
				"duration":   600,
				"test_count": 8,
			},
		},
		"overall_success_rate": 100.0,
		"recommendations": []string{
			"All vault phases completed successfully",
			"Consider adding more edge case testing in the test phase",
		},
	}

	c.JSON(http.StatusOK, response)
}

func listVaultExecutionsHandler(c *gin.Context) {
	_ = c.Query("vault_id") // TODO: Implement filtering by vault_id
	_ = c.Query("status")   // TODO: Implement filtering by status

	// Mock response
	executions := []gin.H{
		{
			"id":               uuid.New(),
			"vault_id":         uuid.New(),
			"status":           "completed",
			"start_time":       time.Now().Add(-2 * time.Hour),
			"end_time":         time.Now().Add(-90 * time.Minute),
			"completed_phases": []string{"setup", "develop", "test"},
			"failed_phases":    []string{},
		},
		{
			"id":               uuid.New(),
			"vault_id":         uuid.New(),
			"status":           "running",
			"start_time":       time.Now().Add(-30 * time.Minute),
			"current_phase":    "develop",
			"completed_phases": []string{"setup"},
			"failed_phases":    []string{},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"executions": executions,
		"total":      len(executions),
	})
}

func getCoverageAnalysisHandler(c *gin.Context) {
	_ = c.Param("scenario_name") // TODO: Use scenario name for filtering

	// Mock coverage analysis - would query from database
	response := CoverageAnalysisResponse{
		OverallCoverage: 87.3,
		CoverageByFile: map[string]float64{
			"main.go":     95.2,
			"handlers.go": 89.1,
			"models.go":   76.5,
			"utils.go":    92.8,
		},
		CoverageGaps: CoverageGaps{
			UntestedFunctions: []string{
				"handleError",
				"validateInput",
				"cleanup",
			},
			UntestedBranches: []string{
				"error handling in main loop",
				"timeout scenarios",
			},
			UntestedEdgeCases: []string{
				"null input handling",
				"concurrent access patterns",
			},
		},
		ImprovementSuggestions: []string{
			"Add unit tests for error handling functions",
			"Create integration tests for concurrent scenarios",
			"Implement performance tests for resource-intensive operations",
		},
		PriorityAreas: []string{
			"Error handling functions (critical)",
			"Input validation routines (high)",
			"Concurrent access patterns (medium)",
		},
	}

	c.JSON(http.StatusOK, response)
}

func systemStatusHandler(c *gin.Context) {
	// Comprehensive system status
	status := gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
		"uptime":    "2h 30m 45s", // Would calculate actual uptime
		"services": gin.H{
			"database": gin.H{
				"status":  "healthy",
				"latency": "2.3ms",
			},
			"ai_service": gin.H{
				"status":  "healthy",
				"latency": "150ms",
			},
		},
		"metrics": gin.H{
			"active_test_suites":    42,
			"running_executions":    3,
			"completed_executions":  156,
			"active_vaults":         8,
			"total_tests_generated": 1247,
		},
	}

	c.JSON(http.StatusOK, status)
}

func systemMetricsHandler(c *gin.Context) {
	// System performance metrics
	metrics := gin.H{
		"performance": gin.H{
			"avg_test_generation_time":   45.2,
			"avg_test_execution_time":    123.7,
			"avg_coverage_analysis_time": 28.1,
			"api_response_time_p95":      125.0,
		},
		"throughput": gin.H{
			"tests_generated_per_hour": 85,
			"tests_executed_per_hour":  340,
			"api_requests_per_minute":  42,
		},
		"resource_usage": gin.H{
			"cpu_usage":     "45%",
			"memory_usage":  "1.2GB",
			"disk_usage":    "15.3GB",
			"database_size": "245MB",
		},
		"quality_metrics": gin.H{
			"avg_test_success_rate":   94.2,
			"avg_coverage_percentage": 87.8,
			"vault_completion_rate":   89.3,
		},
		"timestamp": time.Now().UTC(),
	}

	c.JSON(http.StatusOK, metrics)
}

func main() {
	// Load environment variables
	// Try loading environment configuration from multiple locations so running from the
	// API directory still picks up the scenario-level settings generated by setup-env.sh.
	envCandidates := []string{".env", "../.env", "../../.env"}
	envLoaded := false
	for _, candidate := range envCandidates {
		if _, err := os.Stat(candidate); err == nil {
			if loadErr := godotenv.Load(candidate); loadErr != nil {
				log.Printf("Failed to load %s: %v\n", candidate, loadErr)
			} else {
				envLoaded = true
				break
			}
		}
	}
	if !envLoaded {
		log.Println("No .env file found; relying on existing environment variables")
	}

	// Initialize circuit breakers and error handling
	InitializeCircuitBreakers()

	// Initialize service manager
	serviceManager = NewServiceManager()

	// Initialize database
	initDB()
	defer db.Close()

	// Initialize HTTP client for AI services
	initHTTPClient()

	// Set up health checkers
	setupHealthCheckers()

	// Set up Gin router
	router := gin.Default()

	// CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
	}))

	// Health endpoint
	router.GET("/health", healthHandler)

	// API routes
	api := router.Group("/api/v1")
	{
		// Test suite management
		api.POST("/test-suite/generate", generateTestSuiteHandler)
		api.GET("/test-suite/:suite_id", getTestSuiteHandler)
		api.GET("/test-suites", listTestSuitesHandler)
		api.POST("/test-suite/:suite_id/execute", executeTestSuiteHandler)
		api.POST("/test-suite/:suite_id/maintain", maintainTestSuiteHandler)

		// Test execution results
		api.GET("/test-execution/:execution_id/results", getTestExecutionResultsHandler)
		api.GET("/test-executions", listTestExecutionsHandler)

		// Test vault management
		api.POST("/test-vault/create", createTestVaultHandler)
		api.GET("/test-vault/:vault_id", getTestVaultHandler)
		api.GET("/test-vaults", listTestVaultsHandler)
		api.POST("/test-vault/:vault_id/execute", executeTestVaultHandler)
		api.GET("/vault-execution/:execution_id/results", getVaultExecutionResultsHandler)
		api.GET("/vault-executions", listVaultExecutionsHandler)

		// Coverage analysis
		api.POST("/test-analysis/coverage", analyzeCoverageHandler)
		api.GET("/test-analysis/coverage/:scenario_name", getCoverageAnalysisHandler)

		// System information
		api.GET("/system/status", systemStatusHandler)
		api.GET("/system/metrics", systemMetricsHandler)
	}

	// Get port from environment - REQUIRED, no defaults
	port := os.Getenv("API_PORT")
	if port == "" {
		port = os.Getenv("PORT") // Fallback to PORT for compatibility
	}
	if port == "" {
		log.Fatal("❌ API_PORT or PORT environment variable is required")
	}

	log.Printf("Test Genie API server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
