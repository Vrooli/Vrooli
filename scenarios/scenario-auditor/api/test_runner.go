package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TestCase represents a single test case embedded in a rule file
type TestCase struct {
	ID                 string `json:"id"`
	Description        string `json:"description"`
	Input              string `json:"input"`
	Language           string `json:"language"`
	ShouldFail         bool   `json:"should_fail"`
	ExpectedViolations int    `json:"expected_violations"`
	ExpectedMessage    string `json:"expected_message"`
}

// TestResult represents the result of running a test case
type TestResult struct {
	TestCase         TestCase     `json:"test_case"`
	Passed           bool         `json:"passed"`
	ActualViolations []Violation  `json:"actual_violations"`
	Judge0Result     *Judge0Response `json:"judge0_result,omitempty"`
	ExecutionOutput  ExecutionOutput `json:"execution_output"`
	ExecutedAt       time.Time    `json:"executed_at"`
	Error            string       `json:"error,omitempty"`
}

// ExecutionOutput captures all execution details
type ExecutionOutput struct {
	Stdout        string `json:"stdout,omitempty"`
	Stderr        string `json:"stderr,omitempty"`
	CompileOutput string `json:"compile_output,omitempty"`
	ExitCode      int    `json:"exit_code,omitempty"`
	ExecutionTime string `json:"execution_time,omitempty"`
	Method        string `json:"method"` // "judge0" or "direct"
}

// Judge0Response represents the response from Judge0
type Judge0Response struct {
	Token       string  `json:"token,omitempty"`
	Status      string  `json:"status"`
	Output      string  `json:"stdout,omitempty"`
	Error       string  `json:"stderr,omitempty"`
	CompileOutput string `json:"compile_output,omitempty"`
	Time        float64 `json:"time,omitempty"`
	Memory      int     `json:"memory,omitempty"`
}

// TestCache stores cached test results
type TestCache struct {
	RuleFileHash string                 `json:"rule_file_hash"`
	TestResults  []TestResult           `json:"test_results"`
	TestedAt     time.Time             `json:"tested_at"`
	Judge0Status map[string]string     `json:"judge0_status"`
}

// TestRunner handles test execution and caching
type TestRunner struct {
	mu         sync.RWMutex
	cache      map[string]*TestCache
	judge0URL  string
}

// NewTestRunner creates a new test runner instance
func NewTestRunner() *TestRunner {
	judge0URL := os.Getenv("JUDGE0_URL")
	if judge0URL == "" {
		judge0URL = "http://localhost:2358" // Default Judge0 port
	}
	
	return &TestRunner{
		cache:     make(map[string]*TestCache),
		judge0URL: judge0URL,
	}
}

// ExtractTestCases extracts test cases from a rule file's content
func (tr *TestRunner) ExtractTestCases(content string) ([]TestCase, error) {
	var testCases []TestCase
	
	// Regex to match test case blocks
	testCaseRegex := regexp.MustCompile(`(?s)<test-case\s+id="([^"]+)"(?:\s+should-fail="([^"]+)")?\s*>(.*?)</test-case>`)
	
	matches := testCaseRegex.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) < 4 {
			continue
		}
		
		testCase := TestCase{
			ID: match[1],
		}
		
		// Parse should-fail attribute
		if match[2] != "" {
			testCase.ShouldFail = match[2] == "true"
		}
		
		// Extract inner content
		innerContent := match[3]
		
		// Extract description
		descRegex := regexp.MustCompile(`(?s)<description>(.*?)</description>`)
		if descMatch := descRegex.FindStringSubmatch(innerContent); len(descMatch) > 1 {
			testCase.Description = strings.TrimSpace(descMatch[1])
		}
		
		// Extract input and language
		inputRegex := regexp.MustCompile(`(?s)<input(?:\s+language="([^"]+)")?\s*>(.*?)</input>`)
		if inputMatch := inputRegex.FindStringSubmatch(innerContent); len(inputMatch) > 2 {
			if inputMatch[1] != "" {
				testCase.Language = inputMatch[1]
			} else {
				testCase.Language = "text" // Default language
			}
			testCase.Input = strings.TrimSpace(inputMatch[2])
		}
		
		// Extract expected violations
		violationsRegex := regexp.MustCompile(`<expected-violations>(\d+)</expected-violations>`)
		if violMatch := violationsRegex.FindStringSubmatch(innerContent); len(violMatch) > 1 {
			if v, err := strconv.Atoi(violMatch[1]); err == nil {
				testCase.ExpectedViolations = v
			}
		}
		
		// Extract expected message
		messageRegex := regexp.MustCompile(`(?s)<expected-message>(.*?)</expected-message>`)
		if msgMatch := messageRegex.FindStringSubmatch(innerContent); len(msgMatch) > 1 {
			testCase.ExpectedMessage = strings.TrimSpace(msgMatch[1])
		}
		
		testCases = append(testCases, testCase)
	}
	
	return testCases, nil
}

