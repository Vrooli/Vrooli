package main

import (
	"landing-page-business-suite-api/internal/commerce"
)

// These aliases let package-main tests exercise the commerce rules without
// reintroducing a production compatibility layer.
var (
	mapPlanKind                    = commerce.MapPlanKind
	mapIntroPricingTypeFromString  = commerce.MapIntroPricingType
	normalizePlanOption            = commerce.NormalizePlanOption
	planKindForTier                = commerce.PlanKindForTier
	planRankForTier                = commerce.PlanRankForTier
	mapBillingInterval             = commerce.MapBillingInterval
	ensureStripePriceMatchesBundle = commerce.EnsureStripePriceMatchesBundle
)
