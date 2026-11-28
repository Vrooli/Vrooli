package main

import "testing"

// [REQ:CHART-P0-001-BAR] Bar chart generation with configurable axes and data inputs
func TestBarChart_Generation(t *testing.T) {
	tests := []chartGenerationCase{
		{
			name: "ValidBarChart",
			req: ChartGenerationProcessorRequest{
				ChartType:     "bar",
				Data:          generateTestChartData("bar", 5),
				Style:         "professional",
				ExportFormats: []string{"png"},
				Width:         800,
				Height:        600,
				Title:         "Test Bar Chart",
			},
			expectSuccess: true,
			validateResult: func(t *testing.T, result *ChartGenerationProcessorResponse) {
				if result.ChartID == "" {
					t.Error("Expected non-empty chart ID")
				}
				if len(result.Files) == 0 {
					t.Error("Expected at least one file to be generated")
				}
				if result.Metadata.DataPointCount != 5 {
					t.Errorf("Expected 5 data points, got %d", result.Metadata.DataPointCount)
				}
			},
		},
		{
			name: "EmptyData",
			req: ChartGenerationProcessorRequest{
				ChartType:     "bar",
				Data:          []map[string]interface{}{},
				ExportFormats: []string{"png"},
			},
			expectSuccess: false,
			expectedError: "Data array cannot be empty",
		},
		{
			name: "InvalidChartType",
			req: ChartGenerationProcessorRequest{
				ChartType:     "invalid_type",
				Data:          generateTestChartData("bar", 3),
				ExportFormats: []string{"png"},
			},
			expectSuccess: false,
			expectedError: "Invalid chart_type",
		},
		{
			name: "NilData",
			req: ChartGenerationProcessorRequest{
				ChartType:     "bar",
				Data:          nil,
				ExportFormats: []string{"png"},
			},
			expectSuccess: false,
			expectedError: "Missing data field",
		},
		{
			name: "DimensionsConstrained",
			req: ChartGenerationProcessorRequest{
				ChartType:     "bar",
				Data:          generateTestChartData("bar", 3),
				Width:         5000, // Should be constrained to 2000
				Height:        3000, // Should be constrained to 1500
				ExportFormats: []string{"png"},
			},
			expectSuccess: true,
			validateResult: func(t *testing.T, result *ChartGenerationProcessorResponse) {
				if result.Metadata.Dimensions.Width > 2000 {
					t.Errorf("Width should be constrained to 2000, got %d", result.Metadata.Dimensions.Width)
				}
				if result.Metadata.Dimensions.Height > 1500 {
					t.Errorf("Height should be constrained to 1500, got %d", result.Metadata.Dimensions.Height)
				}
			},
		},
		{
			name: "MinimumDimensionsEnforced",
			req: ChartGenerationProcessorRequest{
				ChartType:     "bar",
				Data:          generateTestChartData("bar", 3),
				Width:         50,  // Should be increased to 200
				Height:        100, // Should be increased to 200
				ExportFormats: []string{"png"},
			},
			expectSuccess: true,
			validateResult: func(t *testing.T, result *ChartGenerationProcessorResponse) {
				if result.Metadata.Dimensions.Width < 200 {
					t.Errorf("Width should be at least 200, got %d", result.Metadata.Dimensions.Width)
				}
				if result.Metadata.Dimensions.Height < 200 {
					t.Errorf("Height should be at least 200, got %d", result.Metadata.Dimensions.Height)
				}
			},
		},
	}

	runChartGenerationCases(t, tests)
}
