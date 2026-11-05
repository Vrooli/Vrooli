package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestHealthHandler tests the health check endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w := makeHTTPRequest(t, testApp.App, req)

		var response map[string]interface{}
		assertJSONResponse(t, w, http.StatusServiceUnavailable, &response)

		if response["service"] != "product-manager-agent-api" {
			t.Errorf("Expected service name 'product-manager-agent-api', got %v", response["service"])
		}

		if _, ok := response["timestamp"]; !ok {
			t.Error("Expected timestamp in response")
		}
	})
}

// TestFeaturesHandler tests the features endpoint
func TestFeaturesHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("GET_Features", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/features",
		}

		w := makeHTTPRequest(t, testApp.App, req)

		var features []Feature
		assertJSONResponse(t, w, http.StatusOK, &features)

		if len(features) == 0 {
			t.Error("Expected default features, got empty list")
		}
	})

	t.Run("POST_CreateFeature", func(t *testing.T) {
		feature := createTestFeature("New Feature")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/features",
			Body:   feature,
		}

		w := makeHTTPRequest(t, testApp.App, req)

		var createdFeature Feature
		assertJSONResponse(t, w, http.StatusOK, &createdFeature)

		if createdFeature.Name != feature.Name {
			t.Errorf("Expected feature name '%s', got '%s'", feature.Name, createdFeature.Name)
		}

		if createdFeature.Score == 0 {
			t.Error("Expected RICE score to be calculated")
		}
	})

	t.Run("PUT_UpdateFeature", func(t *testing.T) {
		feature := createTestFeature("Update Feature")
		feature.Effort = 10 // Change effort to recalculate score

		req := HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/features",
			Body:   feature,
		}

		w := makeHTTPRequest(t, testApp.App, req)

		var updatedFeature Feature
		assertJSONResponse(t, w, http.StatusOK, &updatedFeature)

		if updatedFeature.Effort != 10 {
			t.Errorf("Expected effort 10, got %d", updatedFeature.Effort)
		}
	})

	t.Run("InvalidMethod", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/features",
		}

		w := makeHTTPRequest(t, testApp.App, req)
		assertErrorResponse(t, w, http.StatusMethodNotAllowed)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/features",
			Body:   `{"invalid": "json"`,
		}

		w := makeHTTPRequest(t, testApp.App, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestPrioritizeHandler tests the prioritization endpoint
func TestPrioritizeHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("PrioritizeByRICE", func(t *testing.T) {
		features := createTestFeatures(5)

		reqBody := map[string]interface{}{
			"features": features,
			"strategy": "rice",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/features/prioritize",
			Body:   reqBody,
		}

		w := makeHTTPRequest(t, testApp.App, req)

		var prioritized []Feature
		assertJSONResponse(t, w, http.StatusOK, &prioritized)

		if len(prioritized) != len(features) {
			t.Errorf("Expected %d features, got %d", len(features), len(prioritized))
		}

		// Verify features are sorted by score (descending)
		for i := 1; i < len(prioritized); i++ {
			if prioritized[i].Score > prioritized[i-1].Score {
				t.Error("Features not sorted by RICE score descending")
				break
			}
		}
	})

	t.Run("PrioritizeByValue", func(t *testing.T) {
		features := createTestFeatures(3)

		reqBody := map[string]interface{}{
			"features": features,
			"strategy": "value",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/features/prioritize",
			Body:   reqBody,
		}

		w := makeHTTPRequest(t, testApp.App, req)

		var prioritized []Feature
		assertJSONResponse(t, w, http.StatusOK, &prioritized)

		if len(prioritized) == 0 {
			t.Error("Expected prioritized features")
		}
	})

	t.Run("PrioritizeByEffort", func(t *testing.T) {
		features := createTestFeatures(3)

		reqBody := map[string]interface{}{
			"features": features,
			"strategy": "effort",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/features/prioritize",
			Body:   reqBody,
		}

		w := makeHTTPRequest(t, testApp.App, req)

		var prioritized []Feature
		assertJSONResponse(t, w, http.StatusOK, &prioritized)

		// Verify sorted by effort ascending
		for i := 1; i < len(prioritized); i++ {
			if prioritized[i].Effort < prioritized[i-1].Effort {
				t.Error("Features not sorted by effort ascending")
				break
			}
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/features/prioritize",
			Body:   `{"invalid": "json"`,
		}

		w := makeHTTPRequest(t, testApp.App, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestRICEScoreHandler tests the RICE score calculation endpoint
func TestRICEScoreHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("CalculateRICEScores", func(t *testing.T) {
		features := createTestFeatures(5)

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/features/rice",
			Body:   features,
		}

		w := makeHTTPRequest(t, testApp.App, req)

		var response map[string]interface{}
		assertJSONResponse(t, w, http.StatusOK, &response)

		prioritized, ok := response["prioritized_features"].([]interface{})
		if !ok {
			t.Fatal("Expected prioritized_features in response")
		}

		if len(prioritized) != len(features) {
			t.Errorf("Expected %d features, got %d", len(features), len(prioritized))
		}

		total, ok := response["total"].(float64)
		if !ok || int(total) != len(features) {
			t.Errorf("Expected total %d, got %v", len(features), total)
		}
	})

	t.Run("EmptyFeatures", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/features/rice",
			Body:   []Feature{},
		}

		w := makeHTTPRequest(t, testApp.App, req)

		var response map[string]interface{}
		assertJSONResponse(t, w, http.StatusOK, &response)
	})
}

