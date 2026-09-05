package shared

import "testing"

func TestStandardizeFailureClassPreservesProviderProductFailure(t *testing.T) {
	if got := StandardizeFailureClass(FailureClassTestFailure); got != FailureClassTestFailure {
		t.Fatalf("standardized class = %q, want %q", got, FailureClassTestFailure)
	}
}

func TestStandardizeFailureClassMapsLegacyExecutionFailuresToSystem(t *testing.T) {
	if got := StandardizeFailureClass(FailureClassExecution); got != FailureClassSystem {
		t.Fatalf("standardized class = %q, want %q", got, FailureClassSystem)
	}
}
