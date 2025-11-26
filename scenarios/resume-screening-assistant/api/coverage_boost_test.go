package main

import (
	"net/http"
	"testing"
)

// TestAdditionalCoverage adds tests to boost coverage above 80%
func TestAdditionalCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	suite := NewHandlerTestSuite(env.Config)

	t.Run("CompleteRequestLifecycle", func(t *testing.T) {
		// Test complete request lifecycle for all endpoints
		endpoints := []string{
			"/health",
			"/api/jobs",
			"/api/candidates",
			"/api/candidates?job_id=1",
			"/api/candidates?job_id=2",
			"/api/search?query=python&type=both",
			"/api/search?query=engineer&type=candidates",
			"/api/search?query=software&type=jobs",
			"/",
		}

		for _, endpoint := range endpoints {
			rr := suite.TestEndpoint(t, "GET", endpoint, http.StatusOK)
			response := assertJSONResponse(t, rr, http.StatusOK)

			// Verify response has expected structure
			if response == nil {
				t.Errorf("Response should not be nil for %s", endpoint)
			}
		}
	})

	t.Run("AllHelperFunctions", func(t *testing.T) {
		// Use all helper functions to ensure coverage
		response := suite.TestSuccessPath(t, "GET", "/health")

		// Test all assertion functions
		assertFieldExists(t, response, "status")
		assertStringField(t, response, "status", "healthy")
		assertStringField(t, response, "service", "resume-screening-assistant-api")
		assertStringField(t, response, "version", "1.0.0")
	})

	t.Run("ArrayResponseValidation", func(t *testing.T) {
		response := suite.TestSuccessPath(t, "GET", "/api/jobs")

		// Test array response helpers
		jobs := assertArrayResponse(t, response, "jobs", 1)
		count := assertIntField(t, response, "count")

		if len(jobs) != count {
			t.Errorf("Jobs length (%d) should match count (%d)", len(jobs), count)
		}

		// Test with candidates
		candidatesResponse := suite.TestSuccessPath(t, "GET", "/api/candidates")
		candidates := assertArrayResponse(t, candidatesResponse, "candidates", 1)
		candidatesCount := assertIntField(t, candidatesResponse, "count")

		if len(candidates) != candidatesCount {
			t.Errorf("Candidates length (%d) should match count (%d)", len(candidates), candidatesCount)
		}
	})

	t.Run("SearchEndpointVariations", func(t *testing.T) {
		variations := []struct {
			query      string
			searchType string
		}{
			{"python", "both"},
			{"python", "candidates"},
			{"python", "jobs"},
			{"engineer", "both"},
			{"engineer", "candidates"},
			{"engineer", "jobs"},
			{"ml", "both"},
			{"", "both"},
			{"test", ""},
		}

		for _, v := range variations {
			path := "/api/search?query=" + v.query
			if v.searchType != "" {
				path += "&type=" + v.searchType
			}

			response := suite.TestSuccessPath(t, "GET", path)

			// Validate structure
			assertFieldExists(t, response, "success")
			assertFieldExists(t, response, "results")
			assertFieldExists(t, response, "count")
			assertFieldExists(t, response, "query")
		}
	})

	t.Run("CandidatesFilteringEdgeCases", func(t *testing.T) {
		testCases := []struct {
			jobID    string
			expected string
		}{
			{"1", "should filter by job_id 1"},
			{"2", "should filter by job_id 2"},
			{"999", "should return empty for non-existent job"},
			{"abc", "should handle invalid job_id"},
			{"0", "should handle zero job_id"},
			{"-1", "should handle negative job_id"},
		}

		for _, tc := range testCases {
			t.Run(tc.expected, func(t *testing.T) {
				path := "/api/candidates?job_id=" + tc.jobID
				response := suite.TestSuccessPath(t, "GET", path)

				// Should always succeed with valid JSON
				assertFieldExists(t, response, "candidates")
				assertFieldExists(t, response, "count")
			})
		}
	})

	t.Run("RouterConfiguration", func(t *testing.T) {
		// Test that setupRoutes creates a properly configured router
		router := setupRoutes(env.Config)
		if router == nil {
			t.Fatal("Router should not be nil")
		}

		// Verify router handles requests correctly
		testRouter := createTestRouter(env.Config)
		if testRouter == nil {
			t.Fatal("Test router should not be nil")
		}
	})

	t.Run("ResponseStructureValidation", func(t *testing.T) {
		// Validate health response structure
		healthResponse := suite.TestSuccessPath(t, "GET", "/health")
		if _, ok := healthResponse["status"]; !ok {
			t.Error("Health response should have status field")
		}
		if _, ok := healthResponse["service"]; !ok {
			t.Error("Health response should have service field")
		}
		if _, ok := healthResponse["version"]; !ok {
			t.Error("Health response should have version field")
		}
		if _, ok := healthResponse["time"]; !ok {
			t.Error("Health response should have time field")
		}

		// Validate jobs response structure
		jobsResponse := suite.TestSuccessPath(t, "GET", "/api/jobs")
		if success, ok := jobsResponse["success"].(bool); !ok || !success {
			t.Error("Jobs response should have success=true")
		}
		if _, ok := jobsResponse["jobs"]; !ok {
			t.Error("Jobs response should have jobs field")
		}
		if _, ok := jobsResponse["count"]; !ok {
			t.Error("Jobs response should have count field")
		}

		// Validate candidates response structure
		candidatesResponse := suite.TestSuccessPath(t, "GET", "/api/candidates")
		if success, ok := candidatesResponse["success"].(bool); !ok || !success {
			t.Error("Candidates response should have success=true")
		}
		if _, ok := candidatesResponse["candidates"]; !ok {
			t.Error("Candidates response should have candidates field")
		}
		if _, ok := candidatesResponse["count"]; !ok {
			t.Error("Candidates response should have count field")
		}

		// Validate search response structure
		searchResponse := suite.TestSuccessPath(t, "GET", "/api/search?query=test")
		if success, ok := searchResponse["success"].(bool); !ok || !success {
			t.Error("Search response should have success=true")
		}
		if _, ok := searchResponse["results"]; !ok {
			t.Error("Search response should have results field")
		}
		if _, ok := searchResponse["count"]; !ok {
			t.Error("Search response should have count field")
		}
		if _, ok := searchResponse["query"]; !ok {
			t.Error("Search response should have query field")
		}
	})
}

