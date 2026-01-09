package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	defer ts.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}
		w := makeHTTPRequest(ts, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		response := assertJSONResponse(t, w, http.StatusOK)
		if response == nil {
			return
		}

		// Verify response structure - accept healthy or degraded (test environment may not have all resources)
		if status, ok := response["status"].(string); !ok || (status != "healthy" && status != "degraded") {
			t.Errorf("Expected status 'healthy' or 'degraded', got %v", response["status"])
		}

		if _, ok := response["timestamp"]; !ok {
			t.Error("Expected timestamp in response")
		}

		if _, ok := response["services"]; !ok {
			t.Error("Expected services in response")
		}
	})
}

// TestGetTemplates tests the templates endpoint
func TestGetTemplates(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	defer ts.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/templates",
		}
		w := makeHTTPRequest(ts, req)

		response := assertJSONResponse(t, w, http.StatusOK)
		if response == nil {
			return
		}

		// Verify templates exist
		templates, ok := response["templates"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected templates map in response")
		}

		expectedTemplates := []string{"general", "academic", "market", "technical", "quick-brief"}
		for _, name := range expectedTemplates {
			if _, exists := templates[name]; !exists {
				t.Errorf("Expected template %q not found", name)
			}
		}

		// Verify count matches
		count, ok := response["count"].(float64)
		if !ok || int(count) != len(expectedTemplates) {
			t.Errorf("Expected count %d, got %v", len(expectedTemplates), response["count"])
		}
	})

	t.Run("TemplateStructure", func(t *testing.T) {
		templates := getReportTemplates()

		// Verify academic template
		academic := templates["academic"]
		if academic.Name != "Academic Research" {
			t.Errorf("Academic template name = %q, want %q", academic.Name, "Academic Research")
		}
		if academic.DefaultDepth != "deep" {
			t.Errorf("Academic template depth = %q, want %q", academic.DefaultDepth, "deep")
		}
		if len(academic.RequiredSections) < 5 {
			t.Errorf("Academic template should have at least 5 sections, got %d", len(academic.RequiredSections))
		}
	})
}

// TestGetDepthConfigs tests the depth configs endpoint
func TestGetDepthConfigs(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	defer ts.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/depth-configs",
		}
		w := makeHTTPRequest(ts, req)

		response := assertJSONResponse(t, w, http.StatusOK)
		if response == nil {
			return
		}

		// Verify depth configs exist
		configs, ok := response["depth_configs"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected depth_configs in response")
		}

		// Verify all three depth levels exist
		for _, depth := range []string{"quick", "standard", "deep"} {
			if _, exists := configs[depth]; !exists {
				t.Errorf("Expected depth config %q not found", depth)
			}
		}

		// Verify description
		if _, ok := response["description"]; !ok {
			t.Error("Expected description in response")
		}
	})
}

// TestGetDashboardStats tests the dashboard stats endpoint
func TestGetDashboardStats(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	defer ts.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/dashboard/stats",
		}
		w := makeHTTPRequest(ts, req)

		response := assertJSONResponse(t, w, http.StatusOK)
		if response == nil {
			return
		}

		// Verify required fields
		requiredFields := []string{
			"total_reports",
			"completed_this_month",
			"active_projects",
			"search_activity",
			"ai_insights",
			"knowledge_base",
		}

		for _, field := range requiredFields {
			if _, ok := response[field]; !ok {
				t.Errorf("Expected field %q in response", field)
			}
		}
	})
}

// TestGetCollections tests the collections endpoint
func TestGetCollections(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	defer ts.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/knowledge/collections",
		}
		w := makeHTTPRequest(ts, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Parse as array
		var collections []map[string]interface{}
		if err := parseJSONArray(w.Body.Bytes(), &collections); err != nil {
			t.Fatalf("Failed to parse collections: %v", err)
		}

		if len(collections) == 0 {
			t.Error("Expected at least one collection")
		}

		// Verify collection structure
		for _, collection := range collections {
			requiredFields := []string{"id", "name", "documents", "last_updated", "relevance"}
			for _, field := range requiredFields {
				if _, ok := collection[field]; !ok {
					t.Errorf("Collection missing field %q", field)
				}
			}
		}
	})
}

