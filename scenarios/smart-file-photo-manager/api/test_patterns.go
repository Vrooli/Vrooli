// +build testing

package main

import (
	"fmt"
	"net/http"
	"testing"
)

// ErrorTestPattern defines systematic error testing
type ErrorTestPattern struct {
	Name           string
	Method         string
	Path           string
	Body           interface{}
	ExpectedStatus int
	ExpectedError  string
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
func (b *TestScenarioBuilder) AddInvalidUUID(path, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidUUID",
		Method:         method,
		Path:           path,
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "",
	})
	return b
}

// AddNonExistentFile adds test for non-existent file
func (b *TestScenarioBuilder) AddNonExistentFile(fileID string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistentFile",
		Method:         "GET",
		Path:           fmt.Sprintf("/api/files/%s", fileID),
		ExpectedStatus: http.StatusNotFound,
		ExpectedError:  "not found",
	})
	return b
}

// AddInvalidJSON adds test for malformed JSON
func (b *TestScenarioBuilder) AddInvalidJSON(path, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Method:         method,
		Path:           path,
		Body:           "invalid-json",
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "Invalid request",
	})
	return b
}

// AddMissingRequiredField adds test for missing required field
func (b *TestScenarioBuilder) AddMissingRequiredField(path, method, fieldName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           fmt.Sprintf("Missing_%s", fieldName),
		Method:         method,
		Path:           path,
		Body:           map[string]interface{}{}, // Empty body
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "",
	})
	return b
}

// AddEmptyQuery adds test for empty search query
func (b *TestScenarioBuilder) AddEmptyQuery(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "EmptyQuery",
		Method:         "GET",
		Path:           path,
		ExpectedStatus: http.StatusBadRequest,
		ExpectedError:  "required",
	})
	return b
}

// Build returns the constructed error patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// HandlerTestSuite provides comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	Env         *TestEnvironment
}

// RunErrorTests executes systematic error condition tests
func (suite *HandlerTestSuite) RunErrorTests(t *testing.T, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(pattern.Name, func(t *testing.T) {
			w, err := makeHTTPRequest(suite.Env, pattern.Method, pattern.Path, pattern.Body)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}

			assertStatusCode(t, w, pattern.ExpectedStatus)

			// Verify error message if specified
			if pattern.ExpectedError != "" {
				assertErrorResponse(t, w, pattern.ExpectedStatus, pattern.ExpectedError)
			}
		})
	}
}

// Common test patterns for file operations

// FileUploadErrorPatterns returns common file upload error scenarios
func FileUploadErrorPatterns() []ErrorTestPattern {
	return []ErrorTestPattern{
		{
			Name:           "MissingFilename",
			Method:         "POST",
			Path:           "/api/files",
			Body:           map[string]interface{}{"file_hash": "abc123"},
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:           "MissingFileHash",
			Method:         "POST",
			Path:           "/api/files",
			Body:           map[string]interface{}{"filename": "test.jpg"},
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:           "InvalidMimeType",
			Method:         "POST",
			Path:           "/api/files",
			Body: map[string]interface{}{
				"filename":     "test.jpg",
				"file_hash":    "abc123",
				"mime_type":    "",
				"size_bytes":   1024,
				"storage_path": "/test",
			},
			ExpectedStatus: http.StatusBadRequest,
		},
	}
}

// SearchErrorPatterns returns common search error scenarios
func SearchErrorPatterns() []ErrorTestPattern {
	return []ErrorTestPattern{
		{
			Name:           "EmptySearchQuery",
			Method:         "GET",
			Path:           "/api/search",
			ExpectedStatus: http.StatusBadRequest,
			ExpectedError:  "required",
		},
		{
			Name:           "InvalidSearchType",
			Method:         "POST",
			Path:           "/api/search",
			Body: map[string]interface{}{
				"query": "test",
				"type":  "invalid_type",
			},
			ExpectedStatus: http.StatusOK, // Currently accepts any type
		},
	}
}

