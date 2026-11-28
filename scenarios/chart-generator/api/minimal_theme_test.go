package main

import "testing"

// [REQ:CHART-P0-003-MINIMAL] Minimal theme for clean, distraction-free visualizations
func TestMinimalTheme_Availability(t *testing.T) {
	assertThemeAvailable(t, "minimal")
}
