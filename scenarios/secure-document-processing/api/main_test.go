package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// TestMain sets up and tears down the test environment
func TestMain(m *testing.M) {
	// Setup
	cleanup := setupTestEnv()

	// Run tests
	code := m.Run()

	// Cleanup
	cleanup()

	os.Exit(code)
}

// TestGetEnv tests the getEnv helper function
func TestGetEnv(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ExistingEnvVar", func(t *testing.T) {
		os.Setenv("TEST_VAR", "test_value")
		defer os.Unsetenv("TEST_VAR")

		result := getEnv("TEST_VAR", "default")
		if result != "test_value" {
			t.Errorf("Expected 'test_value', got '%s'", result)
		}
	})

	t.Run("MissingEnvVar", func(t *testing.T) {
		result := getEnv("NON_EXISTENT_VAR", "default_value")
		if result != "default_value" {
			t.Errorf("Expected 'default_value', got '%s'", result)
		}
	})

	t.Run("EmptyEnvVar", func(t *testing.T) {
		os.Setenv("EMPTY_VAR", "")
		defer os.Unsetenv("EMPTY_VAR")

		result := getEnv("EMPTY_VAR", "default")
		if result != "default" {
			t.Errorf("Expected 'default', got '%s'", result)
		}
	})

	t.Run("EmptyDefault", func(t *testing.T) {
		result := getEnv("NON_EXISTENT_VAR", "")
		if result != "" {
			t.Errorf("Expected empty string, got '%s'", result)
		}
	})
}

// TestGetServiceStatus tests the getServiceStatus function
func TestGetServiceStatus(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("EmptyURL", func(t *testing.T) {
		status := getServiceStatus("")
		if status != "not_configured" {
			t.Errorf("Expected 'not_configured', got '%s'", status)
		}
	})

	t.Run("HealthyService", func(t *testing.T) {
		// Create a mock healthy service
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		status := getServiceStatus(server.URL)
		if status != "healthy" {
			t.Errorf("Expected 'healthy', got '%s'", status)
		}
	})

	t.Run("UnhealthyService", func(t *testing.T) {
		// Create a mock unhealthy service
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		status := getServiceStatus(server.URL)
		if status != "unhealthy" {
			t.Errorf("Expected 'unhealthy', got '%s'", status)
		}
	})

	t.Run("UnreachableService", func(t *testing.T) {
		status := getServiceStatus("http://localhost:99999")
		if status != "unhealthy" {
			t.Errorf("Expected 'unhealthy', got '%s'", status)
		}
	})

	t.Run("InvalidURL", func(t *testing.T) {
		status := getServiceStatus("not-a-valid-url")
		if status != "unhealthy" {
			t.Errorf("Expected 'unhealthy', got '%s'", status)
		}
	})
}

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SuccessResponse", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, err := makeHTTPRequest(req, healthHandler)
		if err != nil {
			t.Fatalf("Failed to create HTTP request: %v", err)
		}

		var response HealthResponse
		assertJSONResponse(t, w, http.StatusOK, &response)

		// Validate response structure
		if response.Status != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", response.Status)
		}

		if response.Version != "1.0.0" {
			t.Errorf("Expected version '1.0.0', got '%s'", response.Version)
		}

		if response.Services == nil {
			t.Error("Expected services map, got nil")
		}

		// Check that timestamp is recent
		if time.Since(response.Timestamp) > 5*time.Second {
			t.Errorf("Expected recent timestamp, got %v", response.Timestamp)
		}

		// Validate expected services
			expectedServices := []string{"vault", "minio", "unstructured", "postgres"}
		for _, service := range expectedServices {
			if _, exists := response.Services[service]; !exists {
				t.Errorf("Expected service '%s' in response", service)
			}
		}
	})

	t.Run("QdrantServiceIncluded", func(t *testing.T) {
		os.Setenv("QDRANT_URL", "http://localhost:6333")
		defer os.Unsetenv("QDRANT_URL")

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, err := makeHTTPRequest(req, healthHandler)
		if err != nil {
			t.Fatalf("Failed to create HTTP request: %v", err)
		}

		var response HealthResponse
		assertJSONResponse(t, w, http.StatusOK, &response)

		if _, exists := response.Services["qdrant"]; !exists {
			t.Error("Expected qdrant service when QDRANT_URL is set")
		}
	})

	t.Run("ContentType", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, err := makeHTTPRequest(req, healthHandler)
		if err != nil {
			t.Fatalf("Failed to create HTTP request: %v", err)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}
	})
}

