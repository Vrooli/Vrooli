package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type AlgorithmProcessor struct {
	httpClient    *http.Client
	judge0URL     string
	localExecutor *LocalExecutor
	useLocal      bool
}

// Algorithm Execution types
type AlgorithmExecutionRequest struct {
	Code           string `json:"code"`
	Language       string `json:"language"`
	Stdin          string `json:"stdin,omitempty"`
	ExpectedOutput string `json:"expected_output,omitempty"`
	Timeout        int    `json:"timeout,omitempty"`
}

type AlgorithmExecutionResponse struct {
	Success           bool                   `json:"success"`
	Status            string                 `json:"status"`
	StatusID          int                    `json:"status_id"`
	ExecutionComplete bool                   `json:"execution_complete"`
	Output            string                 `json:"output"`
	ErrorOutput       string                 `json:"error_output"`
	CompileOutput     string                 `json:"compile_output"`
	Message           string                 `json:"message"`
	ExecutionTime     string                 `json:"execution_time"`
	MemoryUsed        string                 `json:"memory_used"`
	TestResult        *TestComparison        `json:"test_result,omitempty"`
	Language          string                 `json:"language"`
	ExecutionID       string                 `json:"execution_id"`
	SubmissionToken   string                 `json:"submission_token"`
	ErrorDetails      *ExecutionErrorDetails `json:"error_details,omitempty"`
}

type TestComparison struct {
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Match    bool   `json:"match"`
}

type ExecutionErrorDetails struct {
	Status       string `json:"status"`
	CompileError string `json:"compile_error,omitempty"`
	RuntimeError string `json:"runtime_error,omitempty"`
	Message      string `json:"message,omitempty"`
}

// Batch Validation types
type BatchValidationRequest struct {
	AlgorithmID    string     `json:"algorithm_id"`
	Language       string     `json:"language"`
	Implementation string     `json:"implementation"`
	TestCases      []TestCase `json:"test_cases"`
}

type TestCase struct {
	Input    map[string]interface{} `json:"input"`
	Expected interface{}            `json:"expected"`
}

type BatchValidationResponse struct {
	BatchID             string                  `json:"batch_id"`
	AlgorithmID         string                  `json:"algorithm_id"`
	Language            string                  `json:"language"`
	ValidationSummary   ValidationSummary       `json:"validation_summary"`
	PerformanceMetrics  BatchPerformanceMetrics `json:"performance_metrics"`
	StatusBreakdown     map[string]int          `json:"status_breakdown"`
	TestResults         []BatchTestResult       `json:"test_results"`
	ValidationTimestamp ValidationTimestamp     `json:"validation_timestamp"`
	Recommendation      string                  `json:"recommendation"`
}

type ValidationSummary struct {
	TotalTests  int     `json:"total_tests"`
	Passed      int     `json:"passed"`
	Failed      int     `json:"failed"`
	SuccessRate float64 `json:"success_rate"`
	AllPassed   bool    `json:"all_passed"`
}

type BatchPerformanceMetrics struct {
	AverageExecutionTime string `json:"average_execution_time"`
	TotalExecutionTime   string `json:"total_execution_time"`
}

type BatchTestResult struct {
	TestIndex      int                    `json:"test_index"`
	Input          map[string]interface{} `json:"input"`
	Passed         bool                   `json:"passed"`
	Status         string                 `json:"status"`
	ActualOutput   string                 `json:"actual_output"`
	ExpectedOutput string                 `json:"expected_output"`
	ExecutionTime  string                 `json:"execution_time"`
	Error          *ExecutionErrorDetails `json:"error,omitempty"`
}

type ValidationTimestamp struct {
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at"`
}

// Judge0 API types
type Judge0Submission struct {
	SourceCode     string `json:"source_code"`
	LanguageID     int    `json:"language_id"`
	Stdin          string `json:"stdin,omitempty"`
	CPUTimeLimit   int    `json:"cpu_time_limit,omitempty"`
	MemoryLimit    int    `json:"memory_limit,omitempty"`
	ExpectedOutput string `json:"expected_output,omitempty"`
}

type Judge0Response struct {
	Token         string       `json:"token"`
	Status        Judge0Status `json:"status"`
	Stdout        string       `json:"stdout,omitempty"`
	Stderr        string       `json:"stderr,omitempty"`
	CompileOutput string       `json:"compile_output,omitempty"`
	Message       string       `json:"message,omitempty"`
	Time          string       `json:"time,omitempty"`
	Memory        int          `json:"memory,omitempty"`
}

