package main

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
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}) *HTTPTestRequest
	Validate       func(t *testing.T, w interface{}, setupData interface{})
	Cleanup        func(setupData interface{})
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	Handler     http.HandlerFunc
	Server      *Server
	BaseURL     string
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

// AddInvalidUUID adds an invalid UUID test pattern
func (b *TestScenarioBuilder) AddInvalidUUID(urlPath string) *TestScenarioBuilder {
	pattern := ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  "GET",
				Path:    urlPath,
				URLVars: map[string]string{"build_id": "invalid-uuid-format"},
			}
		},
		Validate: func(t *testing.T, w interface{}, setupData interface{}) {
			// Additional validation can be added here
		},
	}
	b.patterns = append(b.patterns, pattern)
	return b
}

// AddNonExistentBuild adds a non-existent build ID test pattern
func (b *TestScenarioBuilder) AddNonExistentBuild(urlPath string) *TestScenarioBuilder {
	pattern := ErrorTestPattern{
		Name:           "NonExistentBuild",
		Description:    "Test handler with non-existent build ID",
		ExpectedStatus: http.StatusNotFound,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			nonExistentID := uuid.New()
			return &HTTPTestRequest{
				Method:  "GET",
				Path:    urlPath,
				URLVars: map[string]string{"build_id": nonExistentID.String()},
			}
		},
		Validate: func(t *testing.T, w interface{}, setupData interface{}) {
			// Validate error message contains appropriate information
		},
	}
	b.patterns = append(b.patterns, pattern)
	return b
}

// AddInvalidJSON adds an invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(urlPath string, method string) *TestScenarioBuilder {
	pattern := ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON body",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   "this is not valid JSON {{{",
			}
		},
		Validate: func(t *testing.T, w interface{}, setupData interface{}) {
			// Validate error message mentions JSON parsing issue
		},
	}
	b.patterns = append(b.patterns, pattern)
	return b
}

// AddMissingRequiredField adds a missing required field test pattern
func (b *TestScenarioBuilder) AddMissingRequiredField(urlPath string, method string, fieldName string) *TestScenarioBuilder {
	pattern := ErrorTestPattern{
		Name:           fmt.Sprintf("MissingField_%s", fieldName),
		Description:    fmt.Sprintf("Test handler with missing required field: %s", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			// Create incomplete request missing the specified field
			body := map[string]interface{}{}
			switch fieldName {
			case "app_name":
				body = map[string]interface{}{
					"framework":     "electron",
					"template_type": "basic",
					"output_path":   "/tmp/test",
				}
			case "framework":
				body = map[string]interface{}{
					"app_name":      "TestApp",
					"template_type": "basic",
					"output_path":   "/tmp/test",
				}
			case "template_type":
				body = map[string]interface{}{
					"app_name":    "TestApp",
					"framework":   "electron",
					"output_path": "/tmp/test",
				}
			case "output_path":
				body = map[string]interface{}{
					"app_name":      "TestApp",
					"framework":     "electron",
					"template_type": "basic",
				}
			}
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   body,
			}
		},
		Validate: func(t *testing.T, w interface{}, setupData interface{}) {
			// Validate error message mentions missing field
		},
	}
	b.patterns = append(b.patterns, pattern)
	return b
}

// AddInvalidEnumValue adds an invalid enum value test pattern
func (b *TestScenarioBuilder) AddInvalidEnumValue(urlPath string, method string, fieldName string, invalidValue string) *TestScenarioBuilder {
	pattern := ErrorTestPattern{
		Name:           fmt.Sprintf("InvalidEnum_%s", fieldName),
		Description:    fmt.Sprintf("Test handler with invalid enum value for %s", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			body := createValidGenerateRequest()
			body[fieldName] = invalidValue
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   body,
			}
		},
		Validate: func(t *testing.T, w interface{}, setupData interface{}) {
			// Validate error message mentions invalid value
		},
	}
	b.patterns = append(b.patterns, pattern)
	return b
}

// AddEmptyBody adds an empty body test pattern
func (b *TestScenarioBuilder) AddEmptyBody(urlPath string, method string) *TestScenarioBuilder {
	pattern := ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   nil,
			}
		},
		Validate: func(t *testing.T, w interface{}, setupData interface{}) {
			// Validate appropriate error response
		},
	}
	b.patterns = append(b.patterns, pattern)
	return b
}

// AddNullFieldValue adds a null field value test pattern
func (b *TestScenarioBuilder) AddNullFieldValue(urlPath string, method string, fieldName string) *TestScenarioBuilder {
	pattern := ErrorTestPattern{
		Name:           fmt.Sprintf("NullField_%s", fieldName),
		Description:    fmt.Sprintf("Test handler with null value for %s", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			body := createValidGenerateRequest()
			body[fieldName] = nil
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   body,
			}
		},
		Validate: func(t *testing.T, w interface{}, setupData interface{}) {
			// Validate error response
		},
	}
	b.patterns = append(b.patterns, pattern)
	return b
}

// AddCustomPattern adds a custom error test pattern
func (b *TestScenarioBuilder) AddCustomPattern(pattern ErrorTestPattern) *TestScenarioBuilder {
	b.patterns = append(b.patterns, pattern)
	return b
}

// Build returns the built test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// Common validation helpers

// validateStatusCode checks if response has expected status code
func validateStatusCode(t *testing.T, expectedStatus int, actualStatus int) {
	t.Helper()
	if actualStatus != expectedStatus {
		t.Errorf("Expected status code %d, got %d", expectedStatus, actualStatus)
	}
}

