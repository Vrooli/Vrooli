package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
	TestCase         TestCase         `json:"test_case"`
	Passed           bool             `json:"passed"`
	ActualViolations []Violation      `json:"actual_violations"`
	ExecutionOutput  *ExecutionOutput `json:"execution_output,omitempty"`
	ExecutedAt       time.Time        `json:"executed_at"`
	Error            string           `json:"error,omitempty"`
}

// ExecutionOutput is retained for API compatibility, but unused in static mode
type ExecutionOutput struct {
	Stdout        string `json:"stdout,omitempty"`
	Stderr        string `json:"stderr,omitempty"`
	CompileOutput string `json:"compile_output,omitempty"`
	ExitCode      int    `json:"exit_code,omitempty"`
	ExecutionTime string `json:"execution_time,omitempty"`
	Method        string `json:"method,omitempty"`
}

// TestCache stores cached test results
type TestCache struct {
	RuleFileHash string       `json:"rule_file_hash"`
	TestResults  []TestResult `json:"test_results"`
	TestedAt     time.Time    `json:"tested_at"`
}

// TestRunner handles test execution and caching in static-analysis mode
type TestRunner struct {
	mu    sync.RWMutex
	cache map[string]*TestCache
}

// NewTestRunner creates a new test runner instance
func NewTestRunner() *TestRunner {
	return &TestRunner{
		cache: make(map[string]*TestCache),
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
func (tr *TestRunner) GetFileHash(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
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

	if cache.RuleFileHash != fileHash {
		return nil, false
	}

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

// RunTest executes a single test case using static analysis only
func (tr *TestRunner) RunTest(testCase TestCase, rule RuleInfo) TestResult {
	result := TestResult{
		TestCase:   testCase,
		ExecutedAt: time.Now(),
	}

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
		passed := false
		if testCase.ExpectedViolations > 0 {
			passed = actualViolationCount == testCase.ExpectedViolations
		} else {
			passed = actualViolationCount > 0
		}

		if passed && testCase.ExpectedMessage != "" && actualViolationCount > 0 {
			found := false
			for _, v := range violations {
				if strings.Contains(v.Message, testCase.ExpectedMessage) {
					found = true
					break
				}
			}
			passed = found
		}
		return passed
	}

	return actualViolationCount == 0
}

// RunAllTests runs all test cases for a rule
func (tr *TestRunner) RunAllTests(ruleID string, rule RuleInfo) ([]TestResult, error) {
	content, err := os.ReadFile(rule.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rule file: %v", err)
	}

	fileHash, err := tr.GetFileHash(rule.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to compute file hash: %v", err)
	}

	if cached, valid := tr.GetCachedResults(ruleID, fileHash); valid {
		return cached.TestResults, nil
	}

	testCases, err := tr.ExtractTestCases(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to extract test cases: %v", err)
	}

	var results []TestResult
	for _, testCase := range testCases {
		results = append(results, tr.RunTest(testCase, rule))
	}

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
		ShouldFail:  false,
	}

	result := tr.RunTest(testCase, rule)
	return result, nil
}
