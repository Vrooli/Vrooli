package main

import "fmt"

// FitnessScore represents deployment fitness metrics for a tier.
type FitnessScore struct {
	Overall         int
	Portability     int
	Resources       int
	Licensing       int
	PlatformSupport int
	BlockerReason   string
}

// calculateFitnessScore computes fitness score for a scenario/tier combination.
// Hard-coded fitness rules per PROBLEMS.md recommendation:
//   - Tier 1 (Local/Dev): All scenarios fit perfectly
//   - Tier 2 (Desktop): Good fit for lightweight scenarios
//   - Tier 3 (Mobile): Limited fit, needs resource swaps
//   - Tier 4 (SaaS): Good fit for web-oriented scenarios
//   - Tier 5 (Enterprise): Requires compliance considerations
func calculateFitnessScore(scenario string, tier int) FitnessScore {
	switch tier {
	case 1: // Local/Dev - always 100% fit
		return FitnessScore{
			Overall:         100,
			Portability:     100,
			Resources:       100,
			Licensing:       100,
			PlatformSupport: 100,
		}
	case 2: // Desktop
		return FitnessScore{
			Overall:         75,
			Portability:     80,
			Resources:       70,
			Licensing:       80,
			PlatformSupport: 70,
		}
	case 3: // Mobile
		return FitnessScore{
			Overall:         40,
			Portability:     50,
			Resources:       30,
			Licensing:       60,
			PlatformSupport: 20,
			BlockerReason:   "Mobile tier requires lightweight dependencies (consider swapping postgres->sqlite, ollama->cloud-api)",
		}
	case 4: // SaaS
		return FitnessScore{
			Overall:         85,
			Portability:     90,
			Resources:       80,
			Licensing:       85,
			PlatformSupport: 85,
		}
	case 5: // Enterprise
		return FitnessScore{
			Overall:         60,
			Portability:     70,
			Resources:       80,
			Licensing:       40,
			PlatformSupport: 60,
			BlockerReason:   "Enterprise tier requires license compliance review and audit logging",
		}
	default:
		return FitnessScore{
			Overall:       0,
			BlockerReason: fmt.Sprintf("Invalid tier: %d (must be 1-5)", tier),
		}
	}
}

// getTierName converts a tier number to its display name.
func getTierName(tier int) string {
	switch tier {
	case 1:
		return "local"
	case 2:
		return "desktop"
	case 3:
		return "mobile"
	case 4:
		return "saas"
	case 5:
		return "enterprise"
	default:
		return fmt.Sprintf("tier-%d", tier)
	}
}
