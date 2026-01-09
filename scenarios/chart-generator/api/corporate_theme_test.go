package main

import (
	"context"
	"testing"
)

// [REQ:CHART-P0-003-CORPORATE] Corporate theme with conservative styling
func TestCorporateTheme_Availability(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	processor := NewChartProcessor(nil)
	ctx := context.Background()

	result, err := processor.GetAvailableStyles(ctx)

	if err != nil {
		t.Fatalf("GetAvailableStyles returned error: %v", err)
	}

	hasCorporateTheme := false
	for _, style := range result.Styles {
		if style.ID == "corporate" {
			hasCorporateTheme = true
			break
		}
	}

	if !hasCorporateTheme {
		t.Error("Expected corporate theme to be available")
	}
}
