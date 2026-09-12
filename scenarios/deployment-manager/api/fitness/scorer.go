package fitness

import (
	"encoding/json"
	"os"
	"path/filepath"
)

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
func CalculateScore(scenario string, tier int) Score {
	policy, err := GetTierFitnessPolicy(tier)
	if err != nil {
		// Invalid tier - return policy with zero overall and error reason
		return Score{
			Overall:       policy.Overall,
			BlockerReason: policy.BlockerReason,
		}
	}

	score := Score(policy)
	if adjustment, ok := declaredTierFeasibility(scenario, tier); ok {
		score.Overall = adjustment
		if adjustment < tierWarningThreshold && score.BlockerReason == "" {
			score.BlockerReason = "declared scenario tier feasibility is below the warning threshold"
		}
	}
	return score
}

func declaredTierFeasibility(scenario string, tier int) (int, bool) {
	if scenario == "" {
		return 0, false
	}
	root := os.Getenv("VROOLI_ROOT")
	if root == "" {
		root = findRepoRoot()
	}
	data, err := os.ReadFile(filepath.Join(root, "scenarios", scenario, ".vrooli", "monetization.json"))
	if err != nil {
		return 0, false
	}
	var declaration struct {
		TierFeasibility map[string]int `json:"tier_feasibility"`
	}
	if json.Unmarshal(data, &declaration) != nil {
		return 0, false
	}
	value, ok := declaration.TierFeasibility[GetTierDisplayName(tier)]
	return value, ok
}

func findRepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".vrooli")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}
