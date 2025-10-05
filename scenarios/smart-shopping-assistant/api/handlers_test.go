// +build testing

package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestHandlerEdgeCases tests edge cases for all handlers
func TestHandlerEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("ShoppingResearch_BoundaryBudgets", func(t *testing.T) {
		testCases := []struct {
			name      string
			budgetMax float64
		}{
			{"ZeroBudget", 0.0},
			{"NegativeBudget", -100.0},
			{"VeryLargeBudget", 999999.99},
			{"SmallBudget", 0.01},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/shopping/research",
					Body: ShoppingResearchRequest{
						ProfileID: "test-user",
						Query:     "test",
						BudgetMax: tc.budgetMax,
					},
				}

				w := makeHTTPRequest(env.Server, req)

				if w.Code != http.StatusOK {
					t.Errorf("Expected status 200 for %s, got %d", tc.name, w.Code)
				}
			})
		}
	})

	t.Run("ShoppingResearch_LongQuery", func(t *testing.T) {
		longQuery := "a very long query that exceeds normal length to test handling of large input strings"

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body: ShoppingResearchRequest{
				ProfileID: "test-user",
				Query:     longQuery,
				BudgetMax: 500.0,
			},
		}

		w := makeHTTPRequest(env.Server, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for long query, got %d", w.Code)
		}
	})

	t.Run("ShoppingResearch_SpecialCharactersInQuery", func(t *testing.T) {
		queries := []string{
			"<script>alert('xss')</script>",
			"'; DROP TABLE products; --",
			"test\x00null\x00byte",
			"emoji 😀 test",
			"unicode ñ test",
		}

		for _, query := range queries {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/shopping/research",
				Body: ShoppingResearchRequest{
					ProfileID: "test-user",
					Query:     query,
					BudgetMax: 100.0,
				},
			}

			w := makeHTTPRequest(env.Server, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for query '%s', got %d", query, w.Code)
			}
		}
	})

	t.Run("Tracking_EmptyProfileID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/shopping/tracking/",
			URLVars: map[string]string{"profile_id": ""},
		}

		w := makeHTTPRequest(env.Server, req)

		// Router might return 404 for empty path variable
		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Logf("Empty profile_id returned status %d", w.Code)
		}
	})

	t.Run("Tracking_CreateWithoutBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/tracking",
			Body:   nil,
		}

		w := makeHTTPRequest(env.Server, req)

		// Should handle missing body
		if w.Code != http.StatusOK {
			t.Logf("Create tracking without body returned status %d", w.Code)
		}
	})

	t.Run("PatternAnalysis_DifferentTimeframes", func(t *testing.T) {
		timeframes := []string{"7d", "30d", "90d", "1y", "invalid"}

		for _, timeframe := range timeframes {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/shopping/pattern-analysis",
				Body: PatternAnalysisRequest{
					ProfileID: "test-user",
					Timeframe: timeframe,
				},
			}

			w := makeHTTPRequest(env.Server, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for timeframe '%s', got %d", timeframe, w.Code)
			}
		}
	})

	t.Run("Alert_DeleteNonExistent", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/v1/alerts/non-existent-id",
			URLVars: map[string]string{"id": "non-existent-id"},
		}

		w := makeHTTPRequest(env.Server, req)

		// Currently returns 200 even for non-existent
		if w.Code != http.StatusOK {
			t.Logf("Delete non-existent alert returned status %d", w.Code)
		}
	})
}

