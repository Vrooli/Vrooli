package main

import (
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/logx"
)

func NewLimitsService(db commerce.LimitsStore, dialect string) *commerce.LimitsService {
	return commerce.NewLimitsService(db, dialect, logx.Info)
}
func DollarsToInternalUnits(dollars float64) int64 { return commerce.DollarsToInternalUnits(dollars) }
func InternalUnitsToDollars(units int64) float64   { return commerce.InternalUnitsToDollars(units) }