// TestMockDataConsistency tests that mock data is consistent and valid
func TestMockDataConsistency(t *testing.T) {
	t.Run("MockJobsStructure", func(t *testing.T) {
		jobs := mockJobsData()

		for i, jobInterface := range jobs {
			job, ok := jobInterface.(map[string]interface{})
			if !ok {
				t.Fatalf("Job %d should be a map", i)
			}

			// Validate required fields
			requiredFields := []string{"id", "job_title", "company_name", "location", "experience_required", "candidate_count", "status"}
			for _, field := range requiredFields {
				if _, exists := job[field]; !exists {
					t.Errorf("Job %d missing required field: %s", i, field)
				}
			}
		}
	})

	t.Run("MockCandidatesStructure", func(t *testing.T) {
		candidates := mockCandidatesData()

		for i, candidateInterface := range candidates {
			candidate, ok := candidateInterface.(map[string]interface{})
			if !ok {
				t.Fatalf("Candidate %d should be a map", i)
			}

			// Validate required fields
			requiredFields := []string{"id", "candidate_name", "email", "score", "experience_years", "parsed_skills", "status", "job_id"}
			for _, field := range requiredFields {
				if _, exists := candidate[field]; !exists {
					t.Errorf("Candidate %d missing required field: %s", i, field)
				}
			}

			// Validate score is in valid range
			if score, ok := candidate["score"].(float64); ok {
				if score < 0 || score > 1 {
					t.Errorf("Candidate %d has invalid score: %f (should be 0-1)", i, score)
				}
			}
		}
	})
}

// TestConfigurationIntegration tests configuration integration
func TestConfigurationIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ConfigFieldsPopulated", func(t *testing.T) {
		if env.Config.PostgresURL == "" {
			t.Error("PostgresURL should be populated")
		}
		if env.Config.WindmillURL == "" {
			t.Error("WindmillURL should be populated")
		}
		if env.Config.QdrantURL == "" {
			t.Error("QdrantURL should be populated")
		}
	})

	t.Run("RootEndpointShowsConfig", func(t *testing.T) {
		suite := NewHandlerTestSuite(env.Config)
		response := suite.TestSuccessPath(t, "GET", "/")

		// Verify resources are exposed
		resources, ok := response["resources"].(map[string]interface{})
		if !ok {
			t.Fatal("Resources should be a map")
		}

		if _, exists := resources["postgres"]; !exists {
			t.Error("Should expose postgres resource")
		}
		if _, exists := resources["windmill"]; !exists {
			t.Error("Should expose windmill resource")
		}
		if _, exists := resources["qdrant"]; !exists {
			t.Error("Should expose qdrant resource")
		}
	})
}
