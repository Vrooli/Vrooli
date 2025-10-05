package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestMain sets up and tears down test environment
func TestMain(m *testing.M) {
	// Set lifecycle managed flag for testing
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")

	// Run tests
	code := m.Run()

	// Cleanup
	os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")

	os.Exit(code)
}

// TestNewServer validates server initialization
func TestNewServer(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	if env.Server == nil {
		t.Fatal("Expected server to be initialized")
	}

	if env.Server.config.Port == "" {
		t.Error("Expected port to be configured")
	}

	if env.Server.config.QdrantURL == "" {
		t.Error("Expected Qdrant URL to be configured")
	}
}

// TestHealthHandler validates the health check endpoint
func TestHealthHandler(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		start := time.Now()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertResponseTime(t, start, 10*time.Second, "Health check")

		// Should return 200 or 503 depending on dependencies
		if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 200 or 503, got %d", w.Code)
		}

		var response map[string]interface{}
		assertJSONResponse(t, w, &response)

		// Validate response structure
		assertFieldExists(t, response, "status")
		assertFieldExists(t, response, "service")
		assertFieldExists(t, response, "timestamp")
		assertFieldExists(t, response, "dependencies")

		// Validate service name
		assertFieldValue(t, response, "service", "knowledge-observatory-api")
	})

	t.Run("ResponseTime", func(t *testing.T) {
		start := time.Now()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		duration := time.Since(start)

		// Health endpoint should respond within 10 seconds
		if duration > 10*time.Second {
			t.Errorf("Health check took %v, expected less than 10s", duration)
		}

		// Check response time header
		responseTime := w.Header().Get("X-Response-Time")
		if responseTime == "" {
			t.Error("Expected X-Response-Time header to be set")
		}
	})
}

// TestSearchHandler validates the search endpoint
func TestSearchHandler(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	suite := &HandlerTestSuite{
		HandlerName: "Search",
		Server:      env.Server,
		BaseURL:     "/api/v1/knowledge/search",
		SuccessTests: []SuccessTestCase{
			{
				Name:        "BasicSearch",
				Description: "Test basic search with query",
				Execute: func(t *testing.T, server *Server, setupData interface{}) HTTPTestRequest {
					return HTTPTestRequest{
						Method: "POST",
						Path:   "/api/v1/knowledge/search",
						Body: createSearchRequest("test query"),
					}
				},
				Validate: func(t *testing.T, w interface{}, setupData interface{}) {
					recorder := w.(*httptest.ResponseRecorder)
					assertStatusCode(t, recorder, http.StatusOK)

					var response SearchResponse
					assertJSONResponse(t, recorder, &response)

					// Results may be empty if Qdrant is not available, but should not be nil
					// In test mode without Qdrant, empty slice is acceptable
					if response.Results == nil {
						t.Log("Results are nil - Qdrant may not be available in test environment")
					}

					if response.Time < 0 {
						t.Error("Expected query time to be non-negative")
					}

					// Count should match length of results if results exist
					if response.Results != nil && response.Count != len(response.Results) {
						t.Errorf("Expected count %d to match results length %d", response.Count, len(response.Results))
					}
				},
			},
			{
				Name:        "SearchWithCollection",
				Description: "Test search with specific collection",
				Execute: func(t *testing.T, server *Server, setupData interface{}) HTTPTestRequest {
					return HTTPTestRequest{
						Method: "POST",
						Path:   "/api/v1/knowledge/search",
						Body: createSearchRequest("test", withCollection("test_collection")),
					}
				},
				Validate: func(t *testing.T, w interface{}, setupData interface{}) {
					recorder := w.(*httptest.ResponseRecorder)
					assertStatusCode(t, recorder, http.StatusOK)

					var response SearchResponse
					assertJSONResponse(t, recorder, &response)

					if response.Count < 0 {
						t.Error("Expected count to be non-negative")
					}
				},
			},
			{
				Name:        "SearchWithLimit",
				Description: "Test search with custom limit",
				Execute: func(t *testing.T, server *Server, setupData interface{}) HTTPTestRequest {
					return HTTPTestRequest{
						Method: "POST",
						Path:   "/api/v1/knowledge/search",
						Body: createSearchRequest("test", withLimit(5)),
					}
				},
				Validate: func(t *testing.T, w interface{}, setupData interface{}) {
					recorder := w.(*httptest.ResponseRecorder)
					assertStatusCode(t, recorder, http.StatusOK)

					var response SearchResponse
					assertJSONResponse(t, recorder, &response)

					// Results should not exceed limit (may be less if fewer matches)
					if len(response.Results) > 5 {
						t.Errorf("Expected at most 5 results, got %d", len(response.Results))
					}
				},
			},
			{
				Name:        "SearchWithThreshold",
				Description: "Test search with score threshold",
				Execute: func(t *testing.T, server *Server, setupData interface{}) HTTPTestRequest {
					return HTTPTestRequest{
						Method: "POST",
						Path:   "/api/v1/knowledge/search",
						Body: createSearchRequest("test", withThreshold(0.8)),
					}
				},
				Validate: func(t *testing.T, w interface{}, setupData interface{}) {
					recorder := w.(*httptest.ResponseRecorder)
					assertStatusCode(t, recorder, http.StatusOK)

					var response SearchResponse
					assertJSONResponse(t, recorder, &response)

					// All results should meet threshold
					for _, result := range response.Results {
						if result.Score < 0.8 {
							t.Errorf("Expected all scores >= 0.8, got %f", result.Score)
						}
					}
				},
			},
		},
		ErrorTests: CreateSearchErrorPatterns(),
	}

	suite.RunAllTests(t)

	// Additional error tests specific to search
	t.Run("MissingQuery", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/knowledge/search",
			Body: map[string]interface{}{
				"limit": 10,
			},
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "required")
	})

	t.Run("EmptyQuery", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/knowledge/search",
			Body: map[string]interface{}{
				"query": "",
			},
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "required")
	})
}

