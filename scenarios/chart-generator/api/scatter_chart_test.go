package main

import (
	"context"
	"testing"
)

// [REQ:CHART-P0-001-SCATTER] Scatter chart generation with x/y coordinate plotting
func TestScatterChart_Generation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := NewChartProcessor(nil)
	ctx := context.Background()

	tests := []struct {
		name           string
		req            ChartGenerationProcessorRequest
		expectSuccess  bool
		expectedError  string
		validateResult func(t *testing.T, result *ChartGenerationProcessorResponse)
	}{
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
			} else {
				if result.Success {
					t.Error("Expected failure, got success")
				}

				if result.Error == nil {
					t.Error("Expected error in response, got nil")
				} else if tt.expectedError != "" {
					if !containsString(result.Error.Message, tt.expectedError) {
						t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, result.Error.Message)
					}
				}
			}
		})
	}
}
