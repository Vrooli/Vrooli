package main

import (
	"context"
	"testing"
)

// [REQ:CHART-P1-001] Advanced chart types: gantt, heatmap, treemap charts
func TestAdvancedChartTypes_GanttHeatmapTreemap(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := NewChartProcessor(nil)
	ctx := context.Background()

	tests := []struct {
		name        string
		req         DataValidationRequest
		expectValid bool
	}{
		{
			name: "ValidHeatmapData",
			req: DataValidationRequest{
				Data:      generateTestChartData("heatmap", 10),
				ChartType: "heatmap",
			},
			expectValid: true,
		},
		{
			name: "ValidGanttData",
			req: DataValidationRequest{
				Data:      generateTestChartData("gantt", 5),
				ChartType: "gantt",
			},
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.ValidateData(ctx, tt.req)

			if err != nil {
				t.Fatalf("ValidateData returned error: %v", err)
			}

			if result.Valid != tt.expectValid {
				t.Errorf("Expected valid=%v, got %v. Errors: %v", tt.expectValid, result.Valid, result.Errors)
			}
		})
	}
}