// TestRoadmapHandler tests the roadmap endpoint
func TestRoadmapHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("GET_Roadmap", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/roadmap",
		}

		w := makeHTTPRequest(t, testApp.App, req)

		var roadmap Roadmap
		assertJSONResponse(t, w, http.StatusOK, &roadmap)

		if roadmap.ID == "" {
			t.Error("Expected roadmap with ID")
		}

		if len(roadmap.Features) == 0 {
			t.Error("Expected roadmap with features")
		}
	})

	t.Run("POST_CreateRoadmap", func(t *testing.T) {
		roadmap := Roadmap{
			Name:      "Test Roadmap",
			StartDate: time.Now(),
			EndDate:   time.Now().AddDate(0, 6, 0),
			Features:  []string{"f1", "f2", "f3"},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/roadmap",
			Body:   roadmap,
		}

		w := makeHTTPRequest(t, testApp.App, req)

		var created Roadmap
		assertJSONResponse(t, w, http.StatusOK, &created)

		if created.Name != roadmap.Name {
			t.Errorf("Expected roadmap name '%s', got '%s'", roadmap.Name, created.Name)
		}

		if created.Version != 1 {
			t.Errorf("Expected version 1, got %d", created.Version)
		}
	})

	t.Run("InvalidMethod", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/roadmap",
		}

		w := makeHTTPRequest(t, testApp.App, req)
		assertErrorResponse(t, w, http.StatusMethodNotAllowed)
	})
}

// TestGenerateRoadmapHandler tests the roadmap generation endpoint
func TestGenerateRoadmapHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("GenerateRoadmap", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"feature_ids":     []string{"f1", "f2", "f3"},
			"start_date":      time.Now(),
			"duration_months": 6,
			"team_capacity":   100,
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/roadmap/generate",
			Body:   reqBody,
		}

		w := makeHTTPRequest(t, testApp.App, req)

		var roadmap Roadmap
		assertJSONResponse(t, w, http.StatusOK, &roadmap)

		if roadmap.ID == "" {
			t.Error("Expected generated roadmap with ID")
		}

		if len(roadmap.Features) == 0 {
			t.Error("Expected features in generated roadmap")
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/roadmap/generate",
			Body:   `{"invalid": "json"`,
		}

		w := makeHTTPRequest(t, testApp.App, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestSprintPlanHandler tests the sprint planning endpoint
func TestSprintPlanHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("PlanSprint", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"team_size":      5,
			"velocity":       8,
			"duration_weeks": 2,
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/sprint/plan",
			Body:   reqBody,
		}

		w := makeHTTPRequest(t, testApp.App, req)

		var sprintPlan SprintPlan
		assertJSONResponse(t, w, http.StatusOK, &sprintPlan)

		expectedCapacity := 5 * 8 * 2 // team_size * velocity * duration_weeks
		if sprintPlan.Capacity != expectedCapacity {
			t.Errorf("Expected capacity %d, got %d", expectedCapacity, sprintPlan.Capacity)
		}

		if len(sprintPlan.Features) == 0 {
			t.Error("Expected features in sprint plan")
		}
	})

	t.Run("InvalidCapacity", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"team_size":      0,
			"velocity":       0,
			"duration_weeks": 0,
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/sprint/plan",
			Body:   reqBody,
		}

		w := makeHTTPRequest(t, testApp.App, req)

		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestCurrentSprintHandler tests the current sprint endpoint
func TestCurrentSprintHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("GetCurrentSprint", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/sprint/current",
		}

		w := makeHTTPRequest(t, testApp.App, req)

		var sprint SprintPlan
		assertJSONResponse(t, w, http.StatusOK, &sprint)

		if sprint.ID == "" {
			t.Error("Expected sprint with ID")
		}
	})
}

