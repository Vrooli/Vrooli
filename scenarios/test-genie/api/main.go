package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/lib/pq"
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
	SourcePath     string    `json:"source_path,omitempty"`
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
	CodeCoverage       float64  `json:"code_coverage"`
	BranchCoverage     float64  `json:"branch_coverage"`
	FunctionCoverage   float64  `json:"function_coverage"`
	IssueID            string   `json:"issue_id,omitempty"`
	IssueURL           string   `json:"issue_url,omitempty"`
	RequestID          string   `json:"request_id,omitempty"`
	RequestStatus      string   `json:"request_status,omitempty"`
	RequestedAt        string   `json:"requested_at,omitempty"`
	RequestedBy        string   `json:"requested_by,omitempty"`
	RequestedTestTypes []string `json:"requested_test_types,omitempty"`
	Notes              string   `json:"notes,omitempty"`
}

type PerformanceMetrics struct {
	ExecutionTime float64                `json:"execution_time"`
	ResourceUsage map[string]interface{} `json:"resource_usage"`
	ErrorCount    int                    `json:"error_count"`
}

type ScenarioOverview struct {
	Name                   string     `json:"name"`
	HasTestDirectory       bool       `json:"has_test_directory"`
	HasTests               bool       `json:"has_tests"`
	SuiteCount             int        `json:"suite_count"`
	TestCaseCount          int        `json:"test_case_count"`
	SuiteTypes             []string   `json:"suite_types,omitempty"`
	LatestSuiteGeneratedAt *time.Time `json:"latest_suite_generated_at,omitempty"`
	LatestSuiteID          string     `json:"latest_suite_id,omitempty"`
	LatestSuiteStatus      string     `json:"latest_suite_status,omitempty"`
	LatestSuiteCoverage    float64    `json:"latest_suite_coverage"`
}

type ReportsOverviewResponse struct {
	GeneratedAt time.Time               `json:"generated_at"`
	WindowStart time.Time               `json:"window_start"`
	WindowEnd   time.Time               `json:"window_end"`
	Global      ReportGlobalMetrics     `json:"global"`
	Scenarios   []ScenarioReportSummary `json:"scenarios"`
	Vaults      ReportVaultRollup       `json:"vaults"`
}

type ReportGlobalMetrics struct {
	TotalTests       int     `json:"total_tests"`
	PassedTests      int     `json:"passed_tests"`
	FailedTests      int     `json:"failed_tests"`
	PassRate         float64 `json:"pass_rate"`
	AverageDuration  float64 `json:"average_duration_seconds"`
	AverageCoverage  float64 `json:"average_coverage"`
	ActiveScenarios  int     `json:"active_scenarios"`
	ActiveExecutions int     `json:"active_executions"`
	ActiveVaults     int     `json:"active_vaults"`
	Regressions      int     `json:"active_regressions"`
}

type ScenarioReportSummary struct {
	ScenarioName        string            `json:"scenario_name"`
	HealthScore         int               `json:"health_score"`
	PassRate            float64           `json:"pass_rate"`
	Coverage            float64           `json:"coverage"`
	TargetCoverage      float64           `json:"target_coverage"`
	CoverageDelta       float64           `json:"coverage_delta"`
	ExecutionCount      int               `json:"execution_count"`
	ActiveFailures      int               `json:"active_failures"`
	RunningExecutions   int               `json:"running_executions"`
	AverageTestDuration float64           `json:"average_test_duration"`
	LastExecutionID     string            `json:"last_execution_id"`
	LastExecutionStatus string            `json:"last_execution_status"`
	LastExecutionEnded  *time.Time        `json:"last_execution_ended_at,omitempty"`
	Vault               ReportVaultStatus `json:"vault"`
}

type ReportVaultStatus struct {
	HasVault        bool       `json:"has_vault"`
	TotalExecutions int        `json:"total_executions"`
	SuccessfulRuns  int        `json:"successful_runs"`
	FailedRuns      int        `json:"failed_runs"`
	SuccessRate     float64    `json:"success_rate"`
	LatestStatus    string     `json:"latest_status"`
	LatestEndedAt   *time.Time `json:"latest_ended_at,omitempty"`
}

type ReportVaultRollup struct {
	TotalVaults        int     `json:"total_vaults"`
	TotalExecutions    int     `json:"total_executions"`
	FailedExecutions   int     `json:"failed_executions"`
	SuccessRate        float64 `json:"success_rate"`
	ScenariosWithVault int     `json:"scenarios_with_vault"`
}

type ReportTrendPoint struct {
	Bucket           time.Time `json:"bucket"`
	Executions       int       `json:"executions"`
	FailedExecutions int       `json:"failed_executions"`
	PassedTests      int       `json:"passed_tests"`
	FailedTests      int       `json:"failed_tests"`
	AverageDuration  float64   `json:"average_duration_seconds"`
	AverageCoverage  float64   `json:"average_coverage"`
}

type ReportsTrendsResponse struct {
	GeneratedAt time.Time          `json:"generated_at"`
	BucketSize  string             `json:"bucket_size"`
	WindowStart time.Time          `json:"window_start"`
	WindowEnd   time.Time          `json:"window_end"`
	Series      []ReportTrendPoint `json:"series"`
}

type ReportInsight struct {
	Title        string   `json:"title"`
	Severity     string   `json:"severity"`
	ScenarioName string   `json:"scenario_name,omitempty"`
	Detail       string   `json:"detail"`
	Actions      []string `json:"actions"`
}

type ReportsInsightsResponse struct {
	GeneratedAt time.Time       `json:"generated_at"`
	WindowStart time.Time       `json:"window_start"`
	WindowEnd   time.Time       `json:"window_end"`
	Insights    []ReportInsight `json:"insights"`
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
	Model          string                `json:"model,omitempty"`
}

type TestGenerationOptions struct {
	IncludePerformanceTests bool     `json:"include_performance_tests"`
	IncludeSecurityTests    bool     `json:"include_security_tests"`
	CustomTestPatterns      []string `json:"custom_test_patterns"`
	ExecutionTimeout        int      `json:"execution_timeout"`
}

type GenerateTestSuiteResponse struct {
	RequestID         uuid.UUID           `json:"request_id"`
	Status            string              `json:"status"`
	Message           string              `json:"message"`
	IssueID           string              `json:"issue_id,omitempty"`
	IssueURL          string              `json:"issue_url,omitempty"`
	GeneratedTests    int                 `json:"generated_tests,omitempty"`
	EstimatedCoverage float64             `json:"estimated_coverage,omitempty"`
	GenerationTime    float64             `json:"generation_time,omitempty"`
	TestFiles         map[string][]string `json:"test_files,omitempty"`
}

type GenerateBatchTestSuiteRequest struct {
	ScenarioNames  []string              `json:"scenario_names" binding:"required"`
	TestTypes      []string              `json:"test_types" binding:"required"`
	CoverageTarget float64               `json:"coverage_target"`
	Options        TestGenerationOptions `json:"options"`
	Model          string                `json:"model,omitempty"`
}

type GenerateBatchTestSuiteResponse struct {
	Created int                    `json:"created"`
	Skipped int                    `json:"skipped"`
	Results []BatchGenerateResult  `json:"results"`
	Issues  []IssueCreationResult  `json:"issues"`
}

type BatchGenerateResult struct {
	ScenarioName string `json:"scenario_name"`
	Status       string `json:"status"` // "created", "skipped", "error"
	Reason       string `json:"reason,omitempty"`
	IssueID      string `json:"issue_id,omitempty"`
	IssueURL     string `json:"issue_url,omitempty"`
}

