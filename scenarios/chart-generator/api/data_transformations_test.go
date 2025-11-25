package main

import (
	"testing"
)

// [REQ:CHART-P1-007] Data transformation pipeline (aggregation, filtering, sorting)
func TestDataTransformations_ApplyTransformations(t *testing.T) {
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
