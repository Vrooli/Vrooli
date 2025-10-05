// +build testing

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
	Setup          func(t *testing.T, env *TestEnvironment) interface{}
	Execute        func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest
	Validate       func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(setupData interface{})
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

// AddInvalidID adds a test for invalid article/feed ID
func (b *TestScenarioBuilder) AddInvalidID(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidID",
		Description:    "Test handler with invalid ID format",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  method,
				Path:    path,
				URLVars: map[string]string{"id": "invalid-id-123"},
			}
		},
	})
	return b
}

// AddNonExistentArticle adds a test for non-existent article
func (b *TestScenarioBuilder) AddNonExistentArticle(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistentArticle",
		Description:    "Test handler with non-existent article ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  "GET",
				Path:    path,
				URLVars: map[string]string{"id": "non-existent-article-999"},
			}
		},
	})
	return b
}

// AddNonExistentFeed adds a test for non-existent feed
func (b *TestScenarioBuilder) AddNonExistentFeed(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "NonExistentFeed",
		Description:    "Test handler with non-existent feed ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method:  method,
				Path:    path,
				URLVars: map[string]string{"id": "999999"},
			}
		},
	})
	return b
}

// AddInvalidJSON adds a test for malformed JSON input
func (b *TestScenarioBuilder) AddInvalidJSON(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test handler with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   `{"invalid": "json"`, // Malformed JSON
			}
		},
	})
	return b
}

// AddMissingRequiredFields adds a test for missing required JSON fields
func (b *TestScenarioBuilder) AddMissingRequiredFields(path string, method string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "MissingRequiredFields",
		Description:    "Test handler with missing required fields",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   map[string]interface{}{}, // Empty body
			}
		},
	})
	return b
}

// AddEmptyTopic adds a test for empty topic parameter
func (b *TestScenarioBuilder) AddEmptyTopic(path string) *TestScenarioBuilder {
	b.patterns = append(b.patterns, ErrorTestPattern{
		Name:           "EmptyTopic",
		Description:    "Test handler with empty topic",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "POST",
				Path:   path,
				Body: map[string]interface{}{
					"topic": "",
				},
			}
		},
	})
	return b
}

// Build returns the configured error test patterns
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.patterns
}

// HandlerTestSuite provides a comprehensive test framework for HTTP handlers
type HandlerTestSuite struct {
	HandlerName string
	BaseURL     string
	Env         *TestEnvironment
}

// NewHandlerTestSuite creates a new handler test suite
func NewHandlerTestSuite(name string, env *TestEnvironment) *HandlerTestSuite {
	return &HandlerTestSuite{
		HandlerName: name,
		Env:         env,
	}
}

// RunErrorTests executes a suite of error condition tests
func (suite *HandlerTestSuite) RunErrorTests(t *testing.T, patterns []ErrorTestPattern) {
	for _, pattern := range patterns {
		t.Run(fmt.Sprintf("%s_%s", suite.HandlerName, pattern.Name), func(t *testing.T) {
			// Setup
			var setupData interface{}
			if pattern.Setup != nil {
				setupData = pattern.Setup(t, suite.Env)
			}

			// Cleanup
			if pattern.Cleanup != nil {
				defer pattern.Cleanup(setupData)
			}

			// Execute
			req := pattern.Execute(t, suite.Env, setupData)
			w, err := makeHTTPRequest(suite.Env, req)
			if err != nil {
				t.Fatalf("Failed to make HTTP request: %v", err)
			}

			// Validate status code
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

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name           string
	Description    string
	MaxDuration    time.Duration
	MinThroughput  int // requests per second
	Setup          func(t *testing.T, env *TestEnvironment) interface{}
	Execute        func(t *testing.T, env *TestEnvironment, setupData interface{}) HTTPTestRequest
	Validate       func(t *testing.T, duration time.Duration, throughput float64)
	Cleanup        func(setupData interface{})
}

// RunPerformanceTest executes a performance test
func RunPerformanceTest(t *testing.T, env *TestEnvironment, pattern PerformanceTestPattern) {
	t.Run(fmt.Sprintf("Performance_%s", pattern.Name), func(t *testing.T) {
		// Setup
		var setupData interface{}
		if pattern.Setup != nil {
			setupData = pattern.Setup(t, env)
		}

		// Cleanup
		if pattern.Cleanup != nil {
			defer pattern.Cleanup(setupData)
		}

		// Execute multiple requests and measure performance
		iterations := 100
		start := time.Now()

		for i := 0; i < iterations; i++ {
			req := pattern.Execute(t, env, setupData)
			w, err := makeHTTPRequest(env, req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if w.Code >= 400 {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		duration := time.Since(start)
		throughput := float64(iterations) / duration.Seconds()

		// Validate performance
		if duration > pattern.MaxDuration {
			t.Errorf("Performance test exceeded max duration: %v > %v", duration, pattern.MaxDuration)
		}

		if pattern.MinThroughput > 0 && throughput < float64(pattern.MinThroughput) {
			t.Errorf("Throughput below minimum: %.2f < %d req/s", throughput, pattern.MinThroughput)
		}

		// Custom validation
		if pattern.Validate != nil {
			pattern.Validate(t, duration, throughput)
		}

		t.Logf("Performance test %s: %d requests in %v (%.2f req/s)",
			pattern.Name, iterations, duration, throughput)
	})
}

// Common test data generators

// GenerateTestArticles creates multiple test articles
func GenerateTestArticles(t *testing.T, env *TestEnvironment, count int, source string) []*TestArticle {
	articles := make([]*TestArticle, count)
	for i := 0; i < count; i++ {
		title := fmt.Sprintf("Test Article %d", i+1)
		articles[i] = createTestArticle(t, env, title, source)
	}
	return articles
}

// GenerateTestFeeds creates multiple test feeds
func GenerateTestFeeds(t *testing.T, env *TestEnvironment, count int) []*TestFeed {
	feeds := make([]*TestFeed, count)
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("Test Feed %d", i+1)
		url := fmt.Sprintf("https://example.com/feed/%d.rss", i+1)
		feeds[i] = createTestFeed(t, env, name, url)
	}
	return feeds
}

// CleanupTestArticles removes all test articles
func CleanupTestArticles(articles []*TestArticle) {
	for _, article := range articles {
		if article != nil && article.Cleanup != nil {
			article.Cleanup()
		}
	}
}

// CleanupTestFeeds removes all test feeds
func CleanupTestFeeds(feeds []*TestFeed) {
	for _, feed := range feeds {
		if feed != nil && feed.Cleanup != nil {
			feed.Cleanup()
		}
	}
}