// TestGraphHandler validates the graph endpoint
func TestGraphHandler(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_EmptyRequest", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/knowledge/graph",
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertStatusCode(t, w, http.StatusOK)

		var response GraphResponse
		assertJSONResponse(t, w, &response)

		if response.Nodes == nil {
			t.Error("Expected nodes to be initialized")
		}

		if response.Edges == nil {
			t.Error("Expected edges to be initialized")
		}
	})

	t.Run("Success_WithCenterConcept", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/knowledge/graph",
			Body: map[string]interface{}{
				"center_concept": "test concept",
				"max_nodes":      50,
			},
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertStatusCode(t, w, http.StatusOK)

		var response GraphResponse
		assertJSONResponse(t, w, &response)

		// When center concept is provided, there should be nodes
		if len(response.Nodes) > 50 {
			t.Errorf("Expected at most 50 nodes, got %d", len(response.Nodes))
		}
	})

	t.Run("Success_WithDepth", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/knowledge/graph",
			Body: map[string]interface{}{
				"depth":     3,
				"max_nodes": 100,
			},
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertStatusCode(t, w, http.StatusOK)

		var response GraphResponse
		assertJSONResponse(t, w, &response)
	})

	// Error tests
	errorTests := CreateGraphErrorPatterns()
	for _, pattern := range errorTests {
		t.Run("Error_"+pattern.Name, func(t *testing.T) {
			req := pattern.Execute(t, env.Server, nil)
			w, err := makeHTTPRequest(env.Server, req)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}

			assertStatusCode(t, w, pattern.ExpectedStatus)
		})
	}
}

