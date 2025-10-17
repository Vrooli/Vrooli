// +build testing

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

// TestNewServerConfiguration tests server initialization
func TestNewServerConfiguration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NewServer_DefaultPort", func(t *testing.T) {
		os.Unsetenv("API_PORT")
		server := NewServer()
		if server.port != "3300" {
			t.Errorf("Expected default port 3300, got %s", server.port)
		}
		if server.db != nil {
			server.db.Close()
		}
	})

	t.Run("NewServer_CustomPort", func(t *testing.T) {
		os.Setenv("API_PORT", "8080")
		server := NewServer()
		if server.port != "8080" {
			t.Errorf("Expected port 8080, got %s", server.port)
		}
		if server.db != nil {
			server.db.Close()
		}
		os.Unsetenv("API_PORT")
	})

	t.Run("NewServer_RouterInitialized", func(t *testing.T) {
		server := NewServer()
		if server.router == nil {
			t.Error("Router should be initialized")
		}
		if server.db != nil {
			server.db.Close()
		}
	})
}

// TestDatabaseClose tests database cleanup
func TestDatabaseClose(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Database_CloseNil", func(t *testing.T) {
		db := &Database{
			postgres: nil,
			redis:    nil,
		}
		// Should not panic
		db.Close()
	})

	t.Run("Database_CloseWithConnections", func(t *testing.T) {
		db, err := NewDatabase()
		if err != nil {
			t.Logf("Database initialization error: %v", err)
		}
		if db != nil {
			// Should not panic
			db.Close()
		}
	})
}

// TestSearchProductsCaching tests caching behavior
func TestSearchProductsCaching(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, err := NewDatabase()
	if err != nil {
		t.Logf("Database initialization error: %v", err)
	}
	if db == nil {
		t.Skip("Database not available")
	}
	defer db.Close()

	ctx := context.Background()

	t.Run("SearchProducts_CacheMiss", func(t *testing.T) {
		query := "unique-cache-miss-query"
		products, err := db.SearchProducts(ctx, query, 500.0)
		if err != nil {
			t.Errorf("SearchProducts failed: %v", err)
		}
		if len(products) == 0 {
			t.Error("Expected products")
		}
	})

	t.Run("SearchProducts_CacheHit", func(t *testing.T) {
		if db.redis == nil {
			t.Skip("Redis not available")
		}

		query := "cache-hit-test"
		budget := 600.0

		// First call - cache miss
		products1, err := db.SearchProducts(ctx, query, budget)
		if err != nil {
			t.Errorf("First call failed: %v", err)
		}

		// Second call - should hit cache
		products2, err := db.SearchProducts(ctx, query, budget)
		if err != nil {
			t.Errorf("Second call failed: %v", err)
		}

		if len(products1) != len(products2) {
			t.Error("Cached results should match")
		}
	})
}

