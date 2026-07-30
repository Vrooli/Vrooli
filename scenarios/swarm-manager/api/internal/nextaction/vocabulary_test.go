package nextaction

import "testing"

func TestBlockerVocabularyIsExplicitAndFailClosed(t *testing.T) {
	for _, code := range []string{string(PlanChanged), string(PlanNotAccepted), string(PlanInvalid), string(UnmetDependencies), string(QueueCap), string(CostCap), string(CircuitOpen)} {
		if err := ValidateBlockerCode(code); err != nil {
			t.Fatalf("ValidateBlockerCode(%q) error = %v", code, err)
		}
	}
	if err := ValidateBlockerCode("unmapped"); err == nil {
		t.Fatal("ValidateBlockerCode accepted an unmapped code")
	}
}

func TestActionForBlockerUsesCodesNotMessages(t *testing.T) {
	if got := ActionForBlocker(string(PlanInvalid)); got != RepairPlan {
		t.Fatalf("ActionForBlocker(plan_invalid) = %q, want %q", got, RepairPlan)
	}
	if got := ActionForBlocker(string(QueueCap)); got != Run {
		t.Fatalf("ActionForBlocker(queue_cap) = %q, want %q", got, Run)
	}
}