// GetFileHash computes SHA-256 hash of a file
func (tr *TestRunner) GetFileHash(filepath string) (string, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return "", err
	}
	
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:]), nil
}

// GetCachedResults returns cached test results if valid
func (tr *TestRunner) GetCachedResults(ruleID string, fileHash string) (*TestCache, bool) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	
	cache, exists := tr.cache[ruleID]
	if !exists {
		return nil, false
	}
	
	// Check if file hash matches
	if cache.RuleFileHash != fileHash {
		return nil, false
	}
	
	// Check if cache is not too old (24 hours)
	if time.Since(cache.TestedAt) > 24*time.Hour {
		return nil, false
	}
	
	return cache, true
}

// SaveCache saves test results to cache
func (tr *TestRunner) SaveCache(ruleID string, fileHash string, results []TestResult) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	
	tr.cache[ruleID] = &TestCache{
		RuleFileHash: fileHash,
		TestResults:  results,
		TestedAt:     time.Now(),
		Judge0Status: make(map[string]string),
	}
}

// ClearCache clears cached results for a specific rule
func (tr *TestRunner) ClearCache(ruleID string) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	
	delete(tr.cache, ruleID)
}

// ClearAllCache clears all cached test results
func (tr *TestRunner) ClearAllCache() {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	
	tr.cache = make(map[string]*TestCache)
}

// RunTest executes a single test case against a rule using Judge0 for robust execution
func (tr *TestRunner) RunTest(testCase TestCase, rule RuleInfo) TestResult {
	result := TestResult{
		TestCase:   testCase,
		ExecutedAt: time.Now(),
	}
	
	// Use Judge0 for code execution if language is supported
	if testCase.Language == "go" || testCase.Language == "golang" {
		judge0Result, err := tr.executeWithJudge0(testCase)
		if err != nil {
			// Fallback to direct execution if Judge0 fails
			fmt.Printf("Judge0 execution failed, falling back to direct execution: %v\n", err)
			return tr.runTestDirect(testCase, rule)
		}
		
		result.Judge0Result = judge0Result
		
		// Populate execution output from Judge0 result
		result.ExecutionOutput = ExecutionOutput{
			Stdout:        judge0Result.Output,
			Stderr:        judge0Result.Error,
			CompileOutput: judge0Result.CompileOutput,
			Method:        "judge0",
		}
		
		// If Judge0 execution was successful, run rule check on the code
		if judge0Result.Status == "3" { // Accepted (successful execution)
			violations, err := rule.Check(testCase.Input, fmt.Sprintf("test_%s.%s", testCase.ID, testCase.Language))
			if err != nil {
				result.Error = err.Error()
				result.Passed = false
				return result
			}
			result.ActualViolations = violations
			result.ExecutionOutput.ExitCode = 0
		} else {
			// Code failed to compile/execute - treat as test failure
			result.Error = fmt.Sprintf("Code execution failed: %s", judge0Result.Error)
			result.ActualViolations = []Violation{}
			result.ExecutionOutput.ExitCode = 1
		}
		
		// Determine if test passed based on violations and exit code
		result.Passed = tr.evaluateTestResult(testCase, result.ActualViolations, result.ExecutionOutput.ExitCode)
		return result
	} else {
		// For non-Go languages or when Judge0 is not available, use direct execution
		return tr.runTestDirect(testCase, rule)
	}
}

