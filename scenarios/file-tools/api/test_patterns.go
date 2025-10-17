package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, setupData interface{}) *HTTPTestRequest
	Validate       func(t *testing.T, recorder *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(setupData interface{})
}

// TestScenarioBuilder fluent interface for building test scenarios
type TestScenarioBuilder struct {
	scenarios []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		scenarios: make([]ErrorTestPattern, 0),
	}
}

// AddInvalidJSON adds invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(urlPath string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
		},
	})
	return b
}

// AddMissingAuth adds missing authentication test pattern
func (b *TestScenarioBuilder) AddMissingAuth(urlPath string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "MissingAuth",
		Description:    "Test handler without authentication token",
		ExpectedStatus: http.StatusUnauthorized,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				Headers: map[string]string{
					// No Authorization header
				},
			}
		},
	})
	return b
}

// AddInvalidAuth adds invalid authentication test pattern
func (b *TestScenarioBuilder) AddInvalidAuth(urlPath string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidAuth",
		Description:    "Test handler with invalid authentication token",
		ExpectedStatus: http.StatusUnauthorized,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Headers: map[string]string{
					"Authorization": "Bearer invalid-token",
				},
			}
		},
	})
	return b
}

// AddMissingFile adds missing file test pattern
func (b *TestScenarioBuilder) AddMissingFile(urlPath string, method string, filePath string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "MissingFile",
		Description:    "Test handler with non-existent file",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			body := map[string]interface{}{
				"file_path": filePath,
			}
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   body,
			}
		},
	})
	return b
}

// AddEmptyRequest adds empty request body test pattern
func (b *TestScenarioBuilder) AddEmptyRequest(urlPath string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "EmptyRequest",
		Description:    "Test handler with empty request body",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   map[string]interface{}{},
			}
		},
	})
	return b
}

// AddInvalidFormat adds invalid format test pattern
func (b *TestScenarioBuilder) AddInvalidFormat(urlPath string, method string, formatField string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidFormat",
		Description:    fmt.Sprintf("Test handler with invalid %s format", formatField),
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			body := map[string]interface{}{
				formatField: "invalid-format-xyz",
			}
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   body,
			}
		},
	})
	return b
}

// AddInvalidPath adds invalid path test pattern
func (b *TestScenarioBuilder) AddInvalidPath(urlPath string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidPath",
		Description:    "Test handler with invalid file path",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
			body := map[string]interface{}{
				"file_path": "../../../etc/passwd", // Path traversal attempt
			}
			return &HTTPTestRequest{
				Method: method,
				Path:   urlPath,
				Body:   body,
			}
		},
	})
	return b
}

// Build returns the built scenarios
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.scenarios
}

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name        string
	Description string
	MaxDuration time.Duration
	Setup       func(t *testing.T) interface{}
	Execute     func(t *testing.T, setupData interface{}) time.Duration
	Validate    func(t *testing.T, duration time.Duration, setupData interface{})
	Cleanup     func(setupData interface{})
}

// PerformanceTestBuilder fluent interface for building performance tests
type PerformanceTestBuilder struct {
	tests []PerformanceTestPattern
}

// NewPerformanceTestBuilder creates a new performance test builder
func NewPerformanceTestBuilder() *PerformanceTestBuilder {
	return &PerformanceTestBuilder{
		tests: make([]PerformanceTestPattern, 0),
	}
}

// AddCompressionTest adds compression performance test
func (b *PerformanceTestBuilder) AddCompressionTest(name string, fileCount int, fileSize int64, maxDuration time.Duration) *PerformanceTestBuilder {
	b.tests = append(b.tests, PerformanceTestPattern{
		Name:        name,
		Description: fmt.Sprintf("Compress %d files of %d bytes each", fileCount, fileSize),
		MaxDuration: maxDuration,
		Setup: func(t *testing.T) interface{} {
			env := setupTestDirectory(t)
			files := make(map[string]string)

			// Create test files
			content := generateTestContent(int(fileSize))
			for i := 0; i < fileCount; i++ {
				files[fmt.Sprintf("file%d.txt", i)] = content
			}

			return map[string]interface{}{
				"env":   env,
				"files": createTestFiles(t, env.TempDir, files),
			}
		},
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			start := time.Now()
			// Compression logic would go here
			return time.Since(start)
		},
		Validate: func(t *testing.T, duration time.Duration, setupData interface{}) {
			// Validation logic
		},
		Cleanup: func(setupData interface{}) {
			if data, ok := setupData.(map[string]interface{}); ok {
				if env, ok := data["env"].(*TestEnvironment); ok {
					env.Cleanup()
				}
			}
		},
	})
	return b
}

// Build returns the built performance tests
func (b *PerformanceTestBuilder) Build() []PerformanceTestPattern {
	return b.tests
}

// generateTestContent generates test content of specified size
func generateTestContent(size int) string {
	if size <= 0 {
		return ""
	}

	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	content := make([]byte, size)
	for i := range content {
		content[i] = chars[i%len(chars)]
	}
	return string(content)
}

// Common test patterns for file operations

// compressTestPatterns returns common compression test error patterns
func compressTestPatterns(baseURL string) []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON(baseURL, "POST").
		AddMissingAuth(baseURL, "POST").
		AddInvalidAuth(baseURL, "POST").
		AddEmptyRequest(baseURL, "POST").
		AddInvalidFormat(baseURL, "POST", "archive_format").
		Build()
}

// extractTestPatterns returns common extraction test error patterns
func extractTestPatterns(baseURL string) []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON(baseURL, "POST").
		AddMissingAuth(baseURL, "POST").
		AddInvalidAuth(baseURL, "POST").
		AddEmptyRequest(baseURL, "POST").
		AddMissingFile(baseURL, "POST", "/nonexistent/archive.zip").
		Build()
}

// fileOperationTestPatterns returns common file operation test error patterns
func fileOperationTestPatterns(baseURL string) []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON(baseURL, "POST").
		AddMissingAuth(baseURL, "POST").
		AddInvalidAuth(baseURL, "POST").
		AddEmptyRequest(baseURL, "POST").
		AddInvalidPath(baseURL, "POST").
		Build()
}

// metadataTestPatterns returns common metadata test error patterns
func metadataTestPatterns(baseURL string) []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddMissingAuth(baseURL, "GET").
		AddInvalidAuth(baseURL, "GET").
		AddMissingFile(baseURL, "GET", "/nonexistent/file.txt").
		Build()
}

// checksumTestPatterns returns common checksum test error patterns
func checksumTestPatterns(baseURL string) []ErrorTestPattern {
	return NewTestScenarioBuilder().
		AddInvalidJSON(baseURL, "POST").
		AddMissingAuth(baseURL, "POST").
		AddInvalidAuth(baseURL, "POST").
		AddEmptyRequest(baseURL, "POST").
		AddMissingFile(baseURL, "POST", "/nonexistent/file.txt").
		Build()
}
