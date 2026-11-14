// Canary responsibility: deterministically route sampled challenger runs.
package orchestration

import (
	"crypto/sha256"
	"encoding/binary"
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