// createExecutableProgram wraps a test case in a complete program for runtime validation
func (tr *TestRunner) createExecutableProgram(testCase TestCase, rule RuleInfo) (string, error) {
	if testCase.Language != "go" {
		return "", fmt.Errorf("only Go programs supported for runtime validation")
	}
	
	// Determine rule-specific test wrapper based on rule ID
	switch rule.ID {
	case "content_type_headers":
		return tr.createHTTPTestProgram(testCase)
	case "structured_logging":
		return tr.createLoggingTestProgram(testCase)
	case "health_check":
		return tr.createHealthCheckTestProgram(testCase)
	default:
		// For other rules, just try to create a basic program
		return tr.createBasicTestProgram(testCase)
	}
}

// createHTTPTestProgram creates a complete HTTP test program
func (tr *TestRunner) createHTTPTestProgram(testCase TestCase) (string, error) {
	// Extract function name from the input
	funcName := "handleAPI" // default
	lines := strings.Split(testCase.Input, "\n")
	for _, line := range lines {
		if strings.Contains(line, "func ") && strings.Contains(line, "http.ResponseWriter") {
			// Extract function name
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "func" && i+1 < len(parts) {
					funcName = strings.Split(parts[i+1], "(")[0]
					break
				}
			}
			break
		}
	}
	
	program := fmt.Sprintf(`package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"fmt"
	"strings"
)

%s

func main() {
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	
	%s(w, req)
	
	// Check Content-Type header
	contentType := w.Header().Get("Content-Type")
	body := w.Body.String()
	
	fmt.Printf("Response Headers: %%v\n", w.Header())
	fmt.Printf("Response Body: %%s\n", body)
	
	// Detect if JSON is being written
	if strings.Contains(body, "{") || strings.Contains(testCode, "json.") {
		if contentType == "" {
			fmt.Println("VIOLATION: Missing Content-Type header for JSON response")
		} else if !strings.Contains(contentType, "application/json") {
			fmt.Printf("VIOLATION: Incorrect Content-Type header: %%s (expected application/json)\n", contentType)
		} else {
			fmt.Printf("OK: Correct Content-Type header: %%s\n", contentType)
		}
	} else {
		if contentType == "" {
			fmt.Println("VIOLATION: Missing Content-Type header")
		} else {
			fmt.Printf("OK: Content-Type header present: %%s\n", contentType)
		}
	}
}

const testCode = ` + "`" + `%s` + "`" + `
`, testCase.Input, funcName, testCase.Input)
	
	return program, nil
}

// createLoggingTestProgram creates a program to test logging patterns
func (tr *TestRunner) createLoggingTestProgram(testCase TestCase) (string, error) {
	program := fmt.Sprintf(`package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"io"
)

%s

func main() {
	// Capture stdout to check for unstructured logging
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	
	// Capture stderr for log output
	oldStderr := os.Stderr
	r2, w2, _ := os.Pipe()
	os.Stderr = w2
	
	// Try to call any functions in the test code
	// This is a simple attempt - in practice, we'd need more sophisticated analysis
	
	// Restore stdout/stderr
	w.Close()
	w2.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	
	// Read captured output
	output, _ := io.ReadAll(r)
	errOutput, _ := io.ReadAll(r2)
	
	fmt.Printf("Stdout captured: %%s\n", string(output))
	fmt.Printf("Stderr captured: %%s\n", string(errOutput))
	
	// Check test code for unstructured logging patterns
	testCode := ` + "`" + `%s` + "`" + `
	if strings.Contains(testCode, "fmt.Print") {
		fmt.Println("VIOLATION: Using fmt.Print* for logging")
	} else if strings.Contains(testCode, "log.Print") {
		fmt.Println("VIOLATION: Using basic log.Print* without structure")
	} else if strings.Contains(testCode, "logger.Info") || strings.Contains(testCode, "slog.") {
		fmt.Println("OK: Using structured logging")
	} else {
		fmt.Println("OK: No obvious logging violations")
	}
}
`, testCase.Input, testCase.Input)
	
	return program, nil
}

