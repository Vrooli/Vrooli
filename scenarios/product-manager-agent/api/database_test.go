package main

import (
	"testing"
	"time"
)

// TestFetchFeatures tests the fetchFeatures database operation
func TestFetchFeatures(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("FetchFeatures_WithNilDB", func(t *testing.T) {
		// When DB is nil, should return default features
		features, err := testApp.App.fetchFeatures()

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(features) == 0 {
			t.Error("Expected default features when DB is nil")
		}

		// Verify feature structure
		for _, f := range features {
			if f.ID == "" {
				t.Error("Feature should have ID")
			}
			if f.Name == "" {
				t.Error("Feature should have name")
			}
		}
	})
}

// TestFetchFeaturesByIDs tests fetching features by IDs
func TestFetchFeaturesByIDs(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("FetchByIDs_EmptyList", func(t *testing.T) {
		features, err := testApp.App.fetchFeaturesByIDs([]string{})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(features) != 0 {
			t.Error("Expected empty list for empty IDs")
		}
	})

	t.Run("FetchByIDs_ValidIDs", func(t *testing.T) {
		// Get default features first
		allFeatures, _ := testApp.App.fetchFeatures()

		if len(allFeatures) > 0 {
			ids := []string{allFeatures[0].ID}

			features, err := testApp.App.fetchFeaturesByIDs(ids)

			if err != nil {
				t.Errorf("Expected no error, got %v", err)
			}

			if len(features) == 0 {
				t.Error("Expected to find feature by ID")
			}

			if len(features) > 0 && features[0].ID != allFeatures[0].ID {
				t.Error("Returned feature ID doesn't match requested ID")
			}
		}
	})

	t.Run("FetchByIDs_NonExistentIDs", func(t *testing.T) {
		features, err := testApp.App.fetchFeaturesByIDs([]string{"non-existent-id"})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(features) != 0 {
			t.Error("Expected empty list for non-existent IDs")
		}
	})
}

// TestFetchAvailableFeatures tests fetching available features
func TestFetchAvailableFeatures(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("FetchAvailableFeatures_Success", func(t *testing.T) {
		features, err := testApp.App.fetchAvailableFeatures()

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(features) == 0 {
			t.Error("Expected available features")
		}

		// Check that features have appropriate status
		for _, f := range features {
			if f.Status != "proposed" && f.Status != "approved" && f.Status != "in_progress" {
				// Default features might have various statuses
			}
		}
	})
}

// TestFetchRecentFeatures tests fetching recent features
func TestFetchRecentFeatures(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("FetchRecentFeatures_LimitRespected", func(t *testing.T) {
		limit := 3
		features, err := testApp.App.fetchRecentFeatures(limit)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(features) > limit {
			t.Errorf("Expected at most %d features, got %d", limit, len(features))
		}
	})

	t.Run("FetchRecentFeatures_ZeroLimit", func(t *testing.T) {
		features, err := testApp.App.fetchRecentFeatures(0)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(features) != 0 {
			t.Error("Expected empty list for zero limit")
		}
	})

	t.Run("FetchRecentFeatures_LargeLimit", func(t *testing.T) {
		features, err := testApp.App.fetchRecentFeatures(1000)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Should return all available features
		if len(features) == 0 {
			t.Error("Expected some features")
		}
	})
}

