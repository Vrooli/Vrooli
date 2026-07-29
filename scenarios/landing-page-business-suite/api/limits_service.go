package main

import "landing-page-business-suite-api/internal/commerce"

type (
	LimitsServicer  = commerce.LimitsServicer
	LimitsStore     = commerce.LimitsStore
	LimitsService   = commerce.LimitsService
	TierLimit       = commerce.TierLimit
	TierLimitUpdate = commerce.TierLimitUpdate
)

func NewLimitsService(db LimitsStore, dialect string) *LimitsService {
	return commerce.NewLimitsService(db, dialect, logStructured)
}

func DollarsToInternalUnits(dollars float64) int64 {
	return commerce.DollarsToInternalUnits(dollars)
}

func InternalUnitsToDollars(units int64) float64 {
	return commerce.InternalUnitsToDollars(units)
}