// TestPerformSearch tests the search endpoint
func TestPerformSearch(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Create mock SearXNG server
	mockSearXNG := mockSearXNGServer(t)
	defer mockSearXNG.Close()
	ts.Server.searxngURL = mockSearXNG.URL

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body: map[string]interface{}{
				"query":    "artificial intelligence",
				"limit":    5,
				"category": "general",
			},
		}
		w := makeHTTPRequest(ts, req)

		response := assertJSONResponse(t, w, http.StatusOK)
		if response == nil {
			return
		}

		// Verify response structure
		if _, ok := response["results"]; !ok {
			t.Error("Expected results in response")
		}
		if _, ok := response["results_count"]; !ok {
			t.Error("Expected results_count in response")
		}
		if _, ok := response["average_quality"]; !ok {
			t.Error("Expected average_quality in response")
		}
	})

	t.Run("WithAdvancedFilters", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body: map[string]interface{}{
				"query":        "machine learning",
				"language":     "en",
				"safe_search":  1,
				"file_type":    "pdf",
				"site":         "arxiv.org",
				"exclude_sites": []string{"wikipedia.org"},
				"sort_by":      "quality",
			},
		}
		w := makeHTTPRequest(ts, req)

		response := assertJSONResponse(t, w, http.StatusOK)
		if response == nil {
			return
		}

		// Verify filters were applied
		filters, ok := response["filters_applied"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected filters_applied in response")
		}

		assertEqual(t, filters["language"], "en", "language filter")
		assertEqual(t, filters["file_type"], "pdf", "file_type filter")
	})

	t.Run("QualityMetrics", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/search",
			Body: map[string]interface{}{
				"query": "research",
				"limit": 2,
			},
		}
		w := makeHTTPRequest(ts, req)

		response := assertJSONResponse(t, w, http.StatusOK)
		if response == nil {
			return
		}

		// Verify quality metrics are added to results
		results, ok := response["results"].([]interface{})
		if !ok || len(results) == 0 {
			t.Fatal("Expected results array")
		}

		firstResult, ok := results[0].(map[string]interface{})
		if !ok {
			t.Fatal("Expected result to be a map")
		}

		qualityMetrics, ok := firstResult["quality_metrics"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected quality_metrics in result")
		}

		// Verify quality metric fields
		requiredMetrics := []string{"domain_authority", "recency_score", "content_depth", "overall_quality"}
		for _, metric := range requiredMetrics {
			if _, ok := qualityMetrics[metric]; !ok {
				t.Errorf("Expected metric %q in quality_metrics", metric)
			}
		}
	})
}

// TestDetectContradictions tests the contradiction detection endpoint
func TestDetectContradictions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	ts := setupTestServer(t)
	defer ts.Cleanup()

	// Create mock Ollama server
	mockOllama := mockOllamaServer(t)
	defer mockOllama.Close()
	ts.Server.ollamaURL = mockOllama.URL

	t.Run("InsufficientResults", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/detect-contradictions",
			Body: map[string]interface{}{
				"topic": "AI",
				"results": []map[string]interface{}{
					{
						"title":   "Single result",
						"content": "Not enough for comparison",
						"url":     "https://example.com",
					},
				},
			},
		}
		w := makeHTTPRequest(ts, req)

		response := assertJSONResponse(t, w, http.StatusOK)
		if response == nil {
			return
		}

		// Should return empty contradictions with message
		contradictions, ok := response["contradictions"].([]interface{})
		if !ok {
			t.Fatal("Expected contradictions array")
		}
		if len(contradictions) != 0 {
			t.Error("Expected empty contradictions for single result")
		}
	})

	t.Run("TooManyResults", func(t *testing.T) {
		// Create 6 results (exceeds limit of 5)
		results := make([]map[string]interface{}, 6)
		for i := range results {
			results[i] = map[string]interface{}{
				"title":   "Result " + string(rune('A'+i)),
				"content": "Content " + string(rune('A'+i)),
				"url":     "https://example.com/" + string(rune('A'+i)),
			}
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/detect-contradictions",
			Body: map[string]interface{}{
				"topic":   "AI",
				"results": results,
			},
		}
		w := makeHTTPRequest(ts, req)

		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("ValidContradictionCheck", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/detect-contradictions",
			Body: map[string]interface{}{
				"topic": "AI Safety",
				"results": []map[string]interface{}{
					{
						"title":   "AI is completely safe",
						"content": "Artificial intelligence poses no risks to humanity",
						"url":     "https://example.com/safe",
					},
					{
						"title":   "AI presents serious risks",
						"content": "AI development carries significant existential risks",
						"url":     "https://example.com/risky",
					},
				},
			},
		}
		w := makeHTTPRequest(ts, req)

		response := assertJSONResponse(t, w, http.StatusOK)
		if response == nil {
			return
		}

		// Verify response structure
		if _, ok := response["contradictions"]; !ok {
			t.Error("Expected contradictions in response")
		}
		if _, ok := response["total_results"]; !ok {
			t.Error("Expected total_results in response")
		}
		if _, ok := response["warning"]; !ok {
			t.Error("Expected warning about synchronous processing")
		}
	})
}

// Helper to parse JSON array
func parseJSONArray(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
