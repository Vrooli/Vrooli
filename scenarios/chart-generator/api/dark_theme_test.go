package main

import "testing"

// [REQ:CHART-P0-003-DARK] Dark theme optimized for screens and presentations
func TestDarkTheme_Availability(t *testing.T) {
	assertThemeAvailable(t, "dark")
}
