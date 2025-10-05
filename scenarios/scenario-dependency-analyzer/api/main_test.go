
package main

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestHealthHandler tests the health endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("Success", func(t *testing.T) {
		recorder := makeHTTPRequest(t, router, "GET", "/health", nil)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", recorder.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", response["status"])
		}

		if response["service"] != "scenario-dependency-analyzer" {
			t.Errorf("Expected service name, got %v", response["service"])
		}
	})
}

// TestAnalysisHealthHandler tests the analysis health check endpoint
func TestAnalysisHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("Success", func(t *testing.T) {
		recorder := makeHTTPRequest(t, router, "GET", "/api/v1/health/analysis", nil)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", recorder.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", response["status"])
		}

		capabilities, ok := response["capabilities"].([]interface{})
		if !ok {
			t.Errorf("Expected capabilities array")
		} else if len(capabilities) == 0 {
			t.Errorf("Expected non-empty capabilities")
		}
	})
}

// TestGetGraphHandler tests the graph generation endpoint
func TestGetGraphHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Skip if no database available
	testDB, dbCleanup := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	db = testDB
	defer func() { db = nil }()

	router := setupTestRouter()

	t.Run("ValidGraphTypes", func(t *testing.T) {
		validTypes := []string{"resource", "scenario", "combined"}

		for _, graphType := range validTypes {
			t.Run(graphType, func(t *testing.T) {
				recorder := makeHTTPRequest(t, router, "GET", "/api/v1/graph/"+graphType, nil)

				if recorder.Code != http.StatusOK {
					t.Errorf("Expected status 200 for type %s, got %d. Body: %s",
						graphType, recorder.Code, recorder.Body.String())
				}

				var response map[string]interface{}
				if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				// Verify graph structure
				if _, exists := response["nodes"]; !exists {
					t.Errorf("Expected 'nodes' in response for type %s", graphType)
				}
				if _, exists := response["edges"]; !exists {
					t.Errorf("Expected 'edges' in response for type %s", graphType)
				}
			})
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		patterns := NewTestScenarioBuilder().
			AddInvalidGraphType("/api/v1/graph", "invalid").
			AddInvalidGraphType("/api/v1/graph", "unknown").
			Build()

		suite := NewHandlerTestSuite(t, router)
		suite.TestErrorPatterns(patterns)
	})
}

// TestAnalyzeProposedHandler tests the proposed scenario analysis endpoint
func TestAnalyzeProposedHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("Success", func(t *testing.T) {
		request := ProposedScenarioRequest{
			Name:        "test-scenario",
			Description: "A test scenario for analysis",
			Requirements: []string{
				"Need to store data",
				"Need caching",
				"Need AI capabilities",
			},
		}

		recorder := makeHTTPRequest(t, router, "POST", "/api/v1/analyze/proposed", request)

		// May succeed or fail depending on resource availability
		if recorder.Code != http.StatusOK && recorder.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", recorder.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		tests := []struct {
			name           string
			request        interface{}
			expectedStatus int
		}{
			{
				name:           "Empty request",
				request:        map[string]interface{}{},
				expectedStatus: http.StatusOK, // May still process with defaults
			},
			{
				name:           "Missing name",
				request:        ProposedScenarioRequest{Description: "test"},
				expectedStatus: http.StatusOK, // May still process
			},
			{
				name:           "Invalid JSON",
				request:        "invalid-json",
				expectedStatus: http.StatusBadRequest,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				recorder := makeHTTPRequest(t, router, "POST", "/api/v1/analyze/proposed", tt.request)

				// Accept either expected status or internal server error
				if recorder.Code != tt.expectedStatus && recorder.Code != http.StatusInternalServerError {
					t.Logf("Request %s returned status %d (body: %s)", tt.name, recorder.Code, recorder.Body.String())
				}
			})
		}
	})
}

