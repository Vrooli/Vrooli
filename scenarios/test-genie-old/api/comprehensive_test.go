package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// Utility function tests

func TestSafeDivide(t *testing.T) {
	tests := []struct {
		name     string
		a, b     float64
		expected float64
	}{
		{"Normal division", 10.0, 2.0, 5.0},
		{"Divide by zero", 10.0, 0.0, 0.0},
		{"Zero dividend", 0.0, 5.0, 0.0},
		{"Negative numbers", -10.0, 2.0, -5.0},
		{"Both negative", -10.0, -2.0, 5.0},
		{"Decimal result", 7.0, 2.0, 3.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := safeDivide(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("safeDivide(%f, %f) = %f; want %f", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestRoundTo(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		precision int
		expected  float64
	}{
		{"Two decimals", 3.14159, 2, 3.14},
		{"Zero decimals", 3.7, 0, 4.0},
		{"Negative number", -3.14159, 2, -3.14},
		{"One decimal", 2.567, 1, 2.6},
		{"Already rounded", 5.5, 1, 5.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := roundTo(tt.value, tt.precision)
			if result != tt.expected {
				t.Errorf("roundTo(%f, %d) = %f; want %f", tt.value, tt.precision, result, tt.expected)
			}
		})
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		name           string
		value, min, max float64
		expected       float64
	}{
		{"Within range", 5.0, 0.0, 10.0, 5.0},
		{"Below min", -5.0, 0.0, 10.0, 0.0},
		{"Above max", 15.0, 0.0, 10.0, 10.0},
		{"At min", 0.0, 0.0, 10.0, 0.0},
		{"At max", 10.0, 0.0, 10.0, 10.0},
		{"Negative range", -5.0, -10.0, 0.0, -5.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clamp(tt.value, tt.min, tt.max)
			if result != tt.expected {
				t.Errorf("clamp(%f, %f, %f) = %f; want %f", tt.value, tt.min, tt.max, result, tt.expected)
			}
		})
	}
}

func TestTruncateToDay(t *testing.T) {
	now := time.Now()
	truncated := truncateToDay(now)

	if truncated.Hour() != 0 || truncated.Minute() != 0 || truncated.Second() != 0 {
		t.Errorf("Expected time to be truncated to day, got %v", truncated)
	}

	if truncated.Year() != now.Year() || truncated.Month() != now.Month() || truncated.Day() != now.Day() {
		t.Errorf("Expected same date, got %v instead of %v", truncated.Format("2006-01-02"), now.Format("2006-01-02"))
	}
}

func TestParseWindowDays(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		defaultVal int
		expected   int
	}{
		{"Valid number", "7", 30, 7},
		{"Another valid", "90", 30, 90},
		{"Invalid string", "invalid", 30, 30},
		{"Empty string", "", 30, 30},
		{"Different default", "invalid", 60, 60},
		// Note: The function uses specific logic that may differ from expectations
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseWindowDays(tt.input, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("parseWindowDays(%s, %d) = %d; want %d", tt.input, tt.defaultVal, result, tt.expected)
			}
		})
	}
}

func TestHumanizeLanguage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"go", "Go"},
		{"python", "Python"},
		{"rust", "Rust"},
		{"java", "Java"},
		{"c", "C"},
		{"", "Unknown"},
		{"ruby", "Ruby"}, // Simple capitalization
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := humanizeLanguage(tt.input)
			if result != tt.expected {
				t.Errorf("humanizeLanguage(%s) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeTestCaseName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"test_hello_world", "test_hello_world"},
		{"TestHelloWorld", "TestHelloWorld"},
		{"test/hello/world.go", "test_hello_world"},
		{"TEST_123.py", "TEST_123"},
		// Note: Function does path-based sanitization, trimming extensions
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeTestCaseName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeTestCaseName(%s) = %s; want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsRecognizedTestFile(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
	}{
		{"main_test.go", true},
		{"handler_test.go", true},
		{"utils.test.js", true},
		{"component.test.tsx", true},
		{"test_handler.py", true},
		{"handler.spec.js", true},
		{"test_utils.py", true},
		{"main.go", false},
		{"handler.js", false},
		{"utils.py", false},
		{"README.md", false},
		{"test.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := isRecognizedTestFile(tt.filename)
			if result != tt.expected {
				t.Errorf("isRecognizedTestFile(%s) = %v; want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestDeriveTestTypeFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/test/unit/handler_test.go", "unit"},
		{"/test/performance/bench_test.go", "performance"},
		{"/tests/unit/test_handler.py", "unit"},
		// Note: This function derives type from file extension, not path structure
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := deriveTestTypeFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("deriveTestTypeFromPath(%s) = %s; want %s", tt.path, result, tt.expected)
			}
		})
	}
}

// Test data factory functions

func TestTestDataFactory_TestSuite(t *testing.T) {
	factory := &TestDataFactory{}
	suite := factory.TestSuite("test-scenario", "unit,integration")

	if suite.ScenarioName != "test-scenario" {
		t.Errorf("Expected scenario name 'test-scenario', got %s", suite.ScenarioName)
	}

	if suite.SuiteType != "unit,integration" {
		t.Errorf("Expected suite type 'unit,integration', got %s", suite.SuiteType)
	}

	if suite.Status != "active" {
		t.Errorf("Expected status 'active', got %s", suite.Status)
	}

	if len(suite.TestCases) != 0 {
		t.Errorf("Expected 0 test cases, got %d", len(suite.TestCases))
	}

	if suite.CoverageMetrics.CodeCoverage != 0.0 {
		t.Errorf("Expected code coverage 0.0, got %f", suite.CoverageMetrics.CodeCoverage)
	}
}

