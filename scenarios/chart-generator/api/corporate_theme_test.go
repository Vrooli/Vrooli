package main

import "testing"

// [REQ:CHART-P0-003-CORPORATE] Corporate theme with conservative styling
func TestCorporateTheme_Availability(t *testing.T) {
	assertThemeAvailable(t, "corporate")
}
