package scoring

import (
	"math"
	"testing"
)

// [REQ:SCS-CORE-001] Test completeness score calculation
func TestCalculateCompletenessScore(t *testing.T) {
	metrics := Metrics{
		Scenario: "test-scenario",
		Category: "utility",
		Requirements: MetricCounts{
			Total:   20,
			Passing: 18,
		},
		Targets: MetricCounts{
			Total:   10,
			Passing: 9,
		},
		Tests: MetricCounts{
			Total:   30,
			Passing: 28,
		},
		UI: &UIMetrics{
			IsTemplate:      false,
			FileCount:       25,
			ComponentCount:  10,
			PageCount:       3,
			APIEndpoints:    5,
			APIBeyondHealth: 4,
			HasRouting:      true,
			RouteCount:      4,
			TotalLOC:        800,
		},
	}

	thresholds := DefaultThresholds()
	breakdown := CalculateCompletenessScore(metrics, thresholds, 0)

	// Verify the score is calculated
	if breakdown.Score <= 0 {
		t.Errorf("Expected positive score, got %d", breakdown.Score)
	}

	// Verify all dimensions are present
	if breakdown.Quality.Max != 50 {
		t.Errorf("Quality max should be 50, got %d", breakdown.Quality.Max)
	}
	if breakdown.Coverage.Max != 15 {
		t.Errorf("Coverage max should be 15, got %d", breakdown.Coverage.Max)
	}
	if breakdown.Quantity.Max != 10 {
		t.Errorf("Quantity max should be 10, got %d", breakdown.Quantity.Max)
	}
	if breakdown.UI.Max != 25 {
		t.Errorf("UI max should be 25, got %d", breakdown.UI.Max)
	}

	// Verify base score equals sum of dimensions
	expectedBase := breakdown.Quality.Score + breakdown.Coverage.Score +
		breakdown.Quantity.Score + breakdown.UI.Score
	if breakdown.BaseScore != expectedBase {
		t.Errorf("Base score should be %d, got %d", expectedBase, breakdown.BaseScore)
	}
}

// [REQ:SCS-CORE-001] Test score classification
func TestClassifyScore(t *testing.T) {
	tests := []struct {
		score          int
		classification string
	}{
		{100, "production_ready"},
		{96, "production_ready"},
		{95, "nearly_ready"},
		{81, "nearly_ready"},
		{80, "mostly_complete"},
		{61, "mostly_complete"},
		{60, "functional_incomplete"},
		{41, "functional_incomplete"},
		{40, "foundation_laid"},
		{21, "foundation_laid"},
		{20, "early_stage"},
		{0, "early_stage"},
	}

	for _, tt := range tests {
		t.Run(tt.classification, func(t *testing.T) {
			class, _ := ClassifyScore(tt.score)
			if class != tt.classification {
				t.Errorf("Score %d: expected %s, got %s", tt.score, tt.classification, class)
			}
		})
	}
}

// [REQ:SCS-CORE-001] Test score does not exceed 100 or go below 0
func TestScoreBounds(t *testing.T) {
	// Perfect scenario
	perfectMetrics := Metrics{
		Requirements: MetricCounts{Total: 100, Passing: 100},
		Targets:      MetricCounts{Total: 50, Passing: 50},
		Tests:        MetricCounts{Total: 200, Passing: 200},
		UI: &UIMetrics{
			IsTemplate:      false,
			FileCount:       100,
			APIBeyondHealth: 20,
			RouteCount:      10,
			TotalLOC:        5000,
		},
		Requirements_: []RequirementTree{
			{ID: "R1", Children: []RequirementTree{
				{ID: "R1.1", Children: []RequirementTree{
					{ID: "R1.1.1"},
				}},
			}},
		},
	}
	thresholds := DefaultThresholds()

	result := CalculateCompletenessScore(perfectMetrics, thresholds, 0)
	if result.Score > 100 {
		t.Errorf("Score should not exceed 100, got %d", result.Score)
	}

	// Worst case with heavy penalty
	worstMetrics := Metrics{
		Requirements: MetricCounts{Total: 0, Passing: 0},
		Targets:      MetricCounts{Total: 0, Passing: 0},
		Tests:        MetricCounts{Total: 0, Passing: 0},
		UI:           nil,
	}
	heavyPenalty := 100 // Validation penalty as int

	result2 := CalculateCompletenessScore(worstMetrics, thresholds, heavyPenalty)
	if result2.Score < 0 {
		t.Errorf("Score should not go below 0, got %d", result2.Score)
	}
}

// [REQ:SCS-CORE-001] Test rounding behavior matches JS implementation
func TestRoundingBehavior(t *testing.T) {
	// 0.55 * 20 = 11 (rounded from 11.0)
	metrics := Metrics{
		Requirements: MetricCounts{Total: 20, Passing: 11}, // 55%
		Targets:      MetricCounts{Total: 0, Passing: 0},
		Tests:        MetricCounts{Total: 0, Passing: 0},
	}

	result := CalculateQualityScore(metrics)

	// 0.55 * 20 = 11
	if result.RequirementPassRate.Points != 11 {
		t.Errorf("Expected 11 points for 55%% pass rate, got %d", result.RequirementPassRate.Points)
	}

	// Verify rate is calculated correctly
	expectedRate := 0.55
	if math.Abs(result.RequirementPassRate.Rate-expectedRate) > 0.001 {
		t.Errorf("Expected rate %.2f, got %.2f", expectedRate, result.RequirementPassRate.Rate)
	}
}
