package main

import (
	"testing"
)

// TestGetString tests the getString helper function
func TestGetString(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	tests := []struct {
		name     string
		data     map[string]interface{}
		key      string
		expected string
	}{
		{
			name:     "ValidString",
			data:     map[string]interface{}{"key": "value"},
			key:      "key",
			expected: "value",
		},
		{
			name:     "MissingKey",
			data:     map[string]interface{}{},
			key:      "missing",
			expected: "",
		},
		{
			name:     "NonStringValue",
			data:     map[string]interface{}{"key": 123},
			key:      "key",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getString(tt.data, tt.key)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestGetStringSlice tests the getStringSlice helper function
func TestGetStringSlice(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	tests := []struct {
		name     string
		data     map[string]interface{}
		key      string
		expected int // length of expected slice
	}{
		{
			name:     "ValidSlice",
			data:     map[string]interface{}{"items": []interface{}{"a", "b", "c"}},
			key:      "items",
			expected: 3,
		},
		{
			name:     "EmptySlice",
			data:     map[string]interface{}{"items": []interface{}{}},
			key:      "items",
			expected: 0,
		},
		{
			name:     "MissingKey",
			data:     map[string]interface{}{},
			key:      "missing",
			expected: 0,
		},
		{
			name:     "NonSliceValue",
			data:     map[string]interface{}{"items": "not a slice"},
			key:      "items",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStringSlice(tt.data, tt.key)
			if len(result) != tt.expected {
				t.Errorf("Expected slice length %d, got %d", tt.expected, len(result))
			}
		})
	}
}

// TestGetFloat tests the getFloat helper function
func TestGetFloat(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	tests := []struct {
		name     string
		data     map[string]interface{}
		key      string
		expected float64
	}{
		{
			name:     "ValidFloat",
			data:     map[string]interface{}{"score": 42.5},
			key:      "score",
			expected: 42.5,
		},
		{
			name:     "MissingKey",
			data:     map[string]interface{}{},
			key:      "missing",
			expected: 0.0,
		},
		{
			name:     "NonFloatValue",
			data:     map[string]interface{}{"score": "not a float"},
			key:      "score",
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getFloat(tt.data, tt.key)
			if result != tt.expected {
				t.Errorf("Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

// TestCalculateSprintRisk tests sprint risk calculation
func TestCalculateSprintRisk(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	tests := []struct {
		name       string
		features   []Feature
		expectRisk string
	}{
		{
			name: "HighConfidence",
			features: []Feature{
				{Confidence: 0.9},
				{Confidence: 0.85},
				{Confidence: 0.95},
			},
			expectRisk: "low",
		},
		{
			name: "MediumConfidence",
			features: []Feature{
				{Confidence: 0.7},
				{Confidence: 0.6},
				{Confidence: 0.75},
			},
			expectRisk: "medium",
		},
		{
			name: "LowConfidence",
			features: []Feature{
				{Confidence: 0.5},
				{Confidence: 0.4},
				{Confidence: 0.3},
			},
			expectRisk: "high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			risk := testApp.App.calculateSprintRisk(tt.features)
			if risk != tt.expectRisk {
				t.Errorf("Expected risk '%s', got '%s'", tt.expectRisk, risk)
			}
		})
	}
}

// TestOptimizeSprint tests sprint optimization
func TestOptimizeSprint(t *testing.T) {
	testApp := setupTestApp(t)
	defer testApp.Cleanup()

	t.Run("OptimizeWithinCapacity", func(t *testing.T) {
		features := []Feature{
			{ID: "f1", Effort: 3, Reach: 1000, Impact: 5, Confidence: 0.9},
			{ID: "f2", Effort: 5, Reach: 2000, Impact: 4, Confidence: 0.8},
			{ID: "f3", Effort: 8, Reach: 3000, Impact: 3, Confidence: 0.7},
		}

		capacity := 10

		sprint, err := testApp.App.optimizeSprint(capacity, features)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if sprint.Capacity != capacity {
			t.Errorf("Expected capacity %d, got %d", capacity, sprint.Capacity)
		}

		if sprint.TotalEffort > capacity {
			t.Errorf("Sprint effort %d exceeds capacity %d", sprint.TotalEffort, capacity)
		}

		if len(sprint.Features) == 0 {
			t.Error("Expected features in optimized sprint")
		}
	})

	t.Run("ZeroCapacity", func(t *testing.T) {
		features := createTestFeatures(5)
		capacity := 0

		sprint, err := testApp.App.optimizeSprint(capacity, features)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(sprint.Features) != 0 {
			t.Error("Expected no features with zero capacity")
		}
	})
}
