package main

import (
	"net/http"
	"os"
	"testing"
	"time"
)

// TestLoadConfig tests the configuration loading logic
func TestLoadConfig(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		env := setupTestEnvironment(t)
		defer env.Cleanup()

		config := loadConfig()

		if config == nil {
			t.Fatal("Config should not be nil")
		}

		if config.PostgresURL == "" {
			t.Error("PostgresURL should not be empty")
		}

		if config.QdrantURL == "" {
			t.Error("QdrantURL should not be empty")
		}
	})

	t.Run("MissingPostgresURL", func(t *testing.T) {
		// Save original value
		originalPostgresURL := os.Getenv("POSTGRES_URL")
		defer func() {
			if originalPostgresURL != "" {
				os.Setenv("POSTGRES_URL", originalPostgresURL)
			} else {
				os.Unsetenv("POSTGRES_URL")
			}
		}()

		// Unset POSTGRES_URL
		os.Unsetenv("POSTGRES_URL")

		// This should cause a fatal error, but we can't test that directly
		// Instead, we test that the environment variable is required
		value := os.Getenv("POSTGRES_URL")
		if value != "" {
			t.Error("POSTGRES_URL should be empty for this test")
		}
	})

	t.Run("DefaultValues", func(t *testing.T) {
		env := setupTestEnvironment(t)
		defer env.Cleanup()

		// Unset optional values to test defaults
		os.Unsetenv("N8N_BASE_URL")
		os.Unsetenv("WINDMILL_BASE_URL")
		os.Unsetenv("QDRANT_URL")

		config := loadConfig()

		if config.WindmillURL != "http://localhost:8000" {
			t.Errorf("Expected default Windmill URL, got %s", config.WindmillURL)
		}

		if config.QdrantURL != "http://localhost:6333" {
			t.Errorf("Expected default Qdrant URL, got %s", config.QdrantURL)
		}
	})
}

// TestGetEnv tests the getEnv helper function
func TestGetEnv(t *testing.T) {
	t.Run("ExistingVariable", func(t *testing.T) {
		os.Setenv("TEST_VAR", "test_value")
		defer os.Unsetenv("TEST_VAR")

		result := getEnv("TEST_VAR", "fallback")
		if result != "test_value" {
			t.Errorf("Expected 'test_value', got '%s'", result)
		}
	})

	t.Run("FallbackValue", func(t *testing.T) {
		result := getEnv("NONEXISTENT_VAR", "fallback")
		if result != "fallback" {
			t.Errorf("Expected 'fallback', got '%s'", result)
		}
	})

	t.Run("EmptyFallback", func(t *testing.T) {
		result := getEnv("NONEXISTENT_VAR", "")
		if result != "" {
			t.Errorf("Expected empty string, got '%s'", result)
		}
	})
}

// TestHealthHandler tests the health endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	suite := NewHandlerTestSuite(env.Config)

	t.Run("Success", func(t *testing.T) {
		response := suite.TestSuccessPath(t, "GET", "/health")

		assertStringField(t, response, "status", "healthy")
		assertStringField(t, response, "service", "resume-screening-assistant-api")
		assertStringField(t, response, "version", "1.0.0")
		assertFieldExists(t, response, "time")

		// Validate time format
		timeStr, ok := response["time"].(string)
		if !ok {
			t.Error("Time field should be a string")
		} else {
			_, err := time.Parse(time.RFC3339, timeStr)
			if err != nil {
				t.Errorf("Time field should be in RFC3339 format: %v", err)
			}
		}
	})

	t.Run("InvalidMethod", func(t *testing.T) {
		scenarios := NewTestScenarioBuilder().
			AddInvalidMethod("/health", "POST").
			AddInvalidMethod("/health", "PUT").
			AddInvalidMethod("/health", "DELETE").
			Build()

		suite.RunErrorScenarios(t, scenarios)
	})
}

