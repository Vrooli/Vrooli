package main

import (
	"context"
	"testing"
)

// [REQ:CHART-P0-003-LIGHT] Light theme with professional color palette
func TestLightTheme_Availability(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := NewChartProcessor(nil)
	ctx := context.Background()

	result, err := processor.GetAvailableStyles(ctx)

	if err != nil {
		t.Fatalf("GetAvailableStyles returned error: %v", err)
	}

	hasLightTheme := false
	for _, style := range result.Styles {
		if style.ID == "professional" || style.ID == "light" {
			hasLightTheme = true
			break
		}
	}

	if !hasLightTheme {
		t.Error("Expected light/professional theme to be available")
	}
}
