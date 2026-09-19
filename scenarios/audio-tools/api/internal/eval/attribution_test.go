package eval

import "testing"

func TestAttributeIngressByAblation_PairsExtractionSiblings(t *testing.T) {
	report := &EvalReport{PerStrategy: []StrategyReport{
		{
			Strategy:          "batch/extract_off_verify_off",
			BaseStrategy:      "batch",
			ExtractionEnabled: false,
			EditCounts:        EditCounts{Deletions: 2, Substitutions: 1},
			StageAttribution:  StageAttribution{StrategyLostWords: 3},
		},
		{
			Strategy:          "batch/extract_on_verify_off",
			BaseStrategy:      "batch",
			ExtractionEnabled: true,
			EditCounts:        EditCounts{Deletions: 5, Substitutions: 1},
			StageAttribution:  StageAttribution{StrategyLostWords: 6},
		},
	}}

	AttributeIngressByAblation(report)

	// extraction-off row: ingress is not applicable, stays zero.
	if got := report.PerStrategy[0].StageAttribution.IngressLostWords; got != 0 {
		t.Fatalf("extract-off IngressLostWords = %d, want 0", got)
	}
	// extraction-on row: 5 deletions vs 2 baseline => 3 attributed to ingress,
	// and those 3 moved out of the strategy bucket (6 - 3 = 3).
	on := report.PerStrategy[1].StageAttribution
	if on.IngressLostWords != 3 {
		t.Fatalf("extract-on IngressLostWords = %d, want 3", on.IngressLostWords)
	}
	if on.StrategyLostWords != 3 {
		t.Fatalf("extract-on StrategyLostWords = %d, want 3 (surplus moved to ingress)", on.StrategyLostWords)
	}
}

func TestAttributeStages_PartitionsWithoutRejections(t *testing.T) {
	got := attributeStages(StrategyReport{EditCounts: EditCounts{Substitutions: 2, Deletions: 3}})
	if got.StrategyLostWords != 5 || got.EgressLostWords != 0 || got.IngressLostWords != 0 {
		t.Fatalf("unexpected attribution %+v, want strategy=5 egress=0 ingress=0", got)
	}
}

func TestAttributeStages_MovesDeletionsToEgressOnRejection(t *testing.T) {
	got := attributeStages(StrategyReport{
		EditCounts:            EditCounts{Substitutions: 2, Deletions: 3},
		SpeakerRejectionCount: 1,
	})
	// Deletions belong to egress; strategy keeps only substitutions. The buckets
	// must partition subs+dels with no double count.
	if got.StrategyLostWords != 2 || got.EgressLostWords != 3 {
		t.Fatalf("unexpected attribution %+v, want strategy=2 egress=3", got)
	}
	if sum := got.IngressLostWords + got.StrategyLostWords + got.EgressLostWords; sum != 5 {
		t.Fatalf("buckets sum to %d, want 5 (subs+dels)", sum)
	}
}

func TestAttributeIngressByAblation_ReclaimsFromEgressOnRejection(t *testing.T) {
	report := &EvalReport{PerStrategy: []StrategyReport{
		{
			BaseStrategy: "batch", ConditionGroup: "clean", VerificationEnabled: true,
			EditCounts:       EditCounts{Deletions: 2},
			StageAttribution: attributeStages(StrategyReport{EditCounts: EditCounts{Deletions: 2}, SpeakerRejectionCount: 1}),
		},
		{
			BaseStrategy: "batch", ConditionGroup: "clean", VerificationEnabled: true, ExtractionEnabled: true,
			EditCounts:       EditCounts{Substitutions: 1, Deletions: 5},
			StageAttribution: attributeStages(StrategyReport{EditCounts: EditCounts{Substitutions: 1, Deletions: 5}, SpeakerRejectionCount: 1}),
		},
	}}

	AttributeIngressByAblation(report)

	on := report.PerStrategy[1].StageAttribution
	// surplus = 5 - 2 = 3 attributed to ingress, reclaimed from egress (5 - 3 = 2)
	// where attributeStages parked the deletions, not double-counted.
	if on.IngressLostWords != 3 {
		t.Fatalf("IngressLostWords = %d, want 3", on.IngressLostWords)
	}
	if on.EgressLostWords != 2 {
		t.Fatalf("EgressLostWords = %d, want 2 (surplus reclaimed from egress)", on.EgressLostWords)
	}
	if on.StrategyLostWords != 1 {
		t.Fatalf("StrategyLostWords = %d, want 1 (substitutions untouched)", on.StrategyLostWords)
	}
	if sum := on.IngressLostWords + on.StrategyLostWords + on.EgressLostWords; sum != 6 {
		t.Fatalf("buckets sum to %d, want 6 (subs+dels)", sum)
	}
}

func TestAttributeIngressByAblation_NoSiblingLeavesZero(t *testing.T) {
	report := &EvalReport{PerStrategy: []StrategyReport{{
		Strategy:          "batch/extract_on_verify_off",
		BaseStrategy:      "batch",
		ExtractionEnabled: true,
		EditCounts:        EditCounts{Deletions: 5},
		StageAttribution:  StageAttribution{StrategyLostWords: 5},
	}}}

	AttributeIngressByAblation(report)

	if got := report.PerStrategy[0].StageAttribution.IngressLostWords; got != 0 {
		t.Fatalf("IngressLostWords without a sibling = %d, want 0", got)
	}
}

func TestAttributeIngressByAblation_DistinctGroupsDoNotCross(t *testing.T) {
	report := &EvalReport{PerStrategy: []StrategyReport{
		{Strategy: "batch/off/clean", BaseStrategy: "batch", ConditionGroup: "clean", EditCounts: EditCounts{Deletions: 1}, StageAttribution: StageAttribution{StrategyLostWords: 1}},
		{Strategy: "batch/on/clean", BaseStrategy: "batch", ExtractionEnabled: true, ConditionGroup: "clean", EditCounts: EditCounts{Deletions: 4}, StageAttribution: StageAttribution{StrategyLostWords: 4}},
		{Strategy: "batch/on/noisy", BaseStrategy: "batch", ExtractionEnabled: true, ConditionGroup: "noisy", EditCounts: EditCounts{Deletions: 9}, StageAttribution: StageAttribution{StrategyLostWords: 9}},
	}}

	AttributeIngressByAblation(report)

	if got := report.PerStrategy[1].StageAttribution.IngressLostWords; got != 3 {
		t.Fatalf("clean extract-on IngressLostWords = %d, want 3", got)
	}
	// The noisy extract-on row has no extract-off sibling in its own group, so
	// it must not borrow the clean group's baseline.
	if got := report.PerStrategy[2].StageAttribution.IngressLostWords; got != 0 {
		t.Fatalf("noisy extract-on IngressLostWords = %d, want 0 (no same-group baseline)", got)
	}
}