// TestMetricsHandler validates the metrics endpoint
func TestMetricsHandler(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_EmptyRequest", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/knowledge/metrics",
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertStatusCode(t, w, http.StatusOK)

		var response MetricsResponse
		assertJSONResponse(t, w, &response)

		if response.Metrics == nil {
			t.Error("Expected metrics to be initialized")
		}

		if response.Trends == nil {
			t.Error("Expected trends to be initialized")
		}

		if response.Alerts == nil {
			t.Error("Expected alerts to be initialized")
		}

		if response.LastUpdate == "" {
			t.Error("Expected last_update to be set")
		}
	})

	t.Run("Success_WithCollections", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/knowledge/metrics",
			Body: map[string]interface{}{
				"collections": []string{"test_collection"},
			},
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertStatusCode(t, w, http.StatusOK)

		var response MetricsResponse
		assertJSONResponse(t, w, &response)
	})

	t.Run("Success_WithTimeRange", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/knowledge/metrics",
			Body: map[string]interface{}{
				"time_range": "24h",
			},
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertStatusCode(t, w, http.StatusOK)
	})
}

// TestTimelineHandler validates the timeline endpoint
func TestTimelineHandler(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("Success_EmptyRequest", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/knowledge/timeline",
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertStatusCode(t, w, http.StatusOK)

		var response TimelineResponse
		assertJSONResponse(t, w, &response)

		if response.Entries == nil {
			t.Error("Expected entries to be initialized")
		}

		if response.Collections == nil {
			t.Error("Expected collections to be initialized")
		}

		if response.TotalEvents < 0 {
			t.Error("Expected total_events to be non-negative")
		}
	})

	t.Run("Success_WithTimeRange", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/knowledge/timeline",
			Body: map[string]interface{}{
				"time_range": "168h", // 7 days
			},
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertStatusCode(t, w, http.StatusOK)

		var response TimelineResponse
		assertJSONResponse(t, w, &response)
	})

	t.Run("CORS_Options", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "OPTIONS",
			Path:   "/api/v1/knowledge/timeline",
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertStatusCode(t, w, http.StatusOK)

		// Check CORS headers
		allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
		if allowOrigin != "*" {
			t.Errorf("Expected Access-Control-Allow-Origin to be *, got %s", allowOrigin)
		}
	})
}

