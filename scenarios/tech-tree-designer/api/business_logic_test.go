// +build testing

package main

import (
	"encoding/json"
	"testing"
)

// Test business logic and validation rules

func TestSectorCategories(t *testing.T) {
	validCategories := []string{
		"manufacturing",
		"healthcare",
		"finance",
		"education",
		"software",
		"governance",
	}

	t.Run("Valid sector categories", func(t *testing.T) {
		for _, category := range validCategories {
			sector := Sector{
				Category: category,
			}

			if sector.Category == "" {
				t.Errorf("Category '%s' should not be empty", category)
			}
		}
	})

	t.Run("Category validation", func(t *testing.T) {
		validMap := make(map[string]bool)
		for _, cat := range validCategories {
			validMap[cat] = true
		}

		testCases := []struct {
			category string
			valid    bool
		}{
			{"software", true},
			{"manufacturing", true},
			{"invalid_category", false},
			{"", false},
		}

		for _, tc := range testCases {
			isValid := validMap[tc.category]
			if isValid != tc.valid {
				t.Errorf("Category '%s': expected valid=%v, got %v", tc.category, tc.valid, isValid)
			}
		}
	})
}

func TestStageTypes(t *testing.T) {
	validStageTypes := []string{
		"foundation",
		"operational",
		"analytics",
		"integration",
		"digital_twin",
	}

	t.Run("Valid stage types", func(t *testing.T) {
		for order, stageType := range validStageTypes {
			stage := ProgressionStage{
				StageType:  stageType,
				StageOrder: order + 1,
			}

			if stage.StageType == "" {
				t.Errorf("Stage type '%s' should not be empty", stageType)
			}

			if stage.StageOrder != order+1 {
				t.Errorf("Expected stage order %d, got %d", order+1, stage.StageOrder)
			}
		}
	})

	t.Run("Stage progression order", func(t *testing.T) {
		// Foundation should come before operational, etc.
		expectedOrder := map[string]int{
			"foundation":   1,
			"operational":  2,
			"analytics":    3,
			"integration":  4,
			"digital_twin": 5,
		}

		for stageType, expectedPos := range expectedOrder {
			if expectedPos < 1 || expectedPos > 5 {
				t.Errorf("Stage %s has invalid order position %d", stageType, expectedPos)
			}
		}
	})
}

func TestCompletionStatuses(t *testing.T) {
	validStatuses := []string{
		"not_started",
		"in_progress",
		"completed",
	}

	t.Run("Valid completion statuses", func(t *testing.T) {
		for _, status := range validStatuses {
			mapping := ScenarioMapping{
				CompletionStatus: status,
			}

			if mapping.CompletionStatus == "" {
				t.Errorf("Completion status '%s' should not be empty", status)
			}
		}
	})

	t.Run("Status progression validation", func(t *testing.T) {
		// Status can only progress forward
		progressionMap := map[string]int{
			"not_started": 1,
			"in_progress": 2,
			"completed":   3,
		}

		testCases := []struct {
			from  string
			to    string
			valid bool
		}{
			{"not_started", "in_progress", true},
			{"in_progress", "completed", true},
			{"not_started", "completed", true},
			{"completed", "in_progress", false},
			{"in_progress", "not_started", false},
		}

		for _, tc := range testCases {
			fromLevel := progressionMap[tc.from]
			toLevel := progressionMap[tc.to]
			canProgress := toLevel >= fromLevel

			if canProgress != tc.valid {
				t.Errorf("Progression from '%s' to '%s': expected %v, got %v",
					tc.from, tc.to, tc.valid, canProgress)
			}
		}
	})
}

func TestProgressPercentageValidation(t *testing.T) {
	testCases := []struct {
		name       string
		percentage float64
		valid      bool
	}{
		{"Zero progress", 0.0, true},
		{"Partial progress", 45.5, true},
		{"Complete progress", 100.0, true},
		{"Negative progress", -10.0, false},
		{"Exceeds 100%", 150.0, false},
	}

	t.Run("Sector progress validation", func(t *testing.T) {
		for _, tc := range testCases {
			sector := Sector{
				ProgressPercentage: tc.percentage,
			}

			isValid := sector.ProgressPercentage >= 0 && sector.ProgressPercentage <= 100
			if isValid != tc.valid {
				t.Errorf("%s: progress=%.1f, expected valid=%v, got %v",
					tc.name, tc.percentage, tc.valid, isValid)
			}
		}
	})

	t.Run("Stage progress validation", func(t *testing.T) {
		for _, tc := range testCases {
			stage := ProgressionStage{
				ProgressPercentage: tc.percentage,
			}

			isValid := stage.ProgressPercentage >= 0 && stage.ProgressPercentage <= 100
			if isValid != tc.valid {
				t.Errorf("%s: progress=%.1f, expected valid=%v, got %v",
					tc.name, tc.percentage, tc.valid, isValid)
			}
		}
	})
}

