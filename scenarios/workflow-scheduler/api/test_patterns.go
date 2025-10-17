package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// TestScenario represents a test scenario with setup, execution, and validation
type TestScenario struct {
	Name           string
	Method         string
	Path           string
	Body           interface{}
	QueryParams    map[string]string
	ExpectedStatus int
	Setup          func(t *testing.T, app *TestApp) interface{}
	Validate       func(t *testing.T, rr interface{}, setupData interface{})
	Cleanup        func(t *testing.T, app *TestApp, setupData interface{})
}

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []TestScenario
}

// NewTestScenarioBuilder creates a new scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: make([]TestScenario, 0),
	}
}

// AddInvalidUUID adds a test for invalid UUID format
func (b *TestScenarioBuilder) AddInvalidUUID(path string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "InvalidUUID",
		Method:         method,
		Path:           path,
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddNonExistentSchedule adds a test for non-existent schedule
func (b *TestScenarioBuilder) AddNonExistentSchedule(pathTemplate string, method string) *TestScenarioBuilder {
	nonExistentID := uuid.New().String()
	path := fmt.Sprintf(pathTemplate, nonExistentID)

	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "NonExistentSchedule",
		Method:         method,
		Path:           path,
		ExpectedStatus: http.StatusNotFound,
	})
	return b
}

// AddInvalidJSON adds a test for invalid JSON payload
func (b *TestScenarioBuilder) AddInvalidJSON(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           "InvalidJSON",
		Method:         "POST",
		Path:           path,
		Body:           "invalid json",
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddMissingRequiredField adds a test for missing required field
func (b *TestScenarioBuilder) AddMissingRequiredField(path string, fieldName string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:           fmt.Sprintf("Missing_%s", fieldName),
		Method:         "POST",
		Path:           path,
		Body:           map[string]interface{}{},
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddInvalidCronExpression adds a test for invalid cron expression
func (b *TestScenarioBuilder) AddInvalidCronExpression(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, TestScenario{
		Name:   "InvalidCronExpression",
		Method: "POST",
		Path:   path,
		Body: map[string]interface{}{
			"name":            "Test Schedule",
			"cron_expression": "invalid cron",
			"target_type":     "webhook",
			"target_url":      "http://example.com",
		},
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// Build returns all scenarios
func (b *TestScenarioBuilder) Build() []TestScenario {
	return b.scenarios
}

// ErrorTestPattern defines common error testing patterns
type ErrorTestPattern struct {
	scenarios []TestScenario
}

// NewErrorTestPattern creates a new error test pattern
func NewErrorTestPattern() *ErrorTestPattern {
	return &ErrorTestPattern{
		scenarios: make([]TestScenario, 0),
	}
}

// GetScheduleErrors returns common error scenarios for GET /api/schedules/{id}
func GetScheduleErrors() []TestScenario {
	return NewTestScenarioBuilder().
		AddNonExistentSchedule("/api/schedules/%s", "GET").
		Build()
}

// CreateScheduleErrors returns common error scenarios for POST /api/schedules
func CreateScheduleErrors() []TestScenario {
	builder := NewTestScenarioBuilder()
	builder.AddInvalidCronExpression("/api/schedules")

	// Missing name
	builder.scenarios = append(builder.scenarios, TestScenario{
		Name:   "MissingName",
		Method: "POST",
		Path:   "/api/schedules",
		Body: map[string]interface{}{
			"cron_expression": "0 * * * *",
			"target_type":     "webhook",
		},
		ExpectedStatus: http.StatusBadRequest,
	})

	return builder.Build()
}

// UpdateScheduleErrors returns common error scenarios for PUT /api/schedules/{id}
func UpdateScheduleErrors() []TestScenario {
	builder := NewTestScenarioBuilder()
	builder.AddNonExistentSchedule("/api/schedules/%s", "PUT")
	builder.AddInvalidCronExpression("/api/schedules/test-id")

	return builder.Build()
}

// DeleteScheduleErrors returns common error scenarios for DELETE /api/schedules/{id}
func DeleteScheduleErrors() []TestScenario {
	return NewTestScenarioBuilder().
		AddNonExistentSchedule("/api/schedules/%s", "DELETE").
		Build()
}

// TriggerScheduleErrors returns common error scenarios for POST /api/schedules/{id}/trigger
func TriggerScheduleErrors() []TestScenario {
	return NewTestScenarioBuilder().
		AddNonExistentSchedule("/api/schedules/%s/trigger", "POST").
		Build()
}

// GetExecutionErrors returns common error scenarios for GET /api/executions/{id}
func GetExecutionErrors() []TestScenario {
	return NewTestScenarioBuilder().
		AddNonExistentSchedule("/api/executions/%s", "GET").
		Build()
}

// ValidateCronErrors returns common error scenarios for GET /api/cron/validate
func ValidateCronErrors() []TestScenario {
	return []TestScenario{
		{
			Name:           "MissingExpression",
			Method:         "GET",
			Path:           "/api/cron/validate",
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:   "InvalidExpression",
			Method: "GET",
			Path:   "/api/cron/validate",
			QueryParams: map[string]string{
				"expression": "invalid cron",
			},
			ExpectedStatus: http.StatusOK, // Returns valid:false
		},
	}
}

// RunErrorScenarios executes a set of error scenarios
func RunErrorScenarios(t *testing.T, app *TestApp, scenarios []TestScenario) {
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			// Setup
			var setupData interface{}
			if scenario.Setup != nil {
				setupData = scenario.Setup(t, app)
			}

			// Cleanup
			if scenario.Cleanup != nil {
				defer scenario.Cleanup(t, app, setupData)
			}

			// Build request
			req := HTTPTestRequest{
				Method:      scenario.Method,
				Path:        scenario.Path,
				Body:        scenario.Body,
				QueryParams: scenario.QueryParams,
			}

			// Execute
			rr, err := makeHTTPRequest(app.Router, req)
			if err != nil {
				t.Fatalf("Failed to make HTTP request: %v", err)
			}

			// Validate status code
			if rr.Code != scenario.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					scenario.ExpectedStatus, rr.Code, rr.Body.String())
			}

			// Custom validation
			if scenario.Validate != nil {
				scenario.Validate(t, rr, setupData)
			}
		})
	}
}