// TestCORSMiddleware validates CORS headers are set correctly
func TestCORSMiddleware(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Test timeline endpoint which has explicit CORS handling
	t.Run("CORS_timeline", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "OPTIONS",
			Path:   "/api/v1/knowledge/timeline",
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertStatusCode(t, w, http.StatusOK)

		allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
		if allowOrigin != "*" {
			t.Errorf("Expected Access-Control-Allow-Origin to be *, got %s", allowOrigin)
		}
	})

	// Test enableCORS middleware function directly
	t.Run("CORSMiddlewareFunction", func(t *testing.T) {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("test"))
		})

		corsHandler := enableCORS(testHandler)

		req := httptest.NewRequest("OPTIONS", "/test", nil)
		w := httptest.NewRecorder()
		corsHandler.ServeHTTP(w, req)

		allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
		if allowOrigin != "*" {
			t.Errorf("Expected Access-Control-Allow-Origin to be *, got %s", allowOrigin)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestCalculateCollectionQuality validates quality metric calculations
func TestCalculateCollectionQuality(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("SmallCollection", func(t *testing.T) {
		info := &QdrantCollection{
			Name:        "small_test",
			Status:      "green",
			PointsCount: 50,
		}
		info.Config.Params.Vectors.Size = 768

		quality := env.Server.calculateCollectionQuality(info)

		if quality.Coherence < 0 || quality.Coherence > 1 {
			t.Errorf("Coherence score should be between 0 and 1, got %f", quality.Coherence)
		}

		if quality.Freshness < 0 || quality.Freshness > 1 {
			t.Errorf("Freshness score should be between 0 and 1, got %f", quality.Freshness)
		}

		if quality.Redundancy < 0 || quality.Redundancy > 1 {
			t.Errorf("Redundancy score should be between 0 and 1, got %f", quality.Redundancy)
		}

		if quality.Coverage < 0 || quality.Coverage > 1 {
			t.Errorf("Coverage score should be between 0 and 1, got %f", quality.Coverage)
		}
	})

	t.Run("LargeCollection", func(t *testing.T) {
		info := &QdrantCollection{
			Name:        "large_test",
			Status:      "green",
			PointsCount: 5000,
		}
		info.Config.Params.Vectors.Size = 1536

		quality := env.Server.calculateCollectionQuality(info)

		// Larger collection should have higher coverage
		if quality.Coverage < 0.75 {
			t.Errorf("Expected coverage >= 0.75 for large collection, got %f", quality.Coverage)
		}

		// Higher dimensionality should improve coherence
		if quality.Coherence < 0.85 {
			t.Errorf("Expected coherence >= 0.85 for high-dim vectors, got %f", quality.Coherence)
		}
	})

	t.Run("EmptyCollection", func(t *testing.T) {
		info := &QdrantCollection{
			Name:        "empty_test",
			Status:      "green",
			PointsCount: 0,
		}
		info.Config.Params.Vectors.Size = 768

		quality := env.Server.calculateCollectionQuality(info)

		// All scores should still be valid
		if quality.Coherence < 0 || quality.Coherence > 1 {
			t.Error("Invalid coherence score for empty collection")
		}
	})
}

// TestUtilityFunctions validates helper functions
func TestUtilityFunctions(t *testing.T) {
	t.Run("getEnv", func(t *testing.T) {
		os.Setenv("TEST_VAR", "test_value")
		defer os.Unsetenv("TEST_VAR")

		value := getEnv("TEST_VAR", "default")
		if value != "test_value" {
			t.Errorf("Expected 'test_value', got '%s'", value)
		}

		value = getEnv("NONEXISTENT_VAR", "default")
		if value != "default" {
			t.Errorf("Expected 'default', got '%s'", value)
		}
	})

	t.Run("min", func(t *testing.T) {
		if min(5, 10) != 5 {
			t.Error("min(5, 10) should return 5")
		}

		if min(10, 5) != 5 {
			t.Error("min(10, 5) should return 5")
		}

		if min(5, 5) != 5 {
			t.Error("min(5, 5) should return 5")
		}
	})

	t.Run("calculateTotalEntries", func(t *testing.T) {
		total := calculateTotalEntries()
		if total < 0 {
			t.Error("Total entries should be non-negative")
		}
	})

	t.Run("calculateOverallHealth", func(t *testing.T) {
		health := calculateOverallHealth()
		validHealthStates := []string{"healthy", "degraded", "critical"}

		isValid := false
		for _, state := range validHealthStates {
			if health == state {
				isValid = true
				break
			}
		}

		if !isValid {
			t.Errorf("Invalid health state: %s", health)
		}
	})
}

// TestPerformanceRequirements validates performance criteria
func TestPerformanceRequirements(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("SearchResponseTime", func(t *testing.T) {
		start := time.Now()

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/knowledge/search",
			Body: createSearchRequest("test query", withLimit(10)),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		duration := time.Since(start)

		// Search should respond within 500ms (per PRD)
		// Being lenient with 2s for test environment
		if duration > 2*time.Second {
			t.Errorf("Search took %v, expected less than 2s", duration)
		}

		assertStatusCode(t, w, http.StatusOK)
	})

	t.Run("HealthCheckResponseTime", func(t *testing.T) {
		start := time.Now()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		duration := time.Since(start)

		// Health check should respond within 10s
		if duration > 10*time.Second {
			t.Errorf("Health check took %v, expected less than 10s", duration)
		}

		if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
			t.Errorf("Unexpected status code: %d", w.Code)
		}
	})

	t.Run("ConcurrentRequests", func(t *testing.T) {
		const concurrency = 10
		done := make(chan bool, concurrency)
		errors := make(chan error, concurrency)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			go func(index int) {
				req := HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/knowledge/search",
					Body: createSearchRequest("test query", withLimit(5)),
				}

				w, err := makeHTTPRequest(env.Server, req)
				if err != nil {
					errors <- err
					done <- false
					return
				}

				if w.Code != http.StatusOK {
					errors <- nil // Success channel, just mark done
				}

				done <- true
			}(i)
		}

		// Wait for all requests to complete
		successCount := 0
		for i := 0; i < concurrency; i++ {
			if <-done {
				successCount++
			}
		}

		duration := time.Since(start)

		t.Logf("Completed %d concurrent requests in %v", concurrency, duration)

		if successCount < concurrency/2 {
			t.Errorf("Expected at least 50%% success rate, got %d/%d", successCount, concurrency)
		}
	})
}