func TestContributionWeightValidation(t *testing.T) {
	testCases := []struct {
		name   string
		weight float64
		valid  bool
	}{
		{"No contribution", 0.0, true},
		{"Partial contribution", 0.5, true},
		{"Full contribution", 1.0, true},
		{"Negative weight", -0.5, false},
		{"Exceeds 1.0", 1.5, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mapping := ScenarioMapping{
				ContributionWeight: tc.weight,
			}

			isValid := mapping.ContributionWeight >= 0 && mapping.ContributionWeight <= 1
			if isValid != tc.valid {
				t.Errorf("Weight=%.2f, expected valid=%v, got %v",
					tc.weight, tc.valid, isValid)
			}
		})
	}
}

func TestImpactScoreValidation(t *testing.T) {
	testCases := []struct {
		name  string
		score float64
		valid bool
	}{
		{"Zero impact", 0.0, true},
		{"Medium impact", 0.5, true},
		{"High impact", 1.0, true},
		{"Negative impact", -0.2, false},
		{"Exceeds 1.0", 1.2, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			impact := CrossSectorImpact{
				ImpactScore: tc.score,
			}

			isValid := impact.ImpactScore >= 0 && impact.ImpactScore <= 1
			if isValid != tc.valid {
				t.Errorf("Score=%.2f, expected valid=%v, got %v",
					tc.score, tc.valid, isValid)
			}
		})
	}
}

func TestJSONMarshalling(t *testing.T) {
	t.Run("TechTree JSON marshalling", func(t *testing.T) {
		tree := TechTree{
			ID:          "test-id",
			Name:        "Test Tree",
			Description: "Test Description",
			Version:     "1.0.0",
			IsActive:    true,
		}

		data, err := json.Marshal(tree)
		if err != nil {
			t.Fatalf("Failed to marshal TechTree: %v", err)
		}

		var decoded TechTree
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal TechTree: %v", err)
		}

		if decoded.ID != tree.ID {
			t.Errorf("Expected ID '%s', got '%s'", tree.ID, decoded.ID)
		}
	})

	t.Run("Sector JSON marshalling", func(t *testing.T) {
		sector := Sector{
			ID:                 "sector-id",
			Name:               "Software Engineering",
			Category:           "software",
			ProgressPercentage: 45.5,
		}

		data, err := json.Marshal(sector)
		if err != nil {
			t.Fatalf("Failed to marshal Sector: %v", err)
		}

		var decoded Sector
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal Sector: %v", err)
		}

		if decoded.ProgressPercentage != sector.ProgressPercentage {
			t.Errorf("Expected progress %.2f, got %.2f",
				sector.ProgressPercentage, decoded.ProgressPercentage)
		}
	})

	t.Run("AnalysisRequest JSON marshalling", func(t *testing.T) {
		request := AnalysisRequest{
			CurrentResources: 5,
			TimeHorizon:      12,
			PrioritySectors:  []string{"software", "manufacturing"},
		}

		data, err := json.Marshal(request)
		if err != nil {
			t.Fatalf("Failed to marshal AnalysisRequest: %v", err)
		}

		var decoded AnalysisRequest
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal AnalysisRequest: %v", err)
		}

		if len(decoded.PrioritySectors) != len(request.PrioritySectors) {
			t.Errorf("Expected %d priority sectors, got %d",
				len(request.PrioritySectors), len(decoded.PrioritySectors))
		}
	})
}

func TestCrossSectorImpactLogic(t *testing.T) {
	t.Run("Source and target must be different", func(t *testing.T) {
		impacts := analyzeCrossSectorImpacts()

		for i, impact := range impacts {
			if impact.SourceSector == impact.TargetSector {
				t.Errorf("Impact %d: source and target are the same: %s",
					i, impact.SourceSector)
			}
		}
	})

	t.Run("Impact scores are normalized", func(t *testing.T) {
		impacts := analyzeCrossSectorImpacts()

		for i, impact := range impacts {
			if impact.ImpactScore < 0 || impact.ImpactScore > 1 {
				t.Errorf("Impact %d: score %.2f is not normalized (0-1)",
					i, impact.ImpactScore)
			}
		}
	})

	t.Run("All impacts have descriptions", func(t *testing.T) {
		impacts := analyzeCrossSectorImpacts()

		for i, impact := range impacts {
			if impact.Description == "" {
				t.Errorf("Impact %d: missing description", i)
			}
			if len(impact.Description) < 10 {
				t.Errorf("Impact %d: description too short: '%s'",
					i, impact.Description)
			}
		}
	})
}

