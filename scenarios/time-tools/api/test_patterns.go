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
			w := executeRequest(suite.Handler, req)

			// Basic status validation
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Custom validation
			if pattern.Validate != nil {
				pattern.Validate(t, w, setupData)
			}
		})
	}
}

// TestScenarioBuilder provides fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: []ErrorTestPattern{},
	}
}

// AddInvalidJSON adds test for invalid JSON input
func (b *TestScenarioBuilder) AddInvalidJSON(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test with malformed JSON body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body:   "invalid json",
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusBadRequest, "Invalid request body")
		},
	})
	return b
}

// AddMissingRequiredField adds test for missing required field
func (b *TestScenarioBuilder) AddMissingRequiredField(path string, field string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("Missing_%s", field),
		Description:    fmt.Sprintf("Test with missing %s field", field),
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body:   map[string]interface{}{},
			}
		},
	})
	return b
}

// AddInvalidTimeFormat adds test for invalid time format
func (b *TestScenarioBuilder) AddInvalidTimeFormat(path string, field string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("Invalid_%s_Format", field),
		Description:    fmt.Sprintf("Test with invalid %s format", field),
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			body := map[string]interface{}{
				field: "not-a-valid-time",
			}
			return HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body:   body,
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusBadRequest, "")
		},
	})
	return b
}

// AddInvalidTimezone adds test for invalid timezone
func (b *TestScenarioBuilder) AddInvalidTimezone(path string) *TestScenarioBuilder {
	testTime := getTestTime()
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidTimezone",
		Description:    "Test with invalid timezone",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			body := map[string]interface{}{
				"time":          testTime.RFC3339,
				"from_timezone": "Invalid/Timezone",
				"to_timezone":   "America/New_York",
			}
			return HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body:   body,
			}
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusBadRequest, "timezone")
		},
	})
	return b
}

// AddEmptyBody adds test for empty request body
func (b *TestScenarioBuilder) AddEmptyBody(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body:   "", // Empty string will be sent as body
			}
		},
	})
	return b
}

// AddMethodNotAllowed adds test for wrong HTTP method
func (b *TestScenarioBuilder) AddMethodNotAllowed(path string, wrongMethod string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("MethodNotAllowed_%s", wrongMethod),
		Description:    fmt.Sprintf("Test with wrong HTTP method %s", wrongMethod),
		ExpectedStatus: http.StatusMethodNotAllowed,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: wrongMethod,
				Path:   path,
				Body:   nil,
			}
		},
	})
	return b
}

// Build returns the built test scenarios
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.scenarios
}

// Common test patterns for time-tools specific scenarios

// TimezoneConversionErrorPatterns creates error patterns for timezone conversion
func TimezoneConversionErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON("/api/v1/time/convert").
		AddEmptyBody("/api/v1/time/convert").
		AddInvalidTimeFormat("/api/v1/time/convert", "time").
		AddInvalidTimezone("/api/v1/time/convert").
		Build()
}

// DurationCalculationErrorPatterns creates error patterns for duration calculation
func DurationCalculationErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON("/api/v1/time/duration").
		AddEmptyBody("/api/v1/time/duration").
		AddInvalidTimeFormat("/api/v1/time/duration", "start_time").
		AddInvalidTimeFormat("/api/v1/time/duration", "end_time").
		Build()
}

// FormatTimeErrorPatterns creates error patterns for time formatting
func FormatTimeErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON("/api/v1/time/format").
		AddEmptyBody("/api/v1/time/format").
		AddInvalidTimeFormat("/api/v1/time/format", "time").
		Build()
}

// TimeArithmeticErrorPatterns creates error patterns for time arithmetic
func TimeArithmeticErrorPatterns(path string) []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON(path).
		AddEmptyBody(path).
		AddInvalidTimeFormat(path, "time").
		Build()
}

// ScheduleOptimizationErrorPatterns creates error patterns for schedule optimization
func ScheduleOptimizationErrorPatterns() []ErrorTestPattern {
	builder := NewTestScenarioBuilder().
		AddInvalidJSON("/api/v1/schedule/optimal").
		AddEmptyBody("/api/v1/schedule/optimal")

	// Add invalid date format test
	builder.scenarios = append(builder.scenarios, ErrorTestPattern{
		Name:           "InvalidDateFormat",
		Description:    "Test with invalid date format",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/schedule/optimal",
				Body: map[string]interface{}{
					"earliest_date":      "invalid-date",
					"latest_date":        "2024-01-31",
					"duration_minutes":   60,
					"timezone":           "America/New_York",
					"business_hours_only": true,
				},
			}
		},
	})

	return builder.Build()
}
