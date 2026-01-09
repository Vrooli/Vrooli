package main

import (
	"context"
	"testing"
)

// [REQ:CHART-P0-001-LINE] Line chart generation with time-series and continuous data support
func TestLineChart_Generation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := NewChartProcessor(nil)
	ctx := context.Background()

	tests := []struct {
		name           string
		req            ChartGenerationProcessorRequest
		expectSuccess  bool
		validateResult func(t *testing.T, result *ChartGenerationProcessorResponse)
	}{
		{
			name: "ValidLineChart",
			req: ChartGenerationProcessorRequest{
				ChartType:     "line",
				Data:          generateTestChartData("line", 10),
				ExportFormats: []string{"svg", "png"},
			},
			expectSuccess: true,
			validateResult: func(t *testing.T, result *ChartGenerationProcessorResponse) {
				if result.Metadata.StyleApplied != "professional" { // Should default
					t.Errorf("Expected default style 'professional', got %s", result.Metadata.StyleApplied)
				}
			},
		},
		{
			name: "LineChartWithMultipleDataPoints",
			req: ChartGenerationProcessorRequest{
				ChartType:     "line",
				Data:          generateTestChartData("line", 50),
				ExportFormats: []string{"svg"},
				Title:         "Time Series Data",
			},
			expectSuccess: true,
			validateResult: func(t *testing.T, result *ChartGenerationProcessorResponse) {
				if result.Metadata.DataPointCount != 50 {
					t.Errorf("Expected 50 data points, got %d", result.Metadata.DataPointCount)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.GenerateChart(ctx, tt.req)

			if err != nil {
				t.Fatalf("GenerateChart returned unexpected error: %v", err)
			}

			if tt.expectSuccess {
				if !result.Success {
					errorMsg := ""
					if result.Error != nil {
						errorMsg = result.Error.Message
					}
					t.Errorf("Expected success, got failure: %s", errorMsg)
				}

				if tt.validateResult != nil {
					tt.validateResult(t, result)
				}
			}
		})
	}
}
