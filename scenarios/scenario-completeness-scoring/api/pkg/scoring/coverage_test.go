package scoring

import (
	"testing"
)

// [REQ:SCS-CORE-001B] Test coverage dimension scoring
func TestCalculateCoverageScore(t *testing.T) {
	tests := []struct {
		name         string
		metrics      Metrics
		requirements []RequirementTree
		minScore     int
		maxScore     int
	}{
		{
			name: "high coverage ratio",
			metrics: Metrics{
				Requirements: MetricCounts{Total: 10, Passing: 10},
				Tests:        MetricCounts{Total: 20, Passing: 20}, // 2:1 ratio
			},
			requirements: nil,
			minScore:     8, // Should get full points for 2.0x ratio
			maxScore:     8,
		},
		{
			name: "low coverage ratio",
			metrics: Metrics{
				Requirements: MetricCounts{Total: 10, Passing: 10},
				Tests:        MetricCounts{Total: 5, Passing: 5}, // 0.5:1 ratio
			},
			requirements: nil,
			minScore:     2,
			maxScore:     2,
		},
		{
			name: "with depth",
			metrics: Metrics{
				Requirements: MetricCounts{Total: 10, Passing: 10},
				Tests:        MetricCounts{Total: 20, Passing: 20},
			},
			requirements: []RequirementTree{
				{ID: "R1", Children: []RequirementTree{
					{ID: "R1.1", Children: []RequirementTree{
						{ID: "R1.1.1"},
					}},
				}},
			},
			minScore: 10, // 8 from ratio + some from depth
			maxScore: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateCoverageScore(tt.metrics, tt.requirements)
			if result.Score < tt.minScore || result.Score > tt.maxScore {
				t.Errorf("Expected score between %d and %d, got %d", tt.minScore, tt.maxScore, result.Score)
			}
			if result.Max != 15 {
				t.Errorf("Expected max 15, got %d", result.Max)
			}
		})
	}
}

// [REQ:SCS-CORE-001B] Test depth calculation helper
func TestGetMaxDepth(t *testing.T) {
	tests := []struct {
		name  string
		req   RequirementTree
		depth int
	}{
		{
			name:  "single node",
			req:   RequirementTree{ID: "R1"},
			depth: 1,
		},
		{
			name: "two levels",
			req: RequirementTree{
				ID: "R1",
				Children: []RequirementTree{
					{ID: "R1.1"},
				},
			},
			depth: 2,
		},
		{
			name: "three levels",
			req: RequirementTree{
				ID: "R1",
				Children: []RequirementTree{
					{ID: "R1.1", Children: []RequirementTree{
						{ID: "R1.1.1"},
					}},
				},
			},
			depth: 3,
		},
		{
			name: "multiple branches different depths",
			req: RequirementTree{
				ID: "R1",
				Children: []RequirementTree{
					{ID: "R1.1"},
					{ID: "R1.2", Children: []RequirementTree{
						{ID: "R1.2.1"},
					}},
				},
			},
			depth: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getMaxDepth(tt.req)
			if result != tt.depth {
				t.Errorf("Expected depth %d, got %d", tt.depth, result)
			}
		})
	}
}
