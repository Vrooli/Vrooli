// +build testing

package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestScenarioBuilder provides a fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []ErrorTestScenario
}

// ErrorTestScenario defines a systematic error test
type ErrorTestScenario struct {
	Name           string
	Description    string
	Request        HTTPTestRequest
	ExpectedStatus int
	ErrorContains  string
	Setup          func(t *testing.T, ts *TestServer) interface{}
	Cleanup        func(data interface{})
}

// NewTestScenarioBuilder creates a new scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: []ErrorTestScenario{},
	}
}

// AddInvalidUUID adds an invalid UUID test scenario
func (b *TestScenarioBuilder) AddInvalidUUID(method, path, paramName string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:        "InvalidUUID",
		Description: fmt.Sprintf("Test %s with invalid UUID", path),
		Request: HTTPTestRequest{
			Method:  method,
			Path:    path,
			URLVars: map[string]string{paramName: "invalid-uuid-format"},
		},
		ExpectedStatus: http.StatusBadRequest,
		ErrorContains:  "invalid",
	})
	return b
}

// AddNonExistentResource adds a non-existent resource test scenario
func (b *TestScenarioBuilder) AddNonExistentResource(method, path, paramName string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:        "NonExistentResource",
		Description: fmt.Sprintf("Test %s with non-existent resource", path),
		Request: HTTPTestRequest{
			Method:  method,
			Path:    path,
			URLVars: map[string]string{paramName: uuid.New().String()},
		},
		ExpectedStatus: http.StatusNotFound,
		ErrorContains:  "not found",
	})
	return b
}

// AddInvalidJSON adds a malformed JSON test scenario
func (b *TestScenarioBuilder) AddInvalidJSON(method, path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:        "InvalidJSON",
		Description: fmt.Sprintf("Test %s with malformed JSON", path),
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   `{"invalid": "json"`, // Malformed
		},
		ExpectedStatus: http.StatusBadRequest,
		ErrorContains:  "",
	})
	return b
}

// AddMissingRequiredField adds a missing required field test
func (b *TestScenarioBuilder) AddMissingRequiredField(method, path, field string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:        fmt.Sprintf("Missing%s", field),
		Description: fmt.Sprintf("Test %s with missing %s field", path, field),
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   map[string]interface{}{}, // Empty body
		},
		ExpectedStatus: http.StatusBadRequest,
		ErrorContains:  field,
	})
	return b
}

// AddUnauthorized adds an unauthorized access test
func (b *TestScenarioBuilder) AddUnauthorized(method, path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:        "Unauthorized",
		Description: fmt.Sprintf("Test %s without authentication", path),
		Request: HTTPTestRequest{
			Method:  method,
			Path:    path,
			Headers: map[string]string{}, // No auth header
		},
		ExpectedStatus: http.StatusUnauthorized,
		ErrorContains:  "unauthorized",
	})
	return b
}

// AddEmptyBody adds an empty body test
func (b *TestScenarioBuilder) AddEmptyBody(method, path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:        "EmptyBody",
		Description: fmt.Sprintf("Test %s with empty body", path),
		Request: HTTPTestRequest{
			Method: method,
			Path:   path,
			Body:   "",
		},
		ExpectedStatus: http.StatusBadRequest,
		ErrorContains:  "",
	})
	return b
}

// AddInvalidContentType adds an invalid content type test
func (b *TestScenarioBuilder) AddInvalidContentType(method, path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestScenario{
		Name:        "InvalidContentType",
		Description: fmt.Sprintf("Test %s with invalid content type", path),
		Request: HTTPTestRequest{
			Method:  method,
			Path:    path,
			Headers: map[string]string{"Content-Type": "text/plain"},
			Body:    "plain text body",
		},
		ExpectedStatus: http.StatusBadRequest,
		ErrorContains:  "",
	})
	return b
}

// Build returns the built scenarios
func (b *TestScenarioBuilder) Build() []ErrorTestScenario {
	return b.scenarios
}

// ErrorTestPattern provides reusable error test patterns
type ErrorTestPattern struct {
	Name     string
	Patterns []ErrorTestScenario
}

// RunErrorTests executes all error test scenarios
func RunErrorTests(t *testing.T, ts *TestServer, scenarios []ErrorTestScenario) {
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			var setupData interface{}

			// Setup
			if scenario.Setup != nil {
				setupData = scenario.Setup(t, ts)
			}

			// Cleanup
			if scenario.Cleanup != nil {
				defer scenario.Cleanup(setupData)
			}

			// Execute request
			w, err := ts.makeHTTPRequest(scenario.Request)
			if err != nil {
				t.Fatalf("Failed to make HTTP request: %v", err)
			}

			// Validate response
			if scenario.ErrorContains != "" {
				assertErrorResponse(t, w, scenario.ExpectedStatus, scenario.ErrorContains)
			} else {
				if w.Code != scenario.ExpectedStatus {
					t.Errorf("Expected status %d, got %d. Body: %s",
						scenario.ExpectedStatus, w.Code, w.Body.String())
				}
			}
		})
	}
}

// PerformanceTestPattern defines performance test scenarios
type PerformanceTestPattern struct {
	Name         string
	Description  string
	MaxDuration  time.Duration
	Iterations   int
	Concurrent   bool
	Setup        func(t *testing.T, ts *TestServer) interface{}
	Execute      func(t *testing.T, ts *TestServer, iteration int, setupData interface{})
	Validate     func(t *testing.T, duration time.Duration, iterations int)
	Cleanup      func(setupData interface{})
}