// createHealthCheckTestProgram creates a program to test health check endpoints
func (tr *TestRunner) createHealthCheckTestProgram(testCase TestCase) (string, error) {
	program := fmt.Sprintf(`package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
)

%s

func main() {
	// Try to find health endpoint registration
	testCode := ` + "`" + `%s` + "`" + `
	
	if strings.Contains(testCode, "/health") && strings.Contains(testCode, "HandleFunc") {
		fmt.Println("OK: Health endpoint found")
	} else if strings.Contains(testCode, "/api/health") {
		fmt.Println("OK: Health endpoint found at /api/health")
	} else {
		fmt.Println("VIOLATION: No health check endpoint found")
	}
	
	// If we can identify a main function or router setup, we could test it more thoroughly
	fmt.Printf("Test code analysis: %%s\n", testCode)
}
`, testCase.Input, testCase.Input)
	
	return program, nil
}

// createBasicTestProgram creates a basic test program for other rules
func (tr *TestRunner) createBasicTestProgram(testCase TestCase) (string, error) {
	program := fmt.Sprintf(`package main

import (
	"fmt"
)

%s

func main() {
	fmt.Println("Executing test code...")
	fmt.Printf("Test input: %%s\n", ` + "`" + `%s` + "`" + `)
	
	// Basic execution - the static analysis will handle most validation
	fmt.Println("OK: Code compiled and executed successfully")
}
`, testCase.Input, testCase.Input)
	
	return program, nil
}

// executeProgram compiles and runs a Go program, returning its output and exit code
func (tr *TestRunner) executeProgram(program, language string) (string, int, error) {
	if language != "go" {
		return "", 1, fmt.Errorf("only Go execution supported")
	}
	
	// For now, use Judge0 if available, otherwise return program for inspection
	if tr.hasJudge0() {
		// Create a temporary test case for the generated program
		tempTestCase := TestCase{
			ID:       "runtime-execution",
			Input:    program,
			Language: language,
		}
		
		result, err := tr.executeWithJudge0(tempTestCase)
		if err != nil {
			return "", 1, err
		}
		
		if result.Status == "3" { // Accepted
			return result.Output, 0, nil
		} else {
			return "", 1, fmt.Errorf("execution failed: %s", result.Error)
		}
	}
	
	// Fallback: execute locally using go run
	return tr.executeLocally(program)
}

// executeLocally executes a Go program locally using go run
func (tr *TestRunner) executeLocally(program string) (string, int, error) {
	// Create a temporary directory for the Go file
	tempDir, err := os.MkdirTemp("", "rule_test_*")
	if err != nil {
		return "", 1, fmt.Errorf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up
	
	// Create temporary Go file
	tempFile := filepath.Join(tempDir, "main.go")
	err = os.WriteFile(tempFile, []byte(program), 0644)
	if err != nil {
		return "", 1, fmt.Errorf("failed to write temp file: %v", err)
	}
	
	// Execute with go run
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, "go", "run", tempFile)
	
	// Capture both stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	// Execute the command and get exit code
	err = cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1 // Default to 1 for other errors
		}
	}
	
	// Combine output
	output := stdout.String()
	if stderr.String() != "" {
		if output != "" {
			output += "\n--- STDERR ---\n"
		}
		output += stderr.String()
	}
	
	if err != nil {
		if output == "" {
			output = fmt.Sprintf("Execution failed: %v", err)
		} else {
			output += fmt.Sprintf("\n--- EXECUTION ERROR ---\n%v", err)
		}
	}
	
	return output, exitCode, nil
}

// hasJudge0 checks if Judge0 is available (placeholder implementation)
func (tr *TestRunner) hasJudge0() bool {
	// For now, assume Judge0 is not available and use fallback
	return false
}