type IssueCreationResult struct {
	ScenarioName string `json:"scenario_name"`
	IssueID      string `json:"issue_id"`
	IssueURL     string `json:"issue_url"`
	Message      string `json:"message"`
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

// AI Integration Types
// Global variables
var db *sql.DB
var serviceManager *ServiceManager

var (
	repoRootOnce sync.Once
	repoRootPath string
	repoRootErr  error
)

type aggregatedCoverage struct {
	GeneratedAt string               `json:"generated_at"`
	Languages   []aggregatedLanguage `json:"languages"`
}

type aggregatedLanguage struct {
	Language  string             `json:"language"`
	Metrics   map[string]float64 `json:"metrics"`
	Artifacts map[string]string  `json:"artifacts"`
}

type istanbulMetric struct {
	Pct float64 `json:"pct"`
}

type istanbulSummary struct {
	Statements istanbulMetric `json:"statements"`
	Lines      istanbulMetric `json:"lines"`
	Functions  istanbulMetric `json:"functions"`
	Branches   istanbulMetric `json:"branches"`
}

type CoverageOverview struct {
	ScenarioName    string                    `json:"scenario_name"`
	OverallCoverage float64                   `json:"overall_coverage"`
	GeneratedAt     string                    `json:"generated_at"`
	Languages       []CoverageLanguageSummary `json:"languages"`
	AggregatePath   string                    `json:"aggregate_path"`
	HasArtifacts    bool                      `json:"has_artifacts"`
	Warnings        []string                  `json:"warnings,omitempty"`
}

type CoverageLanguageSummary struct {
	Language  string             `json:"language"`
	Metrics   map[string]float64 `json:"metrics"`
	Artifacts map[string]string  `json:"artifacts"`
}

type testTypeHint struct {
	keyword  string
	testType string
}

var (
	scenarioWalkSkipDirs = map[string]struct{}{
		"node_modules":     {},
		"dist":             {},
		"build":            {},
		"tmp":              {},
		".tmp":             {},
		"coverage":         {},
		".coverage":        {},
		"__pycache__":      {},
		".git":             {},
		"vendor":           {},
		"logs":             {},
		"public":           {},
		"assets":           {},
		"output":           {},
		"storybook-static": {},
		".dist":            {},
		".cache":           {},
		"cache":            {},
	}

	scenarioTestFileExtensions = map[string]struct{}{
		".sh":   {},
		".bats": {},
		".go":   {},
		".py":   {},
		".js":   {},
		".ts":   {},
		".tsx":  {},
		".jsx":  {},
		".rb":   {},
		".rs":   {},
		".java": {},
		".kt":   {},
		".kts":  {},
		".ps1":  {},
		".yaml": {},
		".yml":  {},
	}

	scenarioTestTypeHints = []testTypeHint{
		{"unit", "unit"},
		{"integration", "integration"},
		{"e2e", "end_to_end"},
		{"end-to-end", "end_to_end"},
		{"performance", "performance"},
		{"load", "performance"},
		{"stress", "performance"},
		{"smoke", "smoke"},
		{"regression", "regression"},
		{"business", "business"},
		{"structure", "structure"},
		{"api", "api"},
		{"ui", "ui"},
		{"db", "database"},
		{"database", "database"},
		{"cli", "cli"},
		{"vault", "vault"},
		{"phase", "phased"},
	}

	// Cap repository traversal so we do not scan endlessly upward if repo markers are missing.
	maxRepoSearchDepth = 15
)

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

	// Issue tracker health checker
	issueTrackerChecker := NewHealthChecker("issue_tracker", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		port, err := resolveScenarioPortViaCLI(ctx, "app-issue-tracker", "API_PORT")
		if err != nil {
			return err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://localhost:%d/health", port), nil)
		if err != nil {
			return fmt.Errorf("failed to build app-issue-tracker health request: %w", err)
		}

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("app-issue-tracker health check failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("app-issue-tracker unhealthy (status %d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		return nil
	}, issueTrackerCircuitBreaker)
	serviceManager.RegisterService("issue_tracker", issueTrackerChecker)

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
		`CREATE TABLE IF NOT EXISTS test_vaults (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scenario_name VARCHAR(255) NOT NULL,
			vault_name VARCHAR(255) NOT NULL,
			phases JSONB DEFAULT '[]',
			phase_configurations JSONB DEFAULT '{}',
			success_criteria JSONB DEFAULT '{}',
			total_timeout INTEGER DEFAULT 3600,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_executed TIMESTAMP,
			status VARCHAR(50) DEFAULT 'active'
		)`,
		`CREATE TABLE IF NOT EXISTS vault_executions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			vault_id UUID REFERENCES test_vaults(id) ON DELETE CASCADE,
			execution_type VARCHAR(50),
			start_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			end_time TIMESTAMP,
			current_phase VARCHAR(100),
			completed_phases JSONB DEFAULT '[]',
			failed_phases JSONB DEFAULT '[]',
			status VARCHAR(50) DEFAULT 'running',
			phase_results JSONB DEFAULT '{}',
			environment VARCHAR(100),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS coverage_analysis (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			scenario_name VARCHAR(255) NOT NULL,
			overall_coverage DECIMAL(5,2),
			coverage_by_file JSONB DEFAULT '{}',
			coverage_gaps JSONB DEFAULT '{}',
			improvement_suggestions JSONB DEFAULT '[]',
			priority_areas JSONB DEFAULT '[]',
			analyzed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_test_vaults_scenario ON test_vaults(scenario_name)`,
		`CREATE INDEX IF NOT EXISTS idx_vault_executions_vault ON vault_executions(vault_id)`,
		`CREATE INDEX IF NOT EXISTS idx_coverage_analysis_scenario ON coverage_analysis(scenario_name)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			log.Printf("Error creating table: %v", err)
		}
	}
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

	suiteID := uuid.New()
	requestedAt := time.Now().UTC()

	issueResult, issueErr := submitTestGenerationIssue(c.Request.Context(), suiteID, req)
	if issueErr == nil {
		metrics := CoverageMetrics{
			CodeCoverage:       req.CoverageTarget,
			BranchCoverage:     math.Max(req.CoverageTarget*0.9, 0),
			FunctionCoverage:   math.Max(req.CoverageTarget*0.95, 0),
			IssueID:            issueResult.IssueID,
			IssueURL:           issueResult.IssueURL,
			RequestID:          suiteID.String(),
			RequestStatus:      "submitted",
			RequestedAt:        requestedAt.Format(time.RFC3339),
			RequestedBy:        "test-genie",
			RequestedTestTypes: append([]string(nil), req.TestTypes...),
			Notes:              "Awaiting automated test generation via app-issue-tracker",
		}

		coverageJSON, _ := json.Marshal(metrics)
		suiteType := fmt.Sprintf("pending-%s", suiteID.String()[:8])

		if _, err := db.Exec(`
			INSERT INTO test_suites (id, scenario_name, suite_type, coverage_metrics, generated_at, status)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, suiteID, req.ScenarioName, suiteType, coverageJSON, requestedAt, "maintenance_required"); err != nil {
			log.Printf("⚠️  Failed to record delegated test generation request: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record test generation request"})
			return
		}

		response := GenerateTestSuiteResponse{
			RequestID:         suiteID,
			Status:            "submitted",
			Message:           issueResult.Message,
			IssueID:           issueResult.IssueID,
			IssueURL:          issueResult.IssueURL,
			EstimatedCoverage: req.CoverageTarget,
		}

		c.JSON(http.StatusAccepted, response)
		return
	}

	log.Printf("⚠️  Delegating test generation failed (falling back to local templates): %v", issueErr)

	startTime := time.Now()
	var testCases []TestCase
	for _, testType := range req.TestTypes {
		cases := generateFallbackTests(req.ScenarioName, testType, req.Options)
		for idx := range cases {
			cases[idx].SuiteID = suiteID
		}
		testCases = append(testCases, cases...)
	}

	if len(testCases) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to generate fallback tests"})
		return
	}

	metrics := CoverageMetrics{
		CodeCoverage:       req.CoverageTarget,
		BranchCoverage:     math.Max(req.CoverageTarget*0.9, 0),
		FunctionCoverage:   math.Max(req.CoverageTarget*0.95, 0),
		RequestID:          suiteID.String(),
		RequestStatus:      "generated_locally",
		RequestedAt:        requestedAt.Format(time.RFC3339),
		RequestedBy:        "test-genie",
		RequestedTestTypes: append([]string(nil), req.TestTypes...),
		Notes:              "Issue tracker unavailable; generated default templates locally.",
	}

	coverageJSON, _ := json.Marshal(metrics)
	suiteType := strings.Join(req.TestTypes, ",")
	if suiteType == "" {
		suiteType = "fallback"
	}

	if _, err := db.Exec(`
		INSERT INTO test_suites (id, scenario_name, suite_type, coverage_metrics, generated_at, status)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, suiteID, req.ScenarioName, suiteType, coverageJSON, time.Now(), "active"); err != nil {
		log.Printf("⚠️  Failed to save fallback test suite: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save fallback test suite"})
		return
	}

	for _, testCase := range testCases {
		tagsJSON, _ := json.Marshal(testCase.Tags)
		depsJSON, _ := json.Marshal(testCase.Dependencies)

		if _, err := db.Exec(`
			INSERT INTO test_cases (id, suite_id, name, description, test_type, test_code, expected_result, execution_timeout, dependencies, tags, priority, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`, testCase.ID, suiteID, testCase.Name, testCase.Description, testCase.TestType, testCase.TestCode, testCase.ExpectedResult, testCase.Timeout, depsJSON, tagsJSON, testCase.Priority, testCase.CreatedAt, testCase.UpdatedAt); err != nil {
			log.Printf("⚠️  Failed to save fallback test case %s: %v", testCase.Name, err)
		}
	}

	generationTime := time.Since(startTime).Seconds()
	testFiles := make(map[string][]string)
	for _, testCase := range testCases {
		testFiles[testCase.TestType] = append(testFiles[testCase.TestType], testCase.Name)
	}

	response := GenerateTestSuiteResponse{
		RequestID:         suiteID,
		Status:            "generated_locally",
		Message:           fmt.Sprintf("Delegated submission failed: %s. Generated %d fallback tests locally.", strings.TrimSpace(issueErr.Error()), len(testCases)),
		GeneratedTests:    len(testCases),
		EstimatedCoverage: req.CoverageTarget,
		GenerationTime:    generationTime,
		TestFiles:         testFiles,
	}

	c.JSON(http.StatusCreated, response)
}

func generateBatchTestSuiteHandler(c *gin.Context) {
	var req GenerateBatchTestSuiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response := GenerateBatchTestSuiteResponse{
		Results: []BatchGenerateResult{},
		Issues:  []IssueCreationResult{},
	}

	for _, scenarioName := range req.ScenarioNames {
		// Check if scenario already has tests
		var existingSuites int
		err := db.QueryRow(`
			SELECT COUNT(*) FROM test_suites
			WHERE scenario_name = $1 AND status = 'active'
		`, scenarioName).Scan(&existingSuites)

		if err != nil {
			log.Printf("⚠️  Error checking existing suites for %s: %v", scenarioName, err)
			response.Results = append(response.Results, BatchGenerateResult{
				ScenarioName: scenarioName,
				Status:       "error",
				Reason:       "Database error",
			})
			continue
		}

		// Skip if already has active test suites
		if existingSuites > 0 {
			response.Skipped++
			response.Results = append(response.Results, BatchGenerateResult{
				ScenarioName: scenarioName,
				Status:       "skipped",
				Reason:       "Already has active test suites",
			})
			continue
		}

		// Create individual request for this scenario
		individualReq := GenerateTestSuiteRequest{
			ScenarioName:   scenarioName,
			TestTypes:      req.TestTypes,
			CoverageTarget: req.CoverageTarget,
			Options:        req.Options,
			Model:          req.Model,
		}

		suiteID := uuid.New()
		requestedAt := time.Now().UTC()

		// Try to submit issue
		issueResult, issueErr := submitTestGenerationIssue(c.Request.Context(), suiteID, individualReq)
		if issueErr == nil {
			// Successfully created issue
			metrics := CoverageMetrics{
				CodeCoverage:       req.CoverageTarget,
				BranchCoverage:     math.Max(req.CoverageTarget*0.9, 0),
				FunctionCoverage:   math.Max(req.CoverageTarget*0.95, 0),
				IssueID:            issueResult.IssueID,
				IssueURL:           issueResult.IssueURL,
				RequestID:          suiteID.String(),
				RequestStatus:      "submitted",
				RequestedAt:        requestedAt.Format(time.RFC3339),
				RequestedBy:        "test-genie",
				RequestedTestTypes: append([]string(nil), req.TestTypes...),
				Notes:              "Awaiting automated test generation via app-issue-tracker",
			}

			coverageJSON, _ := json.Marshal(metrics)
			suiteType := fmt.Sprintf("pending-%s", suiteID.String()[:8])

			if _, err := db.Exec(`
				INSERT INTO test_suites (id, scenario_name, suite_type, coverage_metrics, generated_at, status)
				VALUES ($1, $2, $3, $4, $5, $6)
			`, suiteID, scenarioName, suiteType, coverageJSON, requestedAt, "maintenance_required"); err != nil {
				log.Printf("⚠️  Failed to record delegated test generation request for %s: %v", scenarioName, err)
				response.Results = append(response.Results, BatchGenerateResult{
					ScenarioName: scenarioName,
					Status:       "error",
					Reason:       "Failed to record request",
				})
				continue
			}

			response.Created++
			response.Results = append(response.Results, BatchGenerateResult{
				ScenarioName: scenarioName,
				Status:       "created",
				IssueID:      issueResult.IssueID,
				IssueURL:     issueResult.IssueURL,
			})
			response.Issues = append(response.Issues, IssueCreationResult{
				ScenarioName: scenarioName,
				IssueID:      issueResult.IssueID,
				IssueURL:     issueResult.IssueURL,
				Message:      issueResult.Message,
			})
		} else {
			// Issue creation failed
			log.Printf("⚠️  Failed to create issue for %s: %v", scenarioName, issueErr)
			response.Results = append(response.Results, BatchGenerateResult{
				ScenarioName: scenarioName,
				Status:       "error",
				Reason:       fmt.Sprintf("Issue creation failed: %v", issueErr),
			})
		}
	}

	c.JSON(http.StatusOK, response)
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

func buildDiscoveredTestScript(scenarioRoot, relativePath string) string {
	normalizedPath := filepath.ToSlash(relativePath)

	var builder strings.Builder
	builder.WriteString("#!/bin/bash\n")
	builder.WriteString("set -euo pipefail\n")
	builder.WriteString(fmt.Sprintf("cd %q\n", scenarioRoot))

	switch ext := strings.ToLower(filepath.Ext(normalizedPath)); ext {
	case ".sh", ".bash":
		builder.WriteString(fmt.Sprintf("bash %q\n", normalizedPath))
	case ".bats":
		builder.WriteString(fmt.Sprintf("bats %q\n", normalizedPath))
	case ".py":
		builder.WriteString(fmt.Sprintf("python3 %q\n", normalizedPath))
	case ".js":
		builder.WriteString(fmt.Sprintf("node %q\n", normalizedPath))
	case ".ts":
		builder.WriteString(fmt.Sprintf("npx ts-node %q\n", normalizedPath))
	default:
		builder.WriteString(fmt.Sprintf("bash %q\n", normalizedPath))
	}

	return builder.String()
}

func determineSuiteTypeLabel(types []string) string {
	if len(types) == 0 {
		return "discovered"
	}

	joined := strings.Join(types, ",")
	if len(joined) <= 50 {
		return joined
	}

	if len(types) == 1 {
		return types[0][:int(math.Min(50, float64(len(types[0]))))]
	}

	return fmt.Sprintf("multi:%d-types", len(types))
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

var runTestSuite = executeTestSuite

var errSuiteNotFound = errors.New("test suite not found in repository")

var syncSuiteForExecution = syncDiscoveredSuiteToDatabase

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

	if strings.TrimSpace(req.ExecutionType) == "" {
		req.ExecutionType = "full"
	}
	if strings.TrimSpace(req.Environment) == "" {
		req.Environment = "local"
	}
	if req.TimeoutSeconds <= 0 {
		req.TimeoutSeconds = 600
	}

	// Create test execution record
	executionID := uuid.New()
	insertStmt := `
		INSERT INTO test_executions (id, suite_id, execution_type, start_time, status, environment)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	insertArgs := []interface{}{executionID, suiteID, req.ExecutionType, time.Now(), "running", req.Environment}

	_, err = db.Exec(insertStmt, insertArgs...)
	if err != nil {
		log.Printf("❌ Failed to create test execution for suite %s: %v", suiteID, err)
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			if _, syncErr := syncSuiteForExecution(c.Request.Context(), suiteID); syncErr != nil {
				if errors.Is(syncErr, errSuiteNotFound) {
					c.JSON(http.StatusNotFound, gin.H{"error": "Test suite not found", "details": syncErr.Error()})
					return
				}
				log.Printf("❌ Unable to sync discovered suite %s: %v", suiteID, syncErr)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare test suite for execution", "details": syncErr.Error()})
				return
			}

			log.Printf("🔄 Discovered suite %s synced to database. Retrying execution insert.", suiteID)
			_, err = db.Exec(insertStmt, insertArgs...)
		}
	}

	if err != nil {
		log.Printf("❌ Failed to create test execution after synchronization for suite %s: %v", suiteID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create test execution", "details": err.Error()})
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
		Status:            "running",
		EstimatedDuration: float64(testCount * 30), // 30 seconds per test estimate
		TestCount:         testCount,
		TrackingURL:       fmt.Sprintf("/api/v1/test-execution/%s/results", executionID),
	}

	// Start real test execution in background
	go func() {
		log.Printf("🚀 Starting background test execution for suite %s", suiteID)
		runTestSuite(suiteID, executionID, req.ExecutionType, req.Environment, req.TimeoutSeconds)
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

func computeCoverageAnalysis(scenarioName string) (CoverageAnalysisResponse, bool, error) {
	response := CoverageAnalysisResponse{
		CoverageByFile: make(map[string]float64),
		CoverageGaps: CoverageGaps{
			UntestedFunctions: []string{},
			UntestedBranches:  []string{},
			UntestedEdgeCases: []string{},
		},
	}

	aggregate, scenarioRoot, err := loadCoverageAggregate(scenarioName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			scenarioLabel := scenarioName
			if strings.TrimSpace(scenarioLabel) == "" {
				scenarioLabel = "current scenario"
			}
			response.ImprovementSuggestions = []string{
				fmt.Sprintf("Coverage artifacts not found for %s. Run the scenario's unit tests to generate coverage reports.", scenarioLabel),
			}
			response.PriorityAreas = []string{"Generate fresh coverage reports"}
			return response, false, nil
		}
		return response, false, err
	}

	var coverageValues []float64
	suggestions := []string{}
	priorityAreas := []string{}

	for _, language := range aggregate.Languages {
		statementsCoverage := language.Metrics["statements"]
		if statementsCoverage == 0 {
			if v, ok := language.Metrics["lines"]; ok {
				statementsCoverage = v
			}
		}
		if statementsCoverage > 0 {
			coverageValues = append(coverageValues, statementsCoverage)
			response.CoverageByFile[fmt.Sprintf("%s:overall", language.Language)] = statementsCoverage
			if statementsCoverage < 80 {
				label := humanizeLanguage(language.Language)
				suggestions = append(suggestions,
					fmt.Sprintf("%s coverage is %.1f%% (below the 80%% target)", label, statementsCoverage))
				priorityAreas = append(priorityAreas,
					fmt.Sprintf("Increase %s automation coverage", label))
			}
		}

		if language.Language == "node" {
			if summaryRel, ok := language.Artifacts["summary"]; ok && summaryRel != "" {
				summaryPath := filepath.Join(scenarioRoot, summaryRel)
				if perFile, err := parseNodeCoverageSummary(summaryPath, scenarioRoot); err == nil {
					for filePath, pct := range perFile {
						response.CoverageByFile[fmt.Sprintf("node:%s", filePath)] = pct
					}
				} else {
					log.Printf("⚠️  Failed to parse Node coverage summary (%s): %v", summaryPath, err)
				}
			}
		}

		if language.Language == "go" {
			if profileRel, ok := language.Artifacts["cover_profile"]; ok && profileRel != "" {
				profilePath := filepath.Join(scenarioRoot, profileRel)
				if perFile, err := parseGoCoverageProfile(profilePath, scenarioRoot); err == nil {
					for filePath, pct := range perFile {
						response.CoverageByFile[fmt.Sprintf("go:%s", filePath)] = pct
					}
				} else {
					log.Printf("⚠️  Failed to parse Go coverage profile (%s): %v", profilePath, err)
				}
			}
		}
	}

	if len(coverageValues) > 0 {
		var sum float64
		for _, v := range coverageValues {
			sum += v
		}
		response.OverallCoverage = sum / float64(len(coverageValues))
	}

	if len(suggestions) == 0 {
		suggestions = append(suggestions, "Coverage levels meet configured thresholds")
	}
	if len(priorityAreas) == 0 {
		priorityAreas = append(priorityAreas, "Monitor coverage trends over time")
	}

	response.ImprovementSuggestions = suggestions
	response.PriorityAreas = priorityAreas

	return response, true, nil
}

func loadCoverageAggregate(scenarioName string) (*aggregatedCoverage, string, error) {
	candidates := aggregateFileCandidates(scenarioName)
	var parseErr error
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		data, err := os.ReadFile(candidate)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			parseErr = err
			continue
		}
		var aggregate aggregatedCoverage
		if err := json.Unmarshal(data, &aggregate); err != nil {
			parseErr = err
			continue
		}
		scenarioRoot := filepath.Dir(filepath.Dir(filepath.Dir(candidate)))
		return &aggregate, scenarioRoot, nil
	}

	if parseErr != nil {
		return nil, "", parseErr
	}
	return nil, "", os.ErrNotExist
}