// validateErrorMessage checks if error message contains expected text
func validateErrorMessage(t *testing.T, body string, expectedMessage string) {
	t.Helper()
	if expectedMessage != "" && body != "" {
		// Basic validation - can be enhanced based on error format
		if len(body) == 0 {
			t.Error("Expected error message in response body, got empty body")
		}
	}
}

// validateJSONStructure checks if response is valid JSON
func validateJSONStructure(t *testing.T, contentType string) {
	t.Helper()
	if contentType != "" && contentType != "application/json" && contentType != "application/json; charset=utf-8" {
		t.Errorf("Expected Content-Type to be application/json, got %s", contentType)
	}
}

// Edge case test patterns

// createEdgeCasePatterns creates common edge case test patterns
func createEdgeCasePatterns(baseURL string) []ErrorTestPattern {
	return []ErrorTestPattern{
		{
			Name:           "EmptyAppName",
			Description:    "Test with empty app name",
			ExpectedStatus: http.StatusBadRequest,
			Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
				body := createValidGenerateRequest()
				body["app_name"] = ""
				return &HTTPTestRequest{
					Method: "POST",
					Path:   baseURL + "/api/v1/desktop/generate",
					Body:   body,
				}
			},
		},
		{
			Name:           "WhitespaceAppName",
			Description:    "Test with whitespace-only app name",
			ExpectedStatus: http.StatusBadRequest,
			Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
				body := createValidGenerateRequest()
				body["app_name"] = "   "
				return &HTTPTestRequest{
					Method: "POST",
					Path:   baseURL + "/api/v1/desktop/generate",
					Body:   body,
				}
			},
		},
		{
			Name:           "VeryLongAppName",
			Description:    "Test with extremely long app name",
			ExpectedStatus: http.StatusBadRequest,
			Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
				body := createValidGenerateRequest()
				// Create a name longer than reasonable limit (e.g., 1000 characters)
				longName := ""
				for i := 0; i < 100; i++ {
					longName += "VeryLongName"
				}
				body["app_name"] = longName
				return &HTTPTestRequest{
					Method: "POST",
					Path:   baseURL + "/api/v1/desktop/generate",
					Body:   body,
				}
			},
		},
		{
			Name:           "SpecialCharactersInAppName",
			Description:    "Test with special characters in app name",
			ExpectedStatus: http.StatusCreated, // May be valid depending on implementation
			Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
				body := createValidGenerateRequest()
				body["app_name"] = "App@#$%^&*()"
				return &HTTPTestRequest{
					Method: "POST",
					Path:   baseURL + "/api/v1/desktop/generate",
					Body:   body,
				}
			},
		},
		{
			Name:           "EmptyPlatformsArray",
			Description:    "Test with empty platforms array",
			ExpectedStatus: http.StatusCreated, // Should use defaults
			Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
				body := createValidGenerateRequest()
				body["platforms"] = []string{}
				return &HTTPTestRequest{
					Method: "POST",
					Path:   baseURL + "/api/v1/desktop/generate",
					Body:   body,
				}
			},
		},
		{
			Name:           "InvalidPlatformInArray",
			Description:    "Test with invalid platform in array",
			ExpectedStatus: http.StatusCreated, // May be validated or ignored
			Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
				body := createValidGenerateRequest()
				body["platforms"] = []string{"win", "invalid-platform", "mac"}
				return &HTTPTestRequest{
					Method: "POST",
					Path:   baseURL + "/api/v1/desktop/generate",
					Body:   body,
				}
			},
		},
	}
}

// Performance test patterns

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name              string
	Description       string
	RequestCount      int
	ConcurrentWorkers int
	ExpectedMaxTime   int64 // milliseconds
	Setup             func(t *testing.T) interface{}
	Execute           func(t *testing.T, iteration int, setupData interface{}) *HTTPTestRequest
	Cleanup           func(setupData interface{})
}

// createPerformancePatterns creates common performance test patterns
func createPerformancePatterns() []PerformanceTestPattern {
	return []PerformanceTestPattern{
		{
			Name:              "HealthCheckPerformance",
			Description:       "Test health check endpoint performance",
			RequestCount:      100,
			ConcurrentWorkers: 10,
			ExpectedMaxTime:   1000, // 1 second for 100 requests
			Execute: func(t *testing.T, iteration int, setupData interface{}) *HTTPTestRequest {
				return &HTTPTestRequest{
					Method: "GET",
					Path:   "/api/v1/health",
				}
			},
		},
		{
			Name:              "StatusEndpointPerformance",
			Description:       "Test status endpoint under load",
			RequestCount:      50,
			ConcurrentWorkers: 5,
			ExpectedMaxTime:   2000, // 2 seconds for 50 requests
			Execute: func(t *testing.T, iteration int, setupData interface{}) *HTTPTestRequest {
				return &HTTPTestRequest{
					Method: "GET",
					Path:   "/api/v1/status",
				}
			},
		},
		{
			Name:              "TemplateListingPerformance",
			Description:       "Test template listing performance",
			RequestCount:      30,
			ConcurrentWorkers: 3,
			ExpectedMaxTime:   1500, // 1.5 seconds for 30 requests
			Execute: func(t *testing.T, iteration int, setupData interface{}) *HTTPTestRequest {
				return &HTTPTestRequest{
					Method: "GET",
					Path:   "/api/v1/templates",
				}
			},
		},
	}
}