// runTestDirect executes test case directly without Judge0 (fallback method)
func (tr *TestRunner) runTestDirect(testCase TestCase, rule RuleInfo) TestResult {
	result := TestResult{
		TestCase:   testCase,
		ExecutedAt: time.Now(),
		ExecutionOutput: ExecutionOutput{
			Method: "direct",
		},
	}
	
	// Try to create and execute a complete program for runtime validation
	completeProgram, err := tr.createExecutableProgram(testCase, rule)
	if err != nil {
		result.ExecutionOutput.Stderr = fmt.Sprintf("Failed to create executable program: %v", err)
		result.ExecutionOutput.Method = "direct"
		result.ExecutionOutput.ExitCode = 1
	} else {
		// Execute the complete program
		output, exitCode, err := tr.executeProgram(completeProgram, testCase.Language)
		result.ExecutionOutput.Stdout = output
		result.ExecutionOutput.ExitCode = exitCode
		result.ExecutionOutput.Method = "local"
		
		if err != nil {
			if result.ExecutionOutput.Stderr != "" {
				result.ExecutionOutput.Stderr += "\n"
			}
			result.ExecutionOutput.Stderr += fmt.Sprintf("Execution failed: %v", err)
		}
	}
	
	// Run the rule's Check function for static analysis
	staticViolations, err := rule.Check(testCase.Input, fmt.Sprintf("test_%s.%s", testCase.ID, testCase.Language))
	if err != nil {
		result.Error = err.Error()
		result.Passed = false
		return result
	}
	
	// Parse runtime violations from execution output
	runtimeViolations := tr.parseRuntimeViolations(result.ExecutionOutput.Stdout, testCase, rule)
	
	// Combine static and runtime violations (prefer runtime if available)
	if len(runtimeViolations) > 0 {
		result.ActualViolations = runtimeViolations
	} else {
		result.ActualViolations = staticViolations
	}
	
	result.Passed = tr.evaluateTestResult(testCase, result.ActualViolations, result.ExecutionOutput.ExitCode)
	
	return result
}

// parseRuntimeViolations extracts violations from runtime execution output
func (tr *TestRunner) parseRuntimeViolations(executionOutput string, testCase TestCase, rule RuleInfo) []Violation {
	var violations []Violation
	
	if executionOutput == "" {
		return violations
	}
	
	lines := strings.Split(executionOutput, "\n")
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		
		// Look for VIOLATION: messages in the output
		if strings.HasPrefix(trimmedLine, "VIOLATION:") {
			violationMessage := strings.TrimPrefix(trimmedLine, "VIOLATION:")
			violationMessage = strings.TrimSpace(violationMessage)
			
			// Create a violation based on the runtime detection
			violation := Violation{
				RuleID:   rule.ID,
				Severity: "medium", // Default severity, could be parsed from output
				Message:  violationMessage,
				File:     fmt.Sprintf("test_%s.%s", testCase.ID, testCase.Language),
				Line:     i + 1,
			}
			violations = append(violations, violation)
		}
	}
	
	return violations
}

// getRuntimeRecommendation provides rule-specific recommendations for runtime violations
func (tr *TestRunner) getRuntimeRecommendation(ruleID, violationMessage string) string {
	switch ruleID {
	case "content_type_headers":
		if strings.Contains(violationMessage, "Missing Content-Type") {
			return "Add w.Header().Set(\"Content-Type\", \"application/json\") before writing JSON response"
		} else if strings.Contains(violationMessage, "Incorrect Content-Type") {
			return "Set proper Content-Type header: w.Header().Set(\"Content-Type\", \"application/json\")"
		}
		return "Ensure proper Content-Type headers are set for all responses"
	case "structured_logging":
		return "Use structured logging libraries (zap, slog) instead of fmt.Print or log.Print"
	case "health_check":
		return "Add a /health endpoint that returns service status information"
	default:
		return "Follow the coding standards for this rule category"
	}
}

