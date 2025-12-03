package main

// FitnessScore represents deployment fitness metrics for a tier.
// This is the result type returned from fitness calculations.
type FitnessScore struct {
	Overall         int
	Portability     int
	Resources       int
	Licensing       int
	PlatformSupport int
	BlockerReason   string
}

// calculateFitnessScore computes fitness score for a scenario/tier combination.
// Uses the tier fitness policies defined in domain_tiers.go.
//
// Future enhancement: This function can be extended to incorporate
// scenario-specific adjustments based on dependency analysis.
func calculateFitnessScore(scenario string, tier int) FitnessScore {
	policy, err := GetTierFitnessPolicy(tier)
	if err != nil {
		// Invalid tier - return policy with zero overall and error reason
		return FitnessScore{
			Overall:       policy.Overall,
			BlockerReason: policy.BlockerReason,
		}
	}

	return FitnessScore{
		Overall:         policy.Overall,
		Portability:     policy.Portability,
		Resources:       policy.Resources,
		Licensing:       policy.Licensing,
		PlatformSupport: policy.PlatformSupport,
		BlockerReason:   policy.BlockerReason,
	}
}

// getTierName converts a tier number to its display name.
// Delegates to domain_tiers.go for consistent naming.
func getTierName(tier int) string {
	return GetTierDisplayName(tier)
}
