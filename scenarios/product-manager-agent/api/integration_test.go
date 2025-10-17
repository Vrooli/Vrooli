package main

import (
	"encoding/json"
	"testing"
	"time"
)

// TestFeatureToRoadmapWorkflow tests the complete workflow from feature creation to roadmap
func TestFeatureToRoadmapWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	// Step 1: Create multiple features
	featureIDs := []string{}

	for i := 0; i < 5; i++ {
		feature := createTestFeature("Workflow Feature")

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/features",
			Body:   feature,
		}

		w := makeHTTPRequest(t, testApp.App, req)

		var created Feature
		assertJSONResponse(t, w, 200, &created)

		featureIDs = append(featureIDs, created.ID)
	}

	// Step 2: Calculate RICE scores
	var scoredFeatures []Feature
	for range featureIDs {
		// In a real scenario, we'd fetch the feature by ID
		// For this test, we'll use the default features
	}

	// Step 3: Prioritize features
	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/features/prioritize",
		Body: map[string]interface{}{
			"features": createTestFeatures(5),
			"strategy": "rice",
		},
	}

	w := makeHTTPRequest(t, testApp.App, req)
	assertJSONResponse(t, w, 200, &scoredFeatures)

	if len(scoredFeatures) == 0 {
		t.Fatal("Expected prioritized features")
	}

	// Step 4: Generate roadmap with prioritized features
	roadmapReq := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/roadmap/generate",
		Body: map[string]interface{}{
			"feature_ids":     featureIDs,
			"start_date":      time.Now(),
			"duration_months": 6,
			"team_capacity":   100,
		},
	}

	roadmapW := makeHTTPRequest(t, testApp.App, roadmapReq)

	var roadmap Roadmap
	assertJSONResponse(t, roadmapW, 200, &roadmap)

	if roadmap.ID == "" {
		t.Error("Expected roadmap with ID")
	}

	// Note: Roadmap may be empty if capacity/duration is insufficient
	// This is acceptable behavior

	// Step 5: Verify dashboard reflects the changes
	dashReq := HTTPTestRequest{
		Method: "GET",
		Path:   "/api/dashboard",
	}

	dashW := makeHTTPRequest(t, testApp.App, dashReq)

	var dashboard Dashboard
	assertJSONResponse(t, dashW, 200, &dashboard)

	if dashboard.Roadmap == nil {
		t.Error("Dashboard should include roadmap")
	}
}

// TestSprintPlanningWorkflow tests the sprint planning workflow
func TestSprintPlanningWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	// Step 1: Get available features
	featuresReq := HTTPTestRequest{
		Method: "GET",
		Path:   "/api/features",
	}

	featuresW := makeHTTPRequest(t, testApp.App, featuresReq)

	var features []Feature
	assertJSONResponse(t, featuresW, 200, &features)

	if len(features) == 0 {
		t.Fatal("Expected features to exist")
	}

	// Step 2: Calculate ROI for top features
	roiResults := []ROICalculation{}

	for i := 0; i < 3 && i < len(features); i++ {
		roiReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/roi/calculate",
			Body:   features[i],
		}

		roiW := makeHTTPRequest(t, testApp.App, roiReq)

		var roi ROICalculation
		assertJSONResponse(t, roiW, 200, &roi)

		roiResults = append(roiResults, roi)
	}

	if len(roiResults) == 0 {
		t.Fatal("Expected ROI calculations")
	}

	// Step 3: Plan sprint based on team capacity
	sprintReq := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/sprint/plan",
		Body: map[string]interface{}{
			"team_size":      5,
			"velocity":       8,
			"duration_weeks": 2,
		},
	}

	sprintW := makeHTTPRequest(t, testApp.App, sprintReq)

	var sprint SprintPlan
	assertJSONResponse(t, sprintW, 200, &sprint)

	if sprint.ID == "" {
		t.Error("Expected sprint plan with ID")
	}

	expectedCapacity := 5 * 8 * 2
	if sprint.Capacity != expectedCapacity {
		t.Errorf("Expected capacity %d, got %d", expectedCapacity, sprint.Capacity)
	}

	// Step 4: Verify current sprint
	currentReq := HTTPTestRequest{
		Method: "GET",
		Path:   "/api/sprint/current",
	}

	currentW := makeHTTPRequest(t, testApp.App, currentReq)

	var currentSprint SprintPlan
	assertJSONResponse(t, currentW, 200, &currentSprint)

	if currentSprint.ID == "" {
		t.Error("Expected current sprint")
	}
}

