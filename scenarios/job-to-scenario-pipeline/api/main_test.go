
package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, httpReq := makeHTTPRequest(req)
		healthHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status":  "healthy",
			"service": "job-to-scenario-pipeline",
		})

		if response != nil {
			if _, exists := response["timestamp"]; !exists {
				t.Error("Expected timestamp field in health response")
			}
		}
	})
}

// TestListJobsHandler tests the list jobs endpoint
func TestListJobsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("ListAllJobs", func(t *testing.T) {
		// Create test jobs in different states
		job1 := setupTestJob(t, "pending")
		defer job1.Cleanup()

		job2 := setupTestJob(t, "evaluated")
		defer job2.Cleanup()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/jobs",
		}

		w, httpReq := makeHTTPRequest(req)
		listJobsHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			jobs, ok := response["jobs"].([]interface{})
			if !ok {
				t.Error("Expected jobs array in response")
			} else if len(jobs) != 2 {
				t.Errorf("Expected 2 jobs, got %d", len(jobs))
			}

			total, ok := response["total"].(float64)
			if !ok || int(total) != 2 {
				t.Errorf("Expected total to be 2, got %v", response["total"])
			}
		}
	})

	t.Run("FilterByState", func(t *testing.T) {
		job1 := setupTestJob(t, "pending")
		defer job1.Cleanup()

		job2 := setupTestJob(t, "evaluated")
		defer job2.Cleanup()

		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/v1/jobs",
			QueryParams: map[string]string{"state": "pending"},
		}

		w, httpReq := makeHTTPRequest(req)
		listJobsHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			jobs, ok := response["jobs"].([]interface{})
			if !ok {
				t.Error("Expected jobs array in response")
			} else if len(jobs) == 0 {
				t.Error("Expected at least one pending job")
			}
		}
	})

	t.Run("EmptyList", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/jobs",
		}

		w, httpReq := makeHTTPRequest(req)
		listJobsHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"total": float64(0),
		})

		if response != nil {
			// jobs might be nil in JSON, which is acceptable
			jobs := response["jobs"]
			if jobs != nil {
				if jobsArr, ok := jobs.([]interface{}); ok && len(jobsArr) != 0 {
					t.Errorf("Expected empty jobs array, got %d jobs", len(jobsArr))
				}
			}
		}
	})
}

// TestImportJobHandler tests the import job endpoint
func TestImportJobHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("ImportManualJob", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/jobs/import",
			Body: ImportRequest{
				Source: "manual",
				Data:   "Build a task management system\nThis should have user authentication and task tracking.",
			},
		}

		w, httpReq := makeHTTPRequest(req)
		importJobHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"state": "pending",
		})

		if response != nil {
			jobID, ok := response["job_id"].(string)
			if !ok || jobID == "" {
				t.Error("Expected job_id in response")
			} else {
				// Clean up created job
				defer deleteJobFile(jobID, "pending")
			}
		}
	})

	t.Run("ImportUpworkJobJSON", func(t *testing.T) {
		upworkData := map[string]interface{}{
			"title":       "Senior Go Developer Needed",
			"description": "We need an experienced Go developer for microservices.",
			"budget": map[string]interface{}{
				"min": 5000.0,
				"max": 10000.0,
			},
			"skills_required": []string{"Go", "Docker", "Kubernetes"},
			"metadata": map[string]interface{}{
				"source_url":       "https://upwork.com/jobs/123",
				"job_type":         "hourly",
				"experience_level": "expert",
			},
		}

		jsonData, _ := json.Marshal(upworkData)

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/jobs/import",
			Body: ImportRequest{
				Source: "upwork",
				Data:   string(jsonData),
			},
		}

		w, httpReq := makeHTTPRequest(req)
		importJobHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"state": "pending",
		})

		if response != nil {
			if jobID, ok := response["job_id"].(string); ok && jobID != "" {
				defer deleteJobFile(jobID, "pending")
			}
		}
	})

	t.Run("ImportUpworkJobPlainText", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/jobs/import",
			Body: ImportRequest{
				Source: "upwork",
				Data:   "Looking for a Python developer with FastAPI experience",
			},
		}

		w, httpReq := makeHTTPRequest(req)
		importJobHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"state": "pending",
		})

		if response != nil {
			if jobID, ok := response["job_id"].(string); ok && jobID != "" {
				defer deleteJobFile(jobID, "pending")
			}
		}
	})

	// Error cases using test patterns
	t.Run("ErrorCases", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "importJobHandler",
			Handler:     importJobHandler,
			BaseURL:     "/api/v1/jobs/import",
		}

		patterns := NewTestScenarioBuilder().
			AddInvalidJSON("POST", "/api/v1/jobs/import").
			AddInvalidSource().
			Build()

		suite.RunErrorTests(t, patterns)
	})
}