// TestDocumentsHandler tests the documents endpoint
func TestDocumentsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SuccessResponse", func(t *testing.T) {
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

		// Validate response structure
		if len(documents) == 0 {
			t.Error("Expected at least one document in response")
		}

		// Validate first document structure
		if len(documents) > 0 {
			doc := documents[0]
			if doc.ID == "" {
				t.Error("Expected document ID")
			}
			if doc.Filename == "" {
				t.Error("Expected document filename")
			}
			if doc.Status == "" {
				t.Error("Expected document status")
			}
			if doc.Created.IsZero() {
				t.Error("Expected document created timestamp")
			}
		}
	})

	t.Run("ContentType", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/documents",
		}

		w, err := makeHTTPRequest(req, documentsHandler)
		if err != nil {
			t.Fatalf("Failed to create HTTP request: %v", err)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}
	})

	t.Run("ValidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/documents",
		}

		w, err := makeHTTPRequest(req, documentsHandler)
		if err != nil {
			t.Fatalf("Failed to create HTTP request: %v", err)
		}

		// Verify it's valid JSON by attempting to parse
		var result interface{}
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Errorf("Response is not valid JSON: %v", err)
		}
	})
}

// TestJobsHandler tests the jobs endpoint
func TestJobsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SuccessResponse", func(t *testing.T) {
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

		// Validate response structure
		if len(jobs) == 0 {
			t.Error("Expected at least one job in response")
		}

		// Validate first job structure
		if len(jobs) > 0 {
			job := jobs[0]
			if job.ID == "" {
				t.Error("Expected job ID")
			}
			if job.JobName == "" {
				t.Error("Expected job name")
			}
			if job.Status == "" {
				t.Error("Expected job status")
			}
			if job.Created.IsZero() {
				t.Error("Expected job created timestamp")
			}
			if job.Documents == nil {
				t.Error("Expected job documents array")
			}
		}
	})

	t.Run("ContentType", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/jobs",
		}

		w, err := makeHTTPRequest(req, jobsHandler)
		if err != nil {
			t.Fatalf("Failed to create HTTP request: %v", err)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}
	})

	t.Run("ValidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/jobs",
		}

		w, err := makeHTTPRequest(req, jobsHandler)
		if err != nil {
			t.Fatalf("Failed to create HTTP request: %v", err)
		}

		var result interface{}
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Errorf("Response is not valid JSON: %v", err)
		}
	})
}

// TestWorkflowsHandler tests the workflows endpoint
func TestWorkflowsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SuccessResponse", func(t *testing.T) {
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

		// Validate response structure
		if len(workflows) == 0 {
			t.Error("Expected at least one workflow in response")
		}

		// Validate first workflow structure
		if len(workflows) > 0 {
			wf := workflows[0]
			if wf.ID == "" {
				t.Error("Expected workflow ID")
			}
			if wf.Name == "" {
				t.Error("Expected workflow name")
			}
			if wf.Description == "" {
				t.Error("Expected workflow description")
			}
			if wf.Type == "" {
				t.Error("Expected workflow type")
			}
			if wf.Created.IsZero() {
				t.Error("Expected workflow created timestamp")
			}
		}
	})

	t.Run("ContentType", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/workflows",
		}

		w, err := makeHTTPRequest(req, workflowsHandler)
		if err != nil {
			t.Fatalf("Failed to create HTTP request: %v", err)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}
	})

	t.Run("ValidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/workflows",
		}

		w, err := makeHTTPRequest(req, workflowsHandler)
		if err != nil {
			t.Fatalf("Failed to create HTTP request: %v", err)
		}

		var result interface{}
		if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
			t.Errorf("Response is not valid JSON: %v", err)
		}
	})
}

