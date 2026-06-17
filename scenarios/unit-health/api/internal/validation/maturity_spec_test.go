package validation

import "testing"

// specSeverityToLocal maps a maturity.json severity_default onto the lowercase
// severity codeSeverity uses.
func specSeverityToLocal(s string) string {
	switch s {
	case "SEVERITY_ERROR", "SEVERITY_BLOCKER":
		return "error"
	case "SEVERITY_WARNING":
		return "warning"
	case "SEVERITY_INFO":
		return "info"
	default:
		return ""
	}
}

// TestCodeSeverityMatchesSpec proves the engine's codeSeverity map and the
// scenario's .vrooli/maturity.json never drift: every spec finding code has a
// codeSeverity entry with the same severity, and every codeSeverity key is a
// declared spec code. This is the "validate every emitted finding code has a
// mapping" guard from Phase 5.
func TestCodeSeverityMatchesSpec(t *testing.T) {
	spec := loadSpec(t)

	for code, mapping := range spec.Findings {
		want := specSeverityToLocal(mapping.SeverityDefault)
		if want == "" {
			t.Errorf("spec code %q has unrecognized severity_default %q", code, mapping.SeverityDefault)
			continue
		}
		got, ok := codeSeverity[code]
		if !ok {
			t.Errorf("spec code %q has no codeSeverity entry", code)
			continue
		}
		if got != want {
			t.Errorf("code %q severity: codeSeverity=%q spec=%q", code, got, want)
		}
	}

	for code := range codeSeverity {
		if _, ok := spec.Findings[code]; !ok {
			t.Errorf("codeSeverity has code %q with no maturity.json mapping", code)
		}
	}
}

// TestMaturityLadderGateAdvisorySplit encodes the Phase 5 honesty contract:
// L0–L3 are enforced gates (an ERROR/BLOCKER, non-advisory finding can block
// local maturity), while L4–L5 are advisory tiers (measured, never gating).
// This is the anti-drift guard that keeps the ladder from silently overselling
// L4/L5 as enforced — or from a future edit promoting an L4/L5 finding to a
// blocker.
func TestMaturityLadderGateAdvisorySplit(t *testing.T) {
	spec := loadSpec(t)
	advisoryTier := map[string]bool{"L4": true, "L5": true}

	gates := 0
	for code, m := range spec.Findings {
		sev := specSeverityToLocal(m.SeverityDefault)
		isAdvisory := string(m.GlobalImpact) == "advisory"
		// A finding gates local maturity iff it is ERROR/BLOCKER and not advisory
		// (mirrors maturity-go's blocksLocalMaturity).
		gating := sev == "error" && !isAdvisory

		if advisoryTier[m.LocalLevelImpact] {
			if !isAdvisory {
				t.Errorf("L4/L5 finding %q must be global_impact=advisory (non-gating), got %q", code, m.GlobalImpact)
			}
			if sev == "error" {
				t.Errorf("L4/L5 finding %q must not be ERROR severity (advisory tier never blocks)", code)
			}
			if gating {
				t.Errorf("L4/L5 finding %q must never gate local maturity", code)
			}
		} else if gating {
			gates++
		}
	}
	if gates == 0 {
		t.Error("expected at least one enforced L0–L3 gate (ERROR, non-advisory); the ladder would gate nothing")
	}
}