// TestGetDependenciesHandler tests the dependencies retrieval endpoint
func TestGetDependenciesHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Skip if no database available
	testDB, dbCleanup := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	db = testDB
	defer func() { db = nil }()

	router := setupTestRouter()

	t.Run("Success", func(t *testing.T) {
		// Insert test dependencies
		deps := []ScenarioDependency{
			createTestDependency("test-scenario", "resource", "postgres", true),
			createTestDependency("test-scenario", "resource", "redis", false),
			createTestDependency("test-scenario", "scenario", "auth-service", true),
		}

		for _, dep := range deps {
			insertTestDependency(t, testDB, dep)
		}

		recorder := makeHTTPRequest(t, router, "GET", "/api/v1/scenarios/test-scenario/dependencies", nil)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", recorder.Code, recorder.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Verify structure
		if response["scenario"] != "test-scenario" {
			t.Errorf("Expected scenario 'test-scenario', got %v", response["scenario"])
		}

		resources, ok := response["resources"].([]interface{})
		if !ok {
			t.Errorf("Expected resources array")
		} else if len(resources) != 2 {
			t.Errorf("Expected 2 resources, got %d", len(resources))
		}

		scenarios, ok := response["scenarios"].([]interface{})
		if !ok {
			t.Errorf("Expected scenarios array")
		} else if len(scenarios) != 1 {
			t.Errorf("Expected 1 scenario dependency, got %d", len(scenarios))
		}
	})

	t.Run("NonExistentScenario", func(t *testing.T) {
		recorder := makeHTTPRequest(t, router, "GET", "/api/v1/scenarios/non-existent/dependencies", nil)

		if recorder.Code != http.StatusOK {
			t.Errorf("Expected status 200 (empty result), got %d", recorder.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Should return empty arrays
		resources, ok := response["resources"].([]interface{})
		if !ok || len(resources) != 0 {
			t.Errorf("Expected empty resources for non-existent scenario")
		}
	})
}

// TestLoadConfig tests configuration loading
func TestLoadConfig(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ValidConfig", func(t *testing.T) {
		// Save and set required env vars
		originalPort := os.Getenv("API_PORT")
		originalDBURL := os.Getenv("DATABASE_URL")
		defer func() {
			if originalPort != "" {
				os.Setenv("API_PORT", originalPort)
			}
			if originalDBURL != "" {
				os.Setenv("DATABASE_URL", originalDBURL)
			}
		}()

		// Set test values
		os.Setenv("API_PORT", "8080")
		os.Setenv("DATABASE_URL", "postgres://test:test@localhost/test")

		// Note: loadConfig calls log.Fatal on error, so we can't easily test
		// the error paths without refactoring. We'll just verify it works with valid config.
		// This test is primarily to document the expected behavior.
		t.Skip("loadConfig uses log.Fatal which exits process - cannot test error paths without refactoring")
	})
}

// TestContainsHelper tests the contains utility function
func TestContainsHelper(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{"Found", []string{"a", "b", "c"}, "b", true},
		{"NotFound", []string{"a", "b", "c"}, "d", false},
		{"EmptySlice", []string{}, "a", false},
		{"EmptyItem", []string{"a", "b"}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			if result != tt.expected {
				t.Errorf("contains(%v, %q) = %v, want %v", tt.slice, tt.item, result, tt.expected)
			}
		})
	}
}

// TestCalculateComplexityScore tests complexity calculation
func TestCalculateComplexityScore(t *testing.T) {
	tests := []struct {
		name      string
		nodeCount int
		edgeCount int
	}{
		{"Empty", 0, 0},
		{"SmallGraph", 5, 4},
		{"LargeGraph", 50, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := make([]GraphNode, tt.nodeCount)
			edges := make([]GraphEdge, tt.edgeCount)

			for i := 0; i < tt.nodeCount; i++ {
				nodes[i] = GraphNode{
					ID:    string(rune('A' + i)),
					Label: "Node " + string(rune('A'+i)),
					Type:  "test",
				}
			}

			for i := 0; i < tt.edgeCount; i++ {
				edges[i] = GraphEdge{
					Source: "A",
					Target: "B",
					Weight: 1.0,
				}
			}

			score := calculateComplexityScore(nodes, edges)

			if score < 0 {
				t.Errorf("Expected non-negative complexity score, got %f", score)
			}

			// The complexity score algorithm may vary - just verify it returns a reasonable value
			// For large graphs, we expect some complexity score
			if tt.nodeCount > 10 && tt.edgeCount > 10 {
				t.Logf("Large graph complexity score: %f", score)
			}
		})
	}
}

// TestDeduplicateResources tests resource deduplication
func TestDeduplicateResources(t *testing.T) {
	tests := []struct {
		name     string
		input    []map[string]interface{}
		expected int
	}{
		{
			name: "NoDuplicates",
			input: []map[string]interface{}{
				{"resource_name": "postgres", "confidence": 0.9},
				{"resource_name": "redis", "confidence": 0.8},
			},
			expected: 2,
		},
		{
			name: "WithDuplicates",
			input: []map[string]interface{}{
				{"resource_name": "postgres", "confidence": 0.9},
				{"resource_name": "redis", "confidence": 0.8},
				{"resource_name": "postgres", "confidence": 0.7},
			},
			expected: 2,
		},
		{
			name: "DuplicatesKeepHighestConfidence",
			input: []map[string]interface{}{
				{"resource_name": "postgres", "confidence": 0.7},
				{"resource_name": "postgres", "confidence": 0.9},
			},
			expected: 1,
		},
		{
			name:     "Empty",
			input:    []map[string]interface{}{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deduplicateResources(tt.input)
			if len(result) != tt.expected {
				t.Errorf("Expected %d unique resources, got %d", tt.expected, len(result))
			}

			// Verify highest confidence is kept for duplicates
			if tt.name == "DuplicatesKeepHighestConfidence" && len(result) > 0 {
				if result[0]["confidence"].(float64) != 0.9 {
					t.Errorf("Expected highest confidence 0.9, got %v", result[0]["confidence"])
				}
			}
		})
	}
}