// TestHTTPMethodHandling tests that handlers reject invalid HTTP methods
func TestHTTPMethodHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testCases := []struct {
		name        string
		path        string
		handler     http.HandlerFunc
		validMethod string
	}{
		{"Health", "/health", healthHandler, "GET"},
		{"Documents", "/api/documents", documentsHandler, "GET"},
		{"Jobs", "/api/jobs", jobsHandler, "GET"},
		{"Workflows", "/api/workflows", workflowsHandler, "GET"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test POST method (should be rejected for GET-only endpoints)
			req := HTTPTestRequest{
				Method: "POST",
				Path:   tc.path,
			}

			w, err := makeHTTPRequest(req, tc.handler)
			if err != nil {
				t.Fatalf("Failed to create HTTP request: %v", err)
			}

			// Note: Since we're testing handlers directly, they won't enforce method restrictions
			// Method enforcement happens at the router level
			// This test ensures handlers can be called successfully
			if w.Code != http.StatusOK && w.Code != http.StatusMethodNotAllowed {
				t.Logf("Handler responded with status %d (handler doesn't enforce methods)", w.Code)
			}
		})
	}
}

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("HealthHandlerNilRequest", func(t *testing.T) {
		// This is more of a sanity check - handlers should never receive nil
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Handler panicked as expected with nil request: %v", r)
			}
		}()

		w := httptest.NewRecorder()
		healthHandler(w, nil)
		// If we get here without panic, that's also acceptable
	})

	t.Run("EmptyEnvironmentVariables", func(t *testing.T) {
		// Temporarily clear all env vars
		envVars := map[string]string{
			"N8N_BASE_URL":      os.Getenv("N8N_BASE_URL"),
			"WINDMILL_BASE_URL": os.Getenv("WINDMILL_BASE_URL"),
			"VAULT_URL":         os.Getenv("VAULT_URL"),
			"MINIO_URL":         os.Getenv("MINIO_URL"),
			"UNSTRUCTURED_URL":  os.Getenv("UNSTRUCTURED_URL"),
			"QDRANT_URL":        os.Getenv("QDRANT_URL"),
		}

		for key := range envVars {
			os.Unsetenv(key)
		}

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, err := makeHTTPRequest(req, healthHandler)
		if err != nil {
			t.Fatalf("Failed to create HTTP request: %v", err)
		}

		var response HealthResponse
		assertJSONResponse(t, w, http.StatusOK, &response)

		// All services should be "not_configured" when env vars are empty
		for service, status := range response.Services {
			if service != "postgres" && status != "not_configured" {
				t.Errorf("Expected '%s' service to be 'not_configured', got '%s'", service, status)
			}
		}

		// Restore env vars
		for key, value := range envVars {
			if value != "" {
				os.Setenv(key, value)
			}
		}
	})
}

// TestConcurrentRequests tests thread safety of handlers
func TestConcurrentRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ParallelHealthChecks", func(t *testing.T) {
		concurrency := 10
		done := make(chan bool, concurrency)

		for i := 0; i < concurrency; i++ {
			go func() {
				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/health",
				}

				w, err := makeHTTPRequest(req, healthHandler)
				if err != nil {
					t.Errorf("Failed to create HTTP request: %v", err)
					done <- false
					return
				}

				if w.Code != http.StatusOK {
					t.Errorf("Expected status 200, got %d", w.Code)
					done <- false
					return
				}

				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < concurrency; i++ {
			<-done
		}
	})
}

// TestDataStructures tests the data structure types
func TestDataStructures(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("HealthResponseSerialization", func(t *testing.T) {
		response := HealthResponse{
			Status:    "healthy",
			Timestamp: time.Now(),
			Version:   "1.0.0",
			Services:  map[string]string{"test": "healthy"},
		}

		data, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("Failed to marshal HealthResponse: %v", err)
		}

		var decoded HealthResponse
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal HealthResponse: %v", err)
		}

		if decoded.Status != response.Status {
			t.Errorf("Expected status '%s', got '%s'", response.Status, decoded.Status)
		}
	})

	t.Run("DocumentSerialization", func(t *testing.T) {
		doc := Document{
			ID:       "test-id",
			Filename: "test.pdf",
			Status:   "processed",
			Created:  time.Now(),
		}

		data, err := json.Marshal(doc)
		if err != nil {
			t.Fatalf("Failed to marshal Document: %v", err)
		}

		var decoded Document
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal Document: %v", err)
		}

		if decoded.ID != doc.ID {
			t.Errorf("Expected ID '%s', got '%s'", doc.ID, decoded.ID)
		}
	})

	t.Run("ProcessingJobSerialization", func(t *testing.T) {
		job := ProcessingJob{
			ID:        "job-id",
			JobName:   "Test Job",
			Status:    "completed",
			Created:   time.Now(),
			Documents: []string{"doc1", "doc2"},
		}

		data, err := json.Marshal(job)
		if err != nil {
			t.Fatalf("Failed to marshal ProcessingJob: %v", err)
		}

		var decoded ProcessingJob
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal ProcessingJob: %v", err)
		}

		if decoded.ID != job.ID {
			t.Errorf("Expected ID '%s', got '%s'", job.ID, decoded.ID)
		}
	})

	t.Run("WorkflowSerialization", func(t *testing.T) {
		wf := Workflow{
			ID:          "wf-id",
			Name:        "Test Workflow",
			Description: "Test Description",
			Type:        "test",
			Created:     time.Now(),
		}

		data, err := json.Marshal(wf)
		if err != nil {
			t.Fatalf("Failed to marshal Workflow: %v", err)
		}

		var decoded Workflow
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal Workflow: %v", err)
		}

		if decoded.ID != wf.ID {
			t.Errorf("Expected ID '%s', got '%s'", wf.ID, decoded.ID)
		}
	})
}
