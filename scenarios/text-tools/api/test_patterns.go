package main

import (
	"fmt"
	"net/http"
	"testing"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	Request        HTTPTestRequest
	ExpectedStatus int
	ExpectedError  string
}

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	patterns []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		patterns: []ErrorTestPattern{},
	}
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(method, path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:        "InvalidJSON",
		Description: "Test handler with malformed JSON input",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   `{"invalid": "json"`,
		},
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "Invalid request body",
	})
	return b
}

// AddMissingRequiredField adds missing required field test pattern
func (b *TestScenarioBuilder) AddMissingRequiredField(method, path string, body interface{}) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:        "MissingRequiredField",
		Description: "Test handler with missing required field",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   body,
		},
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddEmptyBody adds empty body test pattern
func (b *TestScenarioBuilder) AddEmptyBody(method, path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:        "EmptyBody",
		Description: "Test handler with empty request body",
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   nil,
		},
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddCustomPattern adds a custom test pattern
func (b *TestScenarioBuilder) AddCustomPattern(pattern ErrorTestPattern) *TestScenarioBuilder {
	b.patterns = append(b.patterns, pattern)
	return b
}

// Build returns the configured test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// RunErrorTests executes a suite of error condition tests
func RunErrorTests(t *testing.T, handler http.HandlerFunc, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			w, err := makeHTTPRequest(pattern.Request, handler)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			if pattern.ExpectedError != "" {
				assertErrorResponse(t, w, pattern.ExpectedStatus, pattern.ExpectedError)
			}
		})
	}
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	Name    string
	Handler http.HandlerFunc
	Setup   func(t *testing.T) *TestEnvironment
	Cleanup func(env *TestEnvironment)
}

// RunAllTests executes comprehensive tests for a handler
func (suite *HandlerTestSuite) RunAllTests(t *testing.T) {
	t.Run(suite.Name, func(t *testing.T) {
		// Setup
		loggerCleanup := setupTestLogger()
		defer loggerCleanup()

		var env *TestEnvironment
		if suite.Setup != nil {
			env = suite.Setup(t)
			if suite.Cleanup != nil {
				defer suite.Cleanup(env)
			}
		}

		// Run success tests
		t.Run("Success", func(t *testing.T) {
			suite.testSuccess(t, env)
		})

		// Run error tests
		t.Run("Errors", func(t *testing.T) {
			suite.testErrors(t, env)
		})

		// Run edge cases
		t.Run("EdgeCases", func(t *testing.T) {
			suite.testEdgeCases(t, env)
		})
	})
}

func (suite *HandlerTestSuite) testSuccess(t *testing.T, env *TestEnvironment) {
	// Override in specific test implementations
	t.Skip("Success tests not implemented")
}

func (suite *HandlerTestSuite) testErrors(t *testing.T, env *TestEnvironment) {
	// Override in specific test implementations
	t.Skip("Error tests not implemented")
}

func (suite *HandlerTestSuite) testEdgeCases(t *testing.T, env *TestEnvironment) {
	// Override in specific test implementations
	t.Skip("Edge case tests not implemented")
}

// PerformanceTestHelper provides utilities for performance testing
type PerformanceTestHelper struct {
	Name        string
	Handler     http.HandlerFunc
	Request     HTTPTestRequest
	MaxDuration int64 // in milliseconds
}

// Run executes a performance test
func (p *PerformanceTestHelper) Run(t *testing.T) {
	// Create a simple performance benchmark
	// This is a basic implementation; for detailed performance tests, use Go benchmarks

	w, err := makeHTTPRequest(p.Request, p.Handler)
	if err != nil {
		t.Fatalf("Performance test failed to create request: %v", err)
	}

	if w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Errorf("Performance test got unexpected status: %d", w.Code)
	}

	// Could add timing checks here if needed
	t.Logf("Performance test '%s' completed successfully", p.Name)
}

// ValidationHelper provides common validation utilities
type ValidationHelper struct{}

// ValidateRequiredFields checks that all required fields are present in a response
func (v *ValidationHelper) ValidateRequiredFields(t *testing.T, response map[string]interface{}, requiredFields []string) {
	for _, field := range requiredFields {
		if _, exists := response[field]; !exists {
			t.Errorf("Required field '%s' missing from response", field)
		}
	}
}

// ValidateErrorStructure checks that error responses have the correct structure
func (v *ValidationHelper) ValidateErrorStructure(t *testing.T, response map[string]interface{}) {
	requiredErrorFields := []string{"error", "code"}
	for _, field := range requiredErrorFields {
		if _, exists := response[field]; !exists {
			t.Errorf("Error response missing required field: %s", field)
		}
	}
}

// ValidateV2Response checks v2 API response structure
func (v *ValidationHelper) ValidateV2Response(t *testing.T, response map[string]interface{}) {
	requiredV2Fields := []string{"request_id"}
	for _, field := range requiredV2Fields {
		if _, exists := response[field]; !exists {
			t.Errorf("V2 response missing required field: %s", field)
		}
	}
}

// Global validation helper instance
var Validator = &ValidationHelper{}

// Common test data generators

// GenerateTestText generates test text of various sizes
func GenerateTestText(size string) string {
	switch size {
	case "small":
		return "Hello World"
	case "medium":
		return "This is a medium sized text sample.\nIt has multiple lines.\nAnd some variation in content."
	case "large":
		text := ""
		for i := 0; i < 100; i++ {
			text += fmt.Sprintf("Line %d: This is a test line with some content.\n", i)
		}
		return text
	case "empty":
		return ""
	default:
		return "Default test text"
	}
}

// GenerateTestPattern generates test search patterns
func GenerateTestPattern(patternType string) string {
	switch patternType {
	case "simple":
		return "test"
	case "regex":
		return "[a-z]+"
	case "complex":
		return "^[A-Z][a-z]+\\s+[0-9]+"
	default:
		return "default"
	}
}