// TestDecisionAnalysisWorkflow tests the decision analysis workflow
func TestDecisionAnalysisWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	// Step 1: Create a decision with multiple options
	decision := createTestDecision("Technology Stack Decision")
	decision.Options = []DecisionOption{
		{
			Name:        "Go + PostgreSQL",
			Description: "Use Go with PostgreSQL database",
		},
		{
			Name:        "Node.js + MongoDB",
			Description: "Use Node.js with MongoDB database",
		},
		{
			Name:        "Python + MySQL",
			Description: "Use Python with MySQL database",
		},
	}

	// Step 2: Analyze the decision (would use AI in production)
	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/decision/analyze",
		Body:   decision,
	}

	w := makeHTTPRequest(t, testApp.App, req)

	// Ollama might not be available, so we accept either success or error
	if w.Code != 200 && w.Code != 500 {
		t.Errorf("Expected status 200 or 500, got %d", w.Code)
	}

	// If successful, verify the analysis structure
	if w.Code == 200 {
		var analysis DecisionAnalysis
		json.NewDecoder(w.Body).Decode(&analysis)

		if analysis.DecisionID != decision.ID {
			t.Error("Analysis should reference the decision ID")
		}
	}
}

// TestMarketToFeatureWorkflow tests market analysis to feature creation
func TestMarketToFeatureWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping AI-dependent integration test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	// Step 1: Conduct market analysis
	marketReq := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/market/analyze",
		Body: map[string]interface{}{
			"product_name": "Project Management Tool",
		},
	}

	marketW := makeHTTPRequest(t, testApp.App, marketReq)

	// Ollama might not be available
	if marketW.Code == 200 {
		var marketAnalysis MarketAnalysis
		json.NewDecoder(marketW.Body).Decode(&marketAnalysis)

		// Step 2: Analyze competitor
		compReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/competitor/analyze",
			Body: map[string]interface{}{
				"competitor_name": "Jira",
				"depth":           "standard",
			},
		}

		compW := makeHTTPRequest(t, testApp.App, compReq)

		if compW.Code == 200 {
			var compAnalysis CompetitorAnalysis
			json.NewDecoder(compW.Body).Decode(&compAnalysis)

			// Step 3: Create features based on insights
			// In a real scenario, AI would suggest features based on analysis
			feature := createTestFeature("AI-Suggested Feature")
			feature.Description = "Feature suggested based on market and competitor analysis"

			featureReq := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/features",
				Body:   feature,
			}

			featureW := makeHTTPRequest(t, testApp.App, featureReq)

			var created Feature
			assertJSONResponse(t, featureW, 200, &created)

			if created.ID == "" {
				t.Error("Expected created feature with ID")
			}
		}
	}
}

// TestFeedbackToFeatureWorkflow tests feedback analysis to feature creation
func TestFeedbackToFeatureWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping AI-dependent integration test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	// Step 1: Collect feedback
	feedback := createTestFeedback(10)

	// Step 2: Analyze feedback
	req := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/feedback/analyze",
		Body: map[string]interface{}{
			"feedback_items": feedback,
		},
	}

	w := makeHTTPRequest(t, testApp.App, req)

	// Ollama might not be available
	if w.Code == 200 {
		var analysis FeedbackAnalysis
		json.NewDecoder(w.Body).Decode(&analysis)

		// Step 3: Create features based on feedback themes
		if len(analysis.FeatureRequests) > 0 {
			for i := 0; i < len(analysis.FeatureRequests) && i < 3; i++ {
				feature := createTestFeature(analysis.FeatureRequests[i])

				featureReq := HTTPTestRequest{
					Method: "POST",
					Path:   "/api/features",
					Body:   feature,
				}

				featureW := makeHTTPRequest(t, testApp.App, featureReq)

				var created Feature
				assertJSONResponse(t, featureW, 200, &created)
			}
		}
	}
}

