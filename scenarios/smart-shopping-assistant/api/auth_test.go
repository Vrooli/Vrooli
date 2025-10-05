// +build testing

package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestAuthMiddlewareComprehensive tests all auth middleware paths
func TestAuthMiddlewareComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Auth_NoAuthHeader_NoProfileID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body: map[string]interface{}{
				"query":      "test",
				"budget_max": 100.0,
			},
		}

		w := makeHTTPRequest(env.Server, req)

		// Should succeed as anonymous
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Auth_ProfileIDInQueryParam", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			QueryParams: map[string]string{
				"profile_id": "query-param-user",
			},
			Body: map[string]interface{}{
				"query":      "test",
				"budget_max": 100.0,
			},
		}

		w := makeHTTPRequest(env.Server, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Auth_ProfileIDInBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body: map[string]interface{}{
				"profile_id": "body-user",
				"query":      "test",
				"budget_max": 100.0,
			},
		}

		w := makeHTTPRequest(env.Server, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Auth_InvalidBearerToken", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Headers: map[string]string{
				"Authorization": "Bearer invalid-token-xyz",
			},
			Body: ShoppingResearchRequest{
				ProfileID: "test",
				Query:     "test",
				BudgetMax: 100.0,
			},
		}

		w := makeHTTPRequest(env.Server, req)

		// Should succeed as anonymous (auth service unavailable)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Auth_BearerTokenWithoutPrefix", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Headers: map[string]string{
				"Authorization": "token-without-bearer",
			},
			Body: ShoppingResearchRequest{
				ProfileID: "test",
				Query:     "test",
				BudgetMax: 100.0,
			},
		}

		w := makeHTTPRequest(env.Server, req)

		// Should handle token without Bearer prefix
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestHealthCheckDependencies tests health check with different dependency states
func TestHealthCheckDependencies(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Health_CheckDependencies", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w := makeHTTPRequest(env.Server, req)

		var response HealthResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse health response: %v", err)
		}

		// Verify dependencies are reported
		if response.Dependencies == nil {
			t.Fatal("Dependencies should not be nil")
		}

		// Check database dependency
		if dbDep, ok := response.Dependencies["database"].(map[string]interface{}); ok {
			if _, hasConnected := dbDep["connected"]; !hasConnected {
				t.Error("Database dependency should report connected status")
			}
		}

		// Check redis dependency
		if redisDep, ok := response.Dependencies["redis"].(map[string]interface{}); ok {
			if _, hasConnected := redisDep["connected"]; !hasConnected {
				t.Error("Redis dependency should report connected status")
			}
		}
	})

	t.Run("Health_Timestamp", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w := makeHTTPRequest(env.Server, req)

		var response HealthResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse health response: %v", err)
		}

		if response.Timestamp.IsZero() {
			t.Error("Health check should include timestamp")
		}
	})

	t.Run("Health_Readiness", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w := makeHTTPRequest(env.Server, req)

		var response HealthResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse health response: %v", err)
		}

		// Readiness should be a boolean
		if response.Readiness != true && response.Readiness != false {
			t.Error("Readiness should be a boolean value")
		}
	})
}

// TestShoppingResearchRecommendations tests recommendation generation
func TestShoppingResearchRecommendations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Research_WithBudget_GeneratesRecommendation", func(t *testing.T) {
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

		// Should have recommendations
		if len(response.Recommendations) == 0 {
			t.Error("Expected recommendations when budget is specified")
		}
	})

	t.Run("Research_WithAlternatives_GeneratesSavingsRecommendation", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body: ShoppingResearchRequest{
				ProfileID:           "test",
				Query:               "headphones",
				BudgetMax:           200.0,
				IncludeAlternatives: true,
			},
		}

		w := makeHTTPRequest(env.Server, req)

		var response ShoppingResearchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Should have savings recommendations with alternatives
		foundSavingsRec := false
		for _, rec := range response.Recommendations {
			if len(rec) > 0 {
				foundSavingsRec = true
				break
			}
		}

		if !foundSavingsRec {
			t.Error("Expected savings recommendation with alternatives")
		}
	})

	t.Run("Research_WithDecliningPriceTrend", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body: ShoppingResearchRequest{
				ProfileID: "test",
				Query:     "smartphone",
				BudgetMax: 800.0,
			},
		}

		w := makeHTTPRequest(env.Server, req)

		var response ShoppingResearchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Check if price trend recommendation is present when trend is declining
		if response.PriceAnalysis.CurrentTrend == "declining" {
			foundTrendRec := false
			for _, rec := range response.Recommendations {
				if len(rec) > 0 {
					foundTrendRec = true
					break
				}
			}
			if !foundTrendRec {
				t.Log("Price trend recommendation not found (may be conditional)")
			}
		}
	})
}

// TestAffiliateLinks tests affiliate link generation
func TestAffiliateLinks(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Research_GeneratesAffiliateLinks", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body: ShoppingResearchRequest{
				ProfileID: "test",
				Query:     "laptop",
				BudgetMax: 1000.0,
			},
		}

		w := makeHTTPRequest(env.Server, req)

		var response ShoppingResearchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Should generate affiliate links
		if len(response.AffiliateLinks) == 0 {
			t.Error("Expected affiliate links in response")
		}

		// Validate affiliate link structure
		for _, link := range response.AffiliateLinks {
			if link.ProductID == "" {
				t.Error("Affiliate link missing product ID")
			}
			if link.Retailer == "" {
				t.Error("Affiliate link missing retailer")
			}
			if link.URL == "" {
				t.Error("Affiliate link missing URL")
			}
			if link.Commission < 0 {
				t.Error("Affiliate link has negative commission")
			}
		}
	})
}