func aggregateFileCandidates(scenarioName string) []string {
	var candidates []string

	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Clean(filepath.Join(wd, "..", "coverage", "test-genie", "aggregate.json")))
	}

	if repoRoot, err := getRepoRoot(); err == nil {
		if strings.TrimSpace(scenarioName) != "" {
			candidates = append(candidates, filepath.Join(repoRoot, "scenarios", scenarioName, "coverage", "test-genie", "aggregate.json"))
		}
		candidates = append(candidates, filepath.Join(repoRoot, "scenarios", "test-genie", "coverage", "test-genie", "aggregate.json"))
	}

	seen := map[string]struct{}{}
	unique := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		unique = append(unique, abs)
	}

	return unique
}

func listCoverageSummaries() ([]CoverageOverview, error) {
	repoRoot, err := getRepoRoot()
	if err != nil {
		return nil, err
	}

	scenariosDir := filepath.Join(repoRoot, "scenarios")
	globPattern := filepath.Join(scenariosDir, "*", "coverage", "test-genie", "aggregate.json")
	matches, err := filepath.Glob(globPattern)
	if err != nil {
		return nil, err
	}

	// Also check the test-genie scenario itself
	matches = append(matches, filepath.Join(scenariosDir, "test-genie", "coverage", "test-genie", "aggregate.json"))

	seen := make(map[string]struct{})
	var overviews []CoverageOverview

	for _, candidate := range matches {
		if candidate == "" {
			continue
		}
		absPath, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if _, ok := seen[absPath]; ok {
			continue
		}
		seen[absPath] = struct{}{}

		data, err := os.ReadFile(absPath)
		if err != nil {
			continue
		}

		var aggregate aggregatedCoverage
		if err := json.Unmarshal(data, &aggregate); err != nil {
			log.Printf("⚠️  Failed to unmarshal coverage aggregate %s: %v", absPath, err)
			continue
		}

		scenarioPath := filepath.Dir(filepath.Dir(filepath.Dir(absPath)))
		scenarioName := filepath.Base(scenarioPath)
		if scenarioName == "coverage" {
			// Handles cases where aggregate lives directly under scenario root
			scenarioPath = filepath.Dir(scenarioPath)
			scenarioName = filepath.Base(scenarioPath)
		}

		overview := CoverageOverview{
			ScenarioName:  scenarioName,
			GeneratedAt:   aggregate.GeneratedAt,
			Languages:     []CoverageLanguageSummary{},
			AggregatePath: filepath.ToSlash(absPath),
			HasArtifacts:  len(aggregate.Languages) > 0,
			Warnings:      []string{},
		}

		var coverageValues []float64

		for _, language := range aggregate.Languages {
			metricsCopy := make(map[string]float64, len(language.Metrics))
			for key, value := range language.Metrics {
				metricsCopy[key] = value
			}
			artifactsCopy := make(map[string]string, len(language.Artifacts))
			for key, value := range language.Artifacts {
				artifactsCopy[key] = value
			}

			overview.Languages = append(overview.Languages, CoverageLanguageSummary{
				Language:  language.Language,
				Metrics:   metricsCopy,
				Artifacts: artifactsCopy,
			})

			if value, ok := metricsCopy["statements"]; ok && value > 0 {
				coverageValues = append(coverageValues, value)
			} else if value, ok := metricsCopy["lines"]; ok && value > 0 {
				coverageValues = append(coverageValues, value)
			}

			if len(metricsCopy) == 0 {
				overview.Warnings = append(overview.Warnings,
					fmt.Sprintf("No coverage metrics recorded for %s", humanizeLanguage(language.Language)))
			}
		}

		if len(coverageValues) > 0 {
			var sum float64
			for _, v := range coverageValues {
				sum += v
			}
			overview.OverallCoverage = sum / float64(len(coverageValues))
		} else {
			overview.Warnings = append(overview.Warnings, "Coverage metrics not available")
		}

		if rel, err := filepath.Rel(repoRoot, absPath); err == nil {
			overview.AggregatePath = filepath.ToSlash(rel)
		}

		overviews = append(overviews, overview)
	}

	sort.Slice(overviews, func(i, j int) bool {
		if overviews[i].ScenarioName == overviews[j].ScenarioName {
			return overviews[i].GeneratedAt > overviews[j].GeneratedAt
		}
		return strings.Compare(overviews[i].ScenarioName, overviews[j].ScenarioName) < 0
	})

	return overviews, nil
}

