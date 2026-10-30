package runsignal

import "testing"

// A manifest's declared command semantics reach the resolved fact, so a verify
// command's non-zero exit reads as its verdict rather than a failure.
func TestManifestSemanticsReachTheResolution(t *testing.T) {
	resolution := ResolveCatalogArgs([]string{"agent-manager", "effort", "lint", "/tmp/home"})
	if resolution.Command != "agent-manager effort lint" || resolution.SemanticsKind != "verify" {
		t.Fatalf("resolution = %+v, want agent-manager effort lint with verify semantics", resolution)
	}
	exit := 4
	fact := InvocationFact{Outcome: OutcomeFailure, ExitCode: &exit, SemanticsKind: resolution.SemanticsKind}
	if failed(fact) {
		t.Fatal("a verify command's exit 4 is a verdict, not a failure")
	}
}