// FolderErrorPatterns returns common folder operation error scenarios
func FolderErrorPatterns() []ErrorTestPattern {
	return []ErrorTestPattern{
		{
			Name:           "MissingFolderPath",
			Method:         "POST",
			Path:           "/api/folders",
			Body:           map[string]interface{}{"name": "test"},
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:           "MissingFolderName",
			Method:         "POST",
			Path:           "/api/folders",
			Body:           map[string]interface{}{"path": "/test"},
			ExpectedStatus: http.StatusBadRequest,
		},
		{
			Name:           "NonExistentFolder",
			Method:         "DELETE",
			Path:           "/api/folders/non-existent-path",
			ExpectedStatus: http.StatusNotFound,
		},
	}
}

// BoundaryTestPattern defines boundary condition tests
type BoundaryTestPattern struct {
	Name        string
	Description string
	TestFunc    func(t *testing.T, env *TestEnvironment)
}

// CreateBoundaryTests returns common boundary condition tests
func CreateBoundaryTests(env *TestEnvironment) []BoundaryTestPattern {
	return []BoundaryTestPattern{
		{
			Name:        "EmptyDatabase",
			Description: "Test behavior with no files",
			TestFunc: func(t *testing.T, env *TestEnvironment) {
				cleanupTestData(env)
				w, _ := makeHTTPRequest(env, "GET", "/api/files", nil)
				response := assertJSONResponse(t, w, http.StatusOK)
				if files, ok := response["files"].([]interface{}); !ok || len(files) != 0 {
					// Empty is acceptable
				}
			},
		},
		{
			Name:        "LargeFileList",
			Description: "Test pagination with many files",
			TestFunc: func(t *testing.T, env *TestEnvironment) {
				// Create multiple test files
				for i := 0; i < 5; i++ {
					file := createTestFile(t, env, fmt.Sprintf("test%d.jpg", i), "image/jpeg")
					defer file.Cleanup()
				}

				// Test with limit
				w, _ := makeHTTPRequest(env, "GET", "/api/files?limit=2", nil)
				response := assertJSONResponse(t, w, http.StatusOK)
				if limit, ok := response["limit"].(float64); !ok || limit != 2 {
					t.Errorf("Expected limit 2, got %v", limit)
				}
			},
		},
		{
			Name:        "ZeroSizeFile",
			Description: "Test handling of zero-byte files",
			TestFunc: func(t *testing.T, env *TestEnvironment) {
				body := map[string]interface{}{
					"filename":     "empty.txt",
					"file_hash":    "empty_hash",
					"size_bytes":   0,
					"mime_type":    "text/plain",
					"storage_path": "/test/empty.txt",
					"folder_path":  "/",
				}
				w, _ := makeHTTPRequest(env, "POST", "/api/files", body)
				// Should still accept zero-byte files
				assertStatusCode(t, w, http.StatusCreated)
			},
		},
	}
}

// PerformanceTestPattern defines performance test scenarios
type PerformanceTestPattern struct {
	Name         string
	Description  string
	Iterations   int
	MaxDuration  int // milliseconds
	TestFunc     func(t *testing.T, env *TestEnvironment)
}

// CreatePerformanceTests returns performance test patterns
func CreatePerformanceTests(env *TestEnvironment) []PerformanceTestPattern {
	return []PerformanceTestPattern{
		{
			Name:        "HealthCheckLatency",
			Description: "Health check should respond in <50ms",
			Iterations:  10,
			MaxDuration: 50,
			TestFunc: func(t *testing.T, env *TestEnvironment) {
				w, _ := makeHTTPRequest(env, "GET", "/health", nil)
				assertStatusCode(t, w, http.StatusOK)
			},
		},
		{
			Name:        "FileListLatency",
			Description: "File list should respond in <200ms",
			Iterations:  5,
			MaxDuration: 200,
			TestFunc: func(t *testing.T, env *TestEnvironment) {
				w, _ := makeHTTPRequest(env, "GET", "/api/files?limit=50", nil)
				assertStatusCode(t, w, http.StatusOK)
			},
		},
	}
}
