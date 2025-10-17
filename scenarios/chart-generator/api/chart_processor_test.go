package main

import (
	"context"
	"strings"
	"testing"
)

func TestChartProcessor_GenerateChart(t *testing.T) {
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
			name: "ValidPieChart",
			req: ChartGenerationProcessorRequest{
				ChartType:     "pie",
				Data:          generateTestChartData("pie", 4),
				ExportFormats: []string{"png"},
			},
			expectSuccess: true,
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
					if !strings.Contains(result.Error.Message, tt.expectedError) {
						t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, result.Error.Message)
					}
				}
			}
		})
	}
}

func TestChartProcessor_ValidateData(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := NewChartProcessor(nil)
	ctx := context.Background()

	tests := []struct {
		name        string
		req         DataValidationRequest
		expectValid bool
		expectErrors int
	}{
		{
			name: "ValidBarChartData",
			req: DataValidationRequest{
				Data:      generateTestChartData("bar", 5),
				ChartType: "bar",
			},
			expectValid: true,
			expectErrors: 0,
		},
		{
			name: "ValidPieChartData",
			req: DataValidationRequest{
				Data:      generateTestChartData("pie", 5),
				ChartType: "pie",
			},
			expectValid: true,
			expectErrors: 0,
		},
		{
			name: "EmptyData",
			req: DataValidationRequest{
				Data:      []map[string]interface{}{},
				ChartType: "bar",
			},
			expectValid: false,
			expectErrors: 1,
		},
		{
			name: "NilData",
			req: DataValidationRequest{
				Data:      nil,
				ChartType: "bar",
			},
			expectValid: false,
			expectErrors: 1,
		},
		{
			name: "InvalidChartType",
			req: DataValidationRequest{
				Data:      generateTestChartData("bar", 3),
				ChartType: "invalid_type",
			},
			expectValid: false,
			expectErrors: 1,
		},
		{
			name: "LargeDatasetWarning",
			req: DataValidationRequest{
				Data:      generateTestChartData("bar", 1500),
				ChartType: "bar",
			},
			expectValid: true,
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
		{
			name: "ValidCandlestickData",
			req: DataValidationRequest{
				Data:      generateTestChartData("candlestick", 5),
				ChartType: "candlestick",
			},
			expectValid: true,
		},
		{
			name: "ValidHeatmapData",
			req: DataValidationRequest{
				Data:      generateTestChartData("heatmap", 10),
				ChartType: "heatmap",
			},
			expectValid: true,
		},
		{
			name: "ValidGanttData",
			req: DataValidationRequest{
				Data:      generateTestChartData("gantt", 5),
				ChartType: "gantt",
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

			if tt.expectErrors > 0 && len(result.Errors) != tt.expectErrors {
				t.Errorf("Expected %d errors, got %d: %v", tt.expectErrors, len(result.Errors), result.Errors)
			}

			if result.DataPoints != len(tt.req.Data) {
				t.Errorf("Expected %d data points, got %d", len(tt.req.Data), result.DataPoints)
			}
		})
	}
}

func TestChartProcessor_GetAvailableStyles(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := NewChartProcessor(nil)
	ctx := context.Background()

	result, err := processor.GetAvailableStyles(ctx)

	if err != nil {
		t.Fatalf("GetAvailableStyles returned error: %v", err)
	}

	if len(result.Styles) < 3 {
		t.Errorf("Expected at least 3 styles, got %d", len(result.Styles))
	}

	// Check for default style
	hasDefault := false
	for _, style := range result.Styles {
		if style.IsDefault {
			hasDefault = true
			break
		}
	}

	if !hasDefault {
		t.Error("Expected at least one default style")
	}

	if result.Count != len(result.Styles) {
		t.Errorf("Count mismatch: expected %d, got %d", len(result.Styles), result.Count)
	}
}

func TestChartProcessor_ApplyTransformations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := NewChartProcessor(nil)

	tests := []struct {
		name           string
		data           []map[string]interface{}
		transform      *DataTransform
		expectedCount  int
		validateResult func(t *testing.T, result []map[string]interface{})
	}{
		{
			name: "FilterByValue",
			data: []map[string]interface{}{
				{"x": "A", "y": float64(10)},
				{"x": "B", "y": float64(20)},
				{"x": "C", "y": float64(30)},
			},
			transform: &DataTransform{
				Filter: &FilterConfig{
					Field:    "y",
					Operator: "gt",
					Value:    float64(15),
				},
			},
			expectedCount: 2,
		},
		{
			name: "SortAscending",
			data: []map[string]interface{}{
				{"x": "C", "y": float64(30)},
				{"x": "A", "y": float64(10)},
				{"x": "B", "y": float64(20)},
			},
			transform: &DataTransform{
				Sort: &SortConfig{
					Field:     "y",
					Direction: "asc",
				},
			},
			expectedCount: 3,
			validateResult: func(t *testing.T, result []map[string]interface{}) {
				if result[0]["y"].(float64) != 10 {
					t.Error("First element should have y=10")
				}
				if result[2]["y"].(float64) != 30 {
					t.Error("Last element should have y=30")
				}
			},
		},
		{
			name: "SortDescending",
			data: []map[string]interface{}{
				{"x": "A", "y": float64(10)},
				{"x": "B", "y": float64(20)},
				{"x": "C", "y": float64(30)},
			},
			transform: &DataTransform{
				Sort: &SortConfig{
					Field:     "y",
					Direction: "desc",
				},
			},
			expectedCount: 3,
			validateResult: func(t *testing.T, result []map[string]interface{}) {
				if result[0]["y"].(float64) != 30 {
					t.Error("First element should have y=30")
				}
			},
		},
		{
			name: "AggregateSum",
			data: []map[string]interface{}{
				{"category": "A", "value": float64(10)},
				{"category": "A", "value": float64(20)},
				{"category": "B", "value": float64(15)},
			},
			transform: &DataTransform{
				Aggregate: &AggregateConfig{
					Method:  "sum",
					Field:   "value",
					GroupBy: "category",
				},
			},
			expectedCount: 2,
		},
		{
			name: "AggregateAvg",
			data: []map[string]interface{}{
				{"value": float64(10)},
				{"value": float64(20)},
				{"value": float64(30)},
			},
			transform: &DataTransform{
				Aggregate: &AggregateConfig{
					Method: "avg",
					Field:  "value",
				},
			},
			expectedCount: 1,
			validateResult: func(t *testing.T, result []map[string]interface{}) {
				if result[0]["value"].(float64) != 20 {
					t.Errorf("Expected average of 20, got %v", result[0]["value"])
				}
			},
		},
		{
			name: "GroupByField",
			data: []map[string]interface{}{
				{"category": "A", "value": float64(10)},
				{"category": "A", "value": float64(20)},
				{"category": "B", "value": float64(15)},
			},
			transform: &DataTransform{
				Group: &GroupConfig{
					Field: "category",
				},
			},
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.ApplyTransformations(tt.data, tt.transform)

			if err != nil {
				t.Fatalf("ApplyTransformations returned error: %v", err)
			}

			if len(result) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(result))
			}

			if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestChartProcessor_GenerateCompositeChart(t *testing.T) {
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

func Test_toFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected float64
	}{
		{"Float64", float64(42.5), 42.5},
		{"Float32", float32(10.2), 10.199999809265137}, // Float32 precision
		{"Int", 100, 100.0},
		{"Int32", int32(50), 50.0},
		{"Int64", int64(75), 75.0},
		{"String", "123.45", 123.45},
		{"InvalidString", "abc", 0.0},
		{"Nil", nil, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toFloat64(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func Test_contains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{"Found", []string{"a", "b", "c"}, "b", true},
		{"NotFound", []string{"a", "b", "c"}, "d", false},
		{"EmptySlice", []string{}, "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
