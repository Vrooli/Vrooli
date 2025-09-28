package models

import (
	"github.com/google/uuid"
	"time"
)

// Test Suite Models
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
	PhaseName    string                 `json:"phase_name"`
	Status       string                 `json:"status"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
	TestResults  []TestResult           `json:"test_results"`
	Metrics      map[string]interface{} `json:"metrics"`
	ErrorMessage string                 `json:"error_message,omitempty"`
}

// Health Check Types
type HealthStatus struct {
	Healthy   bool                     `json:"healthy"`
	Message   string                   `json:"message,omitempty"`
	Services  map[string]ServiceHealth `json:"services,omitempty"`
	Timestamp time.Time                `json:"timestamp"`
}

type ServiceHealth struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// AI Integration Types
type AIRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type AIResponse struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Response           string    `json:"response"`
	Done               bool      `json:"done"`
	Context            []int     `json:"context,omitempty"`
	TotalDuration      int64     `json:"total_duration,omitempty"`
	LoadDuration       int64     `json:"load_duration,omitempty"`
	PromptEvalCount    int       `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64     `json:"prompt_eval_duration,omitempty"`
	EvalCount          int       `json:"eval_count,omitempty"`
	EvalDuration       int64     `json:"eval_duration,omitempty"`
}
