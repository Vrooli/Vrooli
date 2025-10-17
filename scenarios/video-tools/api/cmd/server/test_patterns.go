package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

// ErrorTestPattern defines systematic error condition testing
type ErrorTestPattern struct {
	Name           string
	Description    string
	Path           string
	Method         string
	Body           interface{}
	ExpectedStatus int
	ExpectedError  string
}

// TestScenarioBuilder provides fluent interface for building test scenarios
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
func (b *TestScenarioBuilder) AddInvalidUUID(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test with invalid UUID format",
		Path:           path,
		Method:         "GET",
		ExpectedStatus: http.StatusNotFound, // mux returns 404 for invalid routes
	})
	return b
}

// AddNonExistentVideo adds a non-existent video test pattern
func (b *TestScenarioBuilder) AddNonExistentVideo(pathTemplate string) *TestScenarioBuilder {
	nonExistentID := uuid.New().String()
	path := fmt.Sprintf(pathTemplate, nonExistentID)

	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "NonExistentVideo",
		Description:    "Test with non-existent video ID",
		Path:           path,
		Method:         "GET",
		ExpectedStatus: http.StatusNotFound,
		ExpectedError:  "video not found",
	})
	return b
}

// AddInvalidJSON adds an invalid JSON body test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test with malformed JSON body",
		Path:           path,
		Method:         "POST",
		Body:           "{invalid json",
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "invalid request body",
	})
	return b
}

// AddMissingAuth adds a missing authentication test pattern
func (b *TestScenarioBuilder) AddMissingAuth(path string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "MissingAuth",
		Description:    "Test without authentication token",
		Path:           path,
		Method:         method,
		ExpectedStatus: http.StatusUnauthorized,
		ExpectedError:  "unauthorized",
	})
	return b
}

// AddInvalidAuth adds an invalid authentication test pattern
func (b *TestScenarioBuilder) AddInvalidAuth(path string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidAuth",
		Description:    "Test with invalid authentication token",
		Path:           path,
		Method:         method,
		ExpectedStatus: http.StatusUnauthorized,
		ExpectedError:  "unauthorized",
	})
	return b
}

// AddMissingRequiredField adds a missing required field test pattern
func (b *TestScenarioBuilder) AddMissingRequiredField(path string, body interface{}, fieldName string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           fmt.Sprintf("Missing_%s", fieldName),
		Description:    fmt.Sprintf("Test with missing required field: %s", fieldName),
		Path:           path,
		Method:         "POST",
		Body:           body,
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// AddInvalidValue adds an invalid value test pattern
func (b *TestScenarioBuilder) AddInvalidValue(path string, body interface{}, description string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidValue",
		Description:    description,
		Path:           path,
		Method:         "POST",
		Body:           body,
		ExpectedStatus: http.StatusBadRequest,
	})
	return b
}

// Build returns the built test scenarios
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.scenarios
}

// RunErrorTests executes a suite of error condition tests
func RunErrorTests(t *testing.T, server *Server, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			var w *httptest.ResponseRecorder

			switch pattern.Method {
			case "GET":
				w = makeHTTPRequest(server, "GET", pattern.Path, nil, nil)
			case "POST":
				if pattern.Name == "MissingAuth" || pattern.Name == "InvalidAuth" {
					// Test without auth header
					headers := map[string]string{}
					if pattern.Name == "InvalidAuth" {
						headers["Authorization"] = "Bearer invalid-token"
					}
					w = makeHTTPRequest(server, "POST", pattern.Path, nil, headers)
				} else if str, ok := pattern.Body.(string); ok {
					// Invalid JSON string
					w = makeHTTPRequest(server, "POST", pattern.Path, bytes.NewReader([]byte(str)), map[string]string{
						"Content-Type": "application/json",
					})
				} else {
					// Valid JSON body
					w = makeJSONRequest(server, "POST", pattern.Path, pattern.Body)
				}
			}

			if w.Code != pattern.ExpectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s",
					pattern.ExpectedStatus, w.Code, w.Body.String())
			}

			if pattern.ExpectedError != "" {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				if errorMsg, ok := response["error"].(string); ok {
					if errorMsg != pattern.ExpectedError {
						t.Errorf("Expected error '%s', got '%s'",
							pattern.ExpectedError, errorMsg)
					}
				}
			}
		})
	}
}

// HandlerTestSuite provides comprehensive testing framework for HTTP handlers
type HandlerTestSuite struct {
	Name        string
	Server      *Server
	BasePath    string
	CleanupFunc func()
}

// NewHandlerTestSuite creates a new handler test suite
func NewHandlerTestSuite(t *testing.T, name string) *HandlerTestSuite {
	server, cleanup := setupTestServer(t)

	return &HandlerTestSuite{
		Name:        name,
		Server:      server,
		CleanupFunc: cleanup,
	}
}

// Cleanup performs cleanup after tests
func (suite *HandlerTestSuite) Cleanup() {
	if suite.CleanupFunc != nil {
		suite.CleanupFunc()
	}
}

// TestSuccessCase tests a successful API call
func (suite *HandlerTestSuite) TestSuccessCase(t *testing.T, testName string, method, path string, body interface{}, expectedStatus int) map[string]interface{} {
	t.Helper()

	var w *httptest.ResponseRecorder
	if body != nil {
		w = makeJSONRequest(suite.Server, method, path, body)
	} else {
		w = makeHTTPRequest(suite.Server, method, path, nil, nil)
	}

	return assertJSONResponse(t, w, expectedStatus)
}

// TestErrorCase tests an error condition
func (suite *HandlerTestSuite) TestErrorCase(t *testing.T, testName string, method, path string, body interface{}, expectedStatus int, expectedError string) {
	t.Helper()

	var w *httptest.ResponseRecorder
	if body != nil {
		w = makeJSONRequest(suite.Server, method, path, body)
	} else {
		w = makeHTTPRequest(suite.Server, method, path, nil, nil)
	}

	assertErrorResponse(t, w, expectedStatus, expectedError)
}

// Common test data generators

// GenerateTestVideoData generates test video upload data
func GenerateTestVideoData(name string) map[string]interface{} {
	return map[string]interface{}{
		"name":         name,
		"description":  fmt.Sprintf("Test video: %s", name),
		"tags":         []string{"test", "sample"},
		"auto_analyze": false,
	}
}

// GenerateTestConvertRequest generates test conversion request
func GenerateTestConvertRequest(format string) map[string]interface{} {
	return map[string]interface{}{
		"target_format": format,
		"resolution":    "720p",
		"quality":       "medium",
		"preset":        "fast",
	}
}

// GenerateTestEditRequest generates test edit request
func GenerateTestEditRequest() map[string]interface{} {
	return map[string]interface{}{
		"operations": []map[string]interface{}{
			{
				"type": "trim",
				"parameters": map[string]interface{}{
					"start_time": 5.0,
					"end_time":   10.0,
				},
			},
		},
		"output_format": "mp4",
		"quality":       "high",
	}
}
