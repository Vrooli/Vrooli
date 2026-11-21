package main

import (
	"encoding/json"
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
	Execute        func(t *testing.T, setupData interface{}) *HTTPTestRequest
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
			testReq := pattern.Execute(t, setupData)
			httpReq, w, err := makeHTTPRequest(*testReq)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			// Execute handler through router
			router := setupTestRouter()
			router.ServeHTTP(w, httpReq)

			// Validate
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", pattern.ExpectedStatus, w.Code, w.Body.String())
			}

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

// AddInvalidJSON adds a test for invalid JSON input
func (b *TestScenarioBuilder) AddInvalidJSON(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with invalid JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   "invalid-json-string",
			}
		},
	})
	return b
}

// AddMissingRequiredField adds a test for missing required fields
func (b *TestScenarioBuilder) AddMissingRequiredField(path string, method string, fieldName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           fmt.Sprintf("Missing_%s", fieldName),
		Description:    fmt.Sprintf("Test handler with missing %s field", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			// Create request with incomplete data
			return &HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   map[string]interface{}{},
			}
		},
	})
	return b
}

// AddNonExistentWheel adds a test for non-existent wheel ID
func (b *TestScenarioBuilder) AddNonExistentWheel(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistentWheel",
		Description:    "Test handler with non-existent wheel ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  "GET",
				Path:    path,
				URLVars: map[string]string{"id": "non-existent-wheel"},
			}
		},
	})
	return b
}

// AddEmptyOptions adds a test for spin with no options
func (b *TestScenarioBuilder) AddEmptyOptions() *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "EmptyOptions",
		Description:    "Test spin handler with no options",
		ExpectedStatus: http.StatusOK, // API handles gracefully
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   "/api/spin",
				Body: map[string]interface{}{
					"wheel_id": "non-existent",
					"options":  []Option{},
				},
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			var result SpinResult
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Errorf("Failed to unmarshal response: %v", err)
				return
			}
			if result.Result != "No options provided" {
				t.Errorf("Expected 'No options provided', got '%s'", result.Result)
			}
		},
	})
	return b
}

// AddInvalidWeight adds a test for invalid weight values
func (b *TestScenarioBuilder) AddInvalidWeight() *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidWeight",
		Description:    "Test handler with negative weight values",
		ExpectedStatus: http.StatusOK, // API accepts but handles in spin logic
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: "POST",
				Path:   "/api/spin",
				Body: map[string]interface{}{
					"wheel_id": "test",
					"options": []Option{
						{Label: "Option 1", Color: "#FF0000", Weight: -1.0},
						{Label: "Option 2", Color: "#00FF00", Weight: 0.0},
					},
				},
			}
		},
	})
	return b
}

// Build returns the constructed error test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// Common error patterns

// invalidJSONPattern creates a pattern for testing invalid JSON
func invalidJSONPattern(path string, method string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test with malformed JSON body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  method,
				Path:    path,
				Headers: map[string]string{"Content-Type": "application/json"},
				Body:    "this is not valid json{",
			}
		},
	}
}

// missingContentTypePattern creates a pattern for missing content-type header
func missingContentTypePattern(path string, method string) ErrorTestPattern {
	return ErrorTestPattern{
		Name:           "MissingContentType",
		Description:    "Test with missing Content-Type header",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   map[string]string{"test": "data"},
			}
		},
	}
}
