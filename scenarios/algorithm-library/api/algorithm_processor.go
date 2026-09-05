package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

const (
	// LocalExecutionTokenPrefix identifies tokens from local execution.
	LocalExecutionTokenPrefix = "local_"
)

type AlgorithmProcessor struct {
	localExecutor *LocalExecutor
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

func NewAlgorithmProcessor() *AlgorithmProcessor {
	return &AlgorithmProcessor{
		localExecutor: NewLocalExecutor(5 * time.Second),
	}
}

// ExecuteAlgorithm executes code through the scenario's local multi-language executor.
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

	return ap.executeLocal(ctx, req, executionID)
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

func parseExecutionTime(timeStr string) (float64, error) {
	var time float64
	if strings.HasSuffix(timeStr, " ms") {
		_, err := fmt.Sscanf(timeStr, "%f ms", &time)
		return time, err
	}
	return 0, fmt.Errorf("invalid time format: %s", timeStr)
}

// executeLocal runs code through the scenario's local multi-language executor.
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
	statusID := 3
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
		Message:           "Executed using the local executor",
		ExecutionTime:     fmt.Sprintf("%.3f", result.ExecutionTime),
		MemoryUsed:        "N/A",
		TestResult:        testResult,
		Language:          req.Language,
		ExecutionID:       executionID,
		SubmissionToken:   LocalExecutionTokenPrefix + executionID,
		ErrorDetails:      nil,
	}, nil
}
