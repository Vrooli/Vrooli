package assessment

import (
	"path/filepath"
	"runtime"
	"testing"

	intent "intent-go"

	"github.com/stretchr/testify/require"
	maturity "github.com/vrooli/maturity-go/assessment"
)

// scenarioRoot resolves scenarios/business-health from this file's location
// so the test is independent of the working directory.
func scenarioRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

// [REQ:BH-PRV-004] .vrooli/maturity.json loads, validates, and matches identity.
func TestMaturitySpecLoadsAndValidates(t *testing.T) {
	spec, err := maturity.LoadSpecFromScenario(scenarioRoot(t))
	require.NoError(t, err)
	require.NoError(t, maturity.ValidateSpec(*spec))
	require.Equal(t, "business-health", spec.Provider)
	require.Equal(t, "business", spec.Phase)
	require.Len(t, spec.Capabilities, 4)

	caps := map[string]bool{}
	for _, c := range spec.Capabilities {
		caps[c.ID] = true
	}
	for _, want := range []string{"prd_contract", "requirements_registry", "intent_linkage", "evidence_traceability"} {
		require.True(t, caps[want], "capability %q missing", want)
	}
}

// [REQ:BH-FIX-003] Honest fixer accounting: every mapping carries a valid
// fix_class; manual mappings carry a written reason; auto mappings carry a
// fixer_status (pending until the fixer ships in Phase 6).
func TestMaturitySpecFixAccounting(t *testing.T) {
	spec, err := maturity.LoadSpecFromScenario(scenarioRoot(t))
	require.NoError(t, err)
	for code, m := range spec.Findings {
		require.NotEmpty(t, m.FixClass, "finding %q missing fix_class", code)
		switch maturity.FixClass(m.FixClass) {
		case maturity.FixClassManual:
			require.NotEmpty(t, m.FixReason, "manual finding %q missing reason", code)
		case maturity.FixClassAuto, maturity.FixClassExternal:
			require.NotEmpty(t, m.FixerStatus, "fixable finding %q missing fixer_status", code)
		default:
			t.Fatalf("finding %q has unknown fix_class %q", code, m.FixClass)
		}
	}
}

// [REQ:BH-PRV-001] The builder maps neutral findings into a valid shared
// assessment (empty finding set = passing assessment with every capability
// at its clean level).
func TestBuilderEmptyFindings(t *testing.T) {
	spec, err := maturity.LoadSpecFromScenario(scenarioRoot(t))
	require.NoError(t, err)
	b, err := NewBuilder(spec)
	require.NoError(t, err)

	a, err := b.Build("fixture-scenario", nil)
	require.NoError(t, err)
	require.NoError(t, maturity.ValidateAssessment(a))
	require.NoError(t, maturity.RequireIdentity("business-health", "business", a))
	require.Empty(t, a.GetFindings())
}

// [REQ:BH-PRV-003] Severity tokens normalize and unknown values degrade to
// UNSPECIFIED rather than inventing severity.
func TestSeverityToken(t *testing.T) {
	require.Equal(t, "SEVERITY_ERROR", severityToken("error"))
	require.Equal(t, "SEVERITY_WARNING", severityToken("warning"))
	require.Equal(t, "SEVERITY_INFO", severityToken("info"))
	require.Equal(t, "SEVERITY_ERROR", severityToken("SEVERITY_ERROR"))
	require.Equal(t, "SEVERITY_UNSPECIFIED", severityToken("critical"))
}

// [REQ:BH-PRV-001] Findings with known codes map into the assessment with
// their maturity metadata attached from the spec.
func TestBuilderMapsKnownFinding(t *testing.T) {
	spec, err := maturity.LoadSpecFromScenario(scenarioRoot(t))
	require.NoError(t, err)
	b, err := NewBuilder(spec)
	require.NoError(t, err)

	a, err := b.Build("fixture-scenario", []intent.Finding{{
		Code:      intent.CodeOTOrphan,
		Severity:  "warning",
		Message:   "OT-P0-001 has no requirement",
		Locations: []string{"PRD.md"},
	}})
	require.NoError(t, err)
	require.Len(t, a.GetFindings(), 1)
	f := a.GetFindings()[0]
	require.Equal(t, intent.CodeOTOrphan, f.GetCode())
	require.NotNil(t, f.GetMaturity())
	require.Equal(t, "intent_linkage", f.GetMaturity().GetCapabilityId())
}
