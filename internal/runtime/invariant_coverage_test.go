package runtime

import "testing"

func TestSafeguardInvariantCoverageIncludesDeclaredNVIDIAInvariants(t *testing.T) {
	coverage, err := SafeguardInvariantCoverage()
	if err != nil {
		t.Fatal(err)
	}
	if coverage.SitesWalked == 0 || coverage.InvariantsDeclared < 2 || coverage.InvariantsEvaluated < 2 {
		t.Fatalf("coverage = %+v, want the declared NVIDIA invariants evaluated", coverage)
	}
	if len(coverage.Gaps) != 0 {
		t.Fatalf("coverage gaps = %v, want none for the embedded registry", coverage.Gaps)
	}
}