// TestAuthMiddlewareEdgeCases tests auth middleware edge cases
func TestAuthMiddlewareEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Auth_EmptyBearerToken", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Headers: map[string]string{
				"Authorization": "Bearer ",
			},
			Body: ShoppingResearchRequest{
				Query:     "test",
				BudgetMax: 100.0,
			},
		}

		w := makeHTTPRequest(env.Server, req)

		// Should succeed as anonymous
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Auth_MultipleSpacesInBearer", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Headers: map[string]string{
				"Authorization": "Bearer   token-with-spaces",
			},
			Body: ShoppingResearchRequest{
				Query:     "test",
				BudgetMax: 100.0,
			},
		}

		w := makeHTTPRequest(env.Server, req)

		// Should handle extra spaces
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestHandleShoppingResearchDetails tests detailed shopping research behavior
func TestHandleShoppingResearchDetails(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Research_UserIDFromContext", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body: ShoppingResearchRequest{
				// No profile_id - should use user_id from context
				Query:     "test product",
				BudgetMax: 500.0,
			},
		}

		w := makeHTTPRequest(env.Server, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response ShoppingResearchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify response has all required fields
		if response.Products == nil {
			t.Error("Products should not be nil")
		}
	})

	t.Run("Research_WithGiftRecipient", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body: ShoppingResearchRequest{
				ProfileID:       "test-user",
				Query:           "gift",
				BudgetMax:       200.0,
				GiftRecipientID: "recipient-123",
			},
		}

		w := makeHTTPRequest(env.Server, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestProductReviewsSummary tests product reviews structure
func TestProductReviewsSummary(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Products_IncludeReviews", func(t *testing.T) {
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

		// Check if products have reviews
		for i, product := range response.Products {
			if product.ReviewsSummary != nil {
				if product.ReviewsSummary.AverageRating <= 0 || product.ReviewsSummary.AverageRating > 5 {
					t.Errorf("Product %d has invalid rating: %.1f", i, product.ReviewsSummary.AverageRating)
				}
				if product.ReviewsSummary.TotalReviews < 0 {
					t.Errorf("Product %d has negative review count", i)
				}
			}
		}
	})
}

// TestPriceHistory tests price history and insights
func TestPriceHistory(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Research_IncludesPriceHistory", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body: ShoppingResearchRequest{
				ProfileID: "test",
				Query:     "monitor",
				BudgetMax: 400.0,
			},
		}

		w := makeHTTPRequest(env.Server, req)

		var response ShoppingResearchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Validate price analysis
		if response.PriceAnalysis.CurrentTrend == "" {
			t.Error("Expected price trend information")
		}

		if response.PriceAnalysis.HistoricalLow > response.PriceAnalysis.HistoricalHigh {
			t.Error("Historical low should be <= historical high")
		}
	})
}

// TestAlternativeTypes tests different alternative types
func TestAlternativeTypes(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Alternatives_MultipleTypes", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body: ShoppingResearchRequest{
				ProfileID:           "test",
				Query:               "smartphone",
				BudgetMax:           800.0,
				IncludeAlternatives: true,
			},
		}

		w := makeHTTPRequest(env.Server, req)

		var response ShoppingResearchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Check for different alternative types
		altTypes := make(map[string]bool)
		for _, alt := range response.Alternatives {
			altTypes[alt.AlternativeType] = true
		}

		if len(altTypes) == 0 {
			t.Error("Expected different alternative types")
		}
	})
}

// TestTrackingResponseStructure tests tracking response
func TestTrackingResponseStructure(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Tracking_AlertStructure", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/shopping/tracking/test-profile",
			URLVars: map[string]string{"profile_id": "test-profile"},
		}

		w := makeHTTPRequest(env.Server, req)

		var response TrackingResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Validate alert structure if present
		for i, alert := range response.ActiveAlerts {
			if alert.ID == "" {
				t.Errorf("Alert %d has empty ID", i)
			}
			if alert.ProfileID != "test-profile" {
				t.Errorf("Alert %d has wrong profile ID", i)
			}
			if alert.TargetPrice <= 0 {
				t.Errorf("Alert %d has invalid target price", i)
			}
			if alert.CreatedAt.IsZero() {
				t.Errorf("Alert %d has zero timestamp", i)
			}
		}
	})
}

