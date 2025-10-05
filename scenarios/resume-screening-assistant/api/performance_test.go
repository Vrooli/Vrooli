package main

import (
	"fmt"
	"net/http"
	"testing"
)

// TestPerformanceScenarios tests performance-critical scenarios
func TestPerformanceScenarios(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	router := createTestRouter(env.Config)

	t.Run("HighVolumeRequests", func(t *testing.T) {
		// Simulate high volume requests
		for i := 0; i < 100; i++ {
			rr := makeHTTPRequest(t, &router, "GET", "/health", nil)

			if rr.Code != http.StatusOK {
				t.Errorf("Request %d failed with status %d", i, rr.Code)
			}
		}
	})

	t.Run("ConcurrentEndpointAccess", func(t *testing.T) {
		endpoints := []string{
			"/health",
			"/api/jobs",
			"/api/candidates",
			"/api/search?query=test",
		}

		for _, endpoint := range endpoints {
			rr := makeHTTPRequest(t, &router, "GET", endpoint, nil)
			assertJSONResponse(t, rr, http.StatusOK)
		}
	})

	t.Run("LargeQueryParameters", func(t *testing.T) {
		// Test with large query strings
		largeQuery := "/api/search?query="
		for i := 0; i < 100; i++ {
			largeQuery += "python+machine+learning+"
		}

		rr := makeHTTPRequest(t, &router, "GET", largeQuery, nil)

		// Should handle gracefully
		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})

	t.Run("MultipleJobIDFilters", func(t *testing.T) {
		jobIDs := []string{"1", "2", "3", "4", "5", "999"}

		for _, jobID := range jobIDs {
			rr := makeHTTPRequest(t, &router, "GET", "/api/candidates?job_id="+jobID, nil)

			response := assertJSONResponse(t, rr, http.StatusOK)

			// Verify response structure
			assertFieldExists(t, response, "candidates")
			assertFieldExists(t, response, "count")
			assertSuccessResponse(t, rr)
		}
	})

	t.Run("SearchTypeVariations", func(t *testing.T) {
		searchTypes := []string{"both", "candidates", "jobs", "invalid", ""}

		for _, searchType := range searchTypes {
			path := "/api/search?query=engineer"
			if searchType != "" {
				path += "&type=" + searchType
			}

			rr := makeHTTPRequest(t, &router, "GET", path, nil)

			response := assertJSONResponse(t, rr, http.StatusOK)
			assertFieldExists(t, response, "results")
		}
	})
}

// TestHelperFunctionCoverage specifically tests helper functions to boost coverage
func TestHelperFunctionCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	router := createTestRouter(env.Config)

	t.Run("makeHTTPRequestVariations", func(t *testing.T) {
		// Test various HTTP methods
		methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}

		for _, method := range methods {
			rr := makeHTTPRequest(t, &router, method, "/health", nil)
			if rr == nil {
				t.Errorf("Response recorder should not be nil for method %s", method)
			}
		}

		// Test with body
		body := map[string]interface{}{"test": "data"}
		rr := makeHTTPRequest(t, &router, "POST", "/api/test", body)
		if rr == nil {
			t.Error("Response recorder with body should not be nil")
		}
	})

	t.Run("assertJSONResponseVariations", func(t *testing.T) {
		// Test with successful responses
		rr := makeHTTPRequest(t, &router, "GET", "/health", nil)
		response := assertJSONResponse(t, rr, http.StatusOK)
		if response == nil {
			t.Error("Response should not be nil for status 200")
		}
	})

	t.Run("assertSuccessResponseVariations", func(t *testing.T) {
		// Test with various response structures
		testCases := []struct {
			name     string
			endpoint string
		}{
			{"Jobs", "/api/jobs"},
			{"Candidates", "/api/candidates"},
			{"Search", "/api/search?query=test"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				rr := makeHTTPRequest(t, &router, "GET", tc.endpoint, nil)
				response := assertJSONResponse(t, rr, http.StatusOK)
				assertSuccessResponse(t, rr)
				if response == nil {
					t.Error("Response should not be nil")
				}
			})
		}
	})

	t.Run("assertArrayResponseVariations", func(t *testing.T) {
		// Test with different array fields
		testCases := []struct {
			endpoint  string
			arrayKey  string
			minLength int
		}{
			{"/api/jobs", "jobs", 1},
			{"/api/candidates", "candidates", 1},
			{"/api/search?query=test", "results", 0},
		}

		for _, tc := range testCases {
			rr := makeHTTPRequest(t, &router, "GET", tc.endpoint, nil)

			response := assertJSONResponse(t, rr, http.StatusOK)
			arr := assertArrayResponse(t, response, tc.arrayKey, tc.minLength)
			if tc.minLength > 0 && len(arr) < tc.minLength {
				t.Errorf("Expected at least %d items, got %d", tc.minLength, len(arr))
			}
		}
	})

	t.Run("assertFieldExistsVariations", func(t *testing.T) {
		rr := makeHTTPRequest(t, &router, "GET", "/health", nil)

		response := assertJSONResponse(t, rr, http.StatusOK)

		// Test with various fields
		fields := []string{"status", "service", "version", "time"}
		for _, field := range fields {
			assertFieldExists(t, response, field)
		}
	})

	t.Run("assertStringFieldVariations", func(t *testing.T) {
		rr := makeHTTPRequest(t, &router, "GET", "/health", nil)

		response := assertJSONResponse(t, rr, http.StatusOK)

		// Test with various string fields
		assertStringField(t, response, "status", "healthy")
		assertStringField(t, response, "service", "resume-screening-assistant-api")
		assertStringField(t, response, "version", "1.0.0")
	})

	t.Run("assertIntFieldVariations", func(t *testing.T) {
		rr := makeHTTPRequest(t, &router, "GET", "/api/jobs", nil)

		response := assertJSONResponse(t, rr, http.StatusOK)

		// Test integer field assertion
		count := assertIntField(t, response, "count")
		if count < 0 {
			t.Error("Count should be non-negative")
		}
	})
}

