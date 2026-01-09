package fitness

// Score represents deployment fitness metrics for a tier.
// This is the result type returned from fitness calculations.
type Score struct {
	Overall         int
	Portability     int
	Resources       int
	Licensing       int
	PlatformSupport int
	BlockerReason   string
}

// CalculateScore computes fitness score for a scenario/tier combination.
// Uses the tier fitness policies defined in tiers.go.
//
// Future enhancement: This function can be extended to incorporate
// scenario-specific adjustments based on dependency analysis.
func CalculateScore(scenario string, tier int) Score {
	policy, err := GetTierFitnessPolicy(tier)
	if err != nil {
		// Invalid tier - return policy with zero overall and error reason
		return Score{
			Overall:       policy.Overall,
			BlockerReason: policy.BlockerReason,
		}
	}

	return Score{
		Overall:         policy.Overall,
		Portability:     policy.Portability,
		Resources:       policy.Resources,
		Licensing:       policy.Licensing,
		PlatformSupport: policy.PlatformSupport,
		BlockerReason:   policy.BlockerReason,
	}
}
