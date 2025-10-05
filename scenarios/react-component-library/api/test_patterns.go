package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ErrorTestPattern defines a systematic approach to testing error conditions
type ErrorTestPattern struct {
	Name           string
	Description    string
	ExpectedStatus int
	Setup          func(t *testing.T) interface{}
	Execute        func(t *testing.T, router *gin.Engine, setupData interface{}) *httptest.ResponseRecorder
	Validate       func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{})
	Cleanup        func(setupData interface{})
}

// TestScenarioBuilder provides a fluent interface for building error test scenarios
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
func (b *TestScenarioBuilder) AddInvalidUUID(path string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidUUID",
		Description:    "Test endpoint with invalid UUID format",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *httptest.ResponseRecorder {
			invalidPath := fmt.Sprintf(path, "invalid-uuid")
			return makeHTTPRequest(router, HTTPTestRequest{
				Method: "GET",
				Path:   invalidPath,
			})
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusBadRequest, "Invalid component ID")
		},
	})
	return b
}

// AddNonExistentComponent adds a non-existent component test scenario
func (b *TestScenarioBuilder) AddNonExistentComponent(pathFormat string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "NonExistentComponent",
		Description:    "Test endpoint with non-existent component ID",
		ExpectedStatus: http.StatusNotFound,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *httptest.ResponseRecorder {
			nonExistentID := uuid.New()
			path := fmt.Sprintf(pathFormat, nonExistentID.String())
			return makeHTTPRequest(router, HTTPTestRequest{
				Method: "GET",
				Path:   path,
			})
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			assertErrorResponse(t, w, http.StatusNotFound, "Component not found")
		},
	})
	return b
}

// AddInvalidJSON adds an invalid JSON test scenario
func (b *TestScenarioBuilder) AddInvalidJSON(pathFormat string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "InvalidJSON",
		Description:    "Test endpoint with malformed JSON input",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *httptest.ResponseRecorder {
			req, _ := http.NewRequest(method, pathFormat, nil)
			req.Header.Set("Content-Type", "application/json")
			// Send malformed JSON
			req.Body = http.NoBody

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			return w
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", w.Code)
			}
		},
	})
	return b
}

// AddMissingRequiredFields adds a missing required fields test scenario
func (b *TestScenarioBuilder) AddMissingRequiredFields(path string, method string) *TestScenarioBuilder {
	b.scenarios = append(b.scenarios, ErrorTestPattern{
		Name:           "MissingRequiredFields",
		Description:    "Test endpoint with missing required fields",
		ExpectedStatus: http.StatusBadRequest,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) *httptest.ResponseRecorder {
			// Send empty body
			return makeHTTPRequest(router, HTTPTestRequest{
				Method: method,
				Path:   path,
				Body:   map[string]interface{}{},
			})
		},
		Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
			}
		},
	})
	return b
}

// Build returns all configured test scenarios
func (b *TestScenarioBuilder) Build() []ErrorTestPattern {
	return b.scenarios
}

// PerformanceTestPattern defines performance testing scenarios
type PerformanceTestPattern struct {
	Name            string
	Description     string
	MaxDuration     time.Duration
	RequestCount    int
	ConcurrentUsers int
	Setup           func(t *testing.T) interface{}
	Execute         func(t *testing.T, router *gin.Engine, setupData interface{}) time.Duration
	Validate        func(t *testing.T, duration time.Duration, setupData interface{})
	Cleanup         func(setupData interface{})
}

// PerformanceTestBuilder provides a fluent interface for building performance tests
type PerformanceTestBuilder struct {
	tests []PerformanceTestPattern
}

// NewPerformanceTestBuilder creates a new performance test builder
func NewPerformanceTestBuilder() *PerformanceTestBuilder {
	return &PerformanceTestBuilder{
		tests: make([]PerformanceTestPattern, 0),
	}
}

// AddListComponentsPerformanceTest adds a list components performance test
func (b *PerformanceTestBuilder) AddListComponentsPerformanceTest() *PerformanceTestBuilder {
	b.tests = append(b.tests, PerformanceTestPattern{
		Name:            "ListComponents",
		Description:     "Test list components performance",
		MaxDuration:     500 * time.Millisecond,
		RequestCount:    100,
		ConcurrentUsers: 10,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) time.Duration {
			start := time.Now()

			for i := 0; i < 100; i++ {
				w := makeHTTPRequest(router, HTTPTestRequest{
					Method: "GET",
					Path:   "/api/v1/components",
					QueryParams: map[string]string{
						"limit":  "20",
						"offset": "0",
					},
				})

				if w.Code != http.StatusOK {
					t.Errorf("Request %d failed with status %d", i, w.Code)
				}
			}

			return time.Since(start)
		},
		Validate: func(t *testing.T, duration time.Duration, setupData interface{}) {
			avgDuration := duration / 100
			if avgDuration > 50*time.Millisecond {
				t.Logf("Warning: Average request duration %.2fms exceeds recommended 50ms", avgDuration.Seconds()*1000)
			}
		},
	})
	return b
}

