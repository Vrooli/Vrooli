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
