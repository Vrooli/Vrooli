package catalogcoverage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCountMatrixFailuresRequiresFindingAttribution(t *testing.T) {
	output := []byte(`{"cells":[{"verdict":"fail","finding_count":1},{"verdict":"fail","finding_count":0},{"verdict":"pass","finding_count":0},{"verdict":"unmeasured","finding_count":0}]}`)
	got, err := countMatrixFailures(output)
	require.NoError(t, err)
	require.Equal(t, 1, got)
}

func TestBuildCorpusReportEmitsAllPlanInvariants(t *testing.T) {
	root := t.TempDir()
	version := filepath.Join(root, "scenarios/react-component-library/library/components/Panel/versions/1.0.0")
	require.NoError(t, os.MkdirAll(version, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "scenarios/react-component-library/library/components/Panel/component.json"), []byte(`{"latest":"1.0.0"}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(version, "Panel.tsx"), []byte("export function Panel() { return null; }\n"), 0o600))
	report, err := BuildCorpusReport(root)
	require.NoError(t, err)
	require.Equal(t, "corpus-report/v1", report.SchemaVersion)
	require.Len(t, report.Invariants, report.InvariantCount)
	require.Equal(t, 25, report.InvariantCount)
	ids := make(map[string]bool, len(report.Invariants))
	for _, invariant := range report.Invariants {
		require.NotEmpty(t, invariant.ID)
		require.NotEmpty(t, invariant.Unit)
		ids[invariant.ID] = true
	}
	for _, id := range []string{"I7", "I8", "I9", "I12", "I13", "I18", "I21", "I22", "I23", "I25", "I26"} {
		require.True(t, ids[id], "missing invariant %s", id)
	}
	require.False(t, ids["I20"], "removed tautological invariant must not return")
}

func TestNoInvariantIsTautological(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "scenarios/react-component-library/library"), 0o755))
	report, err := BuildCorpusReport(root)
	require.NoError(t, err)
	for _, invariant := range report.Invariants {
		require.NotEqual(t, "I20", invariant.ID, "experience-authority count was a boolean disguised as an invariant")
	}
}

func TestUnavailableLiveMeasurementsAreExplicitFailures(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "scenarios/react-component-library/library"), 0o755))
	report, err := BuildCorpusReport(root)
	require.NoError(t, err)
	byID := make(map[string]CorpusInvariant, len(report.Invariants))
	for _, invariant := range report.Invariants {
		byID[invariant.ID] = invariant
		require.GreaterOrEqual(t, invariant.Value, 0.0, "invariant %s must not use a negative sentinel", invariant.ID)
	}
	for _, id := range []string{"I7", "I8", "I9", "I13"} {
		require.Equal(t, "failed_measurement", byID[id].Status, "invariant %s must explain unavailable probes", id)
	}
}