// TestGetJobHandler tests the get job endpoint
func TestGetJobHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		job := setupTestJob(t, "pending")
		defer job.Cleanup()

		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/jobs/" + job.Job.ID,
			URLVars: map[string]string{"id": job.Job.ID},
		}

		w, httpReq := makeHTTPRequest(req)
		getJobHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"id":    job.Job.ID,
			"title": job.Job.Title,
			"state": "pending",
		})

		if response != nil {
			if _, exists := response["description"]; !exists {
				t.Error("Expected description field in response")
			}
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "getJobHandler",
			Handler:     getJobHandler,
			BaseURL:     "/api/v1/jobs/{id}",
		}

		patterns := NewTestScenarioBuilder().
			AddJobNotFound("/api/v1/jobs/{id}").
			Build()

		suite.RunErrorTests(t, patterns)
	})
}

// TestResearchJobHandler tests the research job endpoint
func TestResearchJobHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		job := setupTestJob(t, "pending")
		defer job.Cleanup()

		req := HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/jobs/" + job.Job.ID + "/research",
			URLVars: map[string]string{"id": job.Job.ID},
		}

		w, httpReq := makeHTTPRequest(req)
		researchJobHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "researching",
			"job_id": job.Job.ID,
		})

		if response == nil {
			t.Error("Expected successful response")
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "researchJobHandler",
			Handler:     researchJobHandler,
			BaseURL:     "/api/v1/jobs/{id}/research",
		}

		patterns := NewTestScenarioBuilder().
			AddJobNotFound("/api/v1/jobs/{id}/research").
			Build()

		suite.RunErrorTests(t, patterns)
	})
}

// TestApproveJobHandler tests the approve job endpoint
func TestApproveJobHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		job := setupTestJobWithResearch(t, "evaluated")
		defer job.Cleanup()

		req := HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/jobs/" + job.Job.ID + "/approve",
			URLVars: map[string]string{"id": job.Job.ID},
		}

		w, httpReq := makeHTTPRequest(req)
		approveJobHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"state": "building",
		})

		if response != nil {
			if _, exists := response["estimated_completion"]; !exists {
				t.Error("Expected estimated_completion in response")
			}
		}
	})

	t.Run("InvalidState", func(t *testing.T) {
		job := setupTestJob(t, "pending")
		defer job.Cleanup()

		req := HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/jobs/" + job.Job.ID + "/approve",
			URLVars: map[string]string{"id": job.Job.ID},
		}

		w, httpReq := makeHTTPRequest(req)
		approveJobHandler(w, httpReq)

		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("ErrorCases", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "approveJobHandler",
			Handler:     approveJobHandler,
			BaseURL:     "/api/v1/jobs/{id}/approve",
		}

		patterns := NewTestScenarioBuilder().
			AddJobNotFound("/api/v1/jobs/{id}/approve").
			Build()

		suite.RunErrorTests(t, patterns)
	})
}

// TestGenerateProposalHandler tests the generate proposal endpoint
func TestGenerateProposalHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("InvalidStateForProposal", func(t *testing.T) {
		job := setupTestJob(t, "pending")
		defer job.Cleanup()

		req := HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/v1/jobs/" + job.Job.ID + "/proposal",
			URLVars: map[string]string{"id": job.Job.ID},
		}

		w, httpReq := makeHTTPRequest(req)
		generateProposalHandler(w, httpReq)

		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("ErrorCases", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "generateProposalHandler",
			Handler:     generateProposalHandler,
			BaseURL:     "/api/v1/jobs/{id}/proposal",
		}

		patterns := NewTestScenarioBuilder().
			AddJobNotFound("/api/v1/jobs/{id}/proposal").
			Build()

		suite.RunErrorTests(t, patterns)
	})
}

