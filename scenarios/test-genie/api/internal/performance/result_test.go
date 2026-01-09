package performance

import (
	"testing"
	"time"
)

func TestBenchmarkSummary_String_AllPassed(t *testing.T) {
	summary := BenchmarkSummary{
		GoBuildDuration: 5 * time.Second,
		GoBuildPassed:   true,
		UIBuildDuration: 30 * time.Second,
		UIBuildPassed:   true,
	}

	str := summary.String()

	if str == "" {
		t.Error("expected non-empty string")
	}
	// Should contain "passed" for both
	if !containsSubstring(str, "passed") {
		t.Errorf("expected 'passed' in string, got %s", str)
	}
}

func TestBenchmarkSummary_String_GoFailed(t *testing.T) {
	summary := BenchmarkSummary{
		GoBuildDuration: 5 * time.Second,
		GoBuildPassed:   false,
		UIBuildDuration: 30 * time.Second,
		UIBuildPassed:   true,
	}

	str := summary.String()

	if !containsSubstring(str, "failed") {
		t.Errorf("expected 'failed' in string, got %s", str)
	}
}

func TestBenchmarkSummary_String_UISkipped(t *testing.T) {
	summary := BenchmarkSummary{
		GoBuildDuration: 5 * time.Second,
		GoBuildPassed:   true,
		UIBuildSkipped:  true,
	}

	str := summary.String()

	if !containsSubstring(str, "skipped") {
		t.Errorf("expected 'skipped' in string, got %s", str)
	}
}

func TestFailMissingDependency(t *testing.T) {
	result := FailMissingDependency(
		ErrTest,
		"Install the dependency",
	)

	if result.Success {
		t.Error("expected failure result")
	}
	if result.FailureClass != FailureClassMissingDependency {
		t.Errorf("expected missing_dependency, got %s", result.FailureClass)
	}
	if result.Remediation != "Install the dependency" {
		t.Errorf("unexpected remediation: %s", result.Remediation)
	}
}

// Helper for tests
var ErrTest = &testError{}

type testError struct{}

func (e *testError) Error() string { return "test error" }

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstringHelper(s, substr))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
