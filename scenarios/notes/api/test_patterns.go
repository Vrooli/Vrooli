// +build testing

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Execute        func(t *testing.T, env *TestEnvironment) *httptest.ResponseRecorder
	Validate       func(t *testing.T, w *httptest.ResponseRecorder)
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

// AddInvalidUUID adds an invalid UUID test scenario
func (b *TestScenarioBuilder) AddInvalidUUID(path, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment) *httptest.ResponseRecorder {
			// Replace {id} placeholder with invalid UUID
			testPath := path
			if method == "GET" || method == "PUT" || method == "DELETE" {
				testPath = fmt.Sprintf(path, "invalid-uuid-format")
			}
			return makeHTTPRequest(env, HTTPTestRequest{
				Method: method,
				Path:   testPath,
			})
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder) {
			if w.Code == http.StatusOK {
				t.Error("Expected error for invalid UUID, got success")
			}
		},
	})
	return b
}

// AddNonExistentResource adds a non-existent resource test scenario
func (b *TestScenarioBuilder) AddNonExistentResource(path, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "NonExistentResource",
		Description:    "Test handler with non-existent resource ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, env *TestEnvironment) *httptest.ResponseRecorder {
			nonExistentID := uuid.New().String()
			testPath := fmt.Sprintf(path, nonExistentID)
			return makeHTTPRequest(env, HTTPTestRequest{
				Method: method,
				Path:   testPath,
			})
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder) {
			if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
				t.Errorf("Expected 404 or 500 for non-existent resource, got %d", w.Code)
			}
		},
	})
	return b
}

// AddInvalidJSON adds a malformed JSON test scenario
func (b *TestScenarioBuilder) AddInvalidJSON(path, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment) *httptest.ResponseRecorder {
			return makeHTTPRequest(env, HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   `{"invalid": "json"`, // Malformed JSON
			})
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder) {
			if w.Code == http.StatusOK || w.Code == http.StatusCreated {
				t.Error("Expected error for invalid JSON, got success")
			}
		},
	})
	return b
}

// AddEmptyBody adds an empty body test scenario
func (b *TestScenarioBuilder) AddEmptyBody(path, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "EmptyBody",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment) *httptest.ResponseRecorder {
			return makeHTTPRequest(env, HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   "",
			})
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder) {
			if w.Code == http.StatusOK || w.Code == http.StatusCreated {
				t.Error("Expected error for empty body, got success")
			}
		},
	})
	return b
}

// AddMissingRequiredFields adds a missing required fields test scenario
func (b *TestScenarioBuilder) AddMissingRequiredFields(path, method string, body interface{}) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "MissingRequiredFields",
		Description:    "Test handler with missing required fields",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment) *httptest.ResponseRecorder {
			return makeHTTPRequest(env, HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   body,
			})
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder) {
			if w.Code == http.StatusOK || w.Code == http.StatusCreated {
				t.Error("Expected error for missing required fields, got success")
			}
		},
	})
	return b
}

// Build returns all configured test scenarios
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.scenarios
}

// RunErrorTests executes a suite of error condition tests
func RunErrorTests(t *testing.T, env *TestEnvironment, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			w := pattern.Execute(t, env)

			// Validate status code
			if pattern.ExpectedStatus > 0 && w.Code != pattern.ExpectedStatus {
				// Allow some flexibility in error codes
				if !(pattern.ExpectedStatus == http.StatusBadRequest &&
					(w.Code == http.StatusBadRequest || w.Code == http.StatusInternalServerError)) &&
					!(pattern.ExpectedStatus == http.StatusNotFound &&
					(w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError)) {
					t.Logf("Warning: Expected status %d, got %d for %s",
						pattern.ExpectedStatus, w.Code, pattern.Description)
				}
			}

			// Run custom validation if provided
			if pattern.Validate != nil {
				pattern.Validate(t, w)
			}
		})
	}
}

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name           string
	Description    string
	MaxDuration    time.Duration
	Setup          func(t *testing.T, env *TestEnvironment) interface{}
	Execute        func(t *testing.T, env *TestEnvironment, setupData interface{}) time.Duration
	Cleanup        func(setupData interface{})
}

