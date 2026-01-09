package main

import (
	"context"
	"testing"
)

// [REQ:CHART-P0-003-DARK] Dark theme optimized for screens and presentations
func TestDarkTheme_Availability(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := NewChartProcessor(nil)
	ctx := context.Background()

	result, err := processor.GetAvailableStyles(ctx)

	if err != nil {
		t.Fatalf("GetAvailableStyles returned error: %v", err)
	}

	hasDarkTheme := false
	for _, style := range result.Styles {
		if style.ID == "dark" {
			hasDarkTheme = true
			break
		}
	}

	if !hasDarkTheme {
		t.Error("Expected dark theme to be available")
	}
}
