package main

import (
	"context"
	"testing"
)

// [REQ:CHART-P0-003-MINIMAL] Minimal theme for clean, distraction-free visualizations
func TestMinimalTheme_Availability(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := NewChartProcessor(nil)
	ctx := context.Background()

	result, err := processor.GetAvailableStyles(ctx)

	if err != nil {
		t.Fatalf("GetAvailableStyles returned error: %v", err)
	}

	hasMinimalTheme := false
	for _, style := range result.Styles {
		if style.ID == "minimal" {
			hasMinimalTheme = true
			break
		}
	}

	if !hasMinimalTheme {
		t.Error("Expected minimal theme to be available")
	}
}
