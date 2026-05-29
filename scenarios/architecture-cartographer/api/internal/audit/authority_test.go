package audit

import (
	"testing"

	"architecture-cartographer/internal/conflicts"
)

func TestDecideOutcomeWithAuthority_LowConfidenceFailsByDefault(t *testing.T) {
	o, reason := decideOutcomeWithAuthority(nil, conflicts.SeverityWarn, "low", false)
	if o != OutcomeFindings {
		t.Fatalf("low confidence must fail by default, got %s", o)
	}
	if reason == "" {
		t.Fatal("expected OutcomeReason describing low authority")
	}
}

func TestDecideOutcomeWithAuthority_LowConfidenceAllowedExplicitly(t *testing.T) {
	o, reason := decideOutcomeWithAuthority(nil, conflicts.SeverityWarn, "low", true)
	if o != OutcomeClean {
		t.Fatalf("--allow-low-authority should permit clean, got %s", o)
	}
	if reason != "" {
		t.Fatalf("expected empty reason on clean, got %q", reason)
	}
}

func TestDecideOutcomeWithAuthority_HighConfidenceUnaffected(t *testing.T) {
	o, _ := decideOutcomeWithAuthority(nil, conflicts.SeverityWarn, "high", false)
	if o != OutcomeClean {
		t.Fatalf("high confidence without findings should be clean, got %s", o)
	}
}

func TestDecideOutcomeWithAuthority_FindingsBeatAuthorityAxis(t *testing.T) {
	in := []conflicts.Conflict{{Type: "cycle", Severity: conflicts.SeverityError}}
	o, reason := decideOutcomeWithAuthority(in, conflicts.SeverityWarn, "low", true)
	if o != OutcomeFindings {
		t.Fatalf("severity findings must override authority axis, got %s", o)
	}
	if reason != "" {
		t.Fatalf("findings outcome should leave reason empty (caller renders findings), got %q", reason)
	}
}
