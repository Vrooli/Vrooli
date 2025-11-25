package main

import (
	"context"
	"testing"
)

// [REQ:CHART-P0-002-JSON] JSON data format parsing and validation
func TestJSONIngestion_ValidateData(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := NewChartProcessor(nil)
	ctx := context.Background()

	tests := []struct {
		name         string
		req          DataValidationRequest
		expectValid  bool
		expectErrors int
	}{
		{
			name: "ValidBarChartData",
			req: DataValidationRequest{
				Data:      generateTestChartData("bar", 5),
				ChartType: "bar",
			},
			expectValid:  true,
			expectErrors: 0,
		},
		{
			name: "ValidPieChartData",
			req: DataValidationRequest{
				Data:      generateTestChartData("pie", 5),
				ChartType: "pie",
			},
			expectValid:  true,
			expectErrors: 0,
		},
		{
			name: "EmptyData",
			req: DataValidationRequest{
				Data:      []map[string]interface{}{},
				ChartType: "bar",
			},
			expectValid:  false,
			expectErrors: 1,
		},
		{
			name: "NilData",
			req: DataValidationRequest{
				Data:      nil,
				ChartType: "bar",
			},
			expectValid:  false,
			expectErrors: 1,
		},
		{
			name: "InvalidChartType",
			req: DataValidationRequest{
				Data:      generateTestChartData("bar", 3),
				ChartType: "invalid_type",
			},
			expectValid:  false,
			expectErrors: 1,
		},
		{
			name: "LargeDatasetWarning",
			req: DataValidationRequest{
				Data:      generateTestChartData("bar", 1500),
				ChartType: "bar",
			},
			expectValid:  true,
			expectErrors: 0,
		},
		{
			name: "MissingRequiredFieldsForPie",
			req: DataValidationRequest{
				Data: []map[string]interface{}{
					{"x": "A"}, // Missing value/y
				},
				ChartType: "pie",
			},
			expectValid: false,
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

			if tt.expectErrors > 0 && len(result.Errors) != tt.expectErrors {
				t.Errorf("Expected %d errors, got %d: %v", tt.expectErrors, len(result.Errors), result.Errors)
			}

			if result.DataPoints != len(tt.req.Data) {
				t.Errorf("Expected %d data points, got %d", len(tt.req.Data), result.DataPoints)
			}
		})
	}
}