// evaluateTestResult determines if a test passed based on expected vs actual violations and exit code
func (tr *TestRunner) evaluateTestResult(testCase TestCase, violations []Violation, exitCode int) bool {
	actualViolationCount := len(violations)
	
	if testCase.ShouldFail {
		// Test expects violations
		passed := false
		if testCase.ExpectedViolations > 0 {
			passed = actualViolationCount == testCase.ExpectedViolations
		} else {
			passed = actualViolationCount > 0
		}
		
		// Check expected message if specified
		if passed && testCase.ExpectedMessage != "" && actualViolationCount > 0 {
			foundExpectedMessage := false
			for _, v := range violations {
				if strings.Contains(v.Message, testCase.ExpectedMessage) {
					foundExpectedMessage = true
					break
				}
			}
			passed = foundExpectedMessage
		}
		return passed
	} else {
		// Test expects no violations AND successful execution
		// For a test to truly pass, it must:
		// 1. Have no violations
		// 2. Execute successfully (exit code 0)
		return actualViolationCount == 0 && exitCode == 0
	}
}

// RunAllTests runs all test cases for a rule
func (tr *TestRunner) RunAllTests(ruleID string, rule RuleInfo) ([]TestResult, error) {
	// Get rule file content
	content, err := os.ReadFile(rule.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rule file: %v", err)
	}
	
	// Get file hash
	fileHash, err := tr.GetFileHash(rule.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to compute file hash: %v", err)
	}
	
	// Check cache
	if cached, valid := tr.GetCachedResults(ruleID, fileHash); valid {
		return cached.TestResults, nil
	}
	
	// Extract test cases
	testCases, err := tr.ExtractTestCases(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to extract test cases: %v", err)
	}
	
	// Run each test case
	var results []TestResult
	for _, testCase := range testCases {
		result := tr.RunTest(testCase, rule)
		results = append(results, result)
	}
	
	// Save to cache
	tr.SaveCache(ruleID, fileHash, results)
	
	return results, nil
}

// ValidateCustomInput tests custom input against a rule
func (tr *TestRunner) ValidateCustomInput(ruleID string, rule RuleInfo, input string, language string) (TestResult, error) {
	testCase := TestCase{
		ID:          "custom",
		Description: "Custom input validation",
		Input:       input,
		Language:    language,
		ShouldFail:  false, // We're just checking what happens
	}
	
	result := tr.RunTest(testCase, rule)
	return result, nil
}

// Judge0 Integration Methods

// Judge0SubmissionRequest represents a submission request to Judge0
type Judge0SubmissionRequest struct {
	SourceCode           string `json:"source_code"`
	LanguageID          int    `json:"language_id"`
	Stdin               string `json:"stdin,omitempty"`
	ExpectedOutput      string `json:"expected_output,omitempty"`
	CPUTimeLimit        string `json:"cpu_time_limit,omitempty"`
	CPUExtraTime        string `json:"cpu_extra_time,omitempty"`
	WallTimeLimit       string `json:"wall_time_limit,omitempty"`
	MemoryLimit         int    `json:"memory_limit,omitempty"`
	StackLimit          int    `json:"stack_limit,omitempty"`
	MaxProcessesAndThreads int `json:"max_processes_and_or_threads,omitempty"`
	EnablePerProcessAndThreadTimeLimit bool `json:"enable_per_process_and_thread_time_limit,omitempty"`
	EnablePerProcessAndThreadMemoryLimit bool `json:"enable_per_process_and_thread_memory_limit,omitempty"`
	MaxFileSize         int    `json:"max_file_size,omitempty"`
}

// Judge0SubmissionResponse represents a submission response from Judge0
type Judge0SubmissionResponse struct {
	Token string `json:"token"`
}