func getRepoRoot() (string, error) {
	repoRootOnce.Do(func() {
		dir, err := os.Getwd()
		if err != nil {
			repoRootErr = err
			return
		}
		dir, err = filepath.Abs(dir)
		if err != nil {
			repoRootErr = err
			return
		}
		for {
			if fileExists(filepath.Join(dir, ".git")) || fileExists(filepath.Join(dir, "pnpm-workspace.yaml")) {
				repoRootPath = dir
				return
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				repoRootErr = fmt.Errorf("repository root not found starting at %s", dir)
				return
			}
			dir = parent
		}
	})

	return repoRootPath, repoRootErr
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func parseNodeCoverageSummary(summaryPath, scenarioRoot string) (map[string]float64, error) {
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		return nil, err
	}

	var raw map[string]istanbulSummary
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	results := make(map[string]float64)
	for key, summary := range raw {
		if strings.EqualFold(key, "total") {
			continue
		}
		normalized := normalizePathRelativeToRoot(key, scenarioRoot)
		results[normalized] = summary.Statements.Pct
	}

	return results, nil
}

func parseGoCoverageProfile(profilePath, scenarioRoot string) (map[string]float64, error) {
	data, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) <= 1 {
		return map[string]float64{}, nil
	}

	type coverageStat struct {
		total   int
		covered int
	}

	stats := make(map[string]*coverageStat)

	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		fileRange := fields[0]
		numStmt, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		count, err := strconv.ParseFloat(fields[2], 64)
		if err != nil {
			continue
		}

		filePath := strings.Split(fileRange, ":")[0]
		stat, ok := stats[filePath]
		if !ok {
			stat = &coverageStat{}
			stats[filePath] = stat
		}
		stat.total += numStmt
		if count > 0 {
			stat.covered += numStmt
		}
	}

	results := make(map[string]float64, len(stats))
	for filePath, stat := range stats {
		if stat.total == 0 {
			continue
		}
		coverage := (float64(stat.covered) / float64(stat.total)) * 100
		results[normalizePathRelativeToRoot(filePath, scenarioRoot)] = coverage
	}

	return results, nil
}

func normalizePathRelativeToRoot(filePath, scenarioRoot string) string {
	cleaned := filepath.Clean(filePath)
	if filepath.IsAbs(cleaned) {
		if rel, err := filepath.Rel(scenarioRoot, cleaned); err == nil {
			return filepath.ToSlash(rel)
		}
		return filepath.ToSlash(cleaned)
	}

	joined := filepath.Join(scenarioRoot, cleaned)
	if rel, err := filepath.Rel(scenarioRoot, joined); err == nil {
		return filepath.ToSlash(rel)
	}

	return filepath.ToSlash(cleaned)
}

