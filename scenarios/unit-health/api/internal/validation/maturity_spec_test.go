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
	if spec.Version != "2.0.0" {
		t.Errorf("spec version = %q, want 2.0.0", spec.Version)
	}
	if len(spec.Capabilities) == 0 {
		t.Error("spec must declare capability ladders")
	}
	if spec.Fallback.CapabilityID == "" {
		t.Error("fallback must declare capability_id")
	}

	for code, mapping := range spec.Findings {
		if mapping.CapabilityID == "" {
			t.Errorf("spec code %q has no capability_id", code)
		}
		if mapping.CleanRequirement == "" {
			t.Errorf("spec code %q has no clean_requirement", code)
		}
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

// TestMaturityCapabilitiesGateAdvisorySplit encodes the honesty contract:
// capability foundations can gate local maturity, while coverage-quality and
// stability-traceability remain measured, non-blocking hardening signals.
func TestMaturityCapabilitiesGateAdvisorySplit(t *testing.T) {
	spec := loadSpec(t)
	advisoryCapabilities := map[string]bool{
		"coverage_quality":       true,
		"stability_traceability": true,
	}

	gates := 0
	for code, m := range spec.Findings {
		sev := specSeverityToLocal(m.SeverityDefault)
		isAdvisory := string(m.GlobalImpact) == "advisory"
		// A finding gates local maturity iff it is ERROR/BLOCKER and not advisory
		// (mirrors maturity-go's blocksLocalMaturity).
		gating := sev == "error" && !isAdvisory

		if advisoryCapabilities[m.CapabilityID] {
			if !isAdvisory {
				t.Errorf("advisory capability finding %q must be global_impact=advisory (non-gating), got %q", code, m.GlobalImpact)
			}
			if sev == "error" {
				t.Errorf("advisory capability finding %q must not be ERROR severity", code)
			}
			if gating {
				t.Errorf("advisory capability finding %q must never gate local maturity", code)
			}
		} else if gating {
			gates++
		}
	}
	if gates == 0 {
		t.Error("expected at least one enforced L0–L3 gate (ERROR, non-advisory); the ladder would gate nothing")
	}
}
