// +build testing

package main

import (
	"testing"
	"time"
)

// Test helper functions and data generation

func TestGenerateStrategicRecommendations(t *testing.T) {
	t.Run("Success - Generate recommendations with valid input", func(t *testing.T) {
		request := AnalysisRequest{
			CurrentResources: 5,
			TimeHorizon:      12,
			PrioritySectors:  []string{"software", "manufacturing"},
		}

		recommendations := generateStrategicRecommendations(request)

		if len(recommendations) == 0 {
			t.Error("Expected at least one recommendation")
		}

		// Validate recommendation structure
		for i, rec := range recommendations {
			if rec.Scenario == "" {
				t.Errorf("Recommendation %d has empty scenario name", i)
			}
			if rec.PriorityScore <= 0 {
				t.Errorf("Recommendation %d has invalid priority score: %f", i, rec.PriorityScore)
			}
			if rec.ImpactMultiplier <= 0 {
				t.Errorf("Recommendation %d has invalid impact multiplier: %f", i, rec.ImpactMultiplier)
			}
			if rec.Reasoning == "" {
				t.Errorf("Recommendation %d has empty reasoning", i)
			}
		}
	})

	t.Run("Edge case - Zero resources", func(t *testing.T) {
		request := AnalysisRequest{
			CurrentResources: 0,
			TimeHorizon:      6,
			PrioritySectors:  []string{},
		}

		recommendations := generateStrategicRecommendations(request)
		// Should still return recommendations
		if len(recommendations) == 0 {
			t.Error("Expected recommendations even with zero resources")
		}
	})

	t.Run("Edge case - High resources", func(t *testing.T) {
		request := AnalysisRequest{
			CurrentResources: 100,
			TimeHorizon:      24,
			PrioritySectors:  []string{"software", "healthcare", "finance"},
		}

		recommendations := generateStrategicRecommendations(request)
		if len(recommendations) == 0 {
			t.Error("Expected recommendations with high resources")
		}
	})
}

func TestCalculateProjectedTimeline(t *testing.T) {
	t.Run("Success - Generate timeline projections", func(t *testing.T) {
		request := AnalysisRequest{
			CurrentResources: 5,
			TimeHorizon:      12,
			PrioritySectors:  []string{"software"},
		}

		timeline := calculateProjectedTimeline(request)

		if len(timeline.Milestones) == 0 {
			t.Error("Expected at least one milestone")
		}

		// Validate milestone structure
		for i, milestone := range timeline.Milestones {
			if milestone.Name == "" {
				t.Errorf("Milestone %d has empty name", i)
			}
			if milestone.EstimatedCompletion.Before(time.Now()) {
				t.Errorf("Milestone %d has completion date in the past", i)
			}
			if milestone.Confidence < 0 || milestone.Confidence > 1 {
				t.Errorf("Milestone %d has invalid confidence: %f (expected 0-1)", i, milestone.Confidence)
			}
		}
	})

	t.Run("Edge case - Short time horizon", func(t *testing.T) {
		request := AnalysisRequest{
			CurrentResources: 3,
			TimeHorizon:      1,
			PrioritySectors:  []string{"software"},
		}

		timeline := calculateProjectedTimeline(request)
		if len(timeline.Milestones) == 0 {
			t.Error("Expected milestones even with short time horizon")
		}
	})

	t.Run("Edge case - Long time horizon", func(t *testing.T) {
		request := AnalysisRequest{
			CurrentResources: 8,
			TimeHorizon:      60,
			PrioritySectors:  []string{"software", "manufacturing", "healthcare"},
		}

		timeline := calculateProjectedTimeline(request)
		if len(timeline.Milestones) == 0 {
			t.Error("Expected milestones with long time horizon")
		}

		// Verify timeline extends into future
		lastMilestone := timeline.Milestones[len(timeline.Milestones)-1]
		if lastMilestone.EstimatedCompletion.Before(time.Now().AddDate(1, 0, 0)) {
			t.Error("Expected long-term milestones for extended time horizon")
		}
	})
}

func TestIdentifyBottlenecks(t *testing.T) {
	t.Run("Success - Identify bottlenecks", func(t *testing.T) {
		bottlenecks := identifyBottlenecks()

		if len(bottlenecks) == 0 {
			t.Error("Expected at least one bottleneck")
		}

		// Validate bottleneck descriptions
		for i, bottleneck := range bottlenecks {
			if bottleneck == "" {
				t.Errorf("Bottleneck %d has empty description", i)
			}
		}
	})
}

