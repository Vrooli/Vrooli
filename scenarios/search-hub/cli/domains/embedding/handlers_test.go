package embedding

import "testing"

// TestRequirePassingCompareRejectsMissingOrNonPassingVerdicts [REQ:REQ-P1-018]
func TestRequirePassingCompareRejectsMissingOrNonPassingVerdicts(t *testing.T) {
	for _, verdict := range []string{"", "withheld", "fail"} {
		if err := requirePassingCompare(migrationState{CompareVerdict: verdict}); err == nil {
			t.Fatalf("verdict %q was accepted without a passing comparison", verdict)
		}
	}
	if err := requirePassingCompare(migrationState{CompareVerdict: "pass"}); err != nil {
		t.Fatalf("passing comparison rejected: %v", err)
	}
}
