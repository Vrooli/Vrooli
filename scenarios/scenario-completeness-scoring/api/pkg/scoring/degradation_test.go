package scoring

import (
	"testing"
)

// [REQ:SCS-CORE-003] Test weight redistribution (validation penalty)
func TestValidationPenalty(t *testing.T) {
	metrics := Metrics{
		Requirements: MetricCounts{Total: 20, Passing: 20},
		Targets:      MetricCounts{Total: 10, Passing: 10},
		Tests:        MetricCounts{Total: 30, Passing: 30},
		UI: &UIMetrics{
			IsTemplate: false,
			FileCount:  50,
		},
	}
	thresholds := DefaultThresholds()

	// Without penalty
	noValidation := CalculateCompletenessScore(metrics, thresholds, 0)

	// With penalty (validation penalty passed as int)
	validationPenalty := 20
	withValidation := CalculateCompletenessScore(metrics, thresholds, validationPenalty)

	if noValidation.ValidationPenalty != 0 {
		t.Errorf("Expected no validation penalty, got %d", noValidation.ValidationPenalty)
	}

	if withValidation.ValidationPenalty != 20 {
		t.Errorf("Expected validation penalty 20, got %d", withValidation.ValidationPenalty)
	}

	expectedDiff := noValidation.Score - withValidation.Score
	if expectedDiff != 20 {
		t.Errorf("Expected score difference of 20, got %d", expectedDiff)
	}
}

// [REQ:SCS-CORE-004] Test graceful degradation with partial data
func TestGracefulDegradation(t *testing.T) {
	// Metrics with missing UI
	metricsNoUI := Metrics{
		Requirements: MetricCounts{Total: 20, Passing: 15},
		Targets:      MetricCounts{Total: 10, Passing: 8},
		Tests:        MetricCounts{Total: 30, Passing: 25},
		UI:           nil,
	}
	thresholds := DefaultThresholds()

	// Should still calculate a score
	breakdown := CalculateCompletenessScore(metricsNoUI, thresholds, 0)

	if breakdown.Score < 0 {
		t.Errorf("Score should be non-negative, got %d", breakdown.Score)
	}

	// Quality, Coverage, Quantity should still be calculated
	if breakdown.Quality.Score <= 0 {
		t.Errorf("Quality score should be positive, got %d", breakdown.Quality.Score)
	}

	// UI should be 0 but present
	if breakdown.UI.Max != 25 {
		t.Errorf("UI max should still be 25, got %d", breakdown.UI.Max)
	}
	if breakdown.UI.Score != 0 {
		t.Errorf("UI score should be 0 for nil UI, got %d", breakdown.UI.Score)
	}
}

// [REQ:SCS-CORE-004] Test graceful degradation with zero totals
func TestGracefulDegradationZeroTotals(t *testing.T) {
	// Metrics with zero totals (division by zero edge case)
	metricsEmpty := Metrics{
		Requirements: MetricCounts{Total: 0, Passing: 0},
		Targets:      MetricCounts{Total: 0, Passing: 0},
		Tests:        MetricCounts{Total: 0, Passing: 0},
		UI:           nil,
	}
	thresholds := DefaultThresholds()

	// Should not panic and should return valid (zero) score
	breakdown := CalculateCompletenessScore(metricsEmpty, thresholds, 0)

	if breakdown.Score < 0 {
		t.Errorf("Score should be non-negative, got %d", breakdown.Score)
	}

	// All pass rates should be 0 (not NaN or panic)
	if breakdown.Quality.RequirementPassRate.Rate != 0 {
		t.Errorf("Expected 0 req pass rate for empty, got %f", breakdown.Quality.RequirementPassRate.Rate)
	}
}

// [REQ:SCS-CORE-003] Test penalty application with breakdown
func TestPenaltyBreakdown(t *testing.T) {
	metrics := Metrics{
		Requirements: MetricCounts{Total: 10, Passing: 10},
		Targets:      MetricCounts{Total: 5, Passing: 5},
		Tests:        MetricCounts{Total: 20, Passing: 20},
		UI: &UIMetrics{
			IsTemplate: false,
			FileCount:  30,
		},
	}
	thresholds := DefaultThresholds()

	// Apply penalty (the detailed breakdown is available from validators.ValidationQualityAnalysis
	// but the scoring engine only needs the total penalty value)
	validationPenalty := 15

	breakdown := CalculateCompletenessScore(metrics, thresholds, validationPenalty)

	if breakdown.ValidationPenalty != 15 {
		t.Errorf("Expected validation penalty 15, got %d", breakdown.ValidationPenalty)
	}

	if breakdown.BaseScore-breakdown.Score != 15 {
		t.Errorf("Expected 15 point reduction, got %d", breakdown.BaseScore-breakdown.Score)
	}
}