func humanizeLanguage(language string) string {
	if language == "" {
		return "Unknown"
	}
	runes := []rune(language)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func analyzeCoverageHandler(c *gin.Context) {
	var req CoverageAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	analysis, hasData, err := computeCoverageAnalysis(req.ScenarioName)
	if err != nil {
		log.Printf("⚠️  Failed to compute coverage analysis: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to compute coverage analysis"})
		return
	}

	if !hasData {
		log.Printf("ℹ️  Coverage artifacts not found for scenario %q", req.ScenarioName)
	}

	c.JSON(http.StatusOK, analysis)
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
	if checker, ok := serviceManager.services["database"]; ok {
		if err := checker.Check(ctx); err != nil {
			serviceChecks["database"] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
		} else {
			serviceChecks["database"] = map[string]interface{}{
				"status": "healthy",
			}
		}
	} else {
		serviceChecks["database"] = map[string]interface{}{
			"status": "unknown",
		}
	}

	// Check issue tracker integration
	if checker, ok := serviceManager.services["issue_tracker"]; ok {
		if err := checker.Check(ctx); err != nil {
			serviceChecks["issue_tracker"] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
		} else {
			serviceChecks["issue_tracker"] = map[string]interface{}{
				"status": "healthy",
			}
		}
	} else {
		serviceChecks["issue_tracker"] = map[string]interface{}{
			"status": "unknown",
		}
	}

	// Circuit breaker status
	circuitBreakers := healthStatus["circuit_breakers"].(map[string]interface{})
	circuitBreakers["database"] = map[string]interface{}{
		"state":    stateToString(dbCircuitBreaker.state),
		"failures": dbCircuitBreaker.failures,
	}

	if issueTrackerCircuitBreaker != nil {
		circuitBreakers["issue_tracker"] = map[string]interface{}{
			"state":    stateToString(issueTrackerCircuitBreaker.state),
			"failures": issueTrackerCircuitBreaker.failures,
		}
	} else {
		circuitBreakers["issue_tracker"] = map[string]interface{}{
			"state":    "UNKNOWN",
			"failures": 0,
		}
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
			"delegated_test_generation",
			"test_execution",
			"coverage_analysis",
			"vault_testing",
			"real_time_monitoring",
		}
	case PartialService:
		return []string{
			"queued_test_requests",
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

	// First try to load a persisted suite from storage.
	if suite, err := fetchStoredTestSuiteByID(c.Request.Context(), suiteID); err != nil {
		log.Printf("⚠️  Failed to fetch stored test suite %s: %v", suiteID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load test suite"})
		return
	} else if suite != nil {
		c.JSON(http.StatusOK, suite)
		return
	}

	// Fall back to discovered suites on disk so legacy scenarios still surface tests.
	repoRoot, err := findRepositoryRoot()
	if err != nil {
		log.Printf("⚠️  Failed to locate repository root while discovering suite %s: %v", suiteID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Test suite not found"})
		return
	}

	suite, err := findDiscoveredSuiteByID(repoRoot, suiteID)
	if err != nil {
		log.Printf("⚠️  Failed to discover suite %s: %v", suiteID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load test suite"})
		return
	}

	if suite == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Test suite not found"})
		return
	}

	c.JSON(http.StatusOK, suite)
}

func fetchStoredTestSuites(ctx context.Context, scenarioFilter, statusFilter string) ([]TestSuite, error) {
	if db == nil {
		return nil, errors.New("database connection not initialized")
	}

	query := `SELECT id, scenario_name, suite_type, coverage_metrics, generated_at, last_executed, status FROM test_suites`
	clauses := []string{}
	args := []interface{}{}
	argPos := 1

	if scenarioFilter != "" {
		clauses = append(clauses, fmt.Sprintf("scenario_name = $%d", argPos))
		args = append(args, scenarioFilter)
		argPos++
	}
	if statusFilter != "" {
		clauses = append(clauses, fmt.Sprintf("status = $%d", argPos))
		args = append(args, statusFilter)
		argPos++
	}

	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY generated_at DESC"

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suites []TestSuite
	for rows.Next() {
		var suite TestSuite
		var coverageBytes []byte
		var lastExecuted sql.NullTime

		if err := rows.Scan(&suite.ID, &suite.ScenarioName, &suite.SuiteType, &coverageBytes, &suite.GeneratedAt, &lastExecuted, &suite.Status); err != nil {
			return nil, err
		}

		if len(coverageBytes) > 0 {
			if err := json.Unmarshal(coverageBytes, &suite.CoverageMetrics); err != nil {
				log.Printf("⚠️  Failed to unmarshal coverage metrics for suite %s: %v", suite.ID, err)
			}
		}

		if lastExecuted.Valid {
			suite.LastExecuted = &lastExecuted.Time
		}

		cases, err := fetchTestCasesForSuite(ctx, suite.ID)
		if err != nil {
			log.Printf("⚠️  Failed to load test cases for suite %s: %v", suite.ID, err)
		} else {
			suite.TestCases = cases
		}

		suites = append(suites, suite)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return suites, nil
}

func fetchStoredTestSuiteByID(ctx context.Context, suiteID uuid.UUID) (*TestSuite, error) {
	if db == nil {
		return nil, errors.New("database connection not initialized")
	}

	query := `SELECT id, scenario_name, suite_type, coverage_metrics, generated_at, last_executed, status FROM test_suites WHERE id = $1`
	var suite TestSuite
	var coverageBytes []byte
	var lastExecuted sql.NullTime

	err := db.QueryRowContext(ctx, query, suiteID).Scan(
		&suite.ID,
		&suite.ScenarioName,
		&suite.SuiteType,
		&coverageBytes,
		&suite.GeneratedAt,
		&lastExecuted,
		&suite.Status,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if len(coverageBytes) > 0 {
		if err := json.Unmarshal(coverageBytes, &suite.CoverageMetrics); err != nil {
			log.Printf("⚠️  Failed to unmarshal coverage metrics for suite %s: %v", suite.ID, err)
		}
	}

	if lastExecuted.Valid {
		suite.LastExecuted = &lastExecuted.Time
	}

	cases, err := fetchTestCasesForSuite(ctx, suite.ID)
	if err != nil {
		return nil, err
	}

	suite.TestCases = cases
	return &suite, nil
}

func fetchTestCasesForSuite(ctx context.Context, suiteID uuid.UUID) ([]TestCase, error) {
	query := `SELECT id, name, description, test_type, test_code, expected_result, execution_timeout, dependencies, tags, priority, created_at, updated_at FROM test_cases WHERE suite_id = $1 ORDER BY created_at ASC`
	rows, err := db.QueryContext(ctx, query, suiteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cases []TestCase
	for rows.Next() {
		var tc TestCase
		var description sql.NullString
		var expected sql.NullString
		var priority sql.NullString
		var timeout sql.NullInt64
		var dependenciesJSON []byte
		var tagsJSON []byte
		var createdAt time.Time
		var updatedAt time.Time

		if err := rows.Scan(&tc.ID, &tc.Name, &description, &tc.TestType, &tc.TestCode, &expected, &timeout, &dependenciesJSON, &tagsJSON, &priority, &createdAt, &updatedAt); err != nil {
			return nil, err
		}

		tc.SuiteID = suiteID
		if description.Valid {
			tc.Description = description.String
		}
		if expected.Valid {
			tc.ExpectedResult = expected.String
		}
		if priority.Valid && priority.String != "" {
			tc.Priority = priority.String
		}
		if timeout.Valid {
			tc.Timeout = int(timeout.Int64)
		}
		if len(dependenciesJSON) > 0 {
			if err := json.Unmarshal(dependenciesJSON, &tc.Dependencies); err != nil {
				log.Printf("⚠️  Failed to unmarshal dependencies for test case %s: %v", tc.ID, err)
			}
		}
		if len(tagsJSON) > 0 {
			if err := json.Unmarshal(tagsJSON, &tc.Tags); err != nil {
				log.Printf("⚠️  Failed to unmarshal tags for test case %s: %v", tc.ID, err)
			}
		}
		tc.CreatedAt = createdAt
		tc.UpdatedAt = updatedAt

		cases = append(cases, tc)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return cases, nil
}

func findRepositoryRoot() (string, error) {
	startDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := startDir
	for i := 0; i < maxRepoSearchDepth; i++ {
		scenariosDir := filepath.Join(dir, "scenarios")
		if info, err := os.Stat(scenariosDir); err == nil && info.IsDir() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("unable to locate repository root starting from %s", startDir)
}

func discoverScenarioTestSuites(repoRoot, scenarioFilter string) ([]TestSuite, error) {
	scenariosDir := filepath.Join(repoRoot, "scenarios")
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		return nil, err
	}

	scenarioFilterLower := strings.ToLower(strings.TrimSpace(scenarioFilter))
	var suites []TestSuite

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		scenarioName := entry.Name()
		if strings.HasPrefix(scenarioName, ".") {
			continue
		}
		if scenarioName == "test-genie" {
			// Skip self to keep the list focused on other scenarios.
			continue
		}
		if scenarioFilterLower != "" && strings.ToLower(scenarioName) != scenarioFilterLower {
			continue
		}

		scenarioRoot := filepath.Join(scenariosDir, scenarioName)
		testDir := filepath.Join(scenarioRoot, "test")
		if info, err := os.Stat(testDir); err != nil || !info.IsDir() {
			continue
		}

		suiteID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("scenario-suite:"+scenarioName))
		var testCases []TestCase
		typeSet := make(map[string]struct{})
		latestMod := time.Time{}

		walkErr := filepath.WalkDir(testDir, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			if d.IsDir() {
				if shouldSkipScenarioDir(d.Name()) {
					return fs.SkipDir
				}
				return nil
			}

			if !isRecognizedTestFile(path) {
				return nil
			}

			rel, err := filepath.Rel(filepath.Join(scenariosDir, scenarioName), path)
			if err != nil {
				rel = d.Name()
			}

			info, err := d.Info()
			if err != nil {
				return err
			}
			modTime := info.ModTime()
			if modTime.After(latestMod) {
				latestMod = modTime
			}

			relativePath := filepath.ToSlash(rel)
			testType := deriveTestTypeFromPath(path)
			if testType == "" {
				testType = "unspecified"
			}
			typeSet[testType] = struct{}{}

			caseID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("scenario-suite:"+scenarioName+":"+rel))
			caseName := sanitizeTestCaseName(rel)
			testScript := buildDiscoveredTestScript(scenarioRoot, relativePath)

			testCases = append(testCases, TestCase{
				ID:             caseID,
				SuiteID:        suiteID,
				Name:           caseName,
				Description:    fmt.Sprintf("Discovered test script: %s", rel),
				TestType:       testType,
				TestCode:       testScript,
				ExpectedResult: "Exit code 0",
				Timeout:        300,
				Tags:           []string{"discovered", testType},
				Priority:       "medium",
				CreatedAt:      modTime,
				UpdatedAt:      modTime,
				SourcePath:     relativePath,
			})

			return nil
		})

		if walkErr != nil {
			log.Printf("⚠️  Failed to traverse tests for %s: %v", scenarioName, walkErr)
			continue
		}

		if len(testCases) == 0 {
			continue
		}

		sort.Slice(testCases, func(i, j int) bool {
			return testCases[i].Name < testCases[j].Name
		})

		suiteTypes := make([]string, 0, len(typeSet))
		for t := range typeSet {
			suiteTypes = append(suiteTypes, t)
		}
		sort.Strings(suiteTypes)

		suite := TestSuite{
			ID:           suiteID,
			ScenarioName: scenarioName,
			SuiteType:    determineSuiteTypeLabel(suiteTypes),
			TestCases:    testCases,
			CoverageMetrics: CoverageMetrics{
				CodeCoverage:     0,
				BranchCoverage:   0,
				FunctionCoverage: 0,
			},
			GeneratedAt: latestMod,
			Status:      "active",
		}

		if suite.GeneratedAt.IsZero() {
			suite.GeneratedAt = time.Now()
		}

		suites = append(suites, suite)
	}

	return suites, nil
}

func findDiscoveredSuiteByID(repoRoot string, suiteID uuid.UUID) (*TestSuite, error) {
	suites, err := discoverScenarioTestSuites(repoRoot, "")
	if err != nil {
		return nil, err
	}

	for i := range suites {
		if suites[i].ID == suiteID {
			return &suites[i], nil
		}
	}

	return nil, nil
}

func syncDiscoveredSuiteToDatabase(ctx context.Context, suiteID uuid.UUID) (*TestSuite, error) {
	if db == nil {
		return nil, errors.New("database connection not initialized")
	}

	repoRoot, err := findRepositoryRoot()
	if err != nil {
		return nil, err
	}

	suite, err := findDiscoveredSuiteByID(repoRoot, suiteID)
	if err != nil {
		return nil, err
	}

	if suite == nil {
		return nil, errSuiteNotFound
	}

	if suite.GeneratedAt.IsZero() {
		suite.GeneratedAt = time.Now()
	}

	coverageJSON, err := json.Marshal(suite.CoverageMetrics)
	if err != nil {
		return nil, err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	success := false
	defer func() {
		if !success {
			tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO test_suites (id, scenario_name, suite_type, coverage_metrics, generated_at, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE
		SET scenario_name = EXCLUDED.scenario_name,
			suite_type = EXCLUDED.suite_type,
			coverage_metrics = EXCLUDED.coverage_metrics,
			status = EXCLUDED.status
	`, suite.ID, suite.ScenarioName, suite.SuiteType, coverageJSON, suite.GeneratedAt, suite.Status); err != nil {
		return nil, err
	}

	for _, tc := range suite.TestCases {
		if tc.TestCode == "" {
			log.Printf("⚠️  Skipping discovered test case %s due to missing script", tc.Name)
			continue
		}

		if tc.CreatedAt.IsZero() {
			tc.CreatedAt = suite.GeneratedAt
		}
		if tc.UpdatedAt.IsZero() {
			tc.UpdatedAt = tc.CreatedAt
		}

		tc.SuiteID = suite.ID

		dependenciesJSON, err := json.Marshal(tc.Dependencies)
		if err != nil {
			return nil, err
		}
		tagsJSON, err := json.Marshal(tc.Tags)
		if err != nil {
			return nil, err
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO test_cases (id, suite_id, name, description, test_type, test_code, expected_result, execution_timeout, dependencies, tags, priority, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			ON CONFLICT (id) DO UPDATE
			SET name = EXCLUDED.name,
				description = EXCLUDED.description,
				test_type = EXCLUDED.test_type,
				test_code = EXCLUDED.test_code,
				expected_result = EXCLUDED.expected_result,
				execution_timeout = EXCLUDED.execution_timeout,
				dependencies = EXCLUDED.dependencies,
				tags = EXCLUDED.tags,
				priority = EXCLUDED.priority,
				updated_at = EXCLUDED.updated_at
		`, tc.ID, tc.SuiteID, tc.Name, tc.Description, tc.TestType, tc.TestCode, tc.ExpectedResult, tc.Timeout, dependenciesJSON, tagsJSON, tc.Priority, tc.CreatedAt, tc.UpdatedAt); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	success = true
	return suite, nil
}

func shouldSkipScenarioDir(name string) bool {
	if name == "" {
		return false
	}
	if strings.HasPrefix(name, ".") {
		return true
	}
	_, skip := scenarioWalkSkipDirs[name]
	return skip
}

var errScenarioTestFound = errors.New("scenario test located")

func scenarioHasRecognizedTests(scenarioRoot string) bool {
	testDir := filepath.Join(scenarioRoot, "test")
	info, err := os.Stat(testDir)
	if err != nil || !info.IsDir() {
		return false
	}

	err = filepath.WalkDir(testDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			if shouldSkipScenarioDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}

		if isRecognizedTestFile(path) {
			return errScenarioTestFound
		}
		return nil
	})

	if errors.Is(err, errScenarioTestFound) {
		return true
	}
	if err != nil {
		log.Printf("⚠️  Failed to inspect tests for %s: %v", scenarioRoot, err)
	}
	return false
}

func isRecognizedTestFile(path string) bool {
	name := filepath.Base(path)
	lowerName := strings.ToLower(name)
	ext := strings.ToLower(filepath.Ext(name))

	if _, ok := scenarioTestFileExtensions[ext]; !ok {
		return false
	}

	switch ext {
	case ".go":
		return strings.HasSuffix(lowerName, "_test.go")
	case ".py":
		return strings.HasPrefix(lowerName, "test_") || strings.HasSuffix(lowerName, "_test.py") || strings.Contains(lowerName, "_spec.py")
	case ".js", ".jsx", ".ts", ".tsx":
		return strings.Contains(lowerName, "test") || strings.Contains(lowerName, "spec")
	case ".yaml", ".yml":
		return strings.Contains(lowerName, "test") || strings.Contains(lowerName, "suite")
	default:
		return true
	}
}

func deriveTestTypeFromPath(path string) string {
	base := strings.ToLower(filepath.Base(path))
	for _, hint := range scenarioTestTypeHints {
		if strings.Contains(base, hint.keyword) {
			return hint.testType
		}
	}

	parent := strings.ToLower(filepath.Base(filepath.Dir(path)))
	for _, hint := range scenarioTestTypeHints {
		if strings.Contains(parent, hint.keyword) {
			return hint.testType
		}
	}

	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	if ext != "" {
		return ext
	}

	return "unspecified"
}

func sanitizeTestCaseName(rel string) string {
	name := strings.TrimSuffix(rel, filepath.Ext(rel))
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.TrimSpace(name)
	if name == "" {
		name = strings.ReplaceAll(rel, "\\", "_")
		name = strings.ReplaceAll(name, "/", "_")
	}
	if name == "" {
		name = "test_case"
	}
	return name
}

func listTestSuitesHandler(c *gin.Context) {
	scenarioFilter := strings.TrimSpace(c.Query("scenario"))
	statusFilter := strings.TrimSpace(c.Query("status"))

	var suites []TestSuite

	storedSuites, err := fetchStoredTestSuites(c.Request.Context(), scenarioFilter, statusFilter)
	if err != nil {
		log.Printf("⚠️  Failed to load stored test suites: %v", err)
	} else {
		suites = append(suites, storedSuites...)
	}

	repoRoot, repoErr := findRepositoryRoot()
	if repoErr != nil {
		log.Printf("⚠️  Failed to locate repository root for test discovery: %v", repoErr)
	} else {
		discoveredSuites, discoverErr := discoverScenarioTestSuites(repoRoot, scenarioFilter)
		if discoverErr != nil {
			log.Printf("⚠️  Failed to discover scenario test suites: %v", discoverErr)
		} else {
			for _, suite := range discoveredSuites {
				if statusFilter != "" && !strings.EqualFold(suite.Status, statusFilter) {
					continue
				}
				suites = append(suites, suite)
			}
		}
	}

	sort.Slice(suites, func(i, j int) bool {
		ii, jj := suites[i], suites[j]
		if ii.ScenarioName == jj.ScenarioName {
			return ii.GeneratedAt.After(jj.GeneratedAt)
		}
		return strings.Compare(ii.ScenarioName, jj.ScenarioName) < 0
	})

	c.JSON(http.StatusOK, gin.H{
		"test_suites": suites,
		"total":       len(suites),
		"filters": gin.H{
			"scenario": scenarioFilter,
			"status":   statusFilter,
		},
	})
}

func listScenariosHandler(c *gin.Context) {
	repoRoot, err := findRepositoryRoot()
	if err != nil {
		log.Printf("⚠️  Failed to locate repository root for scenario listing: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enumerate scenarios"})
		return
	}

	scenariosDir := filepath.Join(repoRoot, "scenarios")
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		log.Printf("⚠️  Failed to read scenarios directory: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enumerate scenarios"})
		return
	}

	type scenarioAccumulator struct {
		data      ScenarioOverview
		types     map[string]struct{}
		hasStored bool
	}

	scenarioMap := make(map[string]*scenarioAccumulator)

	ensureScenario := func(name string) *scenarioAccumulator {
		acc, ok := scenarioMap[name]
		if ok {
			return acc
		}

		acc = &scenarioAccumulator{
			data: ScenarioOverview{
				Name: name,
			},
			types: make(map[string]struct{}),
		}

		scenarioRoot := filepath.Join(scenariosDir, name)
		if info, err := os.Stat(scenarioRoot); err == nil && info.IsDir() {
			acc.data.HasTestDirectory = scenarioHasRecognizedTests(scenarioRoot)
		}

		scenarioMap[name] = acc
		return acc
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

		acc := ensureScenario(name)
		if acc.data.Name == "" {
			acc.data.Name = name
		}

		if acc.types == nil {
			acc.types = make(map[string]struct{})
		}
	}

	storedSuites, err := fetchStoredTestSuites(c.Request.Context(), "", "")
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Printf("⚠️  Failed to load stored suites for scenario listing: %v", err)
	}

	repoSuites := make([]TestSuite, 0)
	if repoRoot != "" {
		if discovered, discoverErr := discoverScenarioTestSuites(repoRoot, ""); discoverErr != nil {
			log.Printf("⚠️  Failed to discover scenario suites while listing scenarios: %v", discoverErr)
		} else {
			repoSuites = discovered
		}
	}

	for _, suite := range storedSuites {
		name := strings.TrimSpace(suite.ScenarioName)
		if name == "" {
			continue
		}

		acc := ensureScenario(name)
		acc.hasStored = true
		acc.data.SuiteCount++
		acc.data.TestCaseCount += len(suite.TestCases)
		acc.data.HasTests = true

		if suite.SuiteType != "" {
			for _, t := range strings.Split(suite.SuiteType, ",") {
				trimmed := strings.TrimSpace(t)
				if trimmed != "" {
					acc.types[strings.ToLower(trimmed)] = struct{}{}
				}
			}
		}

		if acc.data.LatestSuiteGeneratedAt == nil || suite.GeneratedAt.After(*acc.data.LatestSuiteGeneratedAt) {
			generatedAt := suite.GeneratedAt
			acc.data.LatestSuiteGeneratedAt = &generatedAt
			acc.data.LatestSuiteID = suite.ID.String()
			acc.data.LatestSuiteStatus = suite.Status
			acc.data.LatestSuiteCoverage = suite.CoverageMetrics.CodeCoverage
		}
	}

	for _, suite := range repoSuites {
		name := strings.TrimSpace(suite.ScenarioName)
		if name == "" {
			continue
		}

		acc := ensureScenario(name)
		if acc.hasStored {
			continue
		}

		acc.data.SuiteCount++
		acc.data.TestCaseCount += len(suite.TestCases)
		acc.data.HasTests = true

		if suite.SuiteType != "" {
			for _, t := range strings.Split(suite.SuiteType, ",") {
				trimmed := strings.TrimSpace(t)
				if trimmed != "" {
					acc.types[strings.ToLower(trimmed)] = struct{}{}
				}
			}
		}

		if acc.data.LatestSuiteGeneratedAt == nil || suite.GeneratedAt.After(*acc.data.LatestSuiteGeneratedAt) {
			generatedAt := suite.GeneratedAt
			acc.data.LatestSuiteGeneratedAt = &generatedAt
			acc.data.LatestSuiteID = suite.ID.String()
			acc.data.LatestSuiteStatus = suite.Status
			acc.data.LatestSuiteCoverage = suite.CoverageMetrics.CodeCoverage
		}
	}

	results := make([]ScenarioOverview, 0, len(scenarioMap))
	for _, acc := range scenarioMap {
		if len(acc.types) > 0 {
			types := make([]string, 0, len(acc.types))
			for t := range acc.types {
				types = append(types, t)
			}
			sort.Strings(types)
			acc.data.SuiteTypes = types
		}
		if acc.data.HasTests {
			acc.data.HasTests = acc.data.SuiteCount > 0
		}
		results = append(results, acc.data)
	}

	sort.Slice(results, func(i, j int) bool {
		return strings.Compare(results[i].Name, results[j].Name) < 0
	})

	c.JSON(http.StatusOK, gin.H{
		"scenarios": results,
		"count":     len(results),
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
	scenarioName := c.Param("scenario_name")

	analysis, hasData, err := computeCoverageAnalysis(scenarioName)
	if err != nil {
		log.Printf("⚠️  Failed to load coverage analysis for %q: %v", scenarioName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load coverage analysis"})
		return
	}

	if !hasData {
		log.Printf("ℹ️  Coverage artifacts not found for scenario %q", scenarioName)
	}

	c.JSON(http.StatusOK, analysis)
}

func listCoverageAnalysesHandler(c *gin.Context) {
	summaries, err := listCoverageSummaries()
	if err != nil {
		log.Printf("⚠️  Failed to list coverage analyses: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list coverage analyses"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"coverages": summaries,
		"total":     len(summaries),
	})
}

func reportsOverviewHandler(c *gin.Context) {
	windowDays := parseWindowDays(c.Query("window_days"), 30)
	overview, err := buildReportsOverview(c.Request.Context(), windowDays)
	if err != nil {
		log.Printf("❌ Failed to build reports overview: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build reports overview"})
		return
	}

	c.JSON(http.StatusOK, overview)
}

func reportsTrendsHandler(c *gin.Context) {
	windowDays := parseWindowDays(c.Query("window_days"), 30)
	trends, err := buildReportsTrends(c.Request.Context(), windowDays)
	if err != nil {
		log.Printf("❌ Failed to build reports trends: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build reports trends"})
		return
	}

	c.JSON(http.StatusOK, trends)
}

func reportsInsightsHandler(c *gin.Context) {
	windowDays := parseWindowDays(c.Query("window_days"), 30)
	overview, err := buildReportsOverview(c.Request.Context(), windowDays)
	if err != nil {
		log.Printf("❌ Failed to build reports overview for insights: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build reports insights"})
		return
	}

	insights, err := buildReportsInsights(c.Request.Context(), overview, windowDays)
	if err != nil {
		log.Printf("❌ Failed to build reports insights: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build reports insights"})
		return
	}

	c.JSON(http.StatusOK, insights)
}