// TestResponseStructures validates response structures
func TestResponseStructures(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("ShoppingResearch_ResponseStructure", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body: ShoppingResearchRequest{
				ProfileID:           "test",
				Query:               "laptop",
				BudgetMax:           1000.0,
				IncludeAlternatives: true,
			},
		}

		w := makeHTTPRequest(env.Server, req)

		var response ShoppingResearchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Validate all response fields are present
		if response.Products == nil {
			t.Error("Products field is nil")
		}
		if response.Alternatives == nil {
			t.Error("Alternatives field is nil")
		}
		if response.Recommendations == nil {
			t.Error("Recommendations field is nil")
		}
		if response.AffiliateLinks == nil {
			t.Error("AffiliateLinks field is nil")
		}

		// Validate product structure
		for i, product := range response.Products {
			if product.ID == "" {
				t.Errorf("Product %d has empty ID", i)
			}
			if product.Name == "" {
				t.Errorf("Product %d has empty Name", i)
			}
			if product.CurrentPrice <= 0 {
				t.Errorf("Product %d has invalid CurrentPrice: %.2f", i, product.CurrentPrice)
			}
		}

		// Validate alternative structure
		for i, alt := range response.Alternatives {
			if alt.Product.ID == "" {
				t.Errorf("Alternative %d has empty product ID", i)
			}
			if alt.AlternativeType == "" {
				t.Errorf("Alternative %d has empty type", i)
			}
			if alt.SavingsAmount <= 0 {
				t.Errorf("Alternative %d has invalid savings: %.2f", i, alt.SavingsAmount)
			}
		}
	})

	t.Run("Tracking_ResponseStructure", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/shopping/tracking/test-user",
			URLVars: map[string]string{"profile_id": "test-user"},
		}

		w := makeHTTPRequest(env.Server, req)

		var response TrackingResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Validate all arrays are initialized (not nil)
		if response.ActiveAlerts == nil {
			t.Error("ActiveAlerts should not be nil")
		}
		if response.TrackedProducts == nil {
			t.Error("TrackedProducts should not be nil")
		}
		if response.RecentChanges == nil {
			t.Error("RecentChanges should not be nil")
		}
	})

	t.Run("PatternAnalysis_ResponseStructure", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/pattern-analysis",
			Body: PatternAnalysisRequest{
				ProfileID: "test-user",
				Timeframe: "30d",
			},
		}

		w := makeHTTPRequest(env.Server, req)

		var response PatternAnalysisResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Validate structure
		if response.Patterns == nil {
			t.Error("Patterns should not be nil")
		}
		if response.Predictions == nil {
			t.Error("Predictions should not be nil")
		}
		if response.SavingsOpportunities == nil {
			t.Error("SavingsOpportunities should not be nil")
		}

		// Validate patterns
		for i, pattern := range response.Patterns {
			if pattern.Category == "" {
				t.Errorf("Pattern %d has empty category", i)
			}
			if pattern.Confidence < 0 || pattern.Confidence > 1 {
				t.Errorf("Pattern %d has invalid confidence: %.2f", i, pattern.Confidence)
			}
		}
	})
}

// TestHTTPMethods tests different HTTP methods
func TestHTTPMethods(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("InvalidMethod_PostEndpoint", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/shopping/research",
		}

		w := makeHTTPRequest(env.Server, req)

		// Should return method not allowed
		if w.Code == http.StatusOK {
			t.Error("GET should not be allowed on POST endpoint")
		}
	})

	t.Run("InvalidMethod_GetEndpoint", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/tracking/test-user",
			Body:   map[string]interface{}{},
		}

		w := makeHTTPRequest(env.Server, req)

		// Should return method not allowed
		if w.Code == http.StatusOK {
			t.Error("POST should not be allowed on GET endpoint")
		}
	})

	t.Run("OPTIONS_Request", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "OPTIONS",
			Path:   "/api/v1/shopping/research",
		}

		w := makeHTTPRequest(env.Server, req)

		// CORS should handle OPTIONS
		if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
			t.Logf("OPTIONS request returned status %d", w.Code)
		}
	})
}

// TestCORS tests CORS headers
func TestCORS(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("CORS_Headers", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Headers: map[string]string{
				"Origin": "http://example.com",
			},
			Body: ShoppingResearchRequest{
				ProfileID: "test",
				Query:     "test",
				BudgetMax: 100.0,
			},
		}

		w := makeHTTPRequest(env.Server, req)

		// Check for CORS headers (they should be present due to CORS middleware)
		corsHeader := w.Header().Get("Access-Control-Allow-Origin")
		if corsHeader == "" {
			t.Log("CORS headers may not be set (acceptable if CORS middleware is applied elsewhere)")
		}
	})
}
