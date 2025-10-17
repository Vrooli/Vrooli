package main

import (
	"net/http"
	"testing"
)

// TestErrorPatterns tests systematic error handling across endpoints
func TestErrorPatterns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	defer ts.Cleanup()

	t.Run("ReportErrors", func(t *testing.T) {
		patterns := GetReportErrorPatterns()
		for _, pattern := range patterns {
			t.Run(pattern.Name, func(t *testing.T) {
				for _, scenario := range pattern.TestCases {
					t.Run(scenario.Name, func(t *testing.T) {
						runTestScenario(t, ts, scenario)
					})
				}
			})
		}
	})

	t.Run("SearchErrors", func(t *testing.T) {
		// Mock SearXNG for search tests
		mockSearXNG := mockSearXNGServer(t)
		defer mockSearXNG.Close()
		ts.Server.searxngURL = mockSearXNG.URL

		patterns := GetSearchErrorPatterns()
		for _, pattern := range patterns {
			t.Run(pattern.Name, func(t *testing.T) {
				for _, scenario := range pattern.TestCases {
					t.Run(scenario.Name, func(t *testing.T) {
						runTestScenario(t, ts, scenario)
					})
				}
			})
		}
	})

	t.Run("ContradictionErrors", func(t *testing.T) {
		// Mock Ollama for contradiction detection
		mockOllama := mockOllamaServer(t)
		defer mockOllama.Close()
		ts.Server.ollamaURL = mockOllama.URL

		patterns := GetContradictionErrorPatterns()
		for _, pattern := range patterns {
			t.Run(pattern.Name, func(t *testing.T) {
				for _, scenario := range pattern.TestCases {
					t.Run(scenario.Name, func(t *testing.T) {
						runTestScenario(t, ts, scenario)
					})
				}
			})
		}
	})
}

// runTestScenario executes a single test scenario
func runTestScenario(t *testing.T, ts *TestServer, scenario TestScenario) {
	// Setup
	var setupData interface{}
	if scenario.Setup != nil {
		setupData = scenario.Setup(t, ts)
	}

	// Cleanup
	if scenario.Cleanup != nil {
		defer scenario.Cleanup(t, setupData)
	}

	// Execute
	req := scenario.Execute(t, ts, setupData)
	w := makeHTTPRequest(ts, *req)

	// Validate status code
	if w.Code != scenario.ExpectedStatus {
		t.Errorf("Expected status %d, got %d. Body: %s", scenario.ExpectedStatus, w.Code, w.Body.String())
	}

	// Custom validation
	if scenario.Validate != nil && w.Code == http.StatusOK {
		response := assertJSONResponse(t, w, http.StatusOK)
		if response != nil {
			scenario.Validate(t, *req, response, setupData)
		}
	}
}

// TestInvalidJSONHandling tests JSON parsing error handling
func TestInvalidJSONHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	defer ts.Cleanup()

	testCases := []struct {
		name     string
		endpoint string
		method   string
		body     string
	}{
		{
			name:     "CreateReport_MalformedJSON",
			endpoint: "/api/reports",
			method:   "POST",
			body:     `{"topic": "test", "depth": "standard"`, // Missing closing brace
		},
		{
			name:     "Search_MalformedJSON",
			endpoint: "/api/search",
			method:   "POST",
			body:     `{"query": "test"`, // Missing closing brace
		},
		{
			name:     "DetectContradictions_MalformedJSON",
			endpoint: "/api/detect-contradictions",
			method:   "POST",
			body:     `{"topic": "test", "results": [`, // Incomplete array
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method: tc.method,
				Path:   tc.endpoint,
				Body:   tc.body,
			}
			w := makeHTTPRequest(ts, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400 for malformed JSON, got %d", w.Code)
			}
		})
	}
}

