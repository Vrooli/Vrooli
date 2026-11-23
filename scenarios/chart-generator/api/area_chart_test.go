package main

import "testing"

// [REQ:CHART-P0-001-AREA] Area chart generation with filled regions under line plots
func TestAreaChart_Generation(t *testing.T) {
	tests := []chartGenerationCase{
		{
			name: "ValidAreaChart",
			req: ChartGenerationProcessorRequest{
				ChartType:     "area",
				Data:          generateTestChartData("area", 8),
				Style:         "professional",
				ExportFormats: []string{"svg"},
				Width:         800,
				Height:        600,
				Title:         "Test Area Chart",
			},
			expectSuccess: true,
			validateResult: func(t *testing.T, result *ChartGenerationProcessorResponse) {
				if result.ChartID == "" {
					t.Error("Expected non-empty chart ID")
				}
				if len(result.Files) == 0 {
					t.Error("Expected at least one file to be generated")
				}
				if result.Metadata.DataPointCount != 8 {
					t.Errorf("Expected 8 data points, got %d", result.Metadata.DataPointCount)
				}
			},
		},
		{
			name: "EmptyData",
			req: ChartGenerationProcessorRequest{
				ChartType:     "area",
				Data:          []map[string]interface{}{},
				ExportFormats: []string{"svg"},
			},
			expectSuccess: false,
			expectedError: "Data array cannot be empty",
		},
		{
			name: "AreaWithMinimalTheme",
			req: ChartGenerationProcessorRequest{
				ChartType:     "area",
				Data:          generateTestChartData("area", 5),
				Style:         "minimal",
				ExportFormats: []string{"svg"},
			},
			expectSuccess: true,
			validateResult: func(t *testing.T, result *ChartGenerationProcessorResponse) {
				if result.Metadata.StyleApplied != "minimal" {
					t.Errorf("Expected minimal style, got %s", result.Metadata.StyleApplied)
				}
			},
		},
		{
			name: "AreaWithMultipleExports",
			req: ChartGenerationProcessorRequest{
				ChartType:     "area",
				Data:          generateTestChartData("area", 6),
				ExportFormats: []string{"svg", "png"},
			},
			expectSuccess: true,
			validateResult: func(t *testing.T, result *ChartGenerationProcessorResponse) {
				if len(result.Files) < 2 {
					t.Error("Expected at least 2 export files (svg and png)")
				}
			},
		},
	}

	runChartGenerationCases(t, tests)
}
