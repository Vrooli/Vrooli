package scoring

import (
	"testing"
)

// [REQ:SCS-CORE-001A] Test quality dimension scoring
func TestCalculateQualityScore(t *testing.T) {
	tests := []struct {
		name            string
		metrics         Metrics
		expectedScore   int
		expectedReqPts  int
		expectedTgtPts  int
		expectedTestPts int
	}{
		{
			name: "all passing",
			metrics: Metrics{
				Requirements: MetricCounts{Total: 20, Passing: 20},
				Targets:      MetricCounts{Total: 10, Passing: 10},
				Tests:        MetricCounts{Total: 30, Passing: 30},
			},
			expectedScore:   50,
			expectedReqPts:  20,
			expectedTgtPts:  15,
			expectedTestPts: 15,
		},
		{
			name: "half passing",
			metrics: Metrics{
				Requirements: MetricCounts{Total: 20, Passing: 10},
				Targets:      MetricCounts{Total: 10, Passing: 5},
				Tests:        MetricCounts{Total: 30, Passing: 15},
			},
			expectedScore:   26, // 10 + 8 + 8 = 26 (math.Round of 0.5 * 15 rounds to 8)
			expectedReqPts:  10,
			expectedTgtPts:  8,
			expectedTestPts: 8,
		},
		{
			name: "none passing",
			metrics: Metrics{
				Requirements: MetricCounts{Total: 20, Passing: 0},
				Targets:      MetricCounts{Total: 10, Passing: 0},
				Tests:        MetricCounts{Total: 30, Passing: 0},
			},
			expectedScore:   0,
			expectedReqPts:  0,
			expectedTgtPts:  0,
			expectedTestPts: 0,
		},
		{
			name: "no requirements/targets/tests",
			metrics: Metrics{
				Requirements: MetricCounts{Total: 0, Passing: 0},
				Targets:      MetricCounts{Total: 0, Passing: 0},
				Tests:        MetricCounts{Total: 0, Passing: 0},
			},
			expectedScore:   0,
			expectedReqPts:  0,
			expectedTgtPts:  0,
			expectedTestPts: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateQualityScore(tt.metrics)
			if result.Score != tt.expectedScore {
				t.Errorf("Expected score %d, got %d", tt.expectedScore, result.Score)
			}
			if result.RequirementPassRate.Points != tt.expectedReqPts {
				t.Errorf("Expected req points %d, got %d", tt.expectedReqPts, result.RequirementPassRate.Points)
			}
			if result.TargetPassRate.Points != tt.expectedTgtPts {
				t.Errorf("Expected target points %d, got %d", tt.expectedTgtPts, result.TargetPassRate.Points)
			}
			if result.TestPassRate.Points != tt.expectedTestPts {
				t.Errorf("Expected test points %d, got %d", tt.expectedTestPts, result.TestPassRate.Points)
			}
		})
	}
}