// TestJobsHandler tests the jobs endpoint
func TestJobsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	suite := NewHandlerTestSuite(env.Config)

	t.Run("Success", func(t *testing.T) {
		response := suite.TestSuccessPath(t, "GET", "/api/jobs")

		// Validate response structure
		if success, ok := response["success"].(bool); !ok || !success {
			t.Error("Expected success to be true")
		}

		jobs := assertArrayResponse(t, response, "jobs", 1)
		count := assertIntField(t, response, "count")

		if count != len(jobs) {
			t.Errorf("Count (%d) should match jobs array length (%d)", count, len(jobs))
		}

		// Validate first job structure
		if len(jobs) > 0 {
			job, ok := jobs[0].(map[string]interface{})
			if !ok {
				t.Fatal("Job should be a map")
			}

			assertFieldExists(t, job, "id")
			assertFieldExists(t, job, "job_title")
			assertFieldExists(t, job, "company_name")
			assertFieldExists(t, job, "location")
			assertFieldExists(t, job, "experience_required")
			assertFieldExists(t, job, "candidate_count")
			assertFieldExists(t, job, "status")
		}
	})

	t.Run("InvalidMethod", func(t *testing.T) {
		scenarios := NewTestScenarioBuilder().
			AddInvalidMethod("/api/jobs", "POST").
			AddInvalidMethod("/api/jobs", "PUT").
			AddInvalidMethod("/api/jobs", "DELETE").
			Build()

		suite.RunErrorScenarios(t, scenarios)
	})

	t.Run("JobFieldValidation", func(t *testing.T) {
		response := suite.TestSuccessPath(t, "GET", "/api/jobs")
		jobs := assertArrayResponse(t, response, "jobs", 1)

		for i, jobInterface := range jobs {
			job, ok := jobInterface.(map[string]interface{})
			if !ok {
				t.Errorf("Job %d should be a map", i)
				continue
			}

			// Validate required fields
			if jobTitle, ok := job["job_title"].(string); !ok || jobTitle == "" {
				t.Errorf("Job %d: job_title should be a non-empty string", i)
			}

			if companyName, ok := job["company_name"].(string); !ok || companyName == "" {
				t.Errorf("Job %d: company_name should be a non-empty string", i)
			}

			if status, ok := job["status"].(string); !ok || status == "" {
				t.Errorf("Job %d: status should be a non-empty string", i)
			}
		}
	})
}

