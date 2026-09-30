package audit

import (
	"strings"
	"testing"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/domains"
)

// TestDecideOutcomeWithAuthority_Table covers the
// {confidence} × {allowLow} × {findings?} matrix. Today the gate has
// three confidence values (missing/low/high) plus the unspecified
// fallback (treated as clean by default, mirroring legacy behavior).
func TestDecideOutcomeWithAuthority_Table(t *testing.T) {
	withFinding := []conflicts.Conflict{{Type: "cycle", Severity: conflicts.SeverityError, FindingClass: conflicts.FindingClassDeterministic}}

	cases := []struct {
		name        string
		findings    []conflicts.Conflict
		confidence  string
		allowLow    bool
		wantOutcome Outcome
		wantReason  string // empty means "any (don't assert)" handled separately
		reasonHas   string // substring required when wantReason == ""
	}{
		{"missing/strict/no-findings", nil, string(domains.ConfidenceMissing), false, OutcomeFindings, "", "DOMAINS.md"},
		{"missing/allow/no-findings", nil, string(domains.ConfidenceMissing), true, OutcomeClean, "", ""},
		{"missing/strict/with-findings", withFinding, string(domains.ConfidenceMissing), false, OutcomeFindings, "", ""},
		{"low/strict/no-findings", nil, string(domains.ConfidenceLow), false, OutcomeFindings, "", "DOMAINS.md"},
		{"low/allow/no-findings", nil, string(domains.ConfidenceLow), true, OutcomeClean, "", ""},
		{"low/strict/with-findings", withFinding, string(domains.ConfidenceLow), false, OutcomeFindings, "", ""},
		{"high/strict/no-findings", nil, string(domains.ConfidenceHigh), false, OutcomeClean, "", ""},
		{"high/strict/with-findings", withFinding, string(domains.ConfidenceHigh), false, OutcomeFindings, "", ""},
		{"unspecified/strict/no-findings", nil, "", false, OutcomeClean, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o, reason := decideOutcomeWithAuthority("demo", tc.findings, conflicts.SeverityWarn, tc.confidence, tc.allowLow)
			if o != tc.wantOutcome {
				t.Fatalf("outcome=%s, want %s", o, tc.wantOutcome)
			}
			if tc.reasonHas != "" && !strings.Contains(reason, tc.reasonHas) {
				t.Fatalf("reason=%q, want substring %q", reason, tc.reasonHas)
			}
			if tc.wantOutcome == OutcomeClean && reason != "" {
				t.Fatalf("clean outcome must have empty reason, got %q", reason)
			}
		})
	}
}

// TestMissingAuthorityMessage_LeadsWithFix asserts the remediation phrasing
// puts DOMAINS.md before --allow-low-authority. Agents reach for what's
// named first; we want the fix named first.
func TestMissingAuthorityMessage_LeadsWithFix(t *testing.T) {
	msg := missingAuthorityMessage("demo")
	domainsIdx := strings.Index(msg, "DOMAINS.md")
	bypassIdx := strings.Index(msg, "--allow-low-authority")
	if domainsIdx < 0 || bypassIdx < 0 {
		t.Fatalf("missing fix or bypass term: %q", msg)
	}
	if domainsIdx > bypassIdx {
		t.Fatalf("fix must precede bypass in message; got %q", msg)
	}
}

// TestLowAuthorityMessage_LeadsWithFix mirrors the missing case.
func TestLowAuthorityMessage_LeadsWithFix(t *testing.T) {
	msg := lowAuthorityMessage("demo")
	domainsIdx := strings.Index(msg, "DOMAINS.md")
	bypassIdx := strings.Index(msg, "--allow-low-authority")
	if domainsIdx < 0 || bypassIdx < 0 {
		t.Fatalf("missing fix or bypass term: %q", msg)
	}
	if domainsIdx > bypassIdx {
		t.Fatalf("fix must precede bypass in message; got %q", msg)
	}
}
