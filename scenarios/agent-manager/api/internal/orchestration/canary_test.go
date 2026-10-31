package orchestration

import (
	"testing"

	"agent-manager/internal/domain"
)

func TestSelectCanaryArmIsReproducibleAndHonorsBounds(t *testing.T) {
	candidate := domain.ExecutionCandidate{Model: "incumbent", ChallengerModel: "challenger", ChallengerSampleRate: 0.5}
	if first, second := SelectCanaryArm("run-1", candidate), SelectCanaryArm("run-1", candidate); first != second {
		t.Fatalf("same seed produced %q and %q", first, second)
	}
	candidate.ChallengerSampleRate = 0
	if got := SelectCanaryArm("run-1", candidate); got != CanaryArmIncumbent {
		t.Fatalf("zero rate = %q", got)
	}
	candidate.ChallengerSampleRate = 1
	if got := SelectCanaryArm("run-1", candidate); got != CanaryArmChallenger {
		t.Fatalf("one rate = %q", got)
	}
}
