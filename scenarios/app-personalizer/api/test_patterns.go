// +build testing

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
	Execute        func(t *testing.T, setupData interface{}) HTTPTestRequest
	Validate       func(t *testing.T, w *http.Response, setupData interface{})
	Cleanup        func(setupData interface{})
}

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: make([]ErrorTestPattern, 0),
	}
}

// AddInvalidUUID adds an invalid UUID test pattern
func (b *TestScenarioBuilder) AddInvalidUUID(urlPath string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body: map[string]interface{}{
					"app_id": "invalid-uuid",
				},
			}
		},
		Validate: nil,
		Cleanup:  nil,
	})
	return b
}

// AddNonExistentApp adds a non-existent app test pattern
func (b *TestScenarioBuilder) AddNonExistentApp(urlPath string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "NonExistentApp",
		Description:    "Test handler with non-existent app ID",
		ExpectedStatus: http.StatusNotFound,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			nonExistentID := uuid.New()
			return HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body: map[string]interface{}{
					"app_id": nonExistentID.String(),
				},
			}
		},
		Validate: nil,
		Cleanup:  nil,
	})
	return b
}

// AddInvalidJSON adds an invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(urlPath string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body:   "invalid-json{{{",
			}
		},
		Validate: nil,
		Cleanup:  nil,
	})
	return b
}

// AddMissingRequiredField adds a missing required field test pattern
func (b *TestScenarioBuilder) AddMissingRequiredField(urlPath, fieldName string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("Missing_%s", fieldName),
		Description:    fmt.Sprintf("Test handler with missing %s field", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body:   map[string]interface{}{},
			}
		},
		Validate: nil,
		Cleanup:  nil,
	})
	return b
}

// AddNonExistentPath adds a non-existent path test pattern
func (b *TestScenarioBuilder) AddNonExistentPath(urlPath string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "NonExistentPath",
		Description:    "Test handler with non-existent app path",
		ExpectedStatus: http.StatusBadRequest,
		Setup:          nil,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body: map[string]interface{}{
					"app_name":   "test-app",
					"app_path":   "/nonexistent/path",
					"app_type":   "generated",
					"framework":  "react",
					"version":    "1.0.0",
				},
			}
		},
		Validate: nil,
		Cleanup:  nil,
	})
	return b
}

// Build returns the built test scenarios
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.scenarios
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
			w, err := makeHTTPRequest(req)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			// Check status code
			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			// Validate response
			if pattern.Validate != nil {
				// Note: w is *httptest.ResponseRecorder, not *http.Response
				// The Validate function signature should be updated
				t.Log("Custom validation not fully implemented yet")
			}
		})
	}
}

// Common error patterns for all endpoints

// RegisterAppErrorPatterns returns error test patterns for RegisterApp endpoint
func RegisterAppErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON("/api/apps/register").
		AddMissingRequiredField("/api/apps/register", "app_name").
		AddNonExistentPath("/api/apps/register").
		Build()
}

// AnalyzeAppErrorPatterns returns error test patterns for AnalyzeApp endpoint
func AnalyzeAppErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON("/api/apps/analyze").
		AddInvalidUUID("/api/apps/analyze").
		AddNonExistentApp("/api/apps/analyze").
		Build()
}

// PersonalizeAppErrorPatterns returns error test patterns for PersonalizeApp endpoint
func PersonalizeAppErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON("/api/personalize").
		AddInvalidUUID("/api/personalize").
		AddNonExistentApp("/api/personalize").
		Build()
}

// BackupAppErrorPatterns returns error test patterns for BackupApp endpoint
func BackupAppErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON("/api/backup").
		AddMissingRequiredField("/api/backup", "app_path").
		Build()
}

// ValidateAppErrorPatterns returns error test patterns for ValidateApp endpoint
func ValidateAppErrorPatterns() []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON("/api/validate").
		AddMissingRequiredField("/api/validate", "app_path").
		Build()
}