// executeWithJudge0 executes a test case using Judge0
func (tr *TestRunner) executeWithJudge0(testCase TestCase) (*Judge0Response, error) {
	// Get language ID for Judge0
	languageID := tr.getJudge0LanguageID(testCase.Language)
	if languageID == 0 {
		return nil, fmt.Errorf("unsupported language for Judge0: %s", testCase.Language)
	}
	
	// Prepare test code - wrap the input in a simple main function for Go
	sourceCode := tr.prepareTestCode(testCase.Input, testCase.Language)
	
	// Create submission request
	submission := Judge0SubmissionRequest{
		SourceCode:     sourceCode,
		LanguageID:     languageID,
		CPUTimeLimit:   "2",   // 2 seconds
		WallTimeLimit:  "5",   // 5 seconds
		MemoryLimit:    32768, // 32MB
		MaxFileSize:    1024,  // 1KB output limit
	}
	
	// Submit to Judge0
	token, err := tr.submitToJudge0(submission)
	if err != nil {
		return nil, fmt.Errorf("failed to submit to Judge0: %w", err)
	}
	
	// Wait for completion and get result
	result, err := tr.getJudge0Result(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get Judge0 result: %w", err)
	}
	
	return result, nil
}

// getJudge0LanguageID maps our language names to Judge0 language IDs
func (tr *TestRunner) getJudge0LanguageID(language string) int {
	languageMap := map[string]int{
		"go":     60, // Go (1.13.5)
		"golang": 60,
		"python": 71, // Python (3.8.1)
		"javascript": 63, // JavaScript (Node.js 12.14.0)
		"java":   62, // Java (OpenJDK 13.0.1)
		"c":      50, // C (GCC 9.2.0)
		"cpp":    54, // C++ (GCC 9.2.0)
		"rust":   73, // Rust (1.40.0)
	}
	
	return languageMap[strings.ToLower(language)]
}

// prepareTestCode wraps the input code in appropriate boilerplate for execution
func (tr *TestRunner) prepareTestCode(input string, language string) string {
	switch strings.ToLower(language) {
	case "go", "golang":
		// For Go, we need to wrap in a package and imports if not already present
		if !strings.Contains(input, "package ") {
			// Simple wrapper for Go code snippets
			return fmt.Sprintf(`package main

import (
	"net/http"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"os"
)

%s

func main() {
	// Test execution - this is mainly for syntax validation
	fmt.Println("Code syntax validation successful")
}`, input)
		}
		return input
	case "python":
		return input // Python can run as-is
	case "javascript":
		return input // JavaScript can run as-is
	default:
		return input // Return as-is for other languages
	}
}

// submitToJudge0 submits code to Judge0 and returns the submission token
func (tr *TestRunner) submitToJudge0(submission Judge0SubmissionRequest) (string, error) {
	jsonData, err := json.Marshal(submission)
	if err != nil {
		return "", fmt.Errorf("failed to marshal submission: %w", err)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "POST", tr.judge0URL+"/submissions?base64_encoded=false&wait=false", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-RapidAPI-Host", "judge0-ce.p.rapidapi.com")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to submit request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Judge0 submission failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var response Judge0SubmissionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode submission response: %w", err)
	}
	
	return response.Token, nil
}

// getJudge0Result retrieves the result of a Judge0 submission by token
func (tr *TestRunner) getJudge0Result(token string) (*Judge0Response, error) {
	maxRetries := 10
	retryDelay := 500 * time.Millisecond
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		
		req, err := http.NewRequestWithContext(ctx, "GET", tr.judge0URL+"/submissions/"+token+"?base64_encoded=false", nil)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to get result: %w", err)
		}
		
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			cancel()
			return nil, fmt.Errorf("Judge0 result request failed with status %d", resp.StatusCode)
		}
		
		var result Judge0Response
		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		cancel()
		
		if err != nil {
			return nil, fmt.Errorf("failed to decode result: %w", err)
		}
		
		// Check if processing is complete
		if result.Status != "1" && result.Status != "2" { // Not "In Queue" or "Processing"
			return &result, nil
		}
		
		// Wait before retrying
		time.Sleep(retryDelay)
		retryDelay = time.Duration(float64(retryDelay) * 1.5) // Exponential backoff
	}
	
	return nil, fmt.Errorf("Judge0 execution timed out after %d attempts", maxRetries)
}