// TestMarketAnalysisHandler tests the market analysis endpoint
func TestMarketAnalysisHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("AnalyzeMarket", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"product_name": "Test Product",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/market/analyze",
			Body:   reqBody,
		}

		setOllamaJSONGenerateFixture(t, `{"market_size":"$1B","growth_rate":"10%","competitors":["Acme"],"demographics":"builders","opportunities":["automation"],"challenges":["competition"]}`, nil)
		w := makeHTTPRequest(t, testApp.App, req)
		var analysis MarketAnalysis
		assertJSONResponse(t, w, http.StatusOK, &analysis)
		if analysis.ProductName != "Test Product" || analysis.MarketSize != "$1B" || len(analysis.Competitors) != 1 {
			t.Fatalf("unexpected market analysis: %#v", analysis)
		}
	})

	t.Run("MissingProductName", func(t *testing.T) {
		reqBody := map[string]interface{}{}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/market/analyze",
			Body:   reqBody,
		}

		w := makeHTTPRequest(t, testApp.App, req)

		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("DependencyFailure", func(t *testing.T) {
		setOllamaJSONGenerateFixture(t, "", fmt.Errorf("fixture gateway unavailable"))
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/market/analyze",
			Body:   map[string]interface{}{"product_name": "Test Product"},
		}
		w := makeHTTPRequest(t, testApp.App, req)
		assertErrorResponse(t, w, http.StatusInternalServerError)
	})
}

// TestCompetitorAnalysisHandler tests the competitor analysis endpoint
func TestCompetitorAnalysisHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("AnalyzeCompetitor", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"competitor_name": "Competitor Inc",
			"depth":           "standard",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/competitor/analyze",
			Body:   reqBody,
		}

		setOllamaJSONGenerateFixture(t, `{"features":["search"],"pricing":"usage","target_market":"teams","strengths":["speed"],"weaknesses":["cost"],"market_share":"12%"}`, nil)
		w := makeHTTPRequest(t, testApp.App, req)
		var analysis CompetitorAnalysis
		assertJSONResponse(t, w, http.StatusOK, &analysis)
		if analysis.CompetitorName != "Competitor Inc" || analysis.Pricing != "usage" || len(analysis.Features) != 1 {
			t.Fatalf("unexpected competitor analysis: %#v", analysis)
		}
	})
}

// TestFeedbackAnalysisHandler tests the feedback analysis endpoint
func TestFeedbackAnalysisHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("AnalyzeFeedback", func(t *testing.T) {
		feedback := createTestFeedback(5)

		reqBody := map[string]interface{}{
			"feedback_items": feedback,
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/feedback/analyze",
			Body:   reqBody,
		}

		setOllamaJSONGenerateFixture(t, `{"sentiment":"positive","sentiment_score":90,"themes":["speed"],"feature_requests":["export"],"pain_points":[]}`, nil)
		w := makeHTTPRequest(t, testApp.App, req)
		var analysis FeedbackAnalysis
		assertJSONResponse(t, w, http.StatusOK, &analysis)
		if analysis.TotalItems != len(feedback) || analysis.Sentiment != "positive" || analysis.SentimentScore != 90 {
			t.Fatalf("unexpected feedback analysis: %#v", analysis)
		}
	})

	t.Run("EmptyFeedback", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"feedback_items": []FeedbackItem{},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/feedback/analyze",
			Body:   reqBody,
		}

		w := makeHTTPRequest(t, testApp.App, req)

		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestROICalculationHandler tests the ROI calculation endpoint
func TestROICalculationHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("CalculateROI", func(t *testing.T) {
		feature := createTestFeature("ROI Test Feature")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/roi/calculate",
			Body:   feature,
		}

		w := makeHTTPRequest(t, testApp.App, req)

		var calculation ROICalculation
		assertJSONResponse(t, w, http.StatusOK, &calculation)

		if calculation.FeatureID != feature.ID {
			t.Errorf("Expected feature ID '%s', got '%s'", feature.ID, calculation.FeatureID)
		}

		if calculation.ROI == 0 {
			t.Error("Expected ROI to be calculated")
		}

		if len(calculation.Assumptions) == 0 {
			t.Error("Expected assumptions in ROI calculation")
		}
	})

	t.Run("ZeroEffortFeature", func(t *testing.T) {
		feature := createTestFeature("Zero Effort")
		feature.Effort = 0

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/roi/calculate",
			Body:   feature,
		}

		w := makeHTTPRequest(t, testApp.App, req)

		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestDecisionAnalysisHandler tests the decision analysis endpoint
func TestDecisionAnalysisHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("AnalyzeDecision", func(t *testing.T) {
		decision := createTestDecision("Architecture Decision")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/decision/analyze",
			Body:   decision,
		}

		setOllamaJSONGenerateFixture(t, `{"options":[{"pros":["fast"],"cons":["cost"],"risk_level":"low","complexity":"medium","timeline":"2 weeks","success_probability":80,"recommendation_score":90}]}`, nil)
		w := makeHTTPRequest(t, testApp.App, req)
		var analysis DecisionAnalysis
		assertJSONResponse(t, w, http.StatusOK, &analysis)
		if analysis.DecisionID != decision.ID || len(analysis.Options) != len(decision.Options) || analysis.Options[0].Score != 90 {
			t.Fatalf("unexpected decision analysis: %#v", analysis)
		}
	})

	t.Run("NoOptions", func(t *testing.T) {
		decision := createTestDecision("No Options Decision")
		decision.Options = []DecisionOption{}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/decision/analyze",
			Body:   decision,
		}

		w := makeHTTPRequest(t, testApp.App, req)

		assertErrorResponse(t, w, http.StatusBadRequest)
	})
}

