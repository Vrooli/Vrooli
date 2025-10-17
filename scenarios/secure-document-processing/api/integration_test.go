package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestIntegrationEndToEnd tests complete end-to-end scenarios
func TestIntegrationEndToEnd(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	envCleanup := setupTestEnv()
	defer envCleanup()

	router := createTestServer()

	t.Run("CompleteWorkflow", func(t *testing.T) {
		// 1. Check health
		healthReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}
		w, err := makeHTTPRequest(healthReq, router.ServeHTTP)
		if err != nil {
			t.Fatalf("Health check failed: %v", err)
		}

		var healthResp HealthResponse
		assertJSONResponse(t, w, http.StatusOK, &healthResp)

		if healthResp.Status != "healthy" {
			t.Errorf("Expected healthy status, got %s", healthResp.Status)
		}

		// 2. List documents
		docsReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/documents",
		}
		w, err = makeHTTPRequest(docsReq, router.ServeHTTP)
		if err != nil {
			t.Fatalf("Documents request failed: %v", err)
		}

		var documents []Document
		assertJSONResponse(t, w, http.StatusOK, &documents)

		if len(documents) == 0 {
			t.Error("Expected documents in response")
		}

		// 3. List jobs
		jobsReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/jobs",
		}
		w, err = makeHTTPRequest(jobsReq, router.ServeHTTP)
		if err != nil {
			t.Fatalf("Jobs request failed: %v", err)
		}

		var jobs []ProcessingJob
		assertJSONResponse(t, w, http.StatusOK, &jobs)

		if len(jobs) == 0 {
			t.Error("Expected jobs in response")
		}

		// 4. List workflows
		wfReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/workflows",
		}
		w, err = makeHTTPRequest(wfReq, router.ServeHTTP)
		if err != nil {
			t.Fatalf("Workflows request failed: %v", err)
		}

		var workflows []Workflow
		assertJSONResponse(t, w, http.StatusOK, &workflows)

		if len(workflows) == 0 {
			t.Error("Expected workflows in response")
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test 404 on non-existent endpoint
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/nonexistent",
		}
		w, err := makeHTTPRequest(req, router.ServeHTTP)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected 404, got %d", w.Code)
		}
	})

	t.Run("MultipleEndpointsSequential", func(t *testing.T) {
		endpoints := []string{
			"/health",
			"/api/documents",
			"/api/jobs",
			"/api/workflows",
		}

		for _, endpoint := range endpoints {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   endpoint,
			}
			w, err := makeHTTPRequest(req, router.ServeHTTP)
			if err != nil {
				t.Fatalf("Request to %s failed: %v", endpoint, err)
			}

			if w.Code != http.StatusOK {
				t.Errorf("Expected 200 for %s, got %d", endpoint, w.Code)
			}
		}
	})
}

// TestErrorResponses tests error response handling
func TestErrorResponses(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := createTestServer()

	testCases := []struct {
		name           string
		path           string
		method         string
		expectedStatus int
	}{
		{"NotFound", "/api/invalid", "GET", http.StatusNotFound},
		{"RootNotFound", "/", "GET", http.StatusNotFound},
		{"InvalidPath", "/invalid/path/here", "GET", http.StatusNotFound},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method: tc.method,
				Path:   tc.path,
			}
			w, err := makeHTTPRequest(req, router.ServeHTTP)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			assertErrorResponse(t, w, tc.expectedStatus)
		})
	}
}

// TestJSONSerialization tests JSON encoding/decoding
func TestJSONSerialization(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("AllStructures", func(t *testing.T) {
		structures := []interface{}{
			HealthResponse{
				Status:    "healthy",
				Version:   "1.0.0",
				Services:  map[string]string{"test": "healthy"},
			},
			Document{
				ID:       "test-id",
				Filename: "test.pdf",
				Status:   "processed",
			},
			ProcessingJob{
				ID:        "job-id",
				JobName:   "Test Job",
				Status:    "completed",
				Documents: []string{"doc1"},
			},
			Workflow{
				ID:          "wf-id",
				Name:        "Test Workflow",
				Description: "Description",
				Type:        "test",
			},
		}

		for _, structure := range structures {
			data, err := json.Marshal(structure)
			if err != nil {
				t.Errorf("Failed to marshal structure: %v", err)
			}

			// Verify it can be unmarshaled
			var result map[string]interface{}
			if err := json.Unmarshal(data, &result); err != nil {
				t.Errorf("Failed to unmarshal structure: %v", err)
			}
		}
	})
}

// TestServiceStatusIntegration tests service status checking with mocks
func TestServiceStatusIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("WithMockServices", func(t *testing.T) {
		// Create mock services
		healthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer healthyServer.Close()

		unhealthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer unhealthyServer.Close()

		// Test healthy service
		status := getServiceStatus(healthyServer.URL)
		if status != "healthy" {
			t.Errorf("Expected 'healthy', got '%s'", status)
		}

		// Test unhealthy service
		status = getServiceStatus(unhealthyServer.URL)
		if status != "unhealthy" {
			t.Errorf("Expected 'unhealthy', got '%s'", status)
		}
	})
}

// TestHelperFunctions tests all helper functions comprehensively
func TestHelperFunctions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SetupTestEnv", func(t *testing.T) {
		cleanup := setupTestEnv()
		defer cleanup()

		// Verify environment was set
		if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
			t.Error("Expected VROOLI_LIFECYCLE_MANAGED to be set")
		}

		if os.Getenv("API_PORT") != "9999" {
			t.Error("Expected API_PORT to be set to 9999")
		}
	})

	t.Run("CreateTestServer", func(t *testing.T) {
		router := createTestServer()
		if router == nil {
			t.Fatal("Expected router to be created")
		}

		// Verify routes are registered
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected health endpoint to respond with 200, got %d", w.Code)
		}
	})
}

// TestEdgeCasesComprehensive tests additional edge cases
func TestEdgeCasesComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("EmptyHeaderRequest", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/health",
			Headers: map[string]string{},
		}

		w, err := makeHTTPRequest(req, healthHandler)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})

	t.Run("NilBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
			Body:   nil,
		}

		w, err := makeHTTPRequest(req, healthHandler)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})
}

// TestPatternBuilders tests the test pattern builders
func TestPatternBuilders(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("TestScenarioBuilder", func(t *testing.T) {
		builder := NewTestScenarioBuilder()

		scenarios := builder.
			AddInvalidMethod("/api/documents", "GET").
			AddMissingContentType("/api/documents", "POST").
			AddEmptyResponse("/api/documents", "GET").
			Build()

		if len(scenarios) != 3 {
			t.Errorf("Expected 3 scenarios, got %d", len(scenarios))
		}

		// Verify each scenario is properly configured
		for _, scenario := range scenarios {
			if scenario.Name == "" {
				t.Error("Scenario missing name")
			}
			if scenario.Description == "" {
				t.Error("Scenario missing description")
			}
			if scenario.ExpectedStatus == 0 {
				t.Error("Scenario missing expected status")
			}
		}
	})

	t.Run("RunErrorTests", func(t *testing.T) {
		suite := HandlerTestSuite{
			HandlerName: "TestHandler",
			Handler:     healthHandler,
			BaseURL:     "/health",
		}

		patterns := []ErrorTestPattern{
			{
				Name:           "TestPattern",
				Description:    "Test pattern",
				ExpectedStatus: http.StatusOK,
				Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
					return HTTPTestRequest{
						Method: "GET",
						Path:   "/health",
					}
				},
			},
		}

		suite.RunErrorTests(t, patterns)
	})
}