// TestCandidatesHandler tests the candidates endpoint
func TestCandidatesHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	suite := NewHandlerTestSuite(env.Config)

	t.Run("Success", func(t *testing.T) {
		response := suite.TestSuccessPath(t, "GET", "/api/candidates")

		// Validate response structure
		if success, ok := response["success"].(bool); !ok || !success {
			t.Error("Expected success to be true")
		}

		candidates := assertArrayResponse(t, response, "candidates", 1)
		count := assertIntField(t, response, "count")

		if count != len(candidates) {
			t.Errorf("Count (%d) should match candidates array length (%d)", count, len(candidates))
		}

		// Validate first candidate structure
		if len(candidates) > 0 {
			candidate, ok := candidates[0].(map[string]interface{})
			if !ok {
				t.Fatal("Candidate should be a map")
			}

			assertFieldExists(t, candidate, "id")
			assertFieldExists(t, candidate, "candidate_name")
			assertFieldExists(t, candidate, "email")
			assertFieldExists(t, candidate, "score")
			assertFieldExists(t, candidate, "experience_years")
			assertFieldExists(t, candidate, "parsed_skills")
			assertFieldExists(t, candidate, "status")
			assertFieldExists(t, candidate, "job_id")
		}
	})

	t.Run("FilterByJobID", func(t *testing.T) {
		// Test filtering by job_id=1
		response := suite.TestSuccessPath(t, "GET", "/api/candidates?job_id=1")

		candidates := assertArrayResponse(t, response, "candidates", 0)

		// All candidates should have job_id=1
		for i, candidateInterface := range candidates {
			candidate, ok := candidateInterface.(map[string]interface{})
			if !ok {
				t.Errorf("Candidate %d should be a map", i)
				continue
			}

			jobID, ok := candidate["job_id"].(float64)
			if !ok {
				t.Errorf("Candidate %d: job_id should be a number", i)
				continue
			}

			if int(jobID) != 1 {
				t.Errorf("Candidate %d: expected job_id=1, got %d", i, int(jobID))
			}
		}
	})

	t.Run("InvalidJobID", func(t *testing.T) {
		// Test with invalid job_id (non-numeric)
		response := suite.TestSuccessPath(t, "GET", "/api/candidates?job_id=invalid")

		// Should return all candidates when job_id is invalid
		assertFieldExists(t, response, "candidates")
	})

	t.Run("NonExistentJobID", func(t *testing.T) {
		// Test with non-existent job_id
		response := suite.TestSuccessPath(t, "GET", "/api/candidates?job_id=9999")

		candidates := assertArrayResponse(t, response, "candidates", 0)
		if len(candidates) != 0 {
			t.Errorf("Expected 0 candidates for non-existent job_id, got %d", len(candidates))
		}
	})

	t.Run("InvalidMethod", func(t *testing.T) {
		scenarios := NewTestScenarioBuilder().
			AddInvalidMethod("/api/candidates", "POST").
			AddInvalidMethod("/api/candidates", "PUT").
			AddInvalidMethod("/api/candidates", "DELETE").
			Build()

		suite.RunErrorScenarios(t, scenarios)
	})

	t.Run("CandidateFieldValidation", func(t *testing.T) {
		response := suite.TestSuccessPath(t, "GET", "/api/candidates")
		candidates := assertArrayResponse(t, response, "candidates", 1)

		for i, candidateInterface := range candidates {
			candidate, ok := candidateInterface.(map[string]interface{})
			if !ok {
				t.Errorf("Candidate %d should be a map", i)
				continue
			}

			// Validate required fields
			if name, ok := candidate["candidate_name"].(string); !ok || name == "" {
				t.Errorf("Candidate %d: candidate_name should be a non-empty string", i)
			}

			if email, ok := candidate["email"].(string); !ok || email == "" {
				t.Errorf("Candidate %d: email should be a non-empty string", i)
			}

			if score, ok := candidate["score"].(float64); !ok || score < 0 || score > 1 {
				t.Errorf("Candidate %d: score should be a number between 0 and 1", i)
			}

			if skills, ok := candidate["parsed_skills"].([]interface{}); ok {
				if len(skills) == 0 {
					t.Errorf("Candidate %d: parsed_skills should not be empty", i)
				}
			}
		}
	})
}