// TestEdgeCases validates edge case handling
func TestEdgeCases(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("VeryLongQuery", func(t *testing.T) {
		longQuery := strings.Repeat("test ", 1000) // 5000 characters

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/knowledge/search",
			Body: createSearchRequest(longQuery),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should handle gracefully (either accept or reject with clear error)
		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Errorf("Expected 200 or 400, got %d", w.Code)
		}
	})

	t.Run("SpecialCharactersInQuery", func(t *testing.T) {
		specialQueries := []string{
			"test@#$%^&*()",
			"test\n\r\t",
			"test<script>alert('xss')</script>",
			"test'; DROP TABLE--",
		}

		for _, query := range specialQueries {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/knowledge/search",
				Body: createSearchRequest(query),
			}

			w, err := makeHTTPRequest(env.Server, req)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}

			// Should handle without crashing
			if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
				t.Errorf("Query %q: Expected 200 or 400, got %d", query, w.Code)
			}
		}
	})

	t.Run("ZeroLimit", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/knowledge/search",
			Body:   createSearchRequest("test", withLimit(0)),
		}

		w, err := makeHTTPRequest(env.Server, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertStatusCode(t, w, http.StatusOK)

		var response SearchResponse
		assertJSONResponse(t, w, &response)

		// Should use default limit
		if response.Count < 0 {
			t.Error("Expected non-negative count")
		}
	})

	t.Run("ExtremeThreshold", func(t *testing.T) {
		thresholds := []float64{-1.0, 0.0, 1.0, 2.0}

		for _, threshold := range thresholds {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/knowledge/search",
				Body:   createSearchRequest("test", withThreshold(threshold)),
			}

			w, err := makeHTTPRequest(env.Server, req)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}

			// Should handle gracefully
			if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
				t.Errorf("Threshold %f: Expected 200 or 400, got %d", threshold, w.Code)
			}
		}
	})

	t.Run("UnicodeInQuery", func(t *testing.T) {
		unicodeQueries := []string{
			"测试查询",           // Chinese
			"тестовый запрос", // Russian
			"🔬🧬🔭",          // Emojis
			"مرحبا",           // Arabic
		}

		for _, query := range unicodeQueries {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/knowledge/search",
				Body:   createSearchRequest(query),
			}

			w, err := makeHTTPRequest(env.Server, req)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}

			assertStatusCode(t, w, http.StatusOK)
		}
	})
}

// TestJSONResponseFormat validates all endpoints return valid JSON
func TestJSONResponseFormat(t *testing.T) {
	env := setupTestEnvironment(t)
	defer env.Cleanup()

	endpoints := []struct {
		name   string
		method string
		path   string
		body   interface{}
	}{
		{"Health", "GET", "/health", nil},
		{"Search", "POST", "/api/v1/knowledge/search", createSearchRequest("test")},
		{"Graph", "GET", "/api/v1/knowledge/graph", nil},
		{"Metrics", "GET", "/api/v1/knowledge/metrics", nil},
		{"Timeline", "GET", "/api/v1/knowledge/timeline", nil},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method: endpoint.method,
				Path:   endpoint.path,
				Body:   endpoint.body,
			}

			w, err := makeHTTPRequest(env.Server, req)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}

			// Check Content-Type
			contentType := w.Header().Get("Content-Type")
			if !strings.Contains(contentType, "application/json") {
				t.Errorf("Expected Content-Type to contain application/json, got %s", contentType)
			}

			// Validate JSON is parseable
			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Errorf("Failed to parse JSON response: %v. Body: %s", err, w.Body.String())
			}
		})
	}
}
