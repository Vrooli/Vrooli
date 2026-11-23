package main

import "testing"

// [REQ:CHART-P0-001-SCATTER] Scatter chart generation with x/y coordinate plotting
func TestScatterChart_Generation(t *testing.T) {
	tests := []chartGenerationCase{
		{
			name: "ValidScatterChart",
			req: ChartGenerationProcessorRequest{
				ChartType:     "scatter",
				Data:          generateTestChartData("scatter", 10),
				Style:         "professional",
				ExportFormats: []string{"svg"},
				Width:         800,
				Height:        600,
				Title:         "Test Scatter Chart",
			},
			expectSuccess: true,
			validateResult: func(t *testing.T, result *ChartGenerationProcessorResponse) {
				if result.ChartID == "" {
					t.Error("Expected non-empty chart ID")
				}
				if len(result.Files) == 0 {
					t.Error("Expected at least one file to be generated")
				}
				if result.Metadata.DataPointCount != 10 {
					t.Errorf("Expected 10 data points, got %d", result.Metadata.DataPointCount)
				}
			},
		},
		{
			name: "EmptyData",
			req: ChartGenerationProcessorRequest{
				ChartType:     "scatter",
				Data:          []map[string]interface{}{},
				ExportFormats: []string{"svg"},
			},
			expectSuccess: false,
			expectedError: "Data array cannot be empty",
		},
		{
			name: "ScatterWithCustomDimensions",
			req: ChartGenerationProcessorRequest{
				ChartType:     "scatter",
				Data:          generateTestChartData("scatter", 5),
				Width:         1000,
				Height:        800,
				ExportFormats: []string{"svg"},
			},
			expectSuccess: true,
			validateResult: func(t *testing.T, result *ChartGenerationProcessorResponse) {
				if result.Metadata.Dimensions.Width != 1000 {
					t.Errorf("Expected width 1000, got %d", result.Metadata.Dimensions.Width)
				}
				if result.Metadata.Dimensions.Height != 800 {
					t.Errorf("Expected height 800, got %d", result.Metadata.Dimensions.Height)
				}
			},
		},
	}

	runChartGenerationCases(t, tests)
}
