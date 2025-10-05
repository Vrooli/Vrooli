package main

import (
	"net/http"
	"testing"
)

// TestHandlerTestSuite tests the handler test suite framework
func TestHandlerTestSuite(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("HealthHandlerSuite", func(t *testing.T) {
		suite := HandlerTestSuite{
			HandlerName: "Health",
			Handler:     healthHandler,
			BaseURL:     "/health",
		}

		scenarios := NewTestScenarioBuilder().
			AddScenario(TestScenario{
				Name:        "BasicHealthCheck",
				Description: "Basic health check request",
				Request: HTTPTestRequest{
					Method: "GET",
					Path:   "/health",
				},
				ExpectedStatus: http.StatusOK,
			}).
			Build()

		suite.RunTests(t, scenarios)
	})

	t.Run("DocumentsHandlerSuite", func(t *testing.T) {
		suite := HandlerTestSuite{
			HandlerName: "Documents",
			Handler:     documentsHandler,
			BaseURL:     "/api/documents",
		}

		scenarios := NewTestScenarioBuilder().
			AddScenario(TestScenario{
				Name:        "BasicDocumentsList",
				Description: "List all documents",
				Request: HTTPTestRequest{
					Method: "GET",
					Path:   "/api/documents",
				},
				ExpectedStatus: http.StatusOK,
			}).
			Build()

		suite.RunTests(t, scenarios)
	})

	t.Run("JobsHandlerSuite", func(t *testing.T) {
		suite := HandlerTestSuite{
			HandlerName: "Jobs",
			Handler:     jobsHandler,
			BaseURL:     "/api/jobs",
		}

		scenarios := NewTestScenarioBuilder().
			AddScenario(TestScenario{
				Name:        "BasicJobsList",
				Description: "List all jobs",
				Request: HTTPTestRequest{
					Method: "GET",
					Path:   "/api/jobs",
				},
				ExpectedStatus: http.StatusOK,
			}).
			Build()

		suite.RunTests(t, scenarios)
	})

	t.Run("WorkflowsHandlerSuite", func(t *testing.T) {
		suite := HandlerTestSuite{
			HandlerName: "Workflows",
			Handler:     workflowsHandler,
			BaseURL:     "/api/workflows",
		}

		scenarios := NewTestScenarioBuilder().
			AddScenario(TestScenario{
				Name:        "BasicWorkflowsList",
				Description: "List all workflows",
				Request: HTTPTestRequest{
					Method: "GET",
					Path:   "/api/workflows",
				},
				ExpectedStatus: http.StatusOK,
			}).
			Build()

		suite.RunTests(t, scenarios)
	})
}

// TestRouterIntegration tests the complete router setup
func TestRouterIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := createTestServer()

	testCases := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{"HealthEndpoint", "GET", "/health", http.StatusOK},
		{"DocumentsEndpoint", "GET", "/api/documents", http.StatusOK},
		{"JobsEndpoint", "GET", "/api/jobs", http.StatusOK},
		{"WorkflowsEndpoint", "GET", "/api/workflows", http.StatusOK},
		{"InvalidEndpoint", "GET", "/api/invalid", http.StatusNotFound},
		{"RootEndpoint", "GET", "/", http.StatusNotFound},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method: tc.method,
				Path:   tc.path,
			}

			w, err := makeHTTPRequest(req, router.ServeHTTP)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d for %s %s",
					tc.expectedStatus, w.Code, tc.method, tc.path)
			}
		})
	}
}

// TestHandlerResponseHeaders tests that handlers set appropriate headers
func TestHandlerResponseHeaders(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testCases := []struct {
		name    string
		handler http.HandlerFunc
		path    string
	}{
		{"Health", healthHandler, "/health"},
		{"Documents", documentsHandler, "/api/documents"},
		{"Jobs", jobsHandler, "/api/jobs"},
		{"Workflows", workflowsHandler, "/api/workflows"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   tc.path,
			}

			w, err := makeHTTPRequest(req, tc.handler)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
			}
		})
	}
}

// TestHandlerWithCustomHeaders tests handlers with custom request headers
func TestHandlerWithCustomHeaders(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("HealthWithUserAgent", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
			Headers: map[string]string{
				"User-Agent": "TestClient/1.0",
			},
		}

		w, err := makeHTTPRequest(req, healthHandler)
		if err != nil {
			t.Fatalf("Failed to create HTTP request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("DocumentsWithAcceptHeader", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/documents",
			Headers: map[string]string{
				"Accept": "application/json",
			},
		}

		w, err := makeHTTPRequest(req, documentsHandler)
		if err != nil {
			t.Fatalf("Failed to create HTTP request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestHandlerErrorPaths tests various error scenarios
func TestHandlerErrorPaths(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ServiceStatusErrors", func(t *testing.T) {
		// Test with various invalid URLs to ensure getServiceStatus handles them
		testCases := []struct {
			name string
			url  string
		}{
			{"InvalidScheme", "ftp://invalid"},
			{"MalformedURL", "://broken"},
			{"UnreachableHost", "http://0.0.0.0:99999"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				status := getServiceStatus(tc.url)
				if status != "unhealthy" {
					t.Errorf("Expected 'unhealthy' for %s, got '%s'", tc.url, status)
				}
			})
		}
	})
}

// TestHandlerBusinessLogic tests business logic within handlers
func TestHandlerBusinessLogic(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("DocumentsReturnExpectedFormat", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/documents",
		}

		w, err := makeHTTPRequest(req, documentsHandler)
		if err != nil {
			t.Fatalf("Failed to create HTTP request: %v", err)
		}

		var documents []Document
		assertJSONResponse(t, w, http.StatusOK, &documents)

		// Business logic validation
		for _, doc := range documents {
			// Ensure all required fields are present
			if doc.ID == "" || doc.Filename == "" || doc.Status == "" {
				t.Error("Document missing required fields")
			}

			// Validate status values
			validStatuses := map[string]bool{
				"processed":  true,
				"processing": true,
				"pending":    true,
				"error":      true,
			}

			if !validStatuses[doc.Status] {
				t.Errorf("Invalid document status: %s", doc.Status)
			}
		}
	})

	t.Run("JobsContainDocumentReferences", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/jobs",
		}

		w, err := makeHTTPRequest(req, jobsHandler)
		if err != nil {
			t.Fatalf("Failed to create HTTP request: %v", err)
		}

		var jobs []ProcessingJob
		assertJSONResponse(t, w, http.StatusOK, &jobs)

		// Verify jobs reference documents
		for _, job := range jobs {
			if job.Documents == nil {
				t.Error("Job should have documents array (even if empty)")
			}

			// Validate status values
			validStatuses := map[string]bool{
				"completed":  true,
				"processing": true,
				"pending":    true,
				"failed":     true,
			}

			if !validStatuses[job.Status] {
				t.Errorf("Invalid job status: %s", job.Status)
			}
		}
	})

	t.Run("WorkflowsHaveTypes", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/workflows",
		}

		w, err := makeHTTPRequest(req, workflowsHandler)
		if err != nil {
			t.Fatalf("Failed to create HTTP request: %v", err)
		}

		var workflows []Workflow
		assertJSONResponse(t, w, http.StatusOK, &workflows)

		// Verify workflows have proper types
		for _, wf := range workflows {
			if wf.Type == "" {
				t.Error("Workflow should have a type")
			}

			// Validate type values based on mock data
			validTypes := map[string]bool{
				"redaction": true,
				"analysis":  true,
			}

			if !validTypes[wf.Type] {
				t.Logf("Workflow type: %s (may be valid, not in test set)", wf.Type)
			}
		}
	})
}