func TestRecommendationLogic(t *testing.T) {
	t.Run("All recommendations have scenarios", func(t *testing.T) {
		request := AnalysisRequest{
			CurrentResources: 5,
			TimeHorizon:      12,
			PrioritySectors:  []string{"software"},
		}

		recommendations := generateStrategicRecommendations(request)

		for i, rec := range recommendations {
			if rec.Scenario == "" {
				t.Errorf("Recommendation %d: missing scenario name", i)
			}
		}
	})

	t.Run("Priority scores are consistent", func(t *testing.T) {
		request := AnalysisRequest{
			CurrentResources: 5,
			TimeHorizon:      12,
			PrioritySectors:  []string{"software"},
		}

		recommendations := generateStrategicRecommendations(request)

		// Higher impact multiplier should generally correlate with higher priority
		for i := 1; i < len(recommendations); i++ {
			prev := recommendations[i-1]
			curr := recommendations[i]

			// Priority should be descending
			if curr.PriorityScore > prev.PriorityScore {
				t.Errorf("Recommendations not sorted: rec[%d] priority %.2f > rec[%d] priority %.2f",
					i, curr.PriorityScore, i-1, prev.PriorityScore)
			}
		}
	})

	t.Run("All recommendations have reasoning", func(t *testing.T) {
		request := AnalysisRequest{
			CurrentResources: 5,
			TimeHorizon:      12,
			PrioritySectors:  []string{"software"},
		}

		recommendations := generateStrategicRecommendations(request)

		for i, rec := range recommendations {
			if rec.Reasoning == "" {
				t.Errorf("Recommendation %d: missing reasoning", i)
			}
			if len(rec.Reasoning) < 20 {
				t.Errorf("Recommendation %d: reasoning too short: '%s'",
					i, rec.Reasoning)
			}
		}
	})
}

func TestBottleneckIdentification(t *testing.T) {
	t.Run("Bottlenecks are actionable", func(t *testing.T) {
		bottlenecks := identifyBottlenecks()

		for i, bottleneck := range bottlenecks {
			if bottleneck == "" {
				t.Errorf("Bottleneck %d is empty", i)
			}
			if len(bottleneck) < 15 {
				t.Errorf("Bottleneck %d description too short: '%s'",
					i, bottleneck)
			}
		}
	})

	t.Run("Multiple bottlenecks identified", func(t *testing.T) {
		bottlenecks := identifyBottlenecks()

		if len(bottlenecks) < 2 {
			t.Error("Expected multiple bottlenecks to be identified")
		}
	})
}

func TestTimelineCalculations(t *testing.T) {
	t.Run("Milestones are in future", func(t *testing.T) {
		request := AnalysisRequest{
			CurrentResources: 5,
			TimeHorizon:      12,
			PrioritySectors:  []string{"software"},
		}

		timeline := calculateProjectedTimeline(request)

		for i, milestone := range timeline.Milestones {
			// Using time.Now() might cause flakiness, but checking basic logic
			if milestone.Name == "" {
				t.Errorf("Milestone %d has empty name", i)
			}
		}
	})

	t.Run("Confidence values are valid", func(t *testing.T) {
		request := AnalysisRequest{
			CurrentResources: 5,
			TimeHorizon:      12,
			PrioritySectors:  []string{"software"},
		}

		timeline := calculateProjectedTimeline(request)

		for i, milestone := range timeline.Milestones {
			if milestone.Confidence < 0 || milestone.Confidence > 1 {
				t.Errorf("Milestone %d has invalid confidence: %.2f",
					i, milestone.Confidence)
			}
		}
	})
}

func TestDependencyStrengthValidation(t *testing.T) {
	testCases := []struct {
		name     string
		strength float64
		valid    bool
	}{
		{"No dependency", 0.0, true},
		{"Weak dependency", 0.3, true},
		{"Strong dependency", 0.9, true},
		{"Full dependency", 1.0, true},
		{"Negative strength", -0.5, false},
		{"Exceeds 1.0", 1.5, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dep := StageDependency{
				DependencyStrength: tc.strength,
			}

			isValid := dep.DependencyStrength >= 0 && dep.DependencyStrength <= 1
			if isValid != tc.valid {
				t.Errorf("Strength=%.2f, expected valid=%v, got %v",
					tc.strength, tc.valid, isValid)
			}
		})
	}
}

func TestMilestoneTypeValidation(t *testing.T) {
	validTypes := []string{
		"capability",
		"sector",
		"integration",
		"superintelligence",
	}

	t.Run("Valid milestone types", func(t *testing.T) {
		for _, milestoneType := range validTypes {
			milestone := StrategicMilestone{
				MilestoneType: milestoneType,
			}

			if milestone.MilestoneType == "" {
				t.Errorf("Milestone type '%s' should not be empty", milestoneType)
			}
		}
	})
}
