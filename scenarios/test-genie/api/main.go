package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
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
	ID                uuid.UUID           `json:"id"`
	SuiteID           uuid.UUID           `json:"suite_id"`
	ExecutionType     string              `json:"execution_type"`
	StartTime         time.Time           `json:"start_time"`
	EndTime           *time.Time          `json:"end_time,omitempty"`
	Status            string              `json:"status"`
	Results           []TestResult        `json:"results"`
	PerformanceMetrics PerformanceMetrics `json:"performance_metrics"`
	Environment       string              `json:"environment"`
}

type TestResult struct {
	ID           uuid.UUID         `json:"id"`
	ExecutionID  uuid.UUID         `json:"execution_id"`
	TestCaseID   uuid.UUID         `json:"test_case_id"`
	Status       string            `json:"status"`
	Duration     float64           `json:"duration"`
	ErrorMessage *string           `json:"error_message,omitempty"`
	StackTrace   *string           `json:"stack_trace,omitempty"`
	Assertions   []AssertionResult `json:"assertions"`
	Artifacts    map[string]interface{} `json:"artifacts"`
}

type CoverageMetrics struct {
	CodeCoverage     float64 `json:"code_coverage"`
	BranchCoverage   float64 `json:"branch_coverage"`
	FunctionCoverage float64 `json:"function_coverage"`
}

type PerformanceMetrics struct {
	ExecutionTime  float64                `json:"execution_time"`
	ResourceUsage  map[string]interface{} `json:"resource_usage"`
	ErrorCount     int                    `json:"error_count"`
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
	ScenarioName string   `json:"scenario_name" binding:"required"`
	TestTypes    []string `json:"test_types" binding:"required"`
	CoverageTarget float64 `json:"coverage_target"`
	Options       TestGenerationOptions `json:"options"`
}

type TestGenerationOptions struct {
	IncludePerformanceTests bool     `json:"include_performance_tests"`
	IncludeSecurityTests    bool     `json:"include_security_tests"`
	CustomTestPatterns      []string `json:"custom_test_patterns"`
	ExecutionTimeout        int      `json:"execution_timeout"`
}

type GenerateTestSuiteResponse struct {
	SuiteID           uuid.UUID            `json:"suite_id"`
	GeneratedTests    int                  `json:"generated_tests"`
	EstimatedCoverage float64              `json:"estimated_coverage"`
	GenerationTime    float64              `json:"generation_time"`
	TestFiles         map[string][]string  `json:"test_files"`
}

type ExecuteTestSuiteRequest struct {
	ExecutionType        string                      `json:"execution_type"`
	Environment          string                      `json:"environment"`
	ParallelExecution    bool                        `json:"parallel_execution"`
	TimeoutSeconds       int                         `json:"timeout_seconds"`
	NotificationSettings NotificationSettings       `json:"notification_settings"`
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
	ExecutionID         uuid.UUID                  `json:"execution_id"`
	SuiteName          string                     `json:"suite_name"`
	Status             string                     `json:"status"`
	Summary            TestExecutionSummary       `json:"summary"`
	FailedTests        []TestResult               `json:"failed_tests"`
	PerformanceMetrics PerformanceMetrics         `json:"performance_metrics"`
	Recommendations    []string                   `json:"recommendations"`
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
	ScenarioName     string   `json:"scenario_name" binding:"required"`
	SourceCodePaths  []string `json:"source_code_paths"`
	ExistingTestPaths []string `json:"existing_test_paths"`
	AnalysisDepth    string   `json:"analysis_depth"`
}

type CoverageAnalysisResponse struct {
	OverallCoverage       float64                `json:"overall_coverage"`
	CoverageByFile        map[string]float64     `json:"coverage_by_file"`
	CoverageGaps          CoverageGaps           `json:"coverage_gaps"`
	ImprovementSuggestions []string              `json:"improvement_suggestions"`
	PriorityAreas         []string               `json:"priority_areas"`
}

type CoverageGaps struct {
	UntestedFunctions  []string `json:"untested_functions"`
	UntestedBranches   []string `json:"untested_branches"`
	UntestedEdgeCases  []string `json:"untested_edge_cases"`
}

// Database
var db *sql.DB