// AddComponentCreationPerformanceTest adds a component creation performance test
func (b *PerformanceTestBuilder) AddComponentCreationPerformanceTest() *PerformanceTestBuilder {
	b.tests = append(b.tests, PerformanceTestPattern{
		Name:            "CreateComponent",
		Description:     "Test component creation performance",
		MaxDuration:     1 * time.Second,
		RequestCount:    50,
		ConcurrentUsers: 5,
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) time.Duration {
			start := time.Now()

			for i := 0; i < 50; i++ {
				componentData := getValidComponentData()
				componentData.Name = fmt.Sprintf("PerfTest%d", i)

				w := makeHTTPRequest(router, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/components",
					Body:   componentData,
				})

				if w.Code != http.StatusCreated {
					t.Errorf("Request %d failed with status %d", i, w.Code)
				}
			}

			return time.Since(start)
		},
		Validate: func(t *testing.T, duration time.Duration, setupData interface{}) {
			avgDuration := duration / 50
			if avgDuration > 100*time.Millisecond {
				t.Logf("Warning: Average creation time %.2fms exceeds recommended 100ms", avgDuration.Seconds()*1000)
			}
		},
	})
	return b
}

// Build returns all configured performance tests
func (b *PerformanceTestBuilder) Build() []PerformanceTestPattern {
	return b.tests
}

// IntegrationTestPattern defines integration testing scenarios
type IntegrationTestPattern struct {
	Name        string
	Description string
	Setup       func(t *testing.T) interface{}
	Execute     func(t *testing.T, router *gin.Engine, setupData interface{}) bool
	Validate    func(t *testing.T, result bool, setupData interface{})
	Cleanup     func(setupData interface{})
}

// IntegrationTestBuilder provides a fluent interface for building integration tests
type IntegrationTestBuilder struct {
	tests []IntegrationTestPattern
}

// NewIntegrationTestBuilder creates a new integration test builder
func NewIntegrationTestBuilder() *IntegrationTestBuilder {
	return &IntegrationTestBuilder{
		tests: make([]IntegrationTestPattern, 0),
	}
}

// AddFullComponentLifecycleTest adds a full component lifecycle integration test
func (b *IntegrationTestBuilder) AddFullComponentLifecycleTest() *IntegrationTestBuilder {
	b.tests = append(b.tests, IntegrationTestPattern{
		Name:        "FullComponentLifecycle",
		Description: "Test complete component lifecycle: create, read, update, delete",
		Execute: func(t *testing.T, router *gin.Engine, setupData interface{}) bool {
			// Create component
			componentData := getValidComponentData()
			w := makeHTTPRequest(router, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/components",
				Body:   componentData,
			})

			if w.Code != http.StatusCreated {
				t.Errorf("Create failed with status %d", w.Code)
				return false
			}

			var createResponse map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &createResponse); err != nil {
				t.Errorf("Failed to parse create response: %v", err)
				return false
			}

			componentID := createResponse["id"].(string)

			// Read component
			w = makeHTTPRequest(router, HTTPTestRequest{
				Method: "GET",
				Path:   fmt.Sprintf("/api/v1/components/%s", componentID),
			})

			if w.Code != http.StatusOK {
				t.Errorf("Read failed with status %d", w.Code)
				return false
			}

			// Update component
			updateData := map[string]interface{}{
				"description": "Updated description",
			}
			w = makeHTTPRequest(router, HTTPTestRequest{
				Method: "PUT",
				Path:   fmt.Sprintf("/api/v1/components/%s", componentID),
				Body:   updateData,
			})

			if w.Code != http.StatusOK {
				t.Errorf("Update failed with status %d", w.Code)
				return false
			}

			// Delete component
			w = makeHTTPRequest(router, HTTPTestRequest{
				Method: "DELETE",
				Path:   fmt.Sprintf("/api/v1/components/%s", componentID),
			})

			if w.Code != http.StatusOK {
				t.Errorf("Delete failed with status %d", w.Code)
				return false
			}

			// Verify deletion
			w = makeHTTPRequest(router, HTTPTestRequest{
				Method: "GET",
				Path:   fmt.Sprintf("/api/v1/components/%s", componentID),
			})

			if w.Code != http.StatusNotFound {
				t.Errorf("Component still exists after deletion")
				return false
			}

			return true
		},
		Validate: func(t *testing.T, result bool, setupData interface{}) {
			if !result {
				t.Error("Full lifecycle test failed")
			}
		},
	})
	return b
}

// Build returns all configured integration tests
func (b *IntegrationTestBuilder) Build() []IntegrationTestPattern {
	return b.tests
}
