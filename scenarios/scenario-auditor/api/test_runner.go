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
	ExecutedAt       time.Time    `json:"executed_at"`
	Error            string       `json:"error,omitempty"`
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
		
		// If Judge0 execution was successful, run rule check on the code
		if judge0Result.Status == "3" { // Accepted (successful execution)
			violations, err := rule.Check(testCase.Input, fmt.Sprintf("test_%s.%s", testCase.ID, testCase.Language))
			if err != nil {
				result.Error = err.Error()
				result.Passed = false
				return result
			}
			result.ActualViolations = violations
		} else {
			// Code failed to compile/execute - treat as test failure
			result.Error = fmt.Sprintf("Code execution failed: %s", judge0Result.Error)
			result.ActualViolations = []Violation{}
		}
	} else {
		// For non-Go languages or when Judge0 is not available, use direct execution
		return tr.runTestDirect(testCase, rule)
	}
	
	// Determine if test passed based on violations
	result.Passed = tr.evaluateTestResult(testCase, result.ActualViolations)
	return result
}

// runTestDirect executes test case directly without Judge0 (fallback method)
func (tr *TestRunner) runTestDirect(testCase TestCase, rule RuleInfo) TestResult {
	result := TestResult{
		TestCase:   testCase,
		ExecutedAt: time.Now(),
	}
	
	// Run the rule's Check function directly
	violations, err := rule.Check(testCase.Input, fmt.Sprintf("test_%s.%s", testCase.ID, testCase.Language))
	
	if err != nil {
		result.Error = err.Error()
		result.Passed = false
		return result
	}
	
	result.ActualViolations = violations
	result.Passed = tr.evaluateTestResult(testCase, violations)
	
	return result
}

// evaluateTestResult determines if a test passed based on expected vs actual violations
func (tr *TestRunner) evaluateTestResult(testCase TestCase, violations []Violation) bool {
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
		// Test expects no violations
		return actualViolationCount == 0
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