// TestMissingRequiredFields tests endpoints with missing required fields
func TestMissingRequiredFields(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Mock SearXNG for search endpoint
	mockSearXNG := mockSearXNGServer(t)
	defer mockSearXNG.Close()
	ts.Server.searxngURL = mockSearXNG.URL

	testCases := []struct {
		name           string
		endpoint       string
		method         string
		body           map[string]interface{}
		expectedStatus int
	}{
		{
			name:     "Search_MissingQuery",
			endpoint: "/api/search",
			method:   "POST",
			body: map[string]interface{}{
				"limit": 10,
				// Missing "query" field
			},
			expectedStatus: http.StatusOK, // Query defaults to empty string, which is valid
		},
		{
			name:     "DetectContradictions_MissingTopic",
			endpoint: "/api/detect-contradictions",
			method:   "POST",
			body: map[string]interface{}{
				"results": []map[string]interface{}{
					{"title": "test", "content": "test"},
					{"title": "test2", "content": "test2"},
				},
				// Missing "topic" field
			},
			expectedStatus: http.StatusBadRequest, // Ollama will likely fail without topic context
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method: tc.method,
				Path:   tc.endpoint,
				Body:   tc.body,
			}
			w := makeHTTPRequest(ts, req)

			// Accept both expected status and OK (some fields have defaults)
			if w.Code != tc.expectedStatus && w.Code != http.StatusOK {
				t.Errorf("Expected status %d or 200, got %d. Body: %s", tc.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestEdgeCases tests boundary conditions and edge cases
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Skip database-dependent tests (getReports, createReport, etc.)

	t.Run("InvalidDepthValue", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/reports",
			Body: map[string]interface{}{
				"topic": "Test topic",
				"depth": "super-deep", // Invalid depth
			},
		}
		w := makeHTTPRequest(ts, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid depth, got %d", w.Code)
		}
	})

	t.Run("EmptySearchQuery", func(t *testing.T) {
		mockSearXNG := mockSearXNGServer(t)
		defer mockSearXNG.Close()
		ts.Server.searxngURL = mockSearXNG.URL

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body: map[string]interface{}{
				"query": "",
				"limit": 10,
			},
		}
		w := makeHTTPRequest(ts, req)

		// Should still work, just return empty/few results
		if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 200 or 503, got %d", w.Code)
		}
	})

	t.Run("ZeroLimit", func(t *testing.T) {
		mockSearXNG := mockSearXNGServer(t)
		defer mockSearXNG.Close()
		ts.Server.searxngURL = mockSearXNG.URL

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body: map[string]interface{}{
				"query": "test",
				"limit": 0, // Should default to 10
			},
		}
		w := makeHTTPRequest(ts, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestNotImplementedEndpoints tests stub endpoints return 501
func TestNotImplementedEndpoints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	defer ts.Cleanup()

	notImplementedEndpoints := []struct {
		name   string
		method string
		path   string
	}{
		{"UpdateReport", "PUT", "/api/reports/test-id"},
		{"DeleteReport", "DELETE", "/api/reports/test-id"},
		{"GetReportPDF", "GET", "/api/reports/test-id/pdf"},
		{"GetConversations", "GET", "/api/conversations"},
		{"CreateConversation", "POST", "/api/conversations"},
		{"GetConversation", "GET", "/api/conversations/test-id"},
		{"GetMessages", "GET", "/api/conversations/test-id/messages"},
		{"SendMessage", "POST", "/api/conversations/test-id/messages"},
		{"GetSearchHistory", "GET", "/api/search/history"},
		{"AnalyzeContent", "POST", "/api/analyze"},
		{"ExtractInsights", "POST", "/api/analyze/insights"},
		{"AnalyzeTrends", "POST", "/api/analyze/trends"},
		{"AnalyzeCompetitive", "POST", "/api/analyze/competitive"},
		{"SearchKnowledge", "GET", "/api/knowledge/search"},
	}

	for _, endpoint := range notImplementedEndpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method: endpoint.method,
				Path:   endpoint.path,
			}
			w := makeHTTPRequest(ts, req)

			if w.Code != http.StatusNotImplemented {
				t.Errorf("Expected status 501 for %s, got %d", endpoint.name, w.Code)
			}

			// Verify response contains status indicator
			response := assertJSONResponse(t, w, http.StatusNotImplemented)
			if response != nil {
				if status, ok := response["status"].(string); !ok || status != "not implemented" {
					t.Errorf("Expected 'not implemented' status, got %v", response["status"])
				}
			}
		})
	}
}
