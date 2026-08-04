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

func TestCompareCanaryRefusesSmallSampleAndPrintsPromotionEdit(t *testing.T) {
	small := CompareCanary("code.default", "old", "new", CanaryArmMetrics{Count: 2}, CanaryArmMetrics{Count: 2})
	if small.Confidence != "insufficient_sample" || small.Recommendation != "" {
		t.Fatalf("small comparison = %+v", small)
	}
	large := CompareCanary("code.default", "old", "new", CanaryArmMetrics{Count: 30, SuccessRate: .8, MedianMS: 100, CostPerRun: 2}, CanaryArmMetrics{Count: 30, SuccessRate: .9, MedianMS: 90, CostPerRun: 1})
	if large.Recommendation != "promote challenger" || large.PromotionEdit == "" {
		t.Fatalf("large comparison = %+v", large)
	}
}