func TestAnalyzeCrossSectorImpacts(t *testing.T) {
	t.Run("Success - Analyze cross-sector impacts", func(t *testing.T) {
		impacts := analyzeCrossSectorImpacts()

		if len(impacts) == 0 {
			t.Error("Expected at least one cross-sector impact")
		}

		// Validate impact structure
		for i, impact := range impacts {
			if impact.SourceSector == "" {
				t.Errorf("Impact %d has empty source sector", i)
			}
			if impact.TargetSector == "" {
				t.Errorf("Impact %d has empty target sector", i)
			}
			if impact.ImpactScore < 0 || impact.ImpactScore > 1 {
				t.Errorf("Impact %d has invalid score: %f (expected 0-1)", i, impact.ImpactScore)
			}
			if impact.Description == "" {
				t.Errorf("Impact %d has empty description", i)
			}
		}
	})

	t.Run("Validate impact relationships", func(t *testing.T) {
		impacts := analyzeCrossSectorImpacts()

		// Check that source and target are different
		for i, impact := range impacts {
			if impact.SourceSector == impact.TargetSector {
				t.Errorf("Impact %d has same source and target sector: %s", i, impact.SourceSector)
			}
		}
	})
}

func TestAnalysisRequestValidation(t *testing.T) {
	t.Run("Valid analysis request", func(t *testing.T) {
		request := AnalysisRequest{
			CurrentResources: 5,
			TimeHorizon:      12,
			PrioritySectors:  []string{"software", "manufacturing"},
		}

		// Should not panic or error
		_ = generateStrategicRecommendations(request)
		_ = calculateProjectedTimeline(request)
	})

	t.Run("Negative resources", func(t *testing.T) {
		request := AnalysisRequest{
			CurrentResources: -5,
			TimeHorizon:      12,
			PrioritySectors:  []string{"software"},
		}

		// Should handle gracefully
		recommendations := generateStrategicRecommendations(request)
		if len(recommendations) == 0 {
			t.Error("Expected recommendations even with negative resources")
		}
	})

	t.Run("Negative time horizon", func(t *testing.T) {
		request := AnalysisRequest{
			CurrentResources: 5,
			TimeHorizon:      -12,
			PrioritySectors:  []string{"software"},
		}

		// Should handle gracefully
		timeline := calculateProjectedTimeline(request)
		if len(timeline.Milestones) == 0 {
			t.Error("Expected timeline even with negative time horizon")
		}
	})

	t.Run("Nil priority sectors", func(t *testing.T) {
		request := AnalysisRequest{
			CurrentResources: 5,
			TimeHorizon:      12,
			PrioritySectors:  nil,
		}

		// Should handle gracefully
		recommendations := generateStrategicRecommendations(request)
		if len(recommendations) == 0 {
			t.Error("Expected recommendations with nil priority sectors")
		}
	})
}