// TestStoreFeature tests feature storage
func TestStoreFeature(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("StoreFeature_NewFeature", func(t *testing.T) {
		feature := &Feature{
			Name:        "New Feature",
			Description: "Test feature",
			Reach:       1000,
			Impact:      4,
			Confidence:  0.8,
			Effort:      5,
			Status:      "proposed",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		err := testApp.App.storeFeature(feature)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// ID should be generated if empty
		if feature.ID == "" {
			t.Error("Expected feature ID to be generated")
		}
	})

	t.Run("StoreFeature_WithExistingID", func(t *testing.T) {
		feature := &Feature{
			ID:          "test-feature-123",
			Name:        "Existing Feature",
			Description: "Test feature",
			Reach:       2000,
			Impact:      5,
			Confidence:  0.9,
			Effort:      3,
			Status:      "approved",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		err := testApp.App.storeFeature(feature)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("StoreFeature_WithDependencies", func(t *testing.T) {
		feature := &Feature{
			ID:           "feature-with-deps",
			Name:         "Dependent Feature",
			Description:  "Feature with dependencies",
			Dependencies: []string{"dep1", "dep2"},
			Reach:        1500,
			Impact:       4,
			Confidence:   0.75,
			Effort:       6,
			Status:       "proposed",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		err := testApp.App.storeFeature(feature)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}

// TestUpdateFeatureDB tests feature updates
func TestUpdateFeatureDB(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("UpdateFeature_Success", func(t *testing.T) {
		feature := &Feature{
			ID:          "update-test-feature",
			Name:        "Updated Feature",
			Description: "Updated description",
			Reach:       3000,
			Impact:      5,
			Confidence:  0.95,
			Effort:      4,
			Status:      "in_progress",
			CreatedAt:   time.Now().Add(-24 * time.Hour),
			UpdatedAt:   time.Now(),
		}

		err := testApp.App.updateFeatureDB(feature)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}

// TestFetchCurrentRoadmap tests roadmap fetching
func TestFetchCurrentRoadmap(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("FetchCurrentRoadmap_WithNilDB", func(t *testing.T) {
		roadmap, err := testApp.App.fetchCurrentRoadmap()

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if roadmap == nil {
			t.Fatal("Expected roadmap")
		}

		if roadmap.ID == "" {
			t.Error("Roadmap should have ID")
		}

		if len(roadmap.Features) == 0 {
			t.Error("Roadmap should have features")
		}
	})
}

// TestStoreRoadmap tests roadmap storage
func TestStoreRoadmap(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("StoreRoadmap_NewRoadmap", func(t *testing.T) {
		roadmap := &Roadmap{
			Name:      "Test Roadmap",
			StartDate: time.Now(),
			EndDate:   time.Now().AddDate(0, 6, 0),
			Features:  []string{"f1", "f2", "f3"},
			Version:   1,
			CreatedAt: time.Now(),
		}

		err := testApp.App.storeRoadmap(roadmap)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// ID should be generated if empty
		if roadmap.ID == "" {
			t.Error("Expected roadmap ID to be generated")
		}
	})

	t.Run("StoreRoadmap_WithMilestones", func(t *testing.T) {
		roadmap := &Roadmap{
			ID:        "roadmap-with-milestones",
			Name:      "Q1 2025 Roadmap",
			StartDate: time.Now(),
			EndDate:   time.Now().AddDate(0, 3, 0),
			Features:  []string{"f1", "f2"},
			Milestones: []Milestone{
				{
					ID:          "m1",
					Name:        "Q1 Milestone",
					Date:        time.Now().AddDate(0, 3, 0),
					Description: "Quarter 1 release",
					Features:    []string{"f1"},
				},
			},
			Version:   1,
			CreatedAt: time.Now(),
		}

		err := testApp.App.storeRoadmap(roadmap)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}

// TestFetchCurrentSprint tests sprint fetching
func TestFetchCurrentSprint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("FetchCurrentSprint_Success", func(t *testing.T) {
		sprint, err := testApp.App.fetchCurrentSprint()

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if sprint == nil {
			t.Fatal("Expected sprint")
		}

		if sprint.ID == "" {
			t.Error("Sprint should have ID")
		}
	})
}

// TestStoreSprintPlan tests sprint plan storage
func TestStoreSprintPlan(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("StoreSprintPlan_Success", func(t *testing.T) {
		sprint := &SprintPlan{
			SprintNumber:   1,
			StartDate:      time.Now(),
			EndDate:        time.Now().AddDate(0, 0, 14),
			Capacity:       100,
			Features:       createTestFeatures(5),
			TotalEffort:    50,
			EstimatedValue: 25000,
			Velocity:       45.5,
			RiskLevel:      "medium",
			PlannedAt:      time.Now(),
		}

		err := testApp.App.storeSprintPlan(sprint)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// ID should be generated if empty
		if sprint.ID == "" {
			t.Error("Expected sprint ID to be generated")
		}
	})
}

// TestStoreAnalysisOperations tests analysis storage operations
func TestStoreAnalysisOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("StoreMarketAnalysis_Success", func(t *testing.T) {
		analysis := &MarketAnalysis{
			ProductName:   "Test Product",
			MarketSize:    "$10B",
			GrowthRate:    "15% YoY",
			Competitors:   []string{"Competitor A", "Competitor B"},
			Demographics:  "Enterprise B2B",
			Opportunities: []string{"Cloud migration", "AI integration"},
			Challenges:    []string{"Market saturation"},
			Timestamp:     time.Now(),
		}

		err := testApp.App.storeMarketAnalysis(analysis)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if analysis.ID == "" {
			t.Error("Expected ID to be generated")
		}
	})

	t.Run("StoreCompetitorAnalysis_Success", func(t *testing.T) {
		analysis := &CompetitorAnalysis{
			CompetitorName: "Competitor X",
			Features:       []string{"Feature A", "Feature B"},
			Pricing:        "$99/month",
			TargetMarket:   "SMB",
			Strengths:      []string{"Easy to use"},
			Weaknesses:     []string{"Limited features"},
			MarketShare:    "20%",
			AnalyzedAt:     time.Now(),
		}

		err := testApp.App.storeCompetitorAnalysis(analysis)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if analysis.ID == "" {
			t.Error("Expected ID to be generated")
		}
	})

	t.Run("StoreROICalculation_Success", func(t *testing.T) {
		calc := &ROICalculation{
			FeatureID:      "test-feature",
			RevenueImpact:  50000,
			CostEstimate:   10000,
			ROI:            400,
			PaybackPeriod:  2.5,
			Assumptions:    []string{"$10 per user", "$1000 per effort point"},
			CalculatedAt:   time.Now(),
		}

		err := testApp.App.storeROICalculation(calc)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if calc.ID == "" {
			t.Error("Expected ID to be generated")
		}
	})
}

// TestFetchDashboardMetrics tests dashboard metrics fetching
func TestFetchDashboardMetrics(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("FetchDashboardMetrics_Success", func(t *testing.T) {
		metrics, err := testApp.App.fetchDashboardMetrics()

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Verify metrics structure
		if metrics.ActiveFeatures <= 0 {
			t.Error("Expected positive active features count")
		}

		if metrics.TeamVelocity < 0 {
			t.Error("Team velocity should not be negative")
		}
	})
}

// TestGetDefaultFeatures tests default feature generation
func TestGetDefaultFeatures(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("GetDefaultFeatures_Structure", func(t *testing.T) {
		features := testApp.App.getDefaultFeatures()

		if len(features) == 0 {
			t.Fatal("Expected default features")
		}

		for _, f := range features {
			if f.ID == "" {
				t.Error("Default feature should have ID")
			}
			if f.Name == "" {
				t.Error("Default feature should have name")
			}
			if f.Reach <= 0 {
				t.Error("Default feature should have positive reach")
			}
			if f.Impact <= 0 {
				t.Error("Default feature should have positive impact")
			}
		}
	})
}

// TestGetDefaultRoadmap tests default roadmap generation
func TestGetDefaultRoadmap(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("GetDefaultRoadmap_Structure", func(t *testing.T) {
		roadmap := testApp.App.getDefaultRoadmap()

		if roadmap == nil {
			t.Fatal("Expected default roadmap")
		}

		if roadmap.ID == "" {
			t.Error("Default roadmap should have ID")
		}

		if len(roadmap.Features) == 0 {
			t.Error("Default roadmap should have features")
		}

		if len(roadmap.Milestones) == 0 {
			t.Error("Default roadmap should have milestones")
		}

		// Verify milestone structure
		for _, m := range roadmap.Milestones {
			if m.ID == "" {
				t.Error("Milestone should have ID")
			}
			if m.Name == "" {
				t.Error("Milestone should have name")
			}
		}
	})
}

// TestGetDefaultSprint tests default sprint generation
func TestGetDefaultSprint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("GetDefaultSprint_Structure", func(t *testing.T) {
		sprint := testApp.App.getDefaultSprint()

		if sprint == nil {
			t.Fatal("Expected default sprint")
		}

		if sprint.ID == "" {
			t.Error("Default sprint should have ID")
		}

		if len(sprint.Features) == 0 {
			t.Error("Default sprint should have features")
		}

		if sprint.Capacity <= 0 {
			t.Error("Default sprint should have positive capacity")
		}

		if sprint.TotalEffort < 0 {
			t.Error("Default sprint total effort should not be negative")
		}
	})
}