type Judge0Status struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
}

func NewAlgorithmProcessor(judge0URL string) *AlgorithmProcessor {
	if judge0URL == "" {
		judge0URL = "http://localhost:2358" // Default Judge0 port
	}

	// Check if Judge0 is available and functional
	useLocal := false
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(judge0URL + "/about")
	if err != nil || resp.StatusCode != 200 {
		log.Printf("⚠️  Judge0 not available at %s, using local executor fallback", judge0URL)
		useLocal = true
	} else {
		resp.Body.Close()
		// Force local executor due to known cgroup issues with Judge0
		log.Printf("⚠️  Judge0 available at %s but has known cgroup issues, using local executor", judge0URL)
		useLocal = true
	}

	return &AlgorithmProcessor{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		judge0URL:     judge0URL,
		localExecutor: NewLocalExecutor(5 * time.Second),
		useLocal:      useLocal,
	}
}

// ExecuteAlgorithm executes code using Judge0 (replaces algorithm-executor workflow)
func (ap *AlgorithmProcessor) ExecuteAlgorithm(ctx context.Context, req AlgorithmExecutionRequest) (*AlgorithmExecutionResponse, error) {
	executionID := fmt.Sprintf("algo_%d_%s", time.Now().Unix(), generateRandomString(6))

	// Validate required fields
	if req.Code == "" || strings.TrimSpace(req.Code) == "" {
		return &AlgorithmExecutionResponse{
			Success:           false,
			Status:            "error",
			StatusID:          -1,
			ExecutionComplete: true,
			ErrorDetails: &ExecutionErrorDetails{
				Status:  "validation_error",
				Message: "Missing required parameter: code",
			},
			ExecutionID: executionID,
		}, nil
	}

	if req.Language == "" {
		return &AlgorithmExecutionResponse{
			Success:           false,
			Status:            "error",
			StatusID:          -1,
			ExecutionComplete: true,
			ErrorDetails: &ExecutionErrorDetails{
				Status:  "validation_error",
				Message: "Missing required parameter: language",
			},
			ExecutionID: executionID,
		}, nil
	}

	// Use local executor if Judge0 is unavailable or for testing
	if ap.useLocal {
		return ap.executeLocal(ctx, req, executionID)
	}

	// Get language ID for Judge0
	languageID, err := ap.getLanguageID(req.Language)
	if err != nil {
		return &AlgorithmExecutionResponse{
			Success:           false,
			Status:            "error",
			StatusID:          -1,
			ExecutionComplete: true,
			ErrorDetails: &ExecutionErrorDetails{
				Status:  "language_error",
				Message: fmt.Sprintf("Unsupported language: %s", req.Language),
			},
			ExecutionID: executionID,
		}, nil
	}

	// Set timeout limits
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 5
	}
	if timeout > 30 {
		timeout = 30
	}

	// Build Judge0 submission
	submission := Judge0Submission{
		SourceCode:     req.Code,
		LanguageID:     languageID,
		Stdin:          req.Stdin,
		CPUTimeLimit:   timeout,
		MemoryLimit:    128000, // 128MB
		ExpectedOutput: req.ExpectedOutput,
	}

	// Submit to Judge0
	response, err := ap.submitToJudge0(ctx, submission)
	if err != nil {
		return &AlgorithmExecutionResponse{
			Success:           false,
			Status:            "submission_error",
			StatusID:          -1,
			ExecutionComplete: true,
			ErrorDetails: &ExecutionErrorDetails{
				Status:  "submission_error",
				Message: fmt.Sprintf("Failed to submit to Judge0: %v", err),
			},
			ExecutionID: executionID,
		}, nil
	}

	// Wait for execution to complete
	time.Sleep(2 * time.Second)

	// Get results
	result, err := ap.getJudge0Result(ctx, response.Token)
	if err != nil {
		return &AlgorithmExecutionResponse{
			Success:           false,
			Status:            "result_error",
			StatusID:          -1,
			ExecutionComplete: true,
			ErrorDetails: &ExecutionErrorDetails{
				Status:  "result_error",
				Message: fmt.Sprintf("Failed to get results from Judge0: %v", err),
			},
			ExecutionID: executionID,
		}, nil
	}

	// Process and format response
	return ap.formatExecutionResponse(result, req, executionID), nil
}

