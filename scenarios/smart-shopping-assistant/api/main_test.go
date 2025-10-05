// +build testing

package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestMain(m *testing.M) {
	// Setup test environment
	cleanup := setupTestLogger()
	defer cleanup()

	// Run tests
	m.Run()
}

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("HealthCheck_Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w := makeHTTPRequest(env.Server, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response HealthResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse health response: %v", err)
		}

		if response.Service != "smart-shopping-assistant" {
			t.Errorf("Expected service 'smart-shopping-assistant', got %s", response.Service)
		}

		if response.Status != "healthy" && response.Status != "degraded" {
			t.Errorf("Expected status 'healthy' or 'degraded', got %s", response.Status)
		}

		// Verify dependencies structure
		if response.Dependencies == nil {
			t.Error("Expected dependencies map in health response")
		}
	})

	t.Run("HealthCheck_ContainsVersion", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w := makeHTTPRequest(env.Server, req)

		var response HealthResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse health response: %v", err)
		}

		if response.Version == "" {
			t.Error("Expected version in health response")
		}

		// Accept both healthy and degraded status (database may not be available)
		if response.Status != "healthy" && response.Status != "degraded" {
			t.Errorf("Expected status 'healthy' or 'degraded', got %s", response.Status)
		}
	})
}

// TestShoppingResearch tests the shopping research endpoint
func TestShoppingResearch(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("ShoppingResearch_ValidRequest", func(t *testing.T) {
		requestBody := TestData.ShoppingResearchRequest("laptop", 1000.00)

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body:   requestBody,
		}

		w := makeHTTPRequest(env.Server, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
			return
		}

		var response ShoppingResearchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Validate response structure
		if len(response.Products) == 0 {
			t.Error("Expected at least one product in response")
		}

		if response.PriceAnalysis.HistoricalLow == 0 {
			t.Error("Expected price analysis data")
		}

		// Validate products within budget
		for _, product := range response.Products {
			if product.CurrentPrice > requestBody.BudgetMax {
				t.Errorf("Product price %.2f exceeds budget %.2f", product.CurrentPrice, requestBody.BudgetMax)
			}
		}
	})

	t.Run("ShoppingResearch_MissingQuery", func(t *testing.T) {
		requestBody := ShoppingResearchRequest{
			ProfileID: "test-user-1",
			Query:     "", // Missing query
			BudgetMax: 500.00,
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body:   requestBody,
		}

		w := makeHTTPRequest(env.Server, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("ShoppingResearch_InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body:   `{"invalid": json}`,
		}

		w := makeHTTPRequest(env.Server, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("ShoppingResearch_WithAlternatives", func(t *testing.T) {
		requestBody := TestData.ShoppingResearchRequest("headphones", 200.00)
		requestBody.IncludeAlternatives = true

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body:   requestBody,
		}

		w := makeHTTPRequest(env.Server, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
			return
		}

		var response ShoppingResearchResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(response.Alternatives) == 0 {
			t.Error("Expected alternatives when IncludeAlternatives is true")
		}

		// Validate alternatives offer savings
		for _, alt := range response.Alternatives {
			if alt.SavingsAmount <= 0 {
				t.Errorf("Expected positive savings amount, got %.2f", alt.SavingsAmount)
			}
		}
	})

	t.Run("ShoppingResearch_WithAuthentication", func(t *testing.T) {
		requestBody := TestData.ShoppingResearchRequest("monitor", 300.00)

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body:   requestBody,
			Headers: map[string]string{
				"Authorization": "Bearer test-token",
			},
		}

		w := makeHTTPRequest(env.Server, req)

		// Should succeed even with invalid token (fallback to anonymous)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestTrackingEndpoints tests price tracking endpoints
func TestTrackingEndpoints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("GetTracking_ValidProfileID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/shopping/tracking/test-profile-1",
			URLVars: map[string]string{"profile_id": "test-profile-1"},
		}

		w := makeHTTPRequest(env.Server, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
			return
		}

		var response TrackingResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Validate response structure
		if response.ActiveAlerts == nil {
			t.Error("Expected active_alerts array in response")
		}
		if response.TrackedProducts == nil {
			t.Error("Expected tracked_products array in response")
		}
		if response.RecentChanges == nil {
			t.Error("Expected recent_changes array in response")
		}
	})

	t.Run("CreateTracking_Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/tracking",
			Body: map[string]interface{}{
				"profile_id": "test-user-1",
				"product_id": "prod-123",
			},
		}

		w := makeHTTPRequest(env.Server, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestPatternAnalysis tests the pattern analysis endpoint
func TestPatternAnalysis(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("PatternAnalysis_ValidRequest", func(t *testing.T) {
		requestBody := TestData.PatternAnalysisRequest("test-user-1")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/pattern-analysis",
			Body:   requestBody,
		}

		w := makeHTTPRequest(env.Server, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
			return
		}

		var response PatternAnalysisResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Validate response structure
		if response.Patterns == nil {
			t.Error("Expected patterns array in response")
		}
		if response.Predictions == nil {
			t.Error("Expected predictions array in response")
		}
		if response.SavingsOpportunities == nil {
			t.Error("Expected savings_opportunities array in response")
		}
	})

	t.Run("PatternAnalysis_InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/pattern-analysis",
			Body:   `{"invalid": json`,
		}

		w := makeHTTPRequest(env.Server, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestProfileEndpoints tests profile management endpoints
func TestProfileEndpoints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("GetProfiles_Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/profiles",
		}

		w := makeHTTPRequest(env.Server, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("CreateProfile_Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/profiles",
			Body: map[string]interface{}{
				"name": "Test Profile",
			},
		}

		w := makeHTTPRequest(env.Server, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("GetProfile_ByID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/profiles/test-id-123",
			URLVars: map[string]string{"id": "test-id-123"},
		}

		w := makeHTTPRequest(env.Server, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestAlertEndpoints tests price alert endpoints
func TestAlertEndpoints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("GetAlerts_Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/alerts",
		}

		w := makeHTTPRequest(env.Server, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("CreateAlert_Success", func(t *testing.T) {
		alertReq := TestData.PriceAlertRequest("prod-456", 99.99)

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/alerts",
			Body:   alertReq,
		}

		w := makeHTTPRequest(env.Server, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("DeleteAlert_Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/v1/alerts/alert-123",
			URLVars: map[string]string{"id": "alert-123"},
		}

		w := makeHTTPRequest(env.Server, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestAuthMiddleware tests the authentication middleware
func TestAuthMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("Auth_WithBearerToken", func(t *testing.T) {
		requestBody := TestData.ShoppingResearchRequest("keyboard", 150.00)

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body:   requestBody,
			Headers: map[string]string{
				"Authorization": "Bearer test-token-123",
			},
		}

		w := makeHTTPRequest(env.Server, req)

		// Should succeed (falls back to anonymous if auth service unavailable)
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Auth_WithProfileID", func(t *testing.T) {
		requestBody := TestData.ShoppingResearchRequest("mouse", 50.00)

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body:   requestBody,
		}

		w := makeHTTPRequest(env.Server, req)

		// Should succeed with profile_id in body
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Auth_Anonymous", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"query":     "tablet",
			"budget_max": 400.00,
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body:   requestBody,
		}

		w := makeHTTPRequest(env.Server, req)

		// Should succeed as anonymous
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestErrorScenarios uses the TestScenarioBuilder pattern
func TestErrorScenarios(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	// Build comprehensive error scenarios
	scenarios := NewTestScenarioBuilder().
		AddInvalidJSON("/api/v1/shopping/research").
		AddMissingRequiredField("/api/v1/shopping/research", "query").
		AddEmptyBody("/api/v1/shopping/research").
		AddInvalidJSON("/api/v1/shopping/pattern-analysis").
		Build()

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			req := HTTPTestRequest{
				Method:      scenario.Method,
				Path:        scenario.Path,
				Body:        scenario.Body,
				URLVars:     scenario.URLVars,
				QueryParams: scenario.QueryParams,
				Headers:     scenario.Headers,
			}

			w := makeHTTPRequest(env.Server, req)

			if w.Code != scenario.ExpectedStatus {
				t.Logf("Scenario: %s", scenario.Description)
				t.Errorf("Expected status %d, got %d. Response: %s",
					scenario.ExpectedStatus, w.Code, w.Body.String())
			}
		})
	}
}

// TestDatabaseIntegration tests database functionality
func TestDatabaseIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	if env.Server.db == nil {
		t.Skip("Database not available for testing")
	}

	t.Run("Database_SearchProducts", func(t *testing.T) {
		// Test will run if database is available
		requestBody := TestData.ShoppingResearchRequest("camera", 800.00)

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/shopping/research",
			Body:   requestBody,
		}

		w := makeHTTPRequest(env.Server, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}
