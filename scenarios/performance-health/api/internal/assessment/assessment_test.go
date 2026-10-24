package assessment

import "testing"

// [REQ:PH-VALIDATION-002] A nil spec degrades to a nil assessment rather than
// panicking, so the validation handler can surface a degraded result.
func TestBuildNilSpec(t *testing.T) {
	got, err := Build("performance-health", nil, nil)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil assessment for nil spec, got %#v", got)
	}
}
