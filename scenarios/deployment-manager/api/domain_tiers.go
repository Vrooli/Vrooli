package main

import "fmt"

// Tier constants represent deployment target tiers.
// Each tier has different fitness characteristics based on:
// - Portability: How easily the scenario can run on different systems
// - Resources: Memory/CPU/storage constraints of the target platform
// - Licensing: Commercial/OSS licensing compatibility requirements
// - Platform Support: OS/architecture availability of dependencies
const (
	TierLocal      = 1 // Full local development - no constraints
	TierDesktop    = 2 // Standalone desktop app - moderate constraints
	TierMobile     = 3 // Mobile app - heavy resource constraints
	TierSaaS       = 4 // Cloud-hosted SaaS - good web compatibility
	TierEnterprise = 5 // On-premise enterprise - compliance requirements
)

// TierFitnessPolicy defines how a tier evaluates scenario fitness.
// Each policy encapsulates the decision criteria for that deployment target.
type TierFitnessPolicy struct {
	Overall         int
	Portability     int
	Resources       int
	Licensing       int
	PlatformSupport int
	BlockerReason   string
}

// tierFitnessPolicies maps each tier to its fitness evaluation criteria.
// These are the baseline scores before scenario-specific adjustments.
//
// Decision rationale:
// - TierLocal (100%): Development tier accepts everything
// - TierDesktop (75%): Needs lightweight deps, no heavy infra
// - TierMobile (40%): Severely constrained, requires swaps for most resources
// - TierSaaS (85%): Cloud-native, good web compatibility
// - TierEnterprise (60%): Compliance/audit requirements create friction
var tierFitnessPolicies = map[int]TierFitnessPolicy{
	TierLocal: {
		Overall:         100,
		Portability:     100,
		Resources:       100,
		Licensing:       100,
		PlatformSupport: 100,
		BlockerReason:   "", // Local tier has no blockers
	},
	TierDesktop: {
		Overall:         75,
		Portability:     80,
		Resources:       70,
		Licensing:       80,
		PlatformSupport: 70,
		BlockerReason:   "", // Desktop tier has no default blockers
	},
	TierMobile: {
		Overall:         40,
		Portability:     50,
		Resources:       30,
		Licensing:       60,
		PlatformSupport: 20,
		BlockerReason:   "Mobile tier requires lightweight dependencies (consider swapping postgres->sqlite, ollama->cloud-api)",
	},
	TierSaaS: {
		Overall:         85,
		Portability:     90,
		Resources:       80,
		Licensing:       85,
		PlatformSupport: 85,
		BlockerReason:   "", // SaaS tier has no default blockers
	},
	TierEnterprise: {
		Overall:         60,
		Portability:     70,
		Resources:       80,
		Licensing:       40,
		PlatformSupport: 60,
		BlockerReason:   "Enterprise tier requires license compliance review and audit logging",
	},
}

// IsTierBlocked decides whether a tier has blocking issues that prevent deployment.
// A tier is blocked when its overall fitness is zero.
func IsTierBlocked(tier int) bool {
	policy, exists := tierFitnessPolicies[tier]
	if !exists {
		return true // Unknown tiers are blocked by default
	}
	return policy.Overall == 0
}

// IsTierWarningLevel decides whether a tier's fitness score warrants a warning.
// Scores below 50% indicate significant compatibility concerns.
const tierWarningThreshold = 50

func IsTierWarningLevel(score int) bool {
	return score > 0 && score < tierWarningThreshold
}

// GetTierFitnessPolicy returns the fitness policy for a given tier.
// Returns an invalid tier error for unknown tier values.
func GetTierFitnessPolicy(tier int) (TierFitnessPolicy, error) {
	policy, exists := tierFitnessPolicies[tier]
	if !exists {
		return TierFitnessPolicy{
			Overall:       0,
			BlockerReason: fmt.Sprintf("Invalid tier: %d (must be 1-5)", tier),
		}, fmt.Errorf("invalid tier %d", tier)
	}
	return policy, nil
}

// AllTiers returns the list of all valid deployment tiers.
func AllTiers() []int {
	return []int{TierLocal, TierDesktop, TierMobile, TierSaaS, TierEnterprise}
}

// tierNameMap provides human-readable names for each tier.
var tierNameMap = map[int]string{
	TierLocal:      "local",
	TierDesktop:    "desktop",
	TierMobile:     "mobile",
	TierSaaS:       "saas",
	TierEnterprise: "enterprise",
}

// GetTierDisplayName converts a tier number to its display name.
// Returns "tier-N" format for unknown tiers.
func GetTierDisplayName(tier int) string {
	if name, ok := tierNameMap[tier]; ok {
		return name
	}
	return fmt.Sprintf("tier-%d", tier)
}
