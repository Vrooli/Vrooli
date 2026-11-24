package gates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadStampReportPromotesBuildArtifactIntoGovernedState(t *testing.T) {
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	root := t.TempDir()
	buildReport := filepath.Join(root, "ui", "dist", "asset-stamp-report.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(buildReport), 0o700))
	require.NoError(t, os.WriteFile(buildReport, []byte(`{"generatedAt":"build-1","assets":[{"asset":"asset-only","state":"stamped"},{"asset":"asset-fallback","sourceSlot":"source-slot","state":"exempt-permanent","reason":"renders no root"},{"asset":"","identity":"identity-key","state":"unbundled"},{"asset":"","sourceSlot":"","identity":""}]}`), 0o600))

	report, err := LoadStampReport(root)
	require.NoError(t, err)
	require.True(t, report.Present)
	require.Equal(t, "build-1", report.Generated)
	require.Equal(t, StampStamped, report.State("asset-only"))
	require.Equal(t, StampExemptPermanent, report.State("source-slot"), "source slots take precedence over asset IDs")
	require.Equal(t, "renders no root", report.Reasons["source-slot"])
	require.Equal(t, StampUnbundled, report.State("identity-key"))
	require.Equal(t, StampUnknown, report.State("missing"))
	require.False(t, report.Measurable("source-slot"))
	require.True(t, report.Measurable("asset-only"))

	// A subsequent read comes from promoted state, not the mutable build output.
	require.NoError(t, os.WriteFile(buildReport, []byte(`{"generatedAt":"build-2","assets":[{"asset":"asset-only","state":"exempt-backlog"}]}`), 0o600))
	persisted, err := LoadStampReport(root)
	require.NoError(t, err)
	require.Equal(t, "build-1", persisted.Generated)
	require.Equal(t, StampStamped, persisted.State("asset-only"))
}

func TestLoadStampReportTreatsMissingAndMalformedBuildArtifactsExplicitly(t *testing.T) {
	t.Run("no report means unknown state", func(t *testing.T) {
		t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
		report, err := LoadStampReport(t.TempDir())
		require.NoError(t, err)
		require.False(t, report.Present)
		require.Equal(t, StampUnknown, report.State("asset"))
	})
	t.Run("malformed promoted report is rejected", func(t *testing.T) {
		t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
		root := t.TempDir()
		path := filepath.Join(root, "ui", "dist", "asset-stamp-report.json")
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte("{"), 0o600))
		_, err := LoadStampReport(root)
		require.Error(t, err)
	})
}

func TestClassifyUnmeasuredSeparatesUnstampedUncapturedAndPermanentExemptions(t *testing.T) {
	report := StampReport{Present: true, States: map[string]StampState{
		"captured": StampStamped, "permanent": StampExemptPermanent, "backlog": StampExemptBacklog, "unbundled": StampUnbundled,
	}}
	unstamped, uncaptured := classifyUnmeasured(report, []string{"captured", "permanent", "backlog", "unbundled", "unknown"})
	require.Equal(t, []string{"backlog", "unbundled", "unknown"}, unstamped)
	require.Equal(t, []string{"captured"}, uncaptured)

	absent, absentCaptured := classifyUnmeasured(StampReport{}, []string{"unknown"})
	require.Equal(t, []string{"unknown"}, absent)
	require.Empty(t, absentCaptured)
}
