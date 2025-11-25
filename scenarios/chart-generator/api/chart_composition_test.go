package main

import (
	"context"
	"testing"
)

// [REQ:CHART-P1-006] Chart composition (multiple charts in single canvas)
func TestChartComposition_GenerateCompositeChart(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := NewChartProcessor(nil)
	ctx := context.Background()

	tests := []struct {
		name          string
		req           ChartGenerationProcessorRequest
		expectSuccess bool
	}{
		{
			name: "ValidComposite",
			req: ChartGenerationProcessorRequest{
				ChartType: "bar",
				Data:      generateTestChartData("bar", 3),
				Composition: &ChartComposition{
					Layout: "grid",
					Charts: []ChartGenerationProcessorRequest{
						{
							ChartType: "bar",
							Data:      generateTestChartData("bar", 3),
						},
						{
							ChartType: "line",
							Data:      generateTestChartData("line", 5),
						},
					},
				},
				ExportFormats: []string{"png"},
			},
			expectSuccess: true,
		},
		{
			name: "EmptyComposition",
			req: ChartGenerationProcessorRequest{
				ChartType: "bar",
				Data:      generateTestChartData("bar", 3),
				Composition: &ChartComposition{
					Layout: "grid",
					Charts: []ChartGenerationProcessorRequest{},
				},
			},
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.GenerateChart(ctx, tt.req)

			if err != nil {
				t.Fatalf("GenerateChart returned error: %v", err)
			}

			if result.Success != tt.expectSuccess {
				t.Errorf("Expected success=%v, got %v", tt.expectSuccess, result.Success)
			}

			if tt.expectSuccess && result.ChartID == "" {
				t.Error("Expected non-empty chart ID")
			}
		})
	}
}
