package ruleengine

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

	rulespkg "scenario-auditor/rules"
)

// TestCase represents a single test case embedded in a rule file.
type TestCase struct {
	ID                 string `json:"id"`
	Description        string `json:"description"`
	Input              string `json:"input"`
	Language           string `json:"language"`
	ShouldFail         bool   `json:"should_fail"`
	ExpectedViolations int    `json:"expected_violations"`
	ExpectedMessage    string `json:"expected_message"`
	Path               string `json:"path,omitempty"`
	Scenario           string `json:"scenario,omitempty"`
}

// TestResult represents the result of running a test case.
type TestResult struct {
	TestCase         TestCase             `json:"test_case"`
	Passed           bool                 `json:"passed"`
	ActualViolations []rulespkg.Violation `json:"actual_violations"`
	ExecutedAt       time.Time            `json:"executed_at"`
	Error            string               `json:"error,omitempty"`
}

// TestCache stores cached test results.
type TestCache struct {
	RuleFileHash string       `json:"rule_file_hash"`
	TestResults  []TestResult `json:"test_results"`
	TestedAt     time.Time    `json:"tested_at"`
}

// TestHarness handles test execution and caching in static-analysis mode.
type TestHarness struct {
	mu    sync.RWMutex
	cache map[string]*TestCache
}

// NewTestHarness creates a new test harness instance.
func NewTestHarness() *TestHarness {
	return &TestHarness{cache: make(map[string]*TestCache)}
}

// ExtractTestCases extracts test cases from a rule file's content.
func (tr *TestHarness) ExtractTestCases(content string) ([]TestCase, error) {
	var testCases []TestCase

	// Regex to match test case blocks
	testCaseRegex := regexp.MustCompile(`(?s)<test-case\s+([^>]+)>(.*?)</test-case>`)

	matches := testCaseRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) < 3 {
			continue
		}

		attrText := match[1]
		attrMap := parseTestCaseAttributes(attrText)

		testCase := TestCase{ID: attrMap["id"], Path: attrMap["path"], Scenario: attrMap["scenario"]}
		if testCase.ID == "" {
			continue
		}

		if val, ok := attrMap["should-fail"]; ok {
			testCase.ShouldFail = strings.EqualFold(val, "true")
		}

		innerContent := match[2]

		descRegex := regexp.MustCompile(`(?s)<description>(.*?)</description>`)
		if descMatch := descRegex.FindStringSubmatch(innerContent); len(descMatch) > 1 {
			testCase.Description = strings.TrimSpace(descMatch[1])
		}

		inputRegex := regexp.MustCompile(`(?s)<input(?:\s+language="([^"]+)")?\s*>(.*?)</input>`)
		if inputMatch := inputRegex.FindStringSubmatch(innerContent); len(inputMatch) > 2 {
			if inputMatch[1] != "" {
				testCase.Language = inputMatch[1]
			} else {
				testCase.Language = "text"
			}
			testCase.Input = strings.TrimSpace(inputMatch[2])
		}

		violationsRegex := regexp.MustCompile(`<expected-violations>(\d+)</expected-violations>`)
		if violMatch := violationsRegex.FindStringSubmatch(innerContent); len(violMatch) > 1 {
			if v, err := strconv.Atoi(violMatch[1]); err == nil {
				testCase.ExpectedViolations = v
			}
		}

		messageRegex := regexp.MustCompile(`(?s)<expected-message>(.*?)</expected-message>`)
		if msgMatch := messageRegex.FindStringSubmatch(innerContent); len(msgMatch) > 1 {
			testCase.ExpectedMessage = strings.TrimSpace(msgMatch[1])
		}

		testCases = append(testCases, testCase)
	}

	return testCases, nil
}

var testCaseAttrRegex = regexp.MustCompile(`([A-Za-z0-9_-]+)="([^"]*)"`)

func parseTestCaseAttributes(attr string) map[string]string {
	attributes := make(map[string]string)
	matches := testCaseAttrRegex.FindAllStringSubmatch(attr, -1)
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(m[1]))
		attributes[key] = strings.TrimSpace(m[2])
	}
	return attributes
}

// GetFileHash computes SHA-256 hash of a file.
func (tr *TestHarness) GetFileHash(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:]), nil
}

// GetCachedResults returns cached test results if valid.
func (tr *TestHarness) GetCachedResults(ruleID string, fileHash string) (*TestCache, bool) {
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

// SaveCache saves test results to cache.
func (tr *TestHarness) SaveCache(ruleID string, fileHash string, results []TestResult) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.cache[ruleID] = &TestCache{
		RuleFileHash: fileHash,
		TestResults:  results,
		TestedAt:     time.Now(),
	}
}

// ClearCache clears cached results for a specific rule.
func (tr *TestHarness) ClearCache(ruleID string) {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	delete(tr.cache, ruleID)
}

// ClearAllCache clears all cached test results.
func (tr *TestHarness) ClearAllCache() {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.cache = make(map[string]*TestCache)
}

// RunTest executes a single test case using static analysis only.
func (tr *TestHarness) RunTest(testCase TestCase, rule Info) TestResult {
	result := TestResult{
		TestCase:   testCase,
		ExecutedAt: time.Now(),
	}

	pathHint := testCase.Path
	if strings.TrimSpace(pathHint) == "" {
		pathHint = fmt.Sprintf("test_%s.%s", testCase.ID, testCase.Language)
	}

	scenarioName := testCase.Scenario
	violations, err := rule.Check(testCase.Input, pathHint, scenarioName)
	if err != nil {
		result.Error = err.Error()
		result.Passed = false
		return result
	}

	result.ActualViolations = violations
	expected := testCase.ExpectedViolations
	if testCase.ShouldFail {
		result.Passed = expected == len(violations)
	} else {
		result.Passed = len(violations) == expected
	}

	if testCase.ExpectedMessage != "" && len(violations) > 0 {
		matched := false
		for _, v := range violations {
			// Check both Title and Message fields for the expected message
			if strings.Contains(strings.ToLower(v.Title), strings.ToLower(testCase.ExpectedMessage)) ||
				strings.Contains(strings.ToLower(v.Message), strings.ToLower(testCase.ExpectedMessage)) {
				matched = true
				break
			}
		}
		result.Passed = result.Passed && matched
	}

	if testCase.ShouldFail && expected == 0 {
		// Shortcut for boolean assertions
		result.Passed = len(violations) > 0
	}

	return result
}

// RunAllTests runs all test cases for a rule.
func (tr *TestHarness) RunAllTests(ruleID string, rule Info) ([]TestResult, error) {
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

// ValidateCustomInput tests custom input against a rule.
func (tr *TestHarness) ValidateCustomInput(ruleID string, rule Info, input string, language string) (TestResult, error) {
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
