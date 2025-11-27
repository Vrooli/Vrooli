//go:build testing
// +build testing

package testutil

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	URLPath        string
	Method         string
	Body           any
	URLVars        map[string]string
}

// TestScenarioBuilder builds systematic test scenarios
type TestScenarioBuilder struct {
	patterns []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		patterns: make([]ErrorTestPattern, 0),
	}
}

// AddInvalidUUID adds test for invalid UUID format
func (b *TestScenarioBuilder) AddInvalidUUID(urlPath, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		URLPath:        urlPath,
		Method:         method,
		URLVars:        map[string]string{"id": "invalid-uuid"},
	})
	return b
}

// AddNonExistentResource adds test for non-existent resource
func (b *TestScenarioBuilder) AddNonExistentResource(urlPath, method, resourceType string) *TestScenarioBuilder {
	nonExistentID := uuid.New()
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           fmt.Sprintf("NonExistent%s", resourceType),
		Description:    fmt.Sprintf("Test handler with non-existent %s ID", resourceType),
		ExpectedStatus: http.StatusNotFound,
		URLPath:        urlPath,
		Method:         method,
		URLVars:        map[string]string{"id": nonExistentID.String()},
	})
	return b
}

// AddInvalidJSON adds test for malformed JSON
func (b *TestScenarioBuilder) AddInvalidJSON(urlPath, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON body",
		ExpectedStatus: http.StatusBadRequest,
		URLPath:        urlPath,
		Method:         method,
		Body:           "invalid-json{",
	})
	return b
}

// AddMissingRequiredField adds test for missing required field
func (b *TestScenarioBuilder) AddMissingRequiredField(urlPath, method, fieldName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           fmt.Sprintf("Missing%s", fieldName),
		Description:    fmt.Sprintf("Test handler with missing required field: %s", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		URLPath:        urlPath,
		Method:         method,
		Body:           map[string]any{},
	})
	return b
}

// AddEmptyBody adds test for empty request body
func (b *TestScenarioBuilder) AddEmptyBody(urlPath, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		URLPath:        urlPath,
		Method:         method,
		Body:           nil,
	})
	return b
}

// AddCustomPattern adds a custom test pattern
func (b *TestScenarioBuilder) AddCustomPattern(pattern ErrorTestPattern) *TestScenarioBuilder {
	b.patterns = append(b.patterns, pattern)
	return b
}

// Build returns the built test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	BaseURL     string
	t           *testing.T
}

// NewHandlerTestSuite creates a new handler test suite
func NewHandlerTestSuite(t *testing.T, handlerName, baseURL string) *HandlerTestSuite {
	return &HandlerTestSuite{
		HandlerName: handlerName,
		BaseURL:     baseURL,
		t:           t,
	}
}

// RunErrorTests executes a suite of error condition tests
func (suite *HandlerTestSuite) RunErrorTests(patterns []ErrorTestPattern, executeFunc func(pattern ErrorTestPattern) *http.Response) {
	for _, pattern := range patterns {
		testName := fmt.Sprintf("%s_%s", suite.HandlerName, pattern.Name)
		suite.t.Run(testName, func(t *testing.T) {
			resp := executeFunc(pattern)
			if resp == nil {
				t.Fatal("Response is nil")
			}

			if resp.StatusCode != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d", pattern.ExpectedStatus, resp.StatusCode)
			}
		})
	}
}

// Common error patterns

// ProjectErrorPatterns returns common error patterns for project endpoints
func ProjectErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidUUID("/api/v1/projects/{id}", "GET").
		AddNonExistentResource("/api/v1/projects/{id}", "GET", "Project").
		AddInvalidJSON("/api/v1/projects", "POST").
		AddMissingRequiredField("/api/v1/projects", "POST", "Name").
		Build()
}

// WorkflowErrorPatterns returns common error patterns for workflow endpoints
func WorkflowErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidUUID("/api/v1/workflows/{id}", "GET").
		AddNonExistentResource("/api/v1/workflows/{id}", "GET", "Workflow").
		AddInvalidJSON("/api/v1/workflows/create", "POST").
		AddMissingRequiredField("/api/v1/workflows/create", "POST", "Name").
		Build()
}

// ExecutionErrorPatterns returns common error patterns for execution endpoints
func ExecutionErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidUUID("/api/v1/executions/{id}", "GET").
		AddNonExistentResource("/api/v1/executions/{id}", "GET", "Execution").
		Build()
}

// ValidateResponse is a helper function to validate HTTP responses in tests
func ValidateResponse(t *testing.T, w any, expectedStatus int, expectedBodyContains string) {
	t.Helper()

	// This is a placeholder - actual implementation depends on the HTTP framework
	// In practice, this would check response status and body content
}
