package main

import "landing-page-business-suite-api/internal/commerce"

// Test-only convenience names preserve behavior-focused legacy tests while
// production composition calls internal/commerce directly.
func NewLimitsService(db commerce.LimitsStore, dialect string) *commerce.LimitsService {
	return commerce.NewLimitsService(db, dialect, logStructured)
}

func DollarsToInternalUnits(dollars float64) int64 { return commerce.DollarsToInternalUnits(dollars) }

func InternalUnitsToDollars(units int64) float64 { return commerce.InternalUnitsToDollars(units) }