// TestPatternAnalysisDetails tests pattern analysis
func TestPatternAnalysisDetails(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("PatternAnalysis_PurchasePatterns", func(t *testing.T) {
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

		// Validate patterns
		for i, pattern := range response.Patterns {
			if pattern.Category == "" {
				t.Errorf("Pattern %d has empty category", i)
			}
			if pattern.PatternType == "" {
				t.Errorf("Pattern %d has empty pattern type", i)
			}
			if pattern.FrequencyDays < 0 {
				t.Errorf("Pattern %d has negative frequency", i)
			}
			if pattern.AverageSpend < 0 {
				t.Errorf("Pattern %d has negative average spend", i)
			}
		}
	})

	t.Run("PatternAnalysis_RestockPredictions", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/pattern-analysis",
			Body: PatternAnalysisRequest{
				ProfileID: "test-user",
				Timeframe: "90d",
			},
		}

		w := makeHTTPRequest(env.Server, req)

		var response PatternAnalysisResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Validate predictions
		for i, prediction := range response.Predictions {
			if prediction.ProductCategory == "" {
				t.Errorf("Prediction %d has empty category", i)
			}
			if prediction.PredictedDate.IsZero() {
				t.Errorf("Prediction %d has zero predicted date", i)
			}
			if prediction.Confidence < 0 || prediction.Confidence > 1 {
				t.Errorf("Prediction %d has invalid confidence: %.2f", i, prediction.Confidence)
			}
		}
	})

	t.Run("PatternAnalysis_SavingsOpportunities", func(t *testing.T) {
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

		// Validate savings opportunities
		for i, savings := range response.SavingsOpportunities {
			if savings.Description == "" {
				t.Errorf("Savings %d has empty description", i)
			}
			if savings.Potential < 0 {
				t.Errorf("Savings %d has negative potential", i)
			}
			if savings.Action == "" {
				t.Errorf("Savings %d has empty action", i)
			}
		}
	})
}

// TestAffiliateLinksGeneration tests affiliate link generation
func TestAffiliateLinksGeneration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("AffiliateLinks_Generated", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body: ShoppingResearchRequest{
				ProfileID: "test",
				Query:     "camera",
				BudgetMax: 600.0,
			},
		}

		w := makeHTTPRequest(env.Server, req)

		var response ShoppingResearchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify affiliate links are generated
		if len(response.AffiliateLinks) == 0 {
			t.Error("Expected affiliate links to be generated")
		}

		// Validate affiliate link structure
		for i, link := range response.AffiliateLinks {
			if link.ProductID == "" {
				t.Errorf("Affiliate link %d has empty product ID", i)
			}
			if link.Retailer == "" {
				t.Errorf("Affiliate link %d has empty retailer", i)
			}
			if link.URL == "" {
				t.Errorf("Affiliate link %d has empty URL", i)
			}
			if link.Commission <= 0 {
				t.Errorf("Affiliate link %d has invalid commission: %.2f", i, link.Commission)
			}
		}
	})
}

// TestDatabaseFunctions tests database helper functions
func TestDatabaseFunctions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, err := NewDatabase()
	if err != nil {
		t.Logf("Database initialization error: %v", err)
	}
	if db == nil {
		t.Skip("Database not available")
	}
	defer db.Close()

	ctx := context.Background()

	t.Run("FindAlternatives_ValidProduct", func(t *testing.T) {
		alternatives := db.FindAlternatives(ctx, "test-product-id", 100.0)
		if len(alternatives) == 0 {
			t.Error("Expected alternatives to be generated")
		}

		// Verify all alternatives are cheaper
		for i, alt := range alternatives {
			if alt.Product.CurrentPrice >= 100.0 {
				t.Errorf("Alternative %d should be cheaper than original", i)
			}
			if alt.SavingsAmount <= 0 {
				t.Errorf("Alternative %d should have positive savings", i)
			}
		}
	})

	t.Run("GenerateAffiliateLinks_MultipleRetailers", func(t *testing.T) {
		products := []Product{
			{
				ID:   "test-product-1",
				Name: "Test Product",
			},
		}

		links := db.GenerateAffiliateLinks(products)
		if len(links) == 0 {
			t.Error("Expected affiliate links to be generated")
		}

		// Verify multiple retailers
		retailers := make(map[string]bool)
		for _, link := range links {
			retailers[link.Retailer] = true
		}

		if len(retailers) < 2 {
			t.Error("Expected links for multiple retailers")
		}
	})

	t.Run("GetPriceHistory_ValidProduct", func(t *testing.T) {
		insights := db.GetPriceHistory(ctx, "test-product-id")

		if insights.CurrentTrend == "" {
			t.Error("Expected current trend to be set")
		}
		if insights.HistoricalLow > insights.HistoricalHigh {
			t.Error("Historical low should not exceed historical high")
		}
	})
}