func TestDataStructures(t *testing.T) {
	t.Run("TechTree structure", func(t *testing.T) {
		tree := TechTree{
			ID:          "test-id",
			Name:        "Test Tree",
			Description: "Test Description",
			Version:     "1.0.0",
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if tree.ID == "" {
			t.Error("TechTree ID should not be empty")
		}
		if tree.Name == "" {
			t.Error("TechTree Name should not be empty")
		}
	})

	t.Run("Sector structure", func(t *testing.T) {
		sector := Sector{
			ID:                 "sector-id",
			TreeID:             "tree-id",
			Name:               "Software Engineering",
			Category:           "software",
			Description:        "Core software capabilities",
			ProgressPercentage: 45.5,
			PositionX:          100.0,
			PositionY:          200.0,
			Color:              "#3498db",
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}

		if sector.ProgressPercentage < 0 || sector.ProgressPercentage > 100 {
			t.Errorf("Invalid progress percentage: %f", sector.ProgressPercentage)
		}
	})

	t.Run("ProgressionStage structure", func(t *testing.T) {
		stage := ProgressionStage{
			ID:                 "stage-id",
			SectorID:           "sector-id",
			StageType:          "foundation",
			StageOrder:         1,
			Name:               "Foundation Stage",
			Description:        "Core foundation",
			ProgressPercentage: 30.0,
			PositionX:          150.0,
			PositionY:          250.0,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}

		if stage.StageOrder < 1 {
			t.Error("Stage order should be >= 1")
		}
	})

	t.Run("ScenarioMapping structure", func(t *testing.T) {
		mapping := ScenarioMapping{
			ID:                 "mapping-id",
			ScenarioName:       "test-scenario",
			StageID:            "stage-id",
			ContributionWeight: 0.8,
			CompletionStatus:   "in_progress",
			Priority:           1,
			EstimatedImpact:    7.5,
			LastStatusCheck:    time.Now(),
			Notes:              "Test notes",
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}

		if mapping.ContributionWeight < 0 || mapping.ContributionWeight > 1 {
			t.Errorf("Invalid contribution weight: %f", mapping.ContributionWeight)
		}

		validStatuses := []string{"not_started", "in_progress", "completed"}
		validStatus := false
		for _, status := range validStatuses {
			if mapping.CompletionStatus == status {
				validStatus = true
				break
			}
		}
		if !validStatus {
			t.Errorf("Invalid completion status: %s", mapping.CompletionStatus)
		}
	})

	t.Run("StrategicRecommendation structure", func(t *testing.T) {
		rec := StrategicRecommendation{
			Scenario:         "graph-studio",
			PriorityScore:    9.5,
			ImpactMultiplier: 3.2,
			Reasoning:        "Test reasoning",
		}

		if rec.Scenario == "" {
			t.Error("Scenario should not be empty")
		}
		if rec.PriorityScore <= 0 {
			t.Error("Priority score should be positive")
		}
		if rec.ImpactMultiplier <= 0 {
			t.Error("Impact multiplier should be positive")
		}
		if rec.Reasoning == "" {
			t.Error("Reasoning should not be empty")
		}
	})

	t.Run("CrossSectorImpact structure", func(t *testing.T) {
		impact := CrossSectorImpact{
			SourceSector: "Software Engineering",
			TargetSector: "Manufacturing",
			ImpactScore:  0.9,
			Description:  "Software enables manufacturing systems",
		}

		if impact.SourceSector == impact.TargetSector {
			t.Error("Source and target sectors should be different")
		}
		if impact.ImpactScore < 0 || impact.ImpactScore > 1 {
			t.Errorf("Invalid impact score: %f", impact.ImpactScore)
		}
	})
}

func TestRecommendationQuality(t *testing.T) {
	t.Run("Recommendations are prioritized", func(t *testing.T) {
		request := AnalysisRequest{
			CurrentResources: 5,
			TimeHorizon:      12,
			PrioritySectors:  []string{"software"},
		}

		recommendations := generateStrategicRecommendations(request)

		// Verify recommendations are sorted by priority (descending)
		for i := 1; i < len(recommendations); i++ {
			if recommendations[i].PriorityScore > recommendations[i-1].PriorityScore {
				t.Error("Recommendations should be sorted by priority score (descending)")
			}
		}
	})

	t.Run("Recommendations have reasonable values", func(t *testing.T) {
		request := AnalysisRequest{
			CurrentResources: 5,
			TimeHorizon:      12,
			PrioritySectors:  []string{"software"},
		}

		recommendations := generateStrategicRecommendations(request)

		for i, rec := range recommendations {
			// Priority score should be reasonable (0-10 scale)
			if rec.PriorityScore < 0 || rec.PriorityScore > 10 {
				t.Errorf("Recommendation %d has unrealistic priority score: %f", i, rec.PriorityScore)
			}

			// Impact multiplier should be reasonable (1-10 scale)
			if rec.ImpactMultiplier < 1 || rec.ImpactMultiplier > 10 {
				t.Errorf("Recommendation %d has unrealistic impact multiplier: %f", i, rec.ImpactMultiplier)
			}
		}
	})
}

func TestTimelineProjections(t *testing.T) {
	t.Run("Timeline milestones are chronological", func(t *testing.T) {
		request := AnalysisRequest{
			CurrentResources: 5,
			TimeHorizon:      12,
			PrioritySectors:  []string{"software"},
		}

		timeline := calculateProjectedTimeline(request)

		// Verify milestones are in chronological order
		for i := 1; i < len(timeline.Milestones); i++ {
			if timeline.Milestones[i].EstimatedCompletion.Before(timeline.Milestones[i-1].EstimatedCompletion) {
				t.Error("Timeline milestones should be in chronological order")
			}
		}
	})

	t.Run("Confidence decreases for later milestones", func(t *testing.T) {
		request := AnalysisRequest{
			CurrentResources: 5,
			TimeHorizon:      24,
			PrioritySectors:  []string{"software"},
		}

		timeline := calculateProjectedTimeline(request)

		// Later milestones should generally have lower confidence
		if len(timeline.Milestones) >= 2 {
			firstConfidence := timeline.Milestones[0].Confidence
			lastConfidence := timeline.Milestones[len(timeline.Milestones)-1].Confidence

			if lastConfidence > firstConfidence {
				t.Error("Expected confidence to decrease for later milestones")
			}
		}
	})
}