// RunPerformanceTest executes a performance test pattern
func RunPerformanceTest(t *testing.T, ts *TestServer, pattern PerformanceTestPattern) {
	t.Run(pattern.Name, func(t *testing.T) {
		var setupData interface{}

		// Setup
		if pattern.Setup != nil {
			setupData = pattern.Setup(t, ts)
		}

		// Cleanup
		if pattern.Cleanup != nil {
			defer pattern.Cleanup(setupData)
		}

		start := time.Now()

		if pattern.Concurrent {
			// Run iterations concurrently
			done := make(chan bool, pattern.Iterations)
			for i := 0; i < pattern.Iterations; i++ {
				go func(iteration int) {
					pattern.Execute(t, ts, iteration, setupData)
					done <- true
				}(i)
			}

			// Wait for all to complete
			for i := 0; i < pattern.Iterations; i++ {
				<-done
			}
		} else {
			// Run iterations sequentially
			for i := 0; i < pattern.Iterations; i++ {
				pattern.Execute(t, ts, i, setupData)
			}
		}

		duration := time.Since(start)

		// Validate performance
		if duration > pattern.MaxDuration {
			t.Errorf("Performance test exceeded max duration: %v > %v", duration, pattern.MaxDuration)
		}

		if pattern.Validate != nil {
			pattern.Validate(t, duration, pattern.Iterations)
		}

		t.Logf("Performance test completed: %d iterations in %v (avg: %v)",
			pattern.Iterations, duration, duration/time.Duration(pattern.Iterations))
	})
}

// Common test patterns

// DeviceRegistrationErrorPatterns returns common device registration error scenarios
func DeviceRegistrationErrorPatterns() *TestScenarioBuilder {
	return NewTestScenarioBuilder().
		AddInvalidJSON("POST", "/api/v1/devices").
		AddMissingRequiredField("POST", "/api/v1/devices", "name").
		AddEmptyBody("POST", "/api/v1/devices")
}

// DeviceOperationErrorPatterns returns common device operation error scenarios
func DeviceOperationErrorPatterns(deviceID string) *TestScenarioBuilder {
	path := fmt.Sprintf("/api/v1/devices/%s", deviceID)
	return NewTestScenarioBuilder().
		AddInvalidUUID("GET", path, "id").
		AddNonExistentResource("GET", path, "id").
		AddInvalidUUID("PUT", path, "id").
		AddInvalidUUID("DELETE", path, "id")
}

// SyncItemErrorPatterns returns common sync item error scenarios
func SyncItemErrorPatterns() *TestScenarioBuilder {
	return NewTestScenarioBuilder().
		AddInvalidJSON("POST", "/api/v1/sync/upload").
		AddEmptyBody("POST", "/api/v1/sync/upload").
		AddInvalidUUID("GET", "/api/v1/sync/items/{id}", "id").
		AddNonExistentResource("GET", "/api/v1/sync/items/{id}", "id").
		AddInvalidUUID("DELETE", "/api/v1/sync/items/{id}", "id")
}

// FileUploadErrorPatterns returns file upload error scenarios
func FileUploadErrorPatterns() []ErrorTestScenario {
	return []ErrorTestScenario{
		{
			Name:        "FileTooLarge",
			Description: "Test file upload with oversized file",
			Request: HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/files/upload",
			},
			Setup: func(t *testing.T, ts *TestServer) interface{} {
				// Create a file larger than max size
				largeContent := make([]byte, ts.Server.config.MaxFileSize+1024)
				body, contentType := createMultipartFileRequest(t, "file", "large.bin", largeContent, nil)
				return map[string]interface{}{
					"body":        body,
					"contentType": contentType,
				}
			},
			ExpectedStatus: http.StatusBadRequest,
			ErrorContains:  "too large",
		},
		{
			Name:        "MissingFile",
			Description: "Test file upload without file",
			Request: HTTPTestRequest{
				Method:  "POST",
				Path:    "/api/v1/files/upload",
				Headers: map[string]string{"Content-Type": "multipart/form-data"},
			},
			ExpectedStatus: http.StatusBadRequest,
			ErrorContains:  "file",
		},
	}
}

// ClipboardSyncErrorPatterns returns clipboard sync error scenarios
func ClipboardSyncErrorPatterns() *TestScenarioBuilder {
	return NewTestScenarioBuilder().
		AddInvalidJSON("POST", "/api/v1/sync/clipboard").
		AddMissingRequiredField("POST", "/api/v1/sync/clipboard", "content").
		AddEmptyBody("POST", "/api/v1/sync/clipboard")
}

// WebSocketConnectionErrorPatterns returns WebSocket connection error scenarios
func WebSocketConnectionErrorPatterns() []ErrorTestScenario {
	return []ErrorTestScenario{
		{
			Name:        "MissingAuthToken",
			Description: "Test WebSocket connection without auth token",
			Request: HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/sync/ws",
			},
			ExpectedStatus: http.StatusUnauthorized,
			ErrorContains:  "",
		},
		{
			Name:        "InvalidAuthToken",
			Description: "Test WebSocket connection with invalid token",
			Request: HTTPTestRequest{
				Method:      "GET",
				Path:        "/api/v1/sync/ws",
				QueryParams: map[string]string{"token": "invalid-token"},
			},
			ExpectedStatus: http.StatusUnauthorized,
			ErrorContains:  "",
		},
	}
}