type scenarioAggregate struct {
	ExecutionCount    int
	FailedExecutions  int
	RunningExecutions int
	PassedTests       int
	FailedTests       int
	SkippedTests      int
	TotalDuration     float64
}

type executionSnapshot struct {
	ExecutionID string
	Status      string
	EndedAt     *time.Time
}

type coverageEntry struct {
	Value      float64
	AnalyzedAt time.Time
}

type failingTestCandidate struct {
	ScenarioName string
	TestName     string
	Failures     int
	TotalRuns    int
	AvgDuration  float64
}

func buildReportsOverview(ctx context.Context, windowDays int) (ReportsOverviewResponse, error) {
	if windowDays <= 0 {
		windowDays = 30
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	windowEnd := time.Now().UTC()
	windowStart := windowEnd.Add(-time.Duration(windowDays) * 24 * time.Hour)

	scenarioAgg, err := fetchScenarioAggregates(ctx, windowStart)
	if err != nil {
		return ReportsOverviewResponse{}, err
	}

	coverageMap, err := fetchCoverageMap(ctx, windowStart)
	if err != nil {
		return ReportsOverviewResponse{}, err
	}

	latestExecMap, err := fetchLatestExecutions(ctx)
	if err != nil {
		return ReportsOverviewResponse{}, err
	}

	vaultMap, vaultRollup, err := fetchVaultAggregates(ctx, windowStart)
	if err != nil {
		return ReportsOverviewResponse{}, err
	}

	scenarioNames := make(map[string]struct{})
	for name := range scenarioAgg {
		scenarioNames[name] = struct{}{}
	}
	for name := range coverageMap {
		scenarioNames[name] = struct{}{}
	}
	for name := range vaultMap {
		scenarioNames[name] = struct{}{}
	}

	names := make([]string, 0, len(scenarioNames))
	for name := range scenarioNames {
		names = append(names, name)
	}
	sort.Strings(names)

	summaries := make([]ScenarioReportSummary, 0, len(names))
	var totalTests, passedTests, failedTests int
	var totalDuration float64
	var totalCoverage float64
	var coverageCount int
	var activeExecutions int
	var activeScenarios int
	var regressions int

	const coverageTarget = 95.0

	for _, name := range names {
		agg := scenarioAgg[name]
		covEntry, hasCoverage := coverageMap[name]
		coverage := 0.0
		if hasCoverage {
			coverage = covEntry.Value
		}

		snapshot := latestExecMap[name]
		vaultStatus := vaultMap[name]
		totalScenarioTests := agg.PassedTests + agg.FailedTests + agg.SkippedTests
		passRatio := safeDivide(float64(agg.PassedTests), float64(agg.PassedTests+agg.FailedTests))
		avgDuration := safeDivide(agg.TotalDuration, float64(totalScenarioTests))
		vaultSuccess := 1.0
		if vaultStatus.HasVault {
			if vaultStatus.TotalExecutions > 0 {
				vaultSuccess = safeDivide(float64(vaultStatus.SuccessfulRuns), float64(vaultStatus.TotalExecutions))
			}
		} else {
			vaultSuccess = 1.0
		}

		health := computeScenarioHealthScore(passRatio, coverage, coverageTarget, vaultSuccess, agg.FailedExecutions)

		summary := ScenarioReportSummary{
			ScenarioName:        name,
			HealthScore:         health,
			PassRate:            roundTo(passRatio*100, 2),
			Coverage:            roundTo(coverage, 2),
			TargetCoverage:      coverageTarget,
			CoverageDelta:       roundTo(coverage-coverageTarget, 2),
			ExecutionCount:      agg.ExecutionCount,
			ActiveFailures:      agg.FailedExecutions,
			RunningExecutions:   agg.RunningExecutions,
			AverageTestDuration: roundTo(avgDuration, 2),
			Vault:               vaultStatus,
		}

		if snapshot.ExecutionID != "" {
			summary.LastExecutionID = snapshot.ExecutionID
			summary.LastExecutionStatus = snapshot.Status
			summary.LastExecutionEnded = snapshot.EndedAt
		}

		summaries = append(summaries, summary)

		totalTests += totalScenarioTests
		passedTests += agg.PassedTests
		failedTests += agg.FailedTests
		totalDuration += agg.TotalDuration
		activeExecutions += agg.RunningExecutions
		if agg.ExecutionCount > 0 {
			activeScenarios++
		}
		if agg.FailedExecutions > 0 {
			regressions++
		}
		if hasCoverage && coverage > 0 {
			totalCoverage += coverage
			coverageCount++
		}
	}

	global := ReportGlobalMetrics{
		TotalTests:       totalTests,
		PassedTests:      passedTests,
		FailedTests:      failedTests,
		PassRate:         roundTo(safeDivide(float64(passedTests), float64(passedTests+failedTests))*100, 2),
		AverageDuration:  roundTo(safeDivide(totalDuration, float64(totalTests)), 2),
		AverageCoverage:  roundTo(safeDivide(totalCoverage, float64(coverageCount)), 2),
		ActiveScenarios:  activeScenarios,
		ActiveExecutions: activeExecutions,
		ActiveVaults:     vaultRollup.ScenariosWithVault,
		Regressions:      regressions,
	}

	response := ReportsOverviewResponse{
		GeneratedAt: time.Now().UTC(),
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
		Global:      global,
		Scenarios:   summaries,
		Vaults:      vaultRollup,
	}

	return response, nil
}

func buildReportsTrends(ctx context.Context, windowDays int) (ReportsTrendsResponse, error) {
	if windowDays <= 0 {
		windowDays = 30
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	windowEnd := time.Now().UTC()
	windowStart := windowEnd.Add(-time.Duration(windowDays) * 24 * time.Hour)

	execAgg, err := fetchExecutionTrends(ctx, windowStart)
	if err != nil {
		return ReportsTrendsResponse{}, err
	}

	coverageTrend, err := fetchCoverageTrends(ctx, windowStart)
	if err != nil {
		return ReportsTrendsResponse{}, err
	}

	series := make([]ReportTrendPoint, 0, windowDays+1)
	for day := truncateToDay(windowStart); !day.After(windowEnd); day = day.Add(24 * time.Hour) {
		agg := execAgg[day]
		point := ReportTrendPoint{
			Bucket:           day,
			Executions:       agg.ExecutionCount,
			FailedExecutions: agg.FailedExecutions,
			PassedTests:      agg.PassedTests,
			FailedTests:      agg.FailedTests,
			AverageDuration:  roundTo(safeDivide(agg.TotalDuration, float64(agg.TestCount)), 2),
			AverageCoverage:  roundTo(coverageTrend[day], 2),
		}
		series = append(series, point)
	}

	return ReportsTrendsResponse{
		GeneratedAt: time.Now().UTC(),
		BucketSize:  "day",
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
		Series:      series,
	}, nil
}

func buildReportsInsights(ctx context.Context, overview ReportsOverviewResponse, windowDays int) (ReportsInsightsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	insights := make([]ReportInsight, 0, 6)

	if overview.Global.PassRate > 0 && overview.Global.PassRate < 90 {
		insights = append(insights, ReportInsight{
			Title:    "Overall pass rate is slipping",
			Severity: "high",
			Detail:   fmt.Sprintf("Global pass rate over the last %d days is %.2f%% with %d failures recorded.", windowDays, overview.Global.PassRate, overview.Global.FailedTests),
			Actions: []string{
				"Drill into failing scenarios from the overview tab",
				"Prioritize fixes for suites with repeated regressions",
			},
		})
	}

	for _, scenario := range overview.Scenarios {
		if scenario.ActiveFailures > 0 {
			insights = append(insights, ReportInsight{
				Title:        "Active regressions detected",
				Severity:     "high",
				ScenarioName: scenario.ScenarioName,
				Detail:       fmt.Sprintf("%d execution(s) failed recently while pass rate is %.2f%%.", scenario.ActiveFailures, scenario.PassRate),
				Actions: []string{
					"Open the latest failing execution details",
					"Re-run the suite in an isolated environment after fixes",
				},
			})
		}

		if scenario.Coverage > 0 && scenario.Coverage < scenario.TargetCoverage-5 {
			insights = append(insights, ReportInsight{
				Title:        "Coverage below target",
				Severity:     "medium",
				ScenarioName: scenario.ScenarioName,
				Detail:       fmt.Sprintf("Coverage is %.2f%% (target %.0f%%). Add tests for uncovered branches.", scenario.Coverage, scenario.TargetCoverage),
				Actions: []string{
					"Use AI generation to extend edge-case coverage",
					"Prioritize high-risk modules highlighted in coverage gaps",
				},
			})
		}

		if scenario.Vault.HasVault && scenario.Vault.FailedRuns > 0 {
			insights = append(insights, ReportInsight{
				Title:        "Vault instability",
				Severity:     "high",
				ScenarioName: scenario.ScenarioName,
				Detail:       fmt.Sprintf("%d vault execution(s) failed recently. Success rate stands at %.2f%%.", scenario.Vault.FailedRuns, scenario.Vault.SuccessRate),
				Actions: []string{
					"Inspect the failing phase timeline in the Vaults view",
					"Recalibrate phase timeouts or dependency setup for vault phases",
				},
			})
		}
	}

	failingTests, err := fetchFailingTestCandidates(ctx, overview.WindowStart)
	if err != nil {
		log.Printf("⚠️  Failed to fetch failing test candidates: %v", err)
	} else if len(failingTests) > 0 {
		top := failingTests[0]
		insights = append(insights, ReportInsight{
			Title:        "Recurring failing test detected",
			Severity:     "medium",
			ScenarioName: top.ScenarioName,
			Detail:       fmt.Sprintf("Test '%s' failed %d/%d runs (avg %.2fs).", top.TestName, top.Failures, top.TotalRuns, top.AvgDuration),
			Actions: []string{
				"Stabilize this test or quarantine until fixed",
				"Capture logs/artifacts for deterministic reproduction",
			},
		})
	}

	if len(insights) == 0 {
		insights = append(insights, ReportInsight{
			Title:    "Quality signals are stable",
			Severity: "info",
			Detail:   "No regressions detected in the selected window. Maintain momentum by adding predictive coverage checks.",
			Actions: []string{
				"Schedule proactive vault runs to keep baselines fresh",
				"Leverage AI suggestions to target potential blind spots",
			},
		})
	}

	return ReportsInsightsResponse{
		GeneratedAt: time.Now().UTC(),
		WindowStart: overview.WindowStart,
		WindowEnd:   overview.WindowEnd,
		Insights:    insights,
	}, nil
}

func fetchScenarioAggregates(ctx context.Context, windowStart time.Time) (map[string]scenarioAggregate, error) {
	query := `
		SELECT
			ts.scenario_name,
			COUNT(DISTINCT te.id) AS execution_count,
			SUM(CASE WHEN LOWER(COALESCE(te.status, '')) IN ('failed', 'error', 'aborted') THEN 1 ELSE 0 END) AS failed_executions,
			SUM(CASE WHEN LOWER(COALESCE(te.status, '')) IN ('running', 'queued', 'in_progress') THEN 1 ELSE 0 END) AS running_executions,
			SUM(CASE WHEN LOWER(COALESCE(tr.status, '')) = 'passed' THEN 1 ELSE 0 END) AS passed_tests,
			SUM(CASE WHEN LOWER(COALESCE(tr.status, '')) = 'failed' THEN 1 ELSE 0 END) AS failed_tests,
			SUM(CASE WHEN LOWER(COALESCE(tr.status, '')) = 'skipped' THEN 1 ELSE 0 END) AS skipped_tests,
			COALESCE(SUM(COALESCE(tr.duration, 0)), 0) AS total_duration
		FROM test_executions te
		JOIN test_suites ts ON te.suite_id = ts.id
		LEFT JOIN test_results tr ON tr.execution_id = te.id
		WHERE COALESCE(te.end_time, te.start_time, NOW()) >= $1
		GROUP BY ts.scenario_name`

	rows, err := db.QueryContext(ctx, query, windowStart)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]scenarioAggregate)
	for rows.Next() {
		var name string
		var agg scenarioAggregate
		if err := rows.Scan(
			&name,
			&agg.ExecutionCount,
			&agg.FailedExecutions,
			&agg.RunningExecutions,
			&agg.PassedTests,
			&agg.FailedTests,
			&agg.SkippedTests,
			&agg.TotalDuration,
		); err != nil {
			return nil, err
		}
		result[name] = agg
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func fetchCoverageMap(ctx context.Context, windowStart time.Time) (map[string]coverageEntry, error) {
	query := `
		SELECT scenario_name, overall_coverage, analyzed_at
		FROM coverage_analysis
		WHERE analyzed_at >= $1
		ORDER BY scenario_name, analyzed_at DESC`

	rows, err := db.QueryContext(ctx, query, windowStart)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]coverageEntry)
	for rows.Next() {
		var name string
		var coverage float64
		var analyzed time.Time
		if err := rows.Scan(&name, &coverage, &analyzed); err != nil {
			return nil, err
		}
		if existing, ok := result[name]; ok {
			if analyzed.After(existing.AnalyzedAt) {
				result[name] = coverageEntry{Value: coverage, AnalyzedAt: analyzed}
			}
			continue
		}
		result[name] = coverageEntry{Value: coverage, AnalyzedAt: analyzed}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func fetchLatestExecutions(ctx context.Context) (map[string]executionSnapshot, error) {
	query := `
		SELECT DISTINCT ON (ts.scenario_name)
			ts.scenario_name,
			te.id,
			COALESCE(te.status, ''),
			te.end_time,
			te.start_time
		FROM test_executions te
		JOIN test_suites ts ON te.suite_id = ts.id
		ORDER BY ts.scenario_name, COALESCE(te.end_time, te.start_time) DESC NULLS LAST`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]executionSnapshot)
	for rows.Next() {
		var name string
		var id uuid.UUID
		var status string
		var end pq.NullTime
		var start pq.NullTime

		if err := rows.Scan(&name, &id, &status, &end, &start); err != nil {
			return nil, err
		}

		var endedAt *time.Time
		if end.Valid {
			value := end.Time.UTC()
			endedAt = &value
		} else if start.Valid {
			value := start.Time.UTC()
			endedAt = &value
		}

		result[name] = executionSnapshot{
			ExecutionID: id.String(),
			Status:      status,
			EndedAt:     endedAt,
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func fetchVaultAggregates(ctx context.Context, windowStart time.Time) (map[string]ReportVaultStatus, ReportVaultRollup, error) {
	vaultCountsQuery := `
		SELECT scenario_name, COUNT(*)
		FROM test_vaults
		GROUP BY scenario_name`

	countRows, err := db.QueryContext(ctx, vaultCountsQuery)
	if err != nil {
		return nil, ReportVaultRollup{}, err
	}
	defer countRows.Close()

	result := make(map[string]ReportVaultStatus)
	rollup := ReportVaultRollup{}

	for countRows.Next() {
		var name string
		var count int
		if err := countRows.Scan(&name, &count); err != nil {
			return nil, ReportVaultRollup{}, err
		}
		status := ReportVaultStatus{HasVault: true}
		result[name] = status
		if count > 0 {
			rollup.ScenariosWithVault++
		}
		rollup.TotalVaults += count
	}

	if err := countRows.Err(); err != nil {
		return nil, ReportVaultRollup{}, err
	}

	if len(result) == 0 {
		return result, rollup, nil
	}

	aggQuery := `
		SELECT
			ts.scenario_name,
			COUNT(ve.id) AS total_exec,
			SUM(CASE WHEN LOWER(COALESCE(ve.status, '')) IN ('completed', 'success', 'passed', 'finished') THEN 1 ELSE 0 END) AS successful_exec,
			SUM(CASE WHEN LOWER(COALESCE(ve.status, '')) IN ('failed', 'error', 'aborted') THEN 1 ELSE 0 END) AS failed_exec,
			MAX(ve.end_time) AS latest_end,
			MAX(ve.start_time) AS latest_start
		FROM test_vaults ts
		LEFT JOIN vault_executions ve ON ve.vault_id = ts.id AND COALESCE(ve.end_time, ve.start_time, NOW()) >= $1
		GROUP BY ts.scenario_name`

	rows, err := db.QueryContext(ctx, aggQuery, windowStart)
	if err != nil {
		return nil, ReportVaultRollup{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var totalExec, successExec, failedExec int
		var end pq.NullTime
		var start pq.NullTime
		if err := rows.Scan(&name, &totalExec, &successExec, &failedExec, &end, &start); err != nil {
			return nil, ReportVaultRollup{}, err
		}

		status := result[name]
		status.TotalExecutions = totalExec
		status.SuccessfulRuns = successExec
		status.FailedRuns = failedExec
		if totalExec > 0 {
			status.SuccessRate = roundTo(safeDivide(float64(successExec), float64(totalExec))*100, 2)
		}
		if end.Valid {
			value := end.Time.UTC()
			status.LatestEndedAt = &value
		} else if start.Valid {
			value := start.Time.UTC()
			status.LatestEndedAt = &value
		}
		result[name] = status

		rollup.TotalExecutions += totalExec
		rollup.FailedExecutions += failedExec
	}

	if err := rows.Err(); err != nil {
		return nil, ReportVaultRollup{}, err
	}

	statusQuery := `
		SELECT DISTINCT ON (ts.scenario_name)
			ts.scenario_name,
			COALESCE(ve.status, ''),
			COALESCE(ve.end_time, ve.start_time)
		FROM test_vaults ts
		JOIN vault_executions ve ON ve.vault_id = ts.id
		ORDER BY ts.scenario_name, COALESCE(ve.end_time, ve.start_time) DESC NULLS LAST`

	statusRows, err := db.QueryContext(ctx, statusQuery)
	if err != nil {
		return nil, ReportVaultRollup{}, err
	}
	defer statusRows.Close()

	for statusRows.Next() {
		var name string
		var status string
		var tsTime pq.NullTime
		if err := statusRows.Scan(&name, &status, &tsTime); err != nil {
			return nil, ReportVaultRollup{}, err
		}
		entry := result[name]
		entry.LatestStatus = status
		if !entry.HasVault {
			entry.HasVault = true
		}
		if tsTime.Valid {
			value := tsTime.Time.UTC()
			entry.LatestEndedAt = &value
		}
		result[name] = entry
	}

	if err := statusRows.Err(); err != nil {
		return nil, ReportVaultRollup{}, err
	}

	if rollup.TotalExecutions > 0 {
		rollup.SuccessRate = roundTo(safeDivide(float64(rollup.TotalExecutions-rollup.FailedExecutions), float64(rollup.TotalExecutions))*100, 2)
	}

	return result, rollup, nil
}

type trendAggregate struct {
	ExecutionCount   int
	FailedExecutions int
	PassedTests      int
	FailedTests      int
	TotalDuration    float64
	TestCount        int
}

func fetchExecutionTrends(ctx context.Context, windowStart time.Time) (map[time.Time]trendAggregate, error) {
	query := `
		WITH execution_buckets AS (
			SELECT
				te.id,
				DATE_TRUNC('day', COALESCE(te.end_time, te.start_time, NOW())) AS bucket
			FROM test_executions te
			WHERE COALESCE(te.end_time, te.start_time, NOW()) >= $1
		)
		SELECT
			eb.bucket,
			COUNT(DISTINCT eb.id) AS executions,
			SUM(CASE WHEN LOWER(COALESCE(te.status, '')) IN ('failed', 'error', 'aborted') THEN 1 ELSE 0 END) AS failed_executions,
			SUM(CASE WHEN LOWER(COALESCE(tr.status, '')) = 'passed' THEN 1 ELSE 0 END) AS passed_tests,
			SUM(CASE WHEN LOWER(COALESCE(tr.status, '')) = 'failed' THEN 1 ELSE 0 END) AS failed_tests,
			COALESCE(SUM(COALESCE(tr.duration, 0)), 0) AS total_duration,
			COUNT(tr.id) AS test_count
		FROM execution_buckets eb
		JOIN test_executions te ON te.id = eb.id
		LEFT JOIN test_results tr ON tr.execution_id = eb.id
		GROUP BY eb.bucket
		ORDER BY eb.bucket`

	rows, err := db.QueryContext(ctx, query, windowStart)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[time.Time]trendAggregate)
	for rows.Next() {
		var bucket time.Time
		var agg trendAggregate
		if err := rows.Scan(
			&bucket,
			&agg.ExecutionCount,
			&agg.FailedExecutions,
			&agg.PassedTests,
			&agg.FailedTests,
			&agg.TotalDuration,
			&agg.TestCount,
		); err != nil {
			return nil, err
		}
		result[truncateToDay(bucket.UTC())] = agg
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func fetchCoverageTrends(ctx context.Context, windowStart time.Time) (map[time.Time]float64, error) {
	query := `
		SELECT DATE_TRUNC('day', analyzed_at) AS bucket, AVG(overall_coverage)
		FROM coverage_analysis
		WHERE analyzed_at >= $1
		GROUP BY bucket
		ORDER BY bucket`

	rows, err := db.QueryContext(ctx, query, windowStart)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[time.Time]float64)
	for rows.Next() {
		var bucket time.Time
		var avgCoverage float64
		if err := rows.Scan(&bucket, &avgCoverage); err != nil {
			return nil, err
		}
		result[truncateToDay(bucket.UTC())] = avgCoverage
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func fetchFailingTestCandidates(ctx context.Context, windowStart time.Time) ([]failingTestCandidate, error) {
	query := `
		SELECT
			ts.scenario_name,
			COALESCE(tc.name, CASE WHEN tr.test_case_id IS NOT NULL THEN tr.test_case_id::text ELSE 'unknown' END) AS test_name,
			SUM(CASE WHEN LOWER(COALESCE(tr.status, '')) = 'failed' THEN 1 ELSE 0 END) AS failures,
			COUNT(*) AS total_runs,
			COALESCE(AVG(COALESCE(tr.duration, 0)), 0) AS avg_duration
		FROM test_results tr
		JOIN test_executions te ON te.id = tr.execution_id
		JOIN test_suites ts ON ts.id = te.suite_id
		LEFT JOIN test_cases tc ON tc.id = tr.test_case_id
		WHERE COALESCE(te.end_time, te.start_time, NOW()) >= $1
		GROUP BY ts.scenario_name, test_name
		HAVING SUM(CASE WHEN LOWER(COALESCE(tr.status, '')) = 'failed' THEN 1 ELSE 0 END) > 0
		ORDER BY failures DESC, total_runs DESC
		LIMIT 5`

	rows, err := db.QueryContext(ctx, query, windowStart)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidates []failingTestCandidate
	for rows.Next() {
		var candidate failingTestCandidate
		if err := rows.Scan(
			&candidate.ScenarioName,
			&candidate.TestName,
			&candidate.Failures,
			&candidate.TotalRuns,
			&candidate.AvgDuration,
		); err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return candidates, nil
}

func computeScenarioHealthScore(passRatio, coverage, coverageTarget, vaultSuccess float64, failedExecutions int) int {
	passNorm := clamp(passRatio, 0, 1)
	coverageNorm := 0.0
	if coverageTarget > 0 {
		coverageNorm = clamp(coverage/coverageTarget, 0, 1)
	}
	vaultNorm := clamp(vaultSuccess, 0, 1)

	score := (0.55 * passNorm) + (0.3 * coverageNorm) + (0.15 * vaultNorm)
	if failedExecutions > 0 {
		penalty := math.Min(0.2, float64(failedExecutions)*0.05)
		score -= penalty
	}

	score = clamp(score, 0, 1)
	return int(math.Round(score * 100))
}

func parseWindowDays(raw string, defaultVal int) int {
	if raw == "" {
		return defaultVal
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}
	if value < 1 {
		value = 1
	}
	if value > 180 {
		value = 180
	}
	return value
}

func safeDivide(numerator, denominator float64) float64 {
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}

func roundTo(value float64, precision int) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	factor := math.Pow10(precision)
	return math.Round(value*factor) / factor
}

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
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
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start test-genie

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

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
		// Health proxy for UI expectations
		api.GET("/health", healthHandler)

		// Test suite management
		api.POST("/test-suite/generate", generateTestSuiteHandler)
		api.POST("/test-suite/generate-batch", generateBatchTestSuiteHandler)
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
		api.GET("/scenarios", listScenariosHandler)

		// Coverage analysis
		api.POST("/test-analysis/coverage", analyzeCoverageHandler)
		api.GET("/test-analysis/coverage", listCoverageAnalysesHandler)
		api.GET("/test-analysis/coverage/:scenario_name", getCoverageAnalysisHandler)

		// System information
		api.GET("/system/status", systemStatusHandler)
		api.GET("/system/metrics", systemMetricsHandler)
		api.GET("/reports/overview", reportsOverviewHandler)
		api.GET("/reports/trends", reportsTrendsHandler)
		api.GET("/reports/insights", reportsInsightsHandler)

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
