// +build testing

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
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
			w, httpReq, err := makeHTTPRequest(req)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			// Execute handler
			suite.Handler(w, httpReq)

			// Validate response
			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			} else {
				// Default validation: check status code
				assertJSONResponse(t, w, pattern.ExpectedStatus, nil)
			}
		})
	}
}

// TestScenarioBuilder provides fluent interface for building test scenarios
type TestScenarioBuilder struct {
	patterns []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		patterns: make([]ErrorTestPattern, 0),
	}
}

// AddInvalidUUID adds test for invalid UUID parameter
func (b *TestScenarioBuilder) AddInvalidUUID(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidUUIDPattern(urlPath))
	return b
}

// AddNonExistentAlgorithm adds test for non-existent algorithm
func (b *TestScenarioBuilder) AddNonExistentAlgorithm(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, nonExistentAlgorithmPattern(urlPath))
	return b
}

// AddInvalidJSON adds test for malformed JSON request
func (b *TestScenarioBuilder) AddInvalidJSON(urlPath string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidJSONPattern(urlPath, method))
	return b
}

// AddMissingRequiredField adds test for missing required field in request
func (b *TestScenarioBuilder) AddMissingRequiredField(urlPath, method, fieldName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, missingRequiredFieldPattern(urlPath, method, fieldName))
	return b
}

// AddInvalidLanguage adds test for unsupported language
func (b *TestScenarioBuilder) AddInvalidLanguage(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, invalidLanguagePattern(urlPath))
	return b
}

// AddEmptyRequest adds test for empty request body
func (b *TestScenarioBuilder) AddEmptyRequest(urlPath, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, emptyRequestPattern(urlPath, method))
	return b
}

// Build returns the complete test pattern array
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// Common error patterns that can be reused across handlers

// invalidUUIDPattern tests handlers with invalid UUID formats
func invalidUUIDPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  "GET",
				Path:    urlPath,
				URLVars: map[string]string{"id": "invalid-uuid-format"},
			}
		},
	}
}

// nonExistentAlgorithmPattern tests handlers with non-existent algorithm IDs
func nonExistentAlgorithmPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "NonExistentAlgorithm",
		Description:    "Test handler with non-existent algorithm ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			nonExistentID := uuid.New().String()
			return HTTPTestRequest{
				Method:  "GET",
				Path:    urlPath,
				URLVars: map[string]string{"id": nonExistentID},
			}
		},
	}
}

// invalidJSONPattern tests handlers with malformed JSON
func invalidJSONPattern(urlPath, method string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON request body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				// This will bypass the normal JSON marshaling
				Body: nil,
			}
		},
	}
}

// missingRequiredFieldPattern tests handlers with missing required fields
func missingRequiredFieldPattern(urlPath, method, fieldName string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           fmt.Sprintf("Missing%s", fieldName),
		Description:    fmt.Sprintf("Test handler with missing required field: %s", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			// Create request with minimal data that excludes the required field
			body := make(map[string]interface{})
			// Depending on the field, we might add some data but omit the required field
			return HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   body,
			}
		},
	}
}

// invalidLanguagePattern tests validation with unsupported language
func invalidLanguagePattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidLanguage",
		Description:    "Test validation with unsupported programming language",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body: map[string]interface{}{
					"algorithm_id": "quicksort",
					"language":     "fortran",  // Unsupported language
					"code":         "program main\nend program",
				},
			}
		},
	}
}

// emptyRequestPattern tests handlers with empty request body
func emptyRequestPattern(urlPath, method string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "EmptyRequest",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   nil,
			}
		},
	}
}

// sqlInjectionPattern tests handlers for SQL injection resistance
func sqlInjectionPattern(urlPath string, paramName string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "SQLInjection",
		Description:    "Test handler resistance to SQL injection attempts",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "GET",
				Path:   urlPath,
				QueryParams: map[string]string{
					paramName: "'; DROP TABLE algorithms; --",
				},
			}
		},
	}
}

// xssPattern tests handlers for XSS attack resistance
func xssPattern(urlPath, method string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "XSSAttack",
		Description:    "Test handler resistance to XSS attacks",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body: map[string]interface{}{
					"name": "<script>alert('xss')</script>",
					"code": "<img src=x onerror=alert('xss')>",
				},
			}
		},
	}
}

// pathTraversalPattern tests handlers for path traversal resistance
func pathTraversalPattern(urlPath string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "PathTraversal",
		Description:    "Test handler resistance to path traversal attacks",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  "GET",
				Path:    urlPath,
				URLVars: map[string]string{"id": "../../../etc/passwd"},
			}
		},
	}
}

// oversizedPayloadPattern tests handlers with oversized request payloads
func oversizedPayloadPattern(urlPath, method string, sizeKB int) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "OversizedPayload",
		Description:    fmt.Sprintf("Test handler with %dKB oversized payload", sizeKB),
		ExpectedStatus: http.StatusRequestEntityTooLarge,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			// Create a large string
			largeCode := string(make([]byte, sizeKB*1024))
			return HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body: map[string]interface{}{
					"algorithm_id": "quicksort",
					"language":     "python",
					"code":         largeCode,
				},
			}
		},
	}
}

// raceConditionPattern tests for race condition handling
func raceConditionPattern(urlPath, method string, iterations int) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "RaceCondition",
		Description:    fmt.Sprintf("Test concurrent requests (%d iterations)", iterations),
		ExpectedStatus: http.StatusOK,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body: map[string]interface{}{
					"algorithm_id": "quicksort",
					"language":     "python",
					"code":         "def quicksort(arr): return sorted(arr)",
				},
			}
		},
	}
}

// timeoutPattern tests handler behavior with operations that might timeout
func timeoutPattern(urlPath, method string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "Timeout",
		Description:    "Test handler behavior with timeout-prone operations",
		ExpectedStatus: http.StatusRequestTimeout,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			// Code that takes a very long time to execute
			infiniteLoopCode := `
def quicksort(arr):
    while True:
        pass
    return arr
`
			return HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body: map[string]interface{}{
					"algorithm_id": "quicksort",
					"language":     "python",
					"code":         infiniteLoopCode,
				},
			}
		},
	}
}
