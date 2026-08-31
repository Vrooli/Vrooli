package catalogcoverage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildCorpusReportEmitsAllPlanInvariants(t *testing.T) {
	root := t.TempDir()
	version := filepath.Join(root, "scenarios/react-component-library/library/components/Panel/versions/1.0.0")
	require.NoError(t, os.MkdirAll(version, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "scenarios/react-component-library/library/components/Panel/component.json"), []byte(`{"latest":"1.0.0"}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(version, "Panel.tsx"), []byte("export function Panel() { return null; }\n"), 0o600))
	report, err := BuildCorpusReport(root)
	require.NoError(t, err)
	require.Equal(t, "corpus-report/v1", report.SchemaVersion)
	require.Len(t, report.Invariants, 20)
	for _, invariant := range report.Invariants {
		require.NotEmpty(t, invariant.ID)
		require.NotEmpty(t, invariant.Unit)
	}
}