// TestHelperFunctions tests utility functions
func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "SingleLine",
			input:    "Test Title",
			expected: "Test Title",
		},
		{
			name:     "MultiLine",
			input:    "First Line Title\nSecond Line\nThird Line",
			expected: "First Line Title",
		},
		{
			name:     "LongTitle",
			input:    "This is a very long title that exceeds one hundred characters and should be truncated to avoid issues with storage",
			expected: "This is a very long title that exceeds one hundred characters and should be truncated to avoid issue...",
		},
		{
			name:     "EmptyString",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTitle(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestParseUpworkData tests Upwork data parsing
func TestParseUpworkData(t *testing.T) {
	t.Run("CompleteData", func(t *testing.T) {
		data := map[string]interface{}{
			"title":       "Senior Developer",
			"description": "Looking for experienced developer",
			"budget": map[string]interface{}{
				"min": 5000.0,
				"max": 10000.0,
			},
			"skills_required": []interface{}{"Go", "Docker"},
			"metadata": map[string]interface{}{
				"source_url":       "https://upwork.com/123",
				"job_type":         "contract",
				"experience_level": "expert",
			},
		}

		job := parseUpworkData("TEST-123", data)

		if job.Title != "Senior Developer" {
			t.Errorf("Expected title 'Senior Developer', got '%s'", job.Title)
		}

		if job.Budget.Min != 5000.0 {
			t.Errorf("Expected min budget 5000, got %f", job.Budget.Min)
		}

		if len(job.SkillsRequired) != 2 {
			t.Errorf("Expected 2 skills, got %d", len(job.SkillsRequired))
		}

		if job.Metadata.JobType != "contract" {
			t.Errorf("Expected job type 'contract', got '%s'", job.Metadata.JobType)
		}
	})

	t.Run("PartialData", func(t *testing.T) {
		data := map[string]interface{}{
			"title": "Basic Job",
		}

		job := parseUpworkData("TEST-456", data)

		if job.Title != "Basic Job" {
			t.Errorf("Expected title 'Basic Job', got '%s'", job.Title)
		}

		if job.Budget.Currency != "USD" {
			t.Errorf("Expected currency USD, got '%s'", job.Budget.Currency)
		}
	})
}

// TestJobFileOperations tests job save/load/move operations
func TestJobFileOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("SaveAndLoad", func(t *testing.T) {
		job := setupTestJob(t, "pending")
		defer job.Cleanup()

		loadedJob, err := loadJob(job.Job.ID)
		if err != nil {
			t.Fatalf("Failed to load job: %v", err)
		}

		if loadedJob.ID != job.Job.ID {
			t.Errorf("Expected ID %s, got %s", job.Job.ID, loadedJob.ID)
		}

		if loadedJob.Title != job.Job.Title {
			t.Errorf("Expected title '%s', got '%s'", job.Job.Title, loadedJob.Title)
		}
	})

	t.Run("MoveJob", func(t *testing.T) {
		job := setupTestJob(t, "pending")
		defer func() {
			deleteJobFile(job.Job.ID, "researching")
		}()

		err := moveJob(job.Job.ID, "pending", "researching")
		if err != nil {
			t.Fatalf("Failed to move job: %v", err)
		}

		loadedJob, err := loadJob(job.Job.ID)
		if err != nil {
			t.Fatalf("Failed to load moved job: %v", err)
		}

		if loadedJob.State != "researching" {
			t.Errorf("Expected state 'researching', got '%s'", loadedJob.State)
		}

		if len(loadedJob.History) < 2 {
			t.Error("Expected history entry for state transition")
		}
	})

	t.Run("LoadNonExistent", func(t *testing.T) {
		_, err := loadJob("NON-EXISTENT")
		if err == nil {
			t.Error("Expected error when loading non-existent job")
		}
	})
}

// TestCORSMiddleware tests CORS middleware
func TestCORSMiddleware(t *testing.T) {
	t.Run("OptionsRequest", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/v1/jobs", nil)
		w := httptest.NewRecorder()

		handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called for OPTIONS request")
		}))

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
		}

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("Expected CORS headers to be set")
		}
	})

	t.Run("RegularRequest", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/jobs", nil)
		w := httptest.NewRecorder()

		handlerCalled := false
		handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
		}))

		handler.ServeHTTP(w, req)

		if !handlerCalled {
			t.Error("Handler should be called for non-OPTIONS request")
		}

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("Expected CORS headers to be set")
		}
	})
}

// TestRouterConfiguration tests the router setup
func TestRouterConfiguration(t *testing.T) {
	t.Run("AllRoutesConfigured", func(t *testing.T) {
		r := mux.NewRouter()

		// Configure routes like in main()
		r.HandleFunc("/health", healthHandler).Methods("GET")
		r.HandleFunc("/api/v1/jobs", listJobsHandler).Methods("GET")
		r.HandleFunc("/api/v1/jobs/import", importJobHandler).Methods("POST")
		r.HandleFunc("/api/v1/jobs/{id}", getJobHandler).Methods("GET")
		r.HandleFunc("/api/v1/jobs/{id}/research", researchJobHandler).Methods("POST")
		r.HandleFunc("/api/v1/jobs/{id}/approve", approveJobHandler).Methods("POST")
		r.HandleFunc("/api/v1/jobs/{id}/proposal", generateProposalHandler).Methods("POST")

		routes := []struct {
			method string
			path   string
		}{
			{"GET", "/health"},
			{"GET", "/api/v1/jobs"},
			{"POST", "/api/v1/jobs/import"},
			{"GET", "/api/v1/jobs/test-id"},
			{"POST", "/api/v1/jobs/test-id/research"},
			{"POST", "/api/v1/jobs/test-id/approve"},
			{"POST", "/api/v1/jobs/test-id/proposal"},
		}

		for _, route := range routes {
			req := httptest.NewRequest(route.method, route.path, nil)
			var match mux.RouteMatch
			if !r.Match(req, &match) {
				t.Errorf("Route %s %s not configured", route.method, route.path)
			}
		}
	})
}
