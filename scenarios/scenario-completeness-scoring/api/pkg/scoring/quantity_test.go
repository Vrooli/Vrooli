package scoring

import (
	"testing"
)

// [REQ:SCS-CORE-001C] Test quantity dimension scoring
func TestCalculateQuantityScore(t *testing.T) {
	thresholds := DefaultThresholds()

	tests := []struct {
		name     string
		metrics  Metrics
		minScore int
		maxScore int
	}{
		{
			name: "exceeds good thresholds",
			metrics: Metrics{
				Requirements: MetricCounts{Total: 20, Passing: 20}, // Exceeds good (15)
				Targets:      MetricCounts{Total: 15, Passing: 15}, // Exceeds good (12)
				Tests:        MetricCounts{Total: 30, Passing: 30}, // Exceeds good (25)
			},
			minScore: 10,
			maxScore: 10,
		},
		{
			name: "below all thresholds",
			metrics: Metrics{
				Requirements: MetricCounts{Total: 5, Passing: 5},
				Targets:      MetricCounts{Total: 3, Passing: 3},
				Tests:        MetricCounts{Total: 5, Passing: 5},
			},
			minScore: 0,
			maxScore: 5,
		},
		{
			name: "zero counts",
			metrics: Metrics{
				Requirements: MetricCounts{Total: 0, Passing: 0},
				Targets:      MetricCounts{Total: 0, Passing: 0},
				Tests:        MetricCounts{Total: 0, Passing: 0},
			},
			minScore: 0,
			maxScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateQuantityScore(tt.metrics, thresholds)
			if result.Score < tt.minScore || result.Score > tt.maxScore {
				t.Errorf("Expected score between %d and %d, got %d", tt.minScore, tt.maxScore, result.Score)
			}
			if result.Max != 10 {
				t.Errorf("Expected max 10, got %d", result.Max)
			}
		})
	}
}

// [REQ:SCS-CORE-001C] Test threshold level helper
func TestGetThresholdLevel(t *testing.T) {
	thresholds := ThresholdLevels{OK: 10, Good: 15, Excellent: 25}

	tests := []struct {
		count    int
		expected string
	}{
		{30, "excellent"},
		{25, "excellent"},
		{20, "good"},
		{15, "good"},
		{12, "ok"},
		{10, "ok"},
		{5, "below"},
		{0, "below"},
	}

	for _, tt := range tests {
		result := getThresholdLevel(tt.count, thresholds)
		if result != tt.expected {
			t.Errorf("Count %d: expected %s, got %s", tt.count, tt.expected, result)
		}
	}
}

// [REQ:SCS-CORE-003] Test quantity score with zero thresholds (division by zero protection)
// ASSUMPTION: Thresholds are always positive
// HARDENED: This test verifies the safeDivide helper prevents panic
func TestCalculateQuantityScoreZeroThresholds(t *testing.T) {
	// Malformed threshold config with zero values
	zeroThresholds := ThresholdConfig{
		Requirements: ThresholdLevels{OK: 0, Good: 0, Excellent: 0},
		Targets:      ThresholdLevels{OK: 0, Good: 0, Excellent: 0},
		Tests:        ThresholdLevels{OK: 0, Good: 0, Excellent: 0},
	}

	metrics := Metrics{
		Requirements: MetricCounts{Total: 10, Passing: 10},
		Targets:      MetricCounts{Total: 5, Passing: 5},
		Tests:        MetricCounts{Total: 20, Passing: 20},
	}

	// This should NOT panic - safeDivide should return 0 for zero divisors
	result := CalculateQuantityScore(metrics, zeroThresholds)

	// Should return 0 score since all divisions by zero result in 0
	if result.Score != 0 {
		t.Errorf("Expected score 0 for zero thresholds, got %d", result.Score)
	}

	if result.Max != 10 {
		t.Errorf("Expected max 10, got %d", result.Max)
	}

	// Verify component points are all 0
	if result.Requirements.Points != 0 {
		t.Errorf("Expected requirements points 0, got %d", result.Requirements.Points)
	}
	if result.Targets.Points != 0 {
		t.Errorf("Expected targets points 0, got %d", result.Targets.Points)
	}
	if result.Tests.Points != 0 {
		t.Errorf("Expected tests points 0, got %d", result.Tests.Points)
	}
}

// TestCalculateQuantityScoreNegativeThresholds tests behavior with negative thresholds
// ASSUMPTION: Thresholds should never be negative in production
// HARDENED: Verify graceful handling of edge case
func TestCalculateQuantityScoreNegativeThresholds(t *testing.T) {
	negativeThresholds := ThresholdConfig{
		Requirements: ThresholdLevels{OK: -5, Good: -10, Excellent: -15},
		Targets:      ThresholdLevels{OK: -3, Good: -6, Excellent: -9},
		Tests:        ThresholdLevels{OK: -10, Good: -20, Excellent: -30},
	}

	metrics := Metrics{
		Requirements: MetricCounts{Total: 10, Passing: 10},
		Targets:      MetricCounts{Total: 5, Passing: 5},
		Tests:        MetricCounts{Total: 20, Passing: 20},
	}

	// Should NOT panic
	result := CalculateQuantityScore(metrics, negativeThresholds)

	// safeDivide returns 0 for divisor <= 0, so score should be 0
	if result.Score != 0 {
		t.Errorf("Expected score 0 for negative thresholds, got %d", result.Score)
	}
}
