// Canary responsibility: deterministically route sampled challenger runs and
// compare durable incumbent/challenger outcomes without auto-promotion.
package orchestration

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"

	"agent-manager/internal/domain"
)

const (
	CanaryArmIncumbent  = "incumbent"
	CanaryArmChallenger = "challenger"
)

// SelectCanaryArm makes the routing decision exactly once from the run id and
// the challenger declaration copied into the execution snapshot.
func SelectCanaryArm(seed string, candidate domain.ExecutionCandidate) string {
	if candidate.ChallengerModel == "" || candidate.ChallengerSampleRate <= 0 {
		return CanaryArmIncumbent
	}
	if candidate.ChallengerSampleRate >= 1 {
		return CanaryArmChallenger
	}
	digest := sha256.Sum256([]byte(seed + "|model-policy-canary"))
	value := float64(binary.BigEndian.Uint64(digest[:8])) / float64(math.MaxUint64)
	if value < candidate.ChallengerSampleRate {
		return CanaryArmChallenger
	}
	return CanaryArmIncumbent
}

func applyCanary(snapshot *domain.ExecutionPolicySnapshot, seed, selectedModel string) {
	if snapshot == nil || snapshot.SelectedIndex < 0 || snapshot.SelectedIndex >= len(snapshot.Candidates) {
		return
	}
	candidate := &snapshot.Candidates[snapshot.SelectedIndex]
	if candidate.Model != selectedModel || candidate.ChallengerModel == "" {
		return
	}
	arm := SelectCanaryArm(seed, *candidate)
	candidate.CanaryArm = arm
	snapshot.CanaryArm = arm
	if arm == CanaryArmChallenger {
		candidate.Model = candidate.ChallengerModel
	}
	snapshot.SelectedCandidate = *candidate
}

type CanaryArmMetrics struct {
	Count       int64   `json:"count"`
	SuccessRate float64 `json:"success_rate"`
	MedianMS    float64 `json:"median_ms"`
	CostPerRun  float64 `json:"cost_per_run"`
}

type CanaryComparison struct {
	Role           string           `json:"role"`
	Incumbent      CanaryArmMetrics `json:"incumbent"`
	Challenger     CanaryArmMetrics `json:"challenger"`
	Confidence     string           `json:"confidence"`
	Recommendation string           `json:"recommendation"`
	PromotionEdit  string           `json:"promotion_edit,omitempty"`
}

func CompareCanary(role, incumbentModel, challengerModel string, incumbent, challenger CanaryArmMetrics) CanaryComparison {
	result := CanaryComparison{Role: role, Incumbent: incumbent, Challenger: challenger, Confidence: "insufficient_sample"}
	if incumbent.Count < 30 || challenger.Count < 30 {
		return result
	}
	result.Confidence = "directional"
	result.Recommendation = "retain incumbent"
	if challenger.SuccessRate >= incumbent.SuccessRate && challenger.CostPerRun <= incumbent.CostPerRun && challenger.MedianMS <= incumbent.MedianMS {
		result.Recommendation = "promote challenger"
		result.PromotionEdit = fmt.Sprintf("set role %q model to %q (challenger %q)", role, challengerModel, incumbentModel)
	}
	return result
}