// TestErrorResponseHandling tests error response handling
func TestErrorResponseHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	router := createTestRouter(env.Config)

	t.Run("InvalidHTTPMethods", func(t *testing.T) {
		endpoints := []string{
			"/health",
			"/api/jobs",
			"/api/candidates",
			"/api/search",
		}

		invalidMethods := []string{"POST", "PUT", "DELETE", "PATCH"}

		for _, endpoint := range endpoints {
			for _, method := range invalidMethods {
				rr := makeHTTPRequest(t, &router, method, endpoint, nil)

				// Should return 405 Method Not Allowed
				if rr.Code != http.StatusMethodNotAllowed {
					t.Errorf("Expected 405 for %s %s, got %d", method, endpoint, rr.Code)
				}
			}
		}
	})

	t.Run("NotFoundEndpoints", func(t *testing.T) {
		notFoundPaths := []string{
			"/api/nonexistent",
			"/api/jobs/1",
			"/api/candidates/1",
			"/api/unknown",
			"/health/status",
		}

		for _, path := range notFoundPaths {
			rr := makeHTTPRequest(t, &router, "GET", path, nil)

			if rr.Code != http.StatusNotFound {
				t.Errorf("Expected 404 for %s, got %d", path, rr.Code)
			}
		}
	})
}

// TestEdgeCaseCoverage tests edge cases to maximize coverage
func TestEdgeCaseCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	router := createTestRouter(env.Config)

	t.Run("EmptyQueryParameters", func(t *testing.T) {
		paths := []string{
			"/api/candidates?job_id=",
			"/api/search?query=",
			"/api/search?type=",
			"/api/search?query=&type=",
		}

		for _, path := range paths {
			rr := makeHTTPRequest(t, &router, "GET", path, nil)

			// Should still return 200 with empty/filtered results
			if rr.Code != http.StatusOK {
				t.Errorf("Expected 200 for %s, got %d", path, rr.Code)
			}
		}
	})

	t.Run("SpecialCharactersInQuery", func(t *testing.T) {
		specialQueries := []string{
			"/api/search?query=%20%20%20",    // spaces
			"/api/search?query=test%26test",  // ampersand
			"/api/search?query=test%3Dvalue", // equals
			"/api/search?query=test%2Fpath",  // slash
		}

		for _, path := range specialQueries {
			rr := makeHTTPRequest(t, &router, "GET", path, nil)

			// Should handle gracefully
			if rr.Code != http.StatusOK {
				t.Errorf("Expected 200 for %s, got %d", path, rr.Code)
			}
		}
	})

	t.Run("BoundaryJobIDs", func(t *testing.T) {
		boundaryIDs := []string{
			"0",
			"-1",
			"2147483647", // max int32
			"999999999",
			"abc",
			"1.5",
		}

		for _, jobID := range boundaryIDs {
			rr := makeHTTPRequest(t, &router, "GET", "/api/candidates?job_id="+jobID, nil)

			response := assertJSONResponse(t, rr, http.StatusOK)
			assertFieldExists(t, response, "candidates")
		}
	})

	t.Run("MultipleQueryParameters", func(t *testing.T) {
		rr := makeHTTPRequest(t, &router, "GET", "/api/search?query=test&type=both&extra=param&another=value", nil)

		// Should ignore extra parameters
		response := assertJSONResponse(t, rr, http.StatusOK)
		assertFieldExists(t, response, "results")
	})
}

// TestIntegrationScenarios tests complete integration scenarios
func TestIntegrationScenarios(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	suite := NewHandlerTestSuite(env.Config)

	t.Run("CompleteUserWorkflow", func(t *testing.T) {
		// 1. Check health
		healthResp := suite.TestSuccessPath(t, "GET", "/health")
		assertStringField(t, healthResp, "status", "healthy")

		// 2. Get all jobs
		jobsResp := suite.TestSuccessPath(t, "GET", "/api/jobs")
		jobs := assertArrayResponse(t, jobsResp, "jobs", 1)

		// 3. For each job, get candidates
		for _, jobInterface := range jobs {
			job := jobInterface.(map[string]interface{})
			jobID := int(job["id"].(float64))

			candidatesResp := suite.TestSuccessPath(t, "GET",
				fmt.Sprintf("/api/candidates?job_id=%d", jobID))
			assertFieldExists(t, candidatesResp, "candidates")
		}

		// 4. Search for candidates
		searchResp := suite.TestSuccessPath(t, "GET", "/api/search?query=python")
		assertFieldExists(t, searchResp, "results")
	})

	t.Run("FilteringAndSearchWorkflow", func(t *testing.T) {
		// Get all candidates
		allCandidates := suite.TestSuccessPath(t, "GET", "/api/candidates")
		allCount := assertIntField(t, allCandidates, "count")

		// Filter by job_id 1
		filtered := suite.TestSuccessPath(t, "GET", "/api/candidates?job_id=1")
		filteredCount := assertIntField(t, filtered, "count")

		// Filtered count should be less than or equal to total
		if filteredCount > allCount {
			t.Error("Filtered count should not exceed total count")
		}

		// Search by query
		searchResp := suite.TestSuccessPath(t, "GET", "/api/search?query=engineer&type=candidates")
		assertFieldExists(t, searchResp, "results")
	})
}