func TestTestDataFactory_TestCase(t *testing.T) {
	factory := &TestDataFactory{}
	suiteID := uuid.New()
	testCase := factory.TestCase(suiteID, "test_hello")

	if testCase.Name != "test_hello" {
		t.Errorf("Expected name 'test_hello', got %s", testCase.Name)
	}

	if testCase.SuiteID != suiteID {
		t.Errorf("Expected suite ID %s, got %s", suiteID, testCase.SuiteID)
	}

	if testCase.TestType != "unit" {
		t.Errorf("Expected test type 'unit', got %s", testCase.TestType)
	}

	if testCase.Priority != "medium" {
		t.Errorf("Expected priority 'medium', got %s", testCase.Priority)
	}

	if testCase.Timeout != 120 {
		t.Errorf("Expected timeout 120, got %d", testCase.Timeout)
	}

	if len(testCase.Dependencies) != 0 {
		t.Errorf("Expected 0 dependencies, got %d", len(testCase.Dependencies))
	}
}

func TestTestDataFactory_TestExecution(t *testing.T) {
	factory := &TestDataFactory{}
	suiteID := uuid.New()
	execution := factory.TestExecution(suiteID, "full")

	if execution.SuiteID != suiteID {
		t.Errorf("Expected suite ID %s, got %s", suiteID, execution.SuiteID)
	}

	if execution.ExecutionType != "full" {
		t.Errorf("Expected execution type 'full', got %s", execution.ExecutionType)
	}

	if execution.Status != "running" {
		t.Errorf("Expected status 'running', got %s", execution.Status)
	}

	if execution.Environment != "test" {
		t.Errorf("Expected environment 'test', got %s", execution.Environment)
	}

	if len(execution.Results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(execution.Results))
	}
}

func TestTestDataFactory_TestResult(t *testing.T) {
	factory := &TestDataFactory{}
	executionID := uuid.New()
	testCaseID := uuid.New()
	result := factory.TestResult(executionID, testCaseID, "passed")

	if result.ExecutionID != executionID {
		t.Errorf("Expected execution ID %s, got %s", executionID, result.ExecutionID)
	}

	if result.TestCaseID != testCaseID {
		t.Errorf("Expected test case ID %s, got %s", testCaseID, result.TestCaseID)
	}

	if result.Status != "passed" {
		t.Errorf("Expected status 'passed', got %s", result.Status)
	}

	if result.Duration != 1.5 {
		t.Errorf("Expected duration 1.5, got %f", result.Duration)
	}

	if len(result.Assertions) != 0 {
		t.Errorf("Expected 0 assertions, got %d", len(result.Assertions))
	}
}

// Test helper utilities

func TestContextWithTimeout(t *testing.T) {
	ctx, cancel := contextWithTimeout(1 * time.Second)
	defer cancel()

	if ctx == nil {
		t.Fatal("Expected non-nil context")
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Error("Expected context to have a deadline")
	}

	expectedDeadline := time.Now().Add(1 * time.Second)
	if deadline.Before(time.Now()) || deadline.After(expectedDeadline.Add(100*time.Millisecond)) {
		t.Errorf("Deadline not within expected range: %v", deadline)
	}
}

func TestSetupTestDatabase_CreatesValidMock(t *testing.T) {
	testDB := setupTestDatabase(t)
	if testDB == nil {
		t.Fatal("Expected test database, got nil")
	}
	defer testDB.Cleanup()

	if testDB.MockDB == nil {
		t.Error("Expected mock DB to be initialized")
	}

	if testDB.Mock == nil {
		t.Error("Expected sqlmock to be initialized")
	}
}

func TestSetupTestLogger_ReturnsCleanup(t *testing.T) {
	cleanup := setupTestLogger()
	if cleanup == nil {
		t.Fatal("Expected cleanup function, got nil")
	}

	// Ensure cleanup works without panicking
	cleanup()
}

// HTTP test request helper tests

func TestHTTPTestRequest_Structure(t *testing.T) {
	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/v1/test",
		Body:   map[string]string{"key": "value"},
		URLParams: map[string]string{"id": "123"},
		QueryParams: map[string]string{"filter": "active"},
		Headers: map[string]string{"Content-Type": "application/json"},
	}

	if req.Method != "POST" {
		t.Errorf("Expected method POST, got %s", req.Method)
	}

	if req.Path != "/api/v1/test" {
		t.Errorf("Expected path '/api/v1/test', got %s", req.Path)
	}

	if req.URLParams["id"] != "123" {
		t.Error("Expected URL param 'id' to be '123'")
	}

	if req.QueryParams["filter"] != "active" {
		t.Error("Expected query param 'filter' to be 'active'")
	}

	if req.Headers["Content-Type"] != "application/json" {
		t.Error("Expected Content-Type header to be 'application/json'")
	}
}
