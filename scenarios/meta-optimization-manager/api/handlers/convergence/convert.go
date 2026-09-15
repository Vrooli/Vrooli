package convergence

import (
	internalconv "meta-optimization-manager/internal/convergence"

	convergencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/convergence"
)

// This file is the only translation point between the convergence proto wire
// enums and the domain vocabulary (internal/convergence). The domain layer never
// imports proto (api-steer §7).

func tierToProto(t internalconv.FitnessTier) convergencev1.FitnessTier {
	switch t {
	case internalconv.TierStrong:
		return convergencev1.FitnessTier_FITNESS_TIER_STRONG
	case internalconv.TierFair:
		return convergencev1.FitnessTier_FITNESS_TIER_FAIR
	case internalconv.TierWeak:
		return convergencev1.FitnessTier_FITNESS_TIER_WEAK
	default:
		return convergencev1.FitnessTier_FITNESS_TIER_UNSPECIFIED
	}
}

func eligibilityFromProto(e convergencev1.ReferenceEligibility) internalconv.ReferenceEligibility {
	switch e {
	case convergencev1.ReferenceEligibility_REFERENCE_ELIGIBILITY_ELIGIBLE:
		return internalconv.EligibilityEligible
	case convergencev1.ReferenceEligibility_REFERENCE_ELIGIBILITY_CANDIDATE:
		return internalconv.EligibilityCandidate
	case convergencev1.ReferenceEligibility_REFERENCE_ELIGIBILITY_INELIGIBLE:
		return internalconv.EligibilityIneligible
	default:
		return internalconv.EligibilityUnspecified
	}
}

func eligibilityToProto(e internalconv.ReferenceEligibility) convergencev1.ReferenceEligibility {
	switch e {
	case internalconv.EligibilityEligible:
		return convergencev1.ReferenceEligibility_REFERENCE_ELIGIBILITY_ELIGIBLE
	case internalconv.EligibilityCandidate:
		return convergencev1.ReferenceEligibility_REFERENCE_ELIGIBILITY_CANDIDATE
	case internalconv.EligibilityIneligible:
		return convergencev1.ReferenceEligibility_REFERENCE_ELIGIBILITY_INELIGIBLE
	default:
		return convergencev1.ReferenceEligibility_REFERENCE_ELIGIBILITY_UNSPECIFIED
	}
}