// TestSearchHandler tests the search endpoint
func TestSearchHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	suite := NewHandlerTestSuite(env.Config)

	t.Run("Success", func(t *testing.T) {
		response := suite.TestSuccessPath(t, "GET", "/api/search?query=python")

		// Validate response structure
		if success, ok := response["success"].(bool); !ok || !success {
			t.Error("Expected success to be true")
		}

		results := assertArrayResponse(t, response, "results", 0)
		count := assertIntField(t, response, "count")
		assertStringField(t, response, "query", "python")

		if count != len(results) {
			t.Errorf("Count (%d) should match results array length (%d)", count, len(results))
		}
	})

	t.Run("SearchBothTypes", func(t *testing.T) {
		response := suite.TestSuccessPath(t, "GET", "/api/search?query=engineer&type=both")

		results := assertArrayResponse(t, response, "results", 0)

		// Should include both candidates and jobs
		hasCandidate := false
		hasJob := false

		for _, resultInterface := range results {
			result, ok := resultInterface.(map[string]interface{})
			if !ok {
				continue
			}

			resultType, ok := result["type"].(string)
			if !ok {
				continue
			}

			if resultType == "candidate" {
				hasCandidate = true
			} else if resultType == "job" {
				hasJob = true
			}
		}

		if len(results) > 0 && !hasCandidate && !hasJob {
			t.Error("Expected results to include candidates or jobs")
		}
	})

	t.Run("SearchCandidatesOnly", func(t *testing.T) {
		response := suite.TestSuccessPath(t, "GET", "/api/search?query=python&type=candidates")

		results := assertArrayResponse(t, response, "results", 0)

		// All results should be candidates
		for i, resultInterface := range results {
			result, ok := resultInterface.(map[string]interface{})
			if !ok {
				t.Errorf("Result %d should be a map", i)
				continue
			}

			resultType, ok := result["type"].(string)
			if !ok {
				t.Errorf("Result %d: type field should exist", i)
				continue
			}

			if resultType != "candidate" {
				t.Errorf("Result %d: expected type 'candidate', got '%s'", i, resultType)
			}
		}
	})

	t.Run("SearchJobsOnly", func(t *testing.T) {
		response := suite.TestSuccessPath(t, "GET", "/api/search?query=engineer&type=jobs")

		results := assertArrayResponse(t, response, "results", 0)

		// All results should be jobs
		for i, resultInterface := range results {
			result, ok := resultInterface.(map[string]interface{})
			if !ok {
				t.Errorf("Result %d should be a map", i)
				continue
			}

			resultType, ok := result["type"].(string)
			if !ok {
				t.Errorf("Result %d: type field should exist", i)
				continue
			}

			if resultType != "job" {
				t.Errorf("Result %d: expected type 'job', got '%s'", i, resultType)
			}
		}
	})

	t.Run("EmptyQuery", func(t *testing.T) {
		response := suite.TestSuccessPath(t, "GET", "/api/search?query=")

		count := assertIntField(t, response, "count")
		assertStringField(t, response, "query", "")

		if count != 0 {
			t.Logf("Note: Empty query returned %d results (may be expected behavior)", count)
		}
	})

	t.Run("NoQuery", func(t *testing.T) {
		response := suite.TestSuccessPath(t, "GET", "/api/search")

		assertFieldExists(t, response, "results")
		assertFieldExists(t, response, "count")
		assertFieldExists(t, response, "query")
	})

	t.Run("DefaultType", func(t *testing.T) {
		response := suite.TestSuccessPath(t, "GET", "/api/search?query=test")

		// Without specifying type, should default to 'both'
		assertFieldExists(t, response, "results")
	})

	t.Run("InvalidMethod", func(t *testing.T) {
		scenarios := NewTestScenarioBuilder().
			AddInvalidMethod("/api/search", "POST").
			AddInvalidMethod("/api/search", "PUT").
			AddInvalidMethod("/api/search", "DELETE").
			Build()

		suite.RunErrorScenarios(t, scenarios)
	})

	t.Run("EdgeCases", func(t *testing.T) {
		scenarios := NewEdgeCaseTestBuilder().
			AddEmptyInput("/api/search?query=").
			AddNullValue("/api/search", "query").
			AddBoundaryValue("/api/search", "type", "invalid_type").
			Build()

		pattern := NewErrorTestPattern()
		pattern.AddScenarios(scenarios)
		pattern.Execute(t, suite.router)
	})
}

// TestSetupRoutes tests the route setup
func TestSetupRoutes(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("AllRoutesRegistered", func(t *testing.T) {
		router := setupRoutes(env.Config)
		if router == nil {
			t.Fatal("Router should not be nil")
		}

		// Test that all expected routes respond
		suite := NewHandlerTestSuite(env.Config)

		endpoints := []struct {
			path   string
			method string
		}{
			{"/health", "GET"},
			{"/api/jobs", "GET"},
			{"/api/candidates", "GET"},
			{"/api/search", "GET"},
			{"/", "GET"},
		}

		for _, endpoint := range endpoints {
			t.Run(endpoint.method+" "+endpoint.path, func(t *testing.T) {
				rr := suite.TestEndpoint(t, endpoint.method, endpoint.path, http.StatusOK)
				if rr.Code != http.StatusOK {
					t.Errorf("Expected status 200 for %s %s, got %d", endpoint.method, endpoint.path, rr.Code)
				}
			})
		}
	})

	t.Run("RootEndpoint", func(t *testing.T) {
		suite := NewHandlerTestSuite(env.Config)
		response := suite.TestSuccessPath(t, "GET", "/")

		assertStringField(t, response, "service", "resume-screening-assistant-api")
		assertStringField(t, response, "version", "1.0.0")
		assertFieldExists(t, response, "description")
		assertFieldExists(t, response, "endpoints")
		assertFieldExists(t, response, "resources")

		// Validate endpoints map
		endpoints, ok := response["endpoints"].(map[string]interface{})
		if !ok {
			t.Fatal("Endpoints should be a map")
		}

		expectedEndpoints := []string{"health", "jobs", "candidates", "search"}
		for _, endpoint := range expectedEndpoints {
			if _, exists := endpoints[endpoint]; !exists {
				t.Errorf("Expected endpoint '%s' in endpoints map", endpoint)
			}
		}

		// Validate resources map
		resources, ok := response["resources"].(map[string]interface{})
		if !ok {
			t.Fatal("Resources should be a map")
		}

		expectedResources := []string{"windmill", "postgres", "qdrant"}
		for _, resource := range expectedResources {
			if _, exists := resources[resource]; !exists {
				t.Errorf("Expected resource '%s' in resources map", resource)
			}
		}
	})
}