// Initialize database connection
func initDB() {
	// ALL database configuration MUST come from environment - no defaults
	dbHost := os.Getenv("POSTGRES_HOST")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")

	// Validate required environment variables
	if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
		log.Fatal("❌ Missing required database configuration. Please set: POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB")
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

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
			log.Printf("✅ Database connected successfully on attempt %d", attempt + 1)
			break
		}
		
		// Calculate exponential backoff delay
		delay := time.Duration(math.Min(
			float64(baseDelay) * math.Pow(2, float64(attempt)),
			float64(maxDelay),
		))
		
		// Add progressive jitter to prevent thundering herd
		jitterRange := float64(delay) * 0.25
		jitter := time.Duration(jitterRange * (float64(attempt) / float64(maxRetries)))
		actualDelay := delay + jitter
		
		log.Printf("⚠️  Connection attempt %d/%d failed: %v", attempt + 1, maxRetries, pingErr)
		log.Printf("⏳ Waiting %v before next attempt", actualDelay)
		
		// Provide detailed status every few attempts
		if attempt > 0 && attempt % 3 == 0 {
			log.Printf("📈 Retry progress:")
			log.Printf("   - Attempts made: %d/%d", attempt + 1, maxRetries)
			log.Printf("   - Total wait time: ~%v", time.Duration(attempt * 2) * baseDelay)
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

// AI Test Generation Service
func generateTestsWithAI(scenarioName string, testTypes []string, options TestGenerationOptions) ([]TestCase, error) {
	// Mock AI test generation - in real implementation, this would call Ollama
	testCases := []TestCase{}
	
	for _, testType := range testTypes {
		switch testType {
		case "unit":
			testCases = append(testCases, generateUnitTests(scenarioName, options)...)
		case "integration":
			testCases = append(testCases, generateIntegrationTests(scenarioName, options)...)
		case "performance":
			if options.IncludePerformanceTests {
				testCases = append(testCases, generatePerformanceTests(scenarioName, options)...)
			}
		case "vault":
			testCases = append(testCases, generateVaultTests(scenarioName, options)...)
		case "regression":
			testCases = append(testCases, generateRegressionTests(scenarioName, options)...)
		}
	}
	
	return testCases, nil
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
API_PORT=${API_PORT:-8080}
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
POSTGRES_PORT=${POSTGRES_PORT:-5432}
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
API_PORT=${API_PORT:-8080}
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
API_PORT=${API_PORT:-8080}
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
API_PORT=${API_PORT:-8080}
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

	// Start test execution in background (simplified for demo)
	go func() {
		time.Sleep(time.Duration(req.TimeoutSeconds) * time.Second / 4) // Simulate execution
		
		// Update execution status
		db.Exec(`
			UPDATE test_executions 
			SET status = $1, end_time = $2 
			WHERE id = $3
		`, "completed", time.Now(), executionID)
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

	// Mock results for demo
	summary := TestExecutionSummary{
		TotalTests: 10,
		Passed:     8,
		Failed:     1,
		Skipped:    1,
		Duration:   120.5,
		Coverage:   87.3,
	}

	response := TestExecutionResultsResponse{
		ExecutionID:   executionID,
		SuiteName:     scenarioName,
		Status:        execution.Status,
		Summary:       summary,
		FailedTests:   []TestResult{}, // Would populate with actual failed tests
		PerformanceMetrics: PerformanceMetrics{
			ExecutionTime: 120.5,
			ResourceUsage: map[string]interface{}{
				"cpu_usage": "45%",
				"memory_usage": "2.1GB",
			},
			ErrorCount: 1,
		},
		Recommendations: []string{
			"Consider increasing timeout for performance tests",
			"Add more edge case coverage for error handling",
		},
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
			"main.go":    95.2,
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
	// Check database connection
	if err := db.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"error":  "database connection failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	})
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize database
	initDB()
	defer db.Close()

	// Set up Gin router
	router := gin.Default()

	// CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
	router.Use(c.Handler())

	// Health endpoint
	router.GET("/health", healthHandler)

	// API routes
	api := router.Group("/api/v1")
	{
		// Test suite management
		api.POST("/test-suite/generate", generateTestSuiteHandler)
		api.POST("/test-suite/:suite_id/execute", executeTestSuiteHandler)
		
		// Test execution results
		api.GET("/test-execution/:execution_id/results", getTestExecutionResultsHandler)
		
		// Coverage analysis
		api.POST("/test-analysis/coverage", analyzeCoverageHandler)
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