package main

import (
	"context"
	"testing"
)

// [REQ:CHART-P0-001-AREA] Area chart generation with filled regions under line plots
func TestAreaChart_Generation(t *testing.T) {
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