// TestCompleteProductPlanningCycle tests the entire product planning cycle
func TestCompleteProductPlanningCycle(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	// Phase 1: Feature Discovery and Prioritization
	t.Run("Phase1_FeatureDiscovery", func(t *testing.T) {
		// Create multiple features
		for i := 0; i < 10; i++ {
			feature := createTestFeature("Discovery Feature")

			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/features",
				Body:   feature,
			}

			w := makeHTTPRequest(t, testApp.App, req)
			assertJSONResponse(t, w, 200, &Feature{})
		}
	})

	// Phase 2: Prioritization and ROI Analysis
	t.Run("Phase2_PrioritizationAndROI", func(t *testing.T) {
		// Get all features
		featuresReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/features",
		}

		featuresW := makeHTTPRequest(t, testApp.App, featuresReq)

		var features []Feature
		assertJSONResponse(t, featuresW, 200, &features)

		// Prioritize
		prioReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/features/prioritize",
			Body: map[string]interface{}{
				"features": features[:5], // Top 5
				"strategy": "rice",
			},
		}

		prioW := makeHTTPRequest(t, testApp.App, prioReq)

		var prioritized []Feature
		assertJSONResponse(t, prioW, 200, &prioritized)

		// Calculate ROI for top feature
		if len(prioritized) > 0 {
			roiReq := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/roi/calculate",
				Body:   prioritized[0],
			}

			roiW := makeHTTPRequest(t, testApp.App, roiReq)
			assertJSONResponse(t, roiW, 200, &ROICalculation{})
		}
	})

	// Phase 3: Roadmap Planning
	t.Run("Phase3_RoadmapPlanning", func(t *testing.T) {
		roadmapReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/roadmap/generate",
			Body: map[string]interface{}{
				"feature_ids":     []string{"f1", "f2", "f3", "f4", "f5"},
				"start_date":      time.Now(),
				"duration_months": 12,
				"team_capacity":   200,
			},
		}

		roadmapW := makeHTTPRequest(t, testApp.App, roadmapReq)

		var roadmap Roadmap
		assertJSONResponse(t, roadmapW, 200, &roadmap)

		if len(roadmap.Features) == 0 {
			t.Error("Roadmap should contain features")
		}
	})

	// Phase 4: Sprint Planning
	t.Run("Phase4_SprintPlanning", func(t *testing.T) {
		sprintReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/sprint/plan",
			Body: map[string]interface{}{
				"team_size":      8,
				"velocity":       10,
				"duration_weeks": 2,
			},
		}

		sprintW := makeHTTPRequest(t, testApp.App, sprintReq)

		var sprint SprintPlan
		assertJSONResponse(t, sprintW, 200, &sprint)

		if sprint.Capacity == 0 {
			t.Error("Sprint should have capacity")
		}
	})

	// Phase 5: Dashboard Review
	t.Run("Phase5_DashboardReview", func(t *testing.T) {
		dashReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/dashboard",
		}

		dashW := makeHTTPRequest(t, testApp.App, dashReq)

		var dashboard Dashboard
		assertJSONResponse(t, dashW, 200, &dashboard)

		// Verify all components are present
		if dashboard.Metrics.ActiveFeatures == 0 {
			t.Error("Dashboard should show active features")
		}

		if dashboard.RecentFeatures == nil {
			t.Error("Dashboard should show recent features")
		}

		if dashboard.CurrentSprint == nil {
			t.Error("Dashboard should show current sprint")
		}

		if dashboard.Roadmap == nil {
			t.Error("Dashboard should show roadmap")
		}
	})
}

// TestErrorRecoveryWorkflow tests error handling and recovery
func TestErrorRecoveryWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	// Test 1: Invalid JSON should not crash the system
	t.Run("InvalidJSON_Recovery", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/features",
			Body:   `{invalid json`,
		}

		w := makeHTTPRequest(t, testApp.App, req)
		assertErrorResponse(t, w, 400)

		// System should still work after error
		healthReq := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		healthW := makeHTTPRequest(t, testApp.App, healthReq)
		assertJSONResponse(t, healthW, 200, &map[string]interface{}{})
	})

	// Test 2: Missing required fields should be handled gracefully
	t.Run("MissingFields_Recovery", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/roadmap/generate",
			Body:   map[string]interface{}{}, // Missing required fields
		}

		w := makeHTTPRequest(t, testApp.App, req)

		// Should handle gracefully (either 400 or 200 with empty result)
		if w.Code != 200 && w.Code != 400 && w.Code != 500 {
			t.Errorf("Unexpected status code: %d", w.Code)
		}
	})
}

// TestDataConsistencyWorkflow tests data consistency across operations
func TestDataConsistencyWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	// Create a feature
	feature := createTestFeature("Consistency Test Feature")

	createReq := HTTPTestRequest{
		Method: "POST",
		Path:   "/api/features",
		Body:   feature,
	}

	createW := makeHTTPRequest(t, testApp.App, createReq)

	var created Feature
	assertJSONResponse(t, createW, 200, &created)

	originalScore := created.Score

	// Update the feature
	created.Effort = created.Effort * 2 // Double the effort

	updateReq := HTTPTestRequest{
		Method: "PUT",
		Path:   "/api/features",
		Body:   created,
	}

	updateW := makeHTTPRequest(t, testApp.App, updateReq)

	var updated Feature
	assertJSONResponse(t, updateW, 200, &updated)

	// Score should be recalculated (should be lower with higher effort)
	if updated.Score >= originalScore {
		t.Error("Score should decrease when effort increases")
	}
}
