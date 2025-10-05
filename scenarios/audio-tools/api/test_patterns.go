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
	Execute        func(t *testing.T, setupData interface{}) HTTPTestRequest
	Validate       func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(setupData interface{})
}

// TestScenarioBuilder helps build systematic error test scenarios
type TestScenarioBuilder struct {
	patterns []ErrorTestPattern
}

// NewTestScenarioBuilder creates a new test scenario builder
func NewTestScenarioBuilder() *TestScenarioBuilder {
	return &TestScenarioBuilder{
		patterns: make([]ErrorTestPattern, 0),
	}
}

// AddInvalidUUID adds an invalid UUID test pattern
func (b *TestScenarioBuilder) AddInvalidUUID(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test handler with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  "GET",
				Path:    urlPath,
				URLVars: map[string]string{"id": "invalid-uuid-format"},
			}
		},
	})
	return b
}

// AddNonExistentResource adds a non-existent resource test pattern
func (b *TestScenarioBuilder) AddNonExistentResource(urlPath, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistentResource",
		Description:    "Test handler with non-existent resource ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  method,
				Path:    urlPath,
				URLVars: map[string]string{"id": generateTestID()},
			}
		},
	})
	return b
}

// AddInvalidJSON adds an invalid JSON test pattern
func (b *TestScenarioBuilder) AddInvalidJSON(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
		},
	})
	return b
}

// AddMissingRequiredField adds a missing required field test pattern
func (b *TestScenarioBuilder) AddMissingRequiredField(urlPath, fieldName string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           fmt.Sprintf("MissingField_%s", fieldName),
		Description:    fmt.Sprintf("Test handler with missing required field: %s", fieldName),
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   urlPath,
				Body:   map[string]interface{}{},
			}
		},
	})
	return b
}

// AddInvalidFileUpload adds an invalid file upload test pattern
func (b *TestScenarioBuilder) AddInvalidFileUpload(urlPath string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidFileUpload",
		Description:    "Test handler with missing file in multipart request",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  "POST",
				Path:    urlPath,
				Headers: map[string]string{"Content-Type": "multipart/form-data"},
				Body:    "",
			}
		},
	})
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

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name           string
	Description    string
	MaxDuration    time.Duration
	Iterations     int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, iteration int, setupData interface{}) time.Duration
	Validate       func(t *testing.T, avgDuration, maxDuration time.Duration, setupData interface{})
	Cleanup        func(setupData interface{})
}

// RunPerformanceTest executes a performance test pattern
func RunPerformanceTest(t *testing.T, pattern PerformanceTestPattern) {
	t.Helper()

	// Setup
	var setupData interface{}
	if pattern.Setup != nil {
		setupData = pattern.Setup(t)
	}

	// Cleanup
	if pattern.Cleanup != nil {
		defer pattern.Cleanup(setupData)
	}

	// Run iterations
	var totalDuration time.Duration
	var maxIterationDuration time.Duration

	for i := 0; i < pattern.Iterations; i++ {
		duration := pattern.Execute(t, i, setupData)
		totalDuration += duration

		if duration > maxIterationDuration {
			maxIterationDuration = duration
		}

		// Check if we're exceeding max duration
		if duration > pattern.MaxDuration {
			t.Errorf("Iteration %d exceeded max duration: %v > %v", i, duration, pattern.MaxDuration)
		}
	}

	avgDuration := totalDuration / time.Duration(pattern.Iterations)

	// Validate results
	if pattern.Validate != nil {
		pattern.Validate(t, avgDuration, maxIterationDuration, setupData)
	}

	// Log performance metrics
	t.Logf("Performance Test: %s", pattern.Name)
	t.Logf("  Iterations: %d", pattern.Iterations)
	t.Logf("  Avg Duration: %v", avgDuration)
	t.Logf("  Max Duration: %v", maxIterationDuration)
	t.Logf("  Total Duration: %v", totalDuration)
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	BaseURL     string
	Patterns    []ErrorTestPattern
}

// NewHandlerTestSuite creates a new handler test suite
func NewHandlerTestSuite(handlerName, baseURL string) *HandlerTestSuite {
	return &HandlerTestSuite{
		HandlerName: handlerName,
		BaseURL:     baseURL,
		Patterns:    make([]ErrorTestPattern, 0),
	}
}

// AddPattern adds a test pattern to the suite
func (s *HandlerTestSuite) AddPattern(pattern ErrorTestPattern) *HandlerTestSuite {
	s.Patterns = append(s.Patterns, pattern)
	return s
}

// AddPatterns adds multiple test patterns to the suite
func (s *HandlerTestSuite) AddPatterns(patterns []ErrorTestPattern) *HandlerTestSuite {
	s.Patterns = append(s.Patterns, patterns...)
	return s
}

// EdgeCaseTestPattern defines edge case testing scenarios
type EdgeCaseTestPattern struct {
	Name        string
	Description string
	Input       interface{}
	Expected    interface{}
	ShouldError bool
}

// BoundaryTestPattern defines boundary condition tests
type BoundaryTestPattern struct {
	Name          string
	Description   string
	MinValue      interface{}
	MaxValue      interface{}
	InvalidBelow  interface{}
	InvalidAbove  interface{}
}

// CreateAudioProcessingPerformanceTest creates a standard performance test for audio processing
func CreateAudioProcessingPerformanceTest(name string, maxDuration time.Duration, operation func(t *testing.T, env *TestEnvironment) time.Duration) PerformanceTestPattern {
	return PerformanceTestPattern{
		Name:        name,
		Description: fmt.Sprintf("Performance test for %s", name),
		MaxDuration: maxDuration,
		Iterations:  5,
		Setup: func(t *testing.T) interface{} {
			return setupTestDirectory(t)
		},
		Execute: func(t *testing.T, iteration int, setupData interface{}) time.Duration {
			env := setupData.(*TestEnvironment)
			return operation(t, env)
		},
		Validate: func(t *testing.T, avgDuration, maxDuration time.Duration, setupData interface{}) {
			if avgDuration > maxDuration {
				t.Errorf("Average duration %v exceeded max %v", avgDuration, maxDuration)
			}
		},
		Cleanup: func(setupData interface{}) {
			if env, ok := setupData.(*TestEnvironment); ok {
				env.Cleanup()
			}
		},
	}
}

// Common test data generators

// GenerateValidAudioFormats returns a list of valid audio formats for testing
func GenerateValidAudioFormats() []string {
	return []string{"mp3", "wav", "flac", "ogg", "aac"}
}

// GenerateInvalidAudioFormats returns a list of invalid audio formats for testing
func GenerateInvalidAudioFormats() []string {
	return []string{"txt", "jpg", "pdf", "exe", "unknown"}
}

// GenerateEdgeCaseFloats returns edge case float values for testing
func GenerateEdgeCaseFloats() []float64 {
	return []float64{0.0, -1.0, 1.0, 0.001, 1000.0, -1000.0}
}

// GenerateInvalidFloats returns invalid float scenarios
func GenerateInvalidFloats() []interface{} {
	return []interface{}{"not-a-number", nil, "", "abc"}
}