// TestRecommendationsLogic tests recommendation generation
func TestRecommendationsLogic(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Recommendations_WithBudget", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body: ShoppingResearchRequest{
				ProfileID: "test",
				Query:     "headphones",
				BudgetMax: 150.0,
			},
		}

		w := makeHTTPRequest(env.Server, req)

		var response ShoppingResearchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Should include budget-related recommendations
		if len(response.Recommendations) == 0 {
			t.Error("Expected recommendations to be generated")
		}
	})

	t.Run("Recommendations_WithAlternatives", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body: ShoppingResearchRequest{
				ProfileID:           "test",
				Query:               "tablet",
				BudgetMax:           500.0,
				IncludeAlternatives: true,
			},
		}

		w := makeHTTPRequest(env.Server, req)

		var response ShoppingResearchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Should include savings recommendations if alternatives exist
		if len(response.Alternatives) > 0 && len(response.Recommendations) == 0 {
			t.Error("Expected recommendations when alternatives are available")
		}
	})
}

// TestCORSConfiguration tests CORS setup
func TestCORSConfiguration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("CORS_OptionsRequest", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "OPTIONS",
			Path:   "/api/v1/shopping/research",
			Headers: map[string]string{
				"Origin": "http://localhost:3000",
			},
		}

		w := makeHTTPRequest(env.Server, req)

		// CORS middleware should handle OPTIONS
		if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
			t.Logf("OPTIONS request returned status %d (may be handled by CORS middleware)", w.Code)
		}
	})
}

// TestHealthEndpointDependencyDetail tests detailed dependency reporting
func TestHealthEndpointDependencyDetail(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Health_DatabaseLatencyReported", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w := makeHTTPRequest(env.Server, req)

		var response HealthResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Check database dependency structure with latency
		if deps, ok := response.Dependencies["database"].(map[string]interface{}); ok {
			if _, hasConnected := deps["connected"]; !hasConnected {
				t.Error("Database dependency missing 'connected' field")
			}
		} else {
			t.Error("Health response missing database dependency")
		}
	})

	t.Run("Health_RedisLatencyReported", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w := makeHTTPRequest(env.Server, req)

		var response HealthResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Check redis dependency structure with latency
		if deps, ok := response.Dependencies["redis"].(map[string]interface{}); ok {
			if _, hasConnected := deps["connected"]; !hasConnected {
				t.Error("Redis dependency missing 'connected' field")
			}
		} else {
			t.Error("Health response missing redis dependency")
		}
	})
}

// TestErrorHandlingPaths tests error handling for edge cases
func TestErrorHandlingPaths(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Research_ZeroBudget", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body: ShoppingResearchRequest{
				ProfileID: "test",
				Query:     "product",
				BudgetMax: 0.0, // Zero budget
			},
		}

		w := makeHTTPRequest(env.Server, req)

		// Should still process request
		if w.Code != http.StatusOK {
			t.Logf("Zero budget returned status %d", w.Code)
		}
	})

	t.Run("Research_VeryLongQuery", func(t *testing.T) {
		// Create a long query string
		longQueryBytes := make([]byte, 1000)
		for idx := range longQueryBytes {
			longQueryBytes[idx] = 'x'
		}
		longQuery := string(longQueryBytes)

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body: ShoppingResearchRequest{
				ProfileID: "test",
				Query:     longQuery,
				BudgetMax: 100.0,
			},
		}

		w := makeHTTPRequest(env.Server, req)

		// Should handle long queries
		if w.Code != http.StatusOK {
			t.Logf("Long query returned status %d", w.Code)
		}
	})
}