// ValidateBatch runs multiple test cases and compiles results (replaces batch-validator workflow)
func (ap *AlgorithmProcessor) ValidateBatch(ctx context.Context, req BatchValidationRequest) (*BatchValidationResponse, error) {
	batchID := fmt.Sprintf("batch_%d_%s", time.Now().Unix(), generateRandomString(6))
	startTime := time.Now()

	// Validate input
	if req.AlgorithmID == "" {
		return nil, fmt.Errorf("missing required parameter: algorithm_id")
	}
	if req.Language == "" {
		return nil, fmt.Errorf("missing required parameter: language")
	}
	if req.Implementation == "" {
		return nil, fmt.Errorf("missing required parameter: implementation")
	}
	if len(req.TestCases) == 0 {
		return nil, fmt.Errorf("missing or empty test_cases array")
	}

	var testResults []BatchTestResult
	var statusBreakdown = make(map[string]int)
	var executionTimes []float64

	// Process each test case
	for i, testCase := range req.TestCases {
		// Prepare test code
		testCode := ap.prepareTestCode(req.Implementation, req.Language, testCase)
		expectedOutput := ap.formatExpectedOutput(testCase.Expected)

		// Execute test
		execReq := AlgorithmExecutionRequest{
			Code:           testCode,
			Language:       req.Language,
			Stdin:          "",
			ExpectedOutput: expectedOutput,
			Timeout:        5,
		}

		execResult, err := ap.ExecuteAlgorithm(ctx, execReq)
		if err != nil {
			log.Printf("Test case %d execution failed: %v", i, err)
			continue
		}

		// Collect result
		testResult := BatchTestResult{
			TestIndex:      i,
			Input:          testCase.Input,
			Passed:         execResult.Success,
			Status:         execResult.Status,
			ActualOutput:   execResult.Output,
			ExpectedOutput: expectedOutput,
			ExecutionTime:  execResult.ExecutionTime,
			Error:          execResult.ErrorDetails,
		}

		testResults = append(testResults, testResult)

		// Update status breakdown
		statusBreakdown[execResult.Status]++

		// Collect execution time for averaging
		if execResult.ExecutionTime != "" {
			if time, err := parseExecutionTime(execResult.ExecutionTime); err == nil {
				executionTimes = append(executionTimes, time)
			}
		}
	}

	// Calculate summary statistics
	totalTests := len(testResults)
	passedTests := 0
	for _, result := range testResults {
		if result.Passed {
			passedTests++
		}
	}
	failedTests := totalTests - passedTests
	successRate := 0.0
	if totalTests > 0 {
		successRate = float64(passedTests) / float64(totalTests) * 100.0
	}

	// Calculate performance metrics
	avgTime := ""
	totalTime := ""
	if len(executionTimes) > 0 {
		sum := 0.0
		for _, t := range executionTimes {
			sum += t
		}
		avgTime = fmt.Sprintf("%.2f ms", sum/float64(len(executionTimes)))
		totalTime = fmt.Sprintf("%.2f ms", sum)
	}

	// Generate recommendation
	recommendation := ""
	if passedTests == totalTests {
		recommendation = "Implementation is valid and passes all test cases"
	} else {
		recommendation = fmt.Sprintf("Implementation has issues. %d test(s) failed. Review the failed test cases for details.", failedTests)
	}

	completedAt := time.Now()

	return &BatchValidationResponse{
		BatchID:     batchID,
		AlgorithmID: req.AlgorithmID,
		Language:    req.Language,
		ValidationSummary: ValidationSummary{
			TotalTests:  totalTests,
			Passed:      passedTests,
			Failed:      failedTests,
			SuccessRate: successRate,
			AllPassed:   passedTests == totalTests,
		},
		PerformanceMetrics: BatchPerformanceMetrics{
			AverageExecutionTime: avgTime,
			TotalExecutionTime:   totalTime,
		},
		StatusBreakdown: statusBreakdown,
		TestResults:     testResults,
		ValidationTimestamp: ValidationTimestamp{
			StartedAt:   startTime.Format(time.RFC3339),
			CompletedAt: completedAt.Format(time.RFC3339),
		},
		Recommendation: recommendation,
	}, nil
}

// Helper methods