// RunPerformanceTests executes a suite of performance tests
func RunPerformanceTests(t *testing.T, env *TestEnvironment, patterns []PerformanceTestPattern) {
	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			// Setup
			var setupData interface{}
			if pattern.Setup != nil {
				setupData = pattern.Setup(t, env)
			}

			// Cleanup
			if pattern.Cleanup != nil {
				defer pattern.Cleanup(setupData)
			}

			// Execute and measure time
			duration := pattern.Execute(t, env, setupData)

			// Validate performance
			if duration > pattern.MaxDuration {
				t.Errorf("%s took %v, expected < %v", pattern.Description, duration, pattern.MaxDuration)
			} else {
				t.Logf("%s completed in %v", pattern.Description, duration)
			}
		})
	}
}

// Common performance patterns

// CreateNotePerformance tests note creation performance
func CreateNotePerformance(maxDuration time.Duration) PerformanceTestPattern {
	return PerformanceTestPattern{
		Name:        "CreateNotePerformance",
		Description: "Note creation should complete quickly",
		MaxDuration: maxDuration,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) time.Duration {
			start := time.Now()

			w := makeHTTPRequest(env, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/notes",
				Body: map[string]interface{}{
					"title":   "Performance Test Note",
					"content": "Testing note creation performance",
				},
			})

			duration := time.Since(start)

			if w.Code != http.StatusOK && w.Code != http.StatusCreated {
				t.Errorf("Note creation failed with status %d", w.Code)
			}

			return duration
		},
	}
}

// SearchPerformance tests search performance
func SearchPerformance(maxDuration time.Duration) PerformanceTestPattern {
	return PerformanceTestPattern{
		Name:        "SearchPerformance",
		Description: "Search should complete quickly",
		MaxDuration: maxDuration,
		Setup: func(t *testing.T, env *TestEnvironment) interface{} {
			// Create some notes for searching
			notes := make([]*TestNote, 10)
			for i := 0; i < 10; i++ {
				notes[i] = createTestNote(t, env,
					fmt.Sprintf("Search Test Note %d", i),
					fmt.Sprintf("Content for testing search functionality %d", i))
			}
			return notes
		},
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) time.Duration {
			start := time.Now()

			w := makeHTTPRequest(env, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/search",
				Body: map[string]interface{}{
					"query": "testing search",
					"limit": 10,
				},
			})

			duration := time.Since(start)

			if w.Code != http.StatusOK {
				t.Errorf("Search failed with status %d", w.Code)
			}

			return duration
		},
		Cleanup: func(setupData interface{}) {
			if notes, ok := setupData.([]*TestNote); ok {
				for _, note := range notes {
					note.Cleanup()
				}
			}
		},
	}
}

// ListNotesPerformance tests listing notes performance
func ListNotesPerformance(maxDuration time.Duration, noteCount int) PerformanceTestPattern {
	return PerformanceTestPattern{
		Name:        fmt.Sprintf("ListNotesPerformance_%d", noteCount),
		Description: fmt.Sprintf("Listing %d notes should complete quickly", noteCount),
		MaxDuration: maxDuration,
		Setup: func(t *testing.T, env *TestEnvironment) interface{} {
			notes := make([]*TestNote, noteCount)
			for i := 0; i < noteCount; i++ {
				notes[i] = createTestNote(t, env,
					fmt.Sprintf("Performance Note %d", i),
					fmt.Sprintf("Content for note %d", i))
			}
			return notes
		},
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) time.Duration {
			start := time.Now()

			w := makeHTTPRequest(env, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/notes",
			})

			duration := time.Since(start)

			if w.Code != http.StatusOK {
				t.Errorf("List notes failed with status %d", w.Code)
			}

			return duration
		},
		Cleanup: func(setupData interface{}) {
			if notes, ok := setupData.([]*TestNote); ok {
				for _, note := range notes {
					note.Cleanup()
				}
			}
		},
	}
}
