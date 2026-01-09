package main

import (
	"context"
	"testing"
)

// [REQ:CHART-P0-001-PIE] Pie and donut chart generation with percentage labels
func TestPieChart_Generation(t *testing.T) {
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
			name: "ValidPieChart",
			req: ChartGenerationProcessorRequest{
				ChartType:     "pie",
				Data:          generateTestChartData("pie", 4),
				ExportFormats: []string{"png"},
			},
			expectSuccess: true,
		},
		{
			name: "PieChartAcceptsXY",
			req: ChartGenerationProcessorRequest{
				ChartType: "pie",
				Data: []map[string]interface{}{
					{"x": "A", "y": float64(10)}, // Pie accepts x/y OR name/value for flexibility
				},
				ExportFormats: []string{"png"},
			},
			expectSuccess: true,
		},
		{
			name: "InvalidDataStructureForPie",
			req: ChartGenerationProcessorRequest{
				ChartType: "pie",
				Data: []map[string]interface{}{
					{"invalid": "A"}, // Missing both label and value fields
				},
				ExportFormats: []string{"png"},
			},
			expectSuccess: false,
			expectedError: "must have",
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
					// Simple substring check
					found := false
					for i := 0; i <= len(result.Error.Message)-len(tt.expectedError); i++ {
						if result.Error.Message[i:i+len(tt.expectedError)] == tt.expectedError {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, result.Error.Message)
					}
				}
			}
		})
	}
}
