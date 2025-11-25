package main

import (
	"context"
	"testing"
)

// [REQ:CHART-P1-002] Candlestick charts for financial data
func TestCandlestickChart_Validation(t *testing.T) {
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
			name: "ValidCandlestickData",
			req: DataValidationRequest{
				Data:      generateTestChartData("candlestick", 5),
				ChartType: "candlestick",
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
