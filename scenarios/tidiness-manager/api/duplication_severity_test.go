package main

import "testing"

// TestResolveDuplicationSeverity_StructuralIsWarning proves that a uniform
// descriptor block in a declarative-wiring file is downgraded from error
// (high) to warning (medium) — the exact false gate this work removes.
func TestResolveDuplicationSeverity_StructuralIsWarning(t *testing.T) {
	lines := splitLines(descriptorBlock)

	// severityForDuplication(75, 10) => high (a 75-line block).
	base := severityForDuplication(75, 10)
	if base != "high" {
		t.Fatalf("precondition: expected base severity high, got %q", base)
	}

	sev, structural := resolveDuplicationSeverity(base, FileRoleDeclarativeWiring, lines)
	if !structural {
		t.Error("descriptor block should be confirmed structural")
	}
	if sev != "medium" {
		t.Errorf("structural declarative-wiring duplication should cap at medium (warning), got %q", sev)
	}
}

// TestResolveDuplicationSeverity_LogicKeepsError proves a genuine logic block —
// even inside a role-named (declarative-wiring) file — keeps its error severity.
func TestResolveDuplicationSeverity_LogicKeepsError(t *testing.T) {
	lines := splitLines(logicBlock)
	base := severityForDuplication(75, 10) // high

	sev, structural := resolveDuplicationSeverity(base, FileRoleDeclarativeWiring, lines)
	if structural {
		t.Error("branchy logic block must not be treated as structural")
	}
	if sev != "high" {
		t.Errorf("logic duplication keeps normal severity, got %q", sev)
	}
}

// TestResolveDuplicationSeverity_ProductionNeverCaps proves the cap only applies
// to cappable roles: a structural-looking block in ordinary production code is
// NOT downgraded (production duplication is real debt).
func TestResolveDuplicationSeverity_ProductionNeverCaps(t *testing.T) {
	lines := splitLines(descriptorBlock)
	base := severityForDuplication(75, 10) // high

	sev, structural := resolveDuplicationSeverity(base, FileRoleProduction, lines)
	if structural {
		t.Error("production role should never report structural cap")
	}
	if sev != "high" {
		t.Errorf("production duplication keeps normal severity, got %q", sev)
	}
}

func TestCapSeverityAtMedium(t *testing.T) {
	cases := map[string]string{
		"high":   "medium",
		"HIGH":   "medium",
		"medium": "medium",
		"low":    "low",
	}
	for in, want := range cases {
		if got := capSeverityAtMedium(in); got != want {
			t.Errorf("capSeverityAtMedium(%q) = %q, want %q", in, got, want)
		}
	}
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}