// TestDashboardHandler tests the dashboard endpoint
func TestDashboardHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("GetDashboard", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/dashboard",
		}

		w := makeHTTPRequest(t, testApp.App, req)

		var dashboard Dashboard
		assertJSONResponse(t, w, http.StatusOK, &dashboard)

		if dashboard.Metrics.ActiveFeatures == 0 {
			t.Error("Expected metrics in dashboard")
		}

		if dashboard.RecentFeatures == nil {
			t.Error("Expected recent features in dashboard")
		}

		if dashboard.CurrentSprint == nil {
			t.Error("Expected current sprint in dashboard")
		}

		if dashboard.Roadmap == nil {
			t.Error("Expected roadmap in dashboard")
		}
	})
}

// TestCalculateRICE tests the RICE score calculation
func TestCalculateRICE(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("ValidCalculation", func(t *testing.T) {
		feature := Feature{
			Reach:      1000,
			Impact:     5,
			Confidence: 0.8,
			Effort:     4,
		}

		score := testApp.App.calculateRICE(&feature)

		expected := (1000.0 * 5.0 * 0.8) / 4.0
		if score != expected {
			t.Errorf("Expected RICE score %f, got %f", expected, score)
		}
	})

	t.Run("ZeroEffort", func(t *testing.T) {
		feature := Feature{
			Reach:      1000,
			Impact:     5,
			Confidence: 0.8,
			Effort:     0,
		}

		score := testApp.App.calculateRICE(&feature)

		// Should set effort to 1 to prevent division by zero
		if feature.Effort != 1 {
			t.Errorf("Expected effort to be set to 1, got %d", feature.Effort)
		}

		if score == 0 {
			t.Error("Expected non-zero score even with zero effort")
		}
	})
}

// TestDeterminePriority tests the priority determination
func TestDeterminePriority(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	tests := []struct {
		score    float64
		expected string
	}{
		{150.0, "CRITICAL"},
		{75.0, "HIGH"},
		{30.0, "MEDIUM"},
		{10.0, "LOW"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			priority := testApp.App.determinePriority(tt.score)
			if priority != tt.expected {
				t.Errorf("For score %f, expected priority '%s', got '%s'",
					tt.score, tt.expected, priority)
			}
		})
	}
}

// TestSortFeaturesByRICE tests feature sorting
func TestSortFeaturesByRICE(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	features := createTestFeatures(5)

	sorted := testApp.App.sortFeaturesByRICE(features)

	// Verify descending order
	for i := 1; i < len(sorted); i++ {
		if sorted[i].Score > sorted[i-1].Score {
			t.Errorf("Features not sorted correctly: score[%d]=%f > score[%d]=%f",
				i, sorted[i].Score, i-1, sorted[i-1].Score)
		}
	}
}

// TestCORSMiddleware tests CORS handling
func TestCORSMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("OptionsRequest", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "OPTIONS",
			Path:   "/api/features",
		}

		w := makeHTTPRequest(t, testApp.App, req)

		// CORS middleware is applied in the real server, not in makeHTTPRequest
		// This test verifies that the featuresHandler handles OPTIONS correctly
		// In the real handler, the corsMiddleware wraps it
		// For now, we expect the handler to return Method Not Allowed for OPTIONS
		// unless it's handled by the CORS middleware wrapper
		if w.Code != http.StatusOK && w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 200 or 405 for OPTIONS, got %d", w.Code)
		}
	})
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("EmptyFeaturesList", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/features/prioritize",
			Body: map[string]interface{}{
				"features": []Feature{},
				"strategy": "rice",
			},
		}

		w := makeHTTPRequest(t, testApp.App, req)

		var prioritized []Feature
		assertJSONResponse(t, w, http.StatusOK, &prioritized)

		if len(prioritized) != 0 {
			t.Error("Expected empty list for empty input")
		}
	})

	t.Run("LargeFeatureList", func(t *testing.T) {
		features := createTestFeatures(100)

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/features/rice",
			Body:   features,
		}

		w := makeHTTPRequest(t, testApp.App, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected successful handling of large list, got status %d", w.Code)
		}
	})
}