func (ap *AlgorithmProcessor) getLanguageID(language string) (int, error) {
	languageMap := map[string]int{
		"python":     71, // Python 3.8
		"javascript": 63, // JavaScript (Node.js)
		"java":       62, // Java (OpenJDK)
		"cpp":        54, // C++ (GCC)
		"c":          50, // C (GCC)
		"go":         60, // Go
		"rust":       73, // Rust
		"ruby":       72, // Ruby
		"typescript": 74, // TypeScript
		"csharp":     51, // C#
	}

	id, exists := languageMap[strings.ToLower(language)]
	if !exists {
		return 0, fmt.Errorf("unsupported language: %s", language)
	}
	return id, nil
}

func (ap *AlgorithmProcessor) submitToJudge0(ctx context.Context, submission Judge0Submission) (*Judge0Response, error) {
	jsonData, err := json.Marshal(submission)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal submission: %w", err)
	}

	url := fmt.Sprintf("%s/submissions", ap.judge0URL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ap.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to submit to Judge0: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("Judge0 submission failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response Judge0Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

func (ap *AlgorithmProcessor) getJudge0Result(ctx context.Context, token string) (*Judge0Response, error) {
	url := fmt.Sprintf("%s/submissions/%s", ap.judge0URL, token)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := ap.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get results from Judge0: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Judge0 result retrieval failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response Judge0Response
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

func (ap *AlgorithmProcessor) formatExecutionResponse(result *Judge0Response, req AlgorithmExecutionRequest, executionID string) *AlgorithmExecutionResponse {
	statusDescriptions := map[int]string{
		1:  "In Queue",
		2:  "Processing",
		3:  "Accepted",
		4:  "Wrong Answer",
		5:  "Time Limit Exceeded",
		6:  "Compilation Error",
		7:  "Runtime Error (SIGSEGV)",
		8:  "Runtime Error (SIGXFSZ)",
		9:  "Runtime Error (SIGFPE)",
		10: "Runtime Error (SIGABRT)",
		11: "Runtime Error (NZEC)",
		12: "Runtime Error (Other)",
		13: "Internal Error",
		14: "Exec Format Error",
	}

	statusDescription := statusDescriptions[result.Status.ID]
	if statusDescription == "" {
		statusDescription = "Unknown Status"
	}

	passed := result.Status.ID == 3
	executionComplete := result.Status.ID > 2

	// Format execution time
	executionTime := ""
	if result.Time != "" {
		if timeVal := parseFloat(result.Time); timeVal > 0 {
			executionTime = fmt.Sprintf("%.0f ms", timeVal*1000)
		}
	}

	// Format memory usage
	memoryUsed := ""
	if result.Memory > 0 {
		memoryUsed = fmt.Sprintf("%d KB", result.Memory)
	}

	response := &AlgorithmExecutionResponse{
		Success:           passed,
		Status:            statusDescription,
		StatusID:          result.Status.ID,
		ExecutionComplete: executionComplete,
		Output:            result.Stdout,
		ErrorOutput:       result.Stderr,
		CompileOutput:     result.CompileOutput,
		Message:           result.Message,
		ExecutionTime:     executionTime,
		MemoryUsed:        memoryUsed,
		Language:          req.Language,
		ExecutionID:       executionID,
		SubmissionToken:   result.Token,
	}

	// Add test comparison if expected output provided
	if req.ExpectedOutput != "" {
		response.TestResult = &TestComparison{
			Expected: req.ExpectedOutput,
			Actual:   strings.TrimSpace(result.Stdout),
			Match:    passed,
		}
	}

	// Add error details if execution failed
	if !passed && executionComplete {
		response.ErrorDetails = &ExecutionErrorDetails{
			Status:       statusDescription,
			CompileError: result.CompileOutput,
			RuntimeError: result.Stderr,
			Message:      result.Message,
		}
	}

	return response
}

func (ap *AlgorithmProcessor) prepareTestCode(implementation, language string, testCase TestCase) string {
	var testCode string

	switch strings.ToLower(language) {
	case "python":
		testCode = ap.preparePythonTestCode(implementation, testCase)
	case "javascript":
		testCode = ap.prepareJavaScriptTestCode(implementation, testCase)
	default:
		// Generic fallback - just use implementation as-is
		testCode = implementation
	}

	return testCode
}

func (ap *AlgorithmProcessor) preparePythonTestCode(implementation string, testCase TestCase) string {
	inputJSON, _ := json.Marshal(testCase.Input)

	testCode := fmt.Sprintf(`%s

# Test execution
import json
test_input = json.loads('%s')

# Handle different input parameter names
result = None
if 'arr' in test_input:
    result = quicksort(test_input['arr']) if 'quicksort' in globals() else None
elif 'array' in test_input:
    result = quicksort(test_input['array']) if 'quicksort' in globals() else None
elif 'n' in test_input:
    result = fibonacci(test_input['n']) if 'fibonacci' in globals() else None

print(json.dumps(result))
`, implementation, string(inputJSON))

	return testCode
}

func (ap *AlgorithmProcessor) prepareJavaScriptTestCode(implementation string, testCase TestCase) string {
	inputJSON, _ := json.Marshal(testCase.Input)

	testCode := fmt.Sprintf(`%s

// Test execution
const testInput = %s;
let result;

// Handle different input parameter names
if ('arr' in testInput && typeof quicksort !== 'undefined') {
    result = quicksort(testInput.arr);
} else if ('array' in testInput && typeof quicksort !== 'undefined') {
    result = quicksort(testInput.array);
} else if ('n' in testInput && typeof fibonacci !== 'undefined') {
    result = fibonacci(testInput.n);
}

console.log(JSON.stringify(result));
`, implementation, string(inputJSON))

	return testCode
}

func (ap *AlgorithmProcessor) formatExpectedOutput(expected interface{}) string {
	expectedJSON, _ := json.Marshal(expected)
	return string(expectedJSON)
}

// Utility functions
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func parseExecutionTime(timeStr string) (float64, error) {
	var time float64
	if strings.HasSuffix(timeStr, " ms") {
		_, err := fmt.Sscanf(timeStr, "%f ms", &time)
		return time, err
	}
	return 0, fmt.Errorf("invalid time format: %s", timeStr)
}

// executeLocal uses the local executor when Judge0 is unavailable
func (ap *AlgorithmProcessor) executeLocal(ctx context.Context, req AlgorithmExecutionRequest, executionID string) (*AlgorithmExecutionResponse, error) {
	var result *LocalExecutionResult
	var err error

	// Execute based on language
	switch strings.ToLower(req.Language) {
	case "python", "python3", "py":
		result, err = ap.localExecutor.ExecutePython(req.Code, req.Stdin)
	case "javascript", "js", "node":
		result, err = ap.localExecutor.ExecuteJavaScript(req.Code, req.Stdin)
	case "go", "golang":
		result, err = ap.localExecutor.ExecuteGo(req.Code, req.Stdin)
	case "java":
		result, err = ap.localExecutor.ExecuteJava(req.Code, req.Stdin)
	case "cpp", "c++", "cplusplus":
		result, err = ap.localExecutor.ExecuteCPP(req.Code, req.Stdin)
	default:
		return &AlgorithmExecutionResponse{
			Success:           false,
			Status:            "error",
			StatusID:          -1,
			ExecutionComplete: true,
			ErrorDetails: &ExecutionErrorDetails{
				Status:  "language_error",
				Message: fmt.Sprintf("Language %s not supported in local executor", req.Language),
			},
			ExecutionID: executionID,
		}, nil
	}

	if err != nil {
		return &AlgorithmExecutionResponse{
			Success:           false,
			Status:            "execution_error",
			StatusID:          -1,
			ExecutionComplete: true,
			ErrorDetails: &ExecutionErrorDetails{
				Status:  "execution_error",
				Message: fmt.Sprintf("Failed to execute: %v", err),
			},
			ExecutionID: executionID,
		}, nil
	}

	// Compare with expected output if provided
	var testResult *TestComparison
	if req.ExpectedOutput != "" {
		match := strings.TrimSpace(result.Output) == strings.TrimSpace(req.ExpectedOutput)
		testResult = &TestComparison{
			Expected: req.ExpectedOutput,
			Actual:   result.Output,
			Match:    match,
		}
	}

	// Format response
	status := "completed"
	statusID := 3 // Success in Judge0 terms
	if !result.Success {
		status = "runtime_error"
		statusID = 11
	}

	return &AlgorithmExecutionResponse{
		Success:           result.Success,
		Status:            status,
		StatusID:          statusID,
		ExecutionComplete: true,
		Output:            result.Output,
		ErrorOutput:       result.Error,
		CompileOutput:     "",
		Message:           "Executed using local fallback",
		ExecutionTime:     fmt.Sprintf("%.3f", result.ExecutionTime),
		MemoryUsed:        "N/A",
		TestResult:        testResult,
		Language:          req.Language,
		ExecutionID:       executionID,
		SubmissionToken:   "local_" + executionID,
		ErrorDetails:      nil,
	}, nil
}