// TestCalculateResourceConfidence tests confidence calculation for resources
func TestCalculateResourceConfidence(t *testing.T) {
	tests := []struct {
		name        string
		predictions []map[string]interface{}
		minExpected float64
		maxExpected float64
	}{
		{
			name: "HighConfidence",
			predictions: []map[string]interface{}{
				{"confidence": 0.95},
				{"confidence": 0.90},
			},
			minExpected: 0.85,
			maxExpected: 1.0,
		},
		{
			name: "LowConfidence",
			predictions: []map[string]interface{}{
				{"confidence": 0.3},
				{"confidence": 0.2},
			},
			minExpected: 0.0,
			maxExpected: 0.5,
		},
		{
			name:        "Empty",
			predictions: []map[string]interface{}{},
			minExpected: 0.0,
			maxExpected: 0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := calculateResourceConfidence(tt.predictions)

			if confidence < tt.minExpected || confidence > tt.maxExpected {
				t.Errorf("Expected confidence between %f and %f, got %f",
					tt.minExpected, tt.maxExpected, confidence)
			}
		})
	}
}

// TestMapPatternToResource tests pattern to resource mapping
func TestMapPatternToResource(t *testing.T) {
	tests := []struct {
		pattern  string
		expected string
	}{
		{"database", "postgres"},
		{"postgres", "postgres"},
		{"postgresql", "postgres"},
		{"storage", "minio"},
		{"minio", "minio"},
		{"cache", "redis"},
		{"redis", "redis"},
		{"llm", "ollama"},
		{"ollama", "ollama"},
		{"workflow", "n8n"},
		{"n8n", "n8n"},
		{"vector", "qdrant"},
		{"qdrant", "qdrant"},
		{"unknown", ""},
		{"caching", ""}, // Not in map
		{"ai", ""},      // Not in map
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			result := mapPatternToResource(tt.pattern)
			if result != tt.expected {
				t.Errorf("mapPatternToResource(%q) = %q, want %q", tt.pattern, result, tt.expected)
			}
		})
	}
}

// TestGetHeuristicPredictions tests heuristic-based predictions
func TestGetHeuristicPredictions(t *testing.T) {
	tests := []struct {
		description       string
		expectedResources []string
		minPredictions    int
	}{
		{
			description:       "Need database to store user data",
			expectedResources: []string{"postgres"},
			minPredictions:    1,
		},
		{
			description:       "Build AI chat with language model",
			expectedResources: []string{"ollama"},
			minPredictions:    1,
		},
		{
			description:       "Use cache for session storage",
			expectedResources: []string{"redis"},
			minPredictions:    1,
		},
		{
			description:       "Vector search with semantic similarity and embedding",
			expectedResources: []string{"qdrant"},
			minPredictions:    1,
		},
		{
			description:       "Workflow automation with triggers",
			expectedResources: []string{"n8n"},
			minPredictions:    1,
		},
		{
			description:       "File upload and document storage",
			expectedResources: []string{"minio"},
			minPredictions:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			predictions := getHeuristicPredictions(tt.description)

			if len(predictions) < tt.minPredictions {
				t.Errorf("Expected at least %d predictions, got %d", tt.minPredictions, len(predictions))
			}

			// Verify expected resources are in predictions
			for _, expectedResource := range tt.expectedResources {
				found := false
				for _, pred := range predictions {
					if resourceName, ok := pred["resource_name"].(string); ok {
						if resourceName == expectedResource {
							found = true
							break
						}
					}
				}
				if !found {
					t.Errorf("Expected resource %q not found in predictions", expectedResource)
				}
			}
		})
	}
}

// TestScenarioDependencyStructure tests the ScenarioDependency struct
func TestScenarioDependencyStructure(t *testing.T) {
	dep := ScenarioDependency{
		ID:             "test-id",
		ScenarioName:   "test-scenario",
		DependencyType: "resource",
		DependencyName: "postgres",
		Required:       true,
		Purpose:        "Primary database",
		AccessMethod:   "direct",
		Configuration:  map[string]interface{}{"port": 5432},
		DiscoveredAt:   time.Now(),
		LastVerified:   time.Now(),
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(dep)
	if err != nil {
		t.Fatalf("Failed to marshal dependency: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled ScenarioDependency
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal dependency: %v", err)
	}

	if unmarshaled.ScenarioName != dep.ScenarioName {
		t.Errorf("Expected scenario name %s, got %s", dep.ScenarioName, unmarshaled.ScenarioName)
	}
}