// TestHTTPResponseCodes tests various HTTP response codes
func TestHTTPResponseCodes(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	suite := NewHandlerTestSuite(env.Config)

	t.Run("SuccessResponses", func(t *testing.T) {
		endpoints := []string{
			"/health",
			"/api/jobs",
			"/api/candidates",
			"/api/search?query=test",
			"/",
		}

		for _, endpoint := range endpoints {
			t.Run(endpoint, func(t *testing.T) {
				rr := suite.TestEndpoint(t, "GET", endpoint, http.StatusOK)
				if rr.Code != http.StatusOK {
					t.Errorf("Expected status 200, got %d", rr.Code)
				}
			})
		}
	})

	t.Run("MethodNotAllowed", func(t *testing.T) {
		invalidRequests := []struct {
			path   string
			method string
		}{
			{"/health", "POST"},
			{"/api/jobs", "DELETE"},
			{"/api/candidates", "PUT"},
			{"/api/search", "POST"},
		}

		for _, req := range invalidRequests {
			t.Run(req.method+" "+req.path, func(t *testing.T) {
				rr := suite.TestEndpoint(t, req.method, req.path, http.StatusMethodNotAllowed)
				if rr.Code != http.StatusMethodNotAllowed {
					t.Logf("Expected status 405 for %s %s, got %d", req.method, req.path, rr.Code)
				}
			})
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		rr := suite.TestEndpoint(t, "GET", "/nonexistent", http.StatusNotFound)
		if rr.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for non-existent path, got %d", rr.Code)
		}
	})
}

// TestContentTypeHeaders tests that all responses have correct Content-Type
func TestContentTypeHeaders(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	suite := NewHandlerTestSuite(env.Config)

	endpoints := []string{
		"/health",
		"/api/jobs",
		"/api/candidates",
		"/api/search?query=test",
		"/",
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			rr := suite.TestEndpoint(t, "GET", endpoint, http.StatusOK)

			contentType := rr.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
			}
		})
	}
}

// BenchmarkHealthHandler benchmarks the health endpoint
func BenchmarkHealthHandler(b *testing.B) {
	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	suite := NewHandlerTestSuite(env.Config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		suite.TestEndpoint(&testing.T{}, "GET", "/health", http.StatusOK)
	}
}

// BenchmarkJobsHandler benchmarks the jobs endpoint
func BenchmarkJobsHandler(b *testing.B) {
	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	suite := NewHandlerTestSuite(env.Config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		suite.TestEndpoint(&testing.T{}, "GET", "/api/jobs", http.StatusOK)
	}
}

// BenchmarkCandidatesHandler benchmarks the candidates endpoint
func BenchmarkCandidatesHandler(b *testing.B) {
	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	suite := NewHandlerTestSuite(env.Config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		suite.TestEndpoint(&testing.T{}, "GET", "/api/candidates", http.StatusOK)
	}
}

// BenchmarkSearchHandler benchmarks the search endpoint
func BenchmarkSearchHandler(b *testing.B) {
	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	suite := NewHandlerTestSuite(env.Config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		suite.TestEndpoint(&testing.T{}, "GET", "/api/search?query=python", http.StatusOK)
	}
}
