package catalogcoverage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/databasetest"
)

func TestBuildCorpusReportEmitsAllPlanInvariants(t *testing.T) {
	root := t.TempDir()
	version := filepath.Join(root, "scenarios/react-component-library/library/components/Panel/versions/1.0.0")
	require.NoError(t, os.MkdirAll(version, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "scenarios/react-component-library/library/components/Panel/component.json"), []byte(`{"latest":"1.0.0"}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(version, "Panel.tsx"), []byte("export function Panel() { return null; }\n"), 0o600))
	report, err := BuildCorpusReport(root, nil)
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
	report, err := BuildCorpusReport(root, nil)
	require.NoError(t, err)
	for _, invariant := range report.Invariants {
		require.NotEqual(t, "I20", invariant.ID, "experience-authority count was a boolean disguised as an invariant")
	}
}

func TestUnavailableLiveMeasurementsAreExplicitFailures(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "scenarios/react-component-library/library"), 0o755))
	report, err := BuildCorpusReport(root, nil)
	require.NoError(t, err)
	byID := make(map[string]CorpusInvariant, len(report.Invariants))
	for _, invariant := range report.Invariants {
		byID[invariant.ID] = invariant
		require.GreaterOrEqual(t, invariant.Value, 0.0, "invariant %s must not use a negative sentinel", invariant.ID)
	}
	for _, id := range []string{"I1", "I2", "I7", "I8", "I9", "I13"} {
		require.Equal(t, "failed_measurement", byID[id].Status, "invariant %s must explain unavailable probes", id)
	}
}

// The DB-backed invariants measure the catalog the caller supplies, never the
// operator's live database: a fixture with one evicted version and no mirror
// reports exactly that.
func TestCatalogInvariantsMeasureTheSuppliedDatabase(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "scenarios/react-component-library/library"), 0o755))
	catalog := databasetest.NewSQLite(t)
	_, err := catalog.Exec(`
CREATE TABLE components (id TEXT PRIMARY KEY, library_id TEXT NOT NULL, latest_version TEXT NOT NULL, draft_version TEXT NOT NULL, manifest_path TEXT NOT NULL DEFAULT '');
CREATE TABLE component_versions (id TEXT PRIMARY KEY, component_id TEXT NOT NULL, library_id TEXT NOT NULL, version TEXT NOT NULL, status TEXT NOT NULL, source_path TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT '', released_at TEXT NOT NULL DEFAULT '', presence TEXT NOT NULL DEFAULT 'materialized');
CREATE TABLE component_version_files (version_id TEXT NOT NULL, path TEXT NOT NULL, content TEXT NOT NULL, content_sha256 TEXT NOT NULL);
CREATE TABLE adoption_records (id TEXT, component_id TEXT, library_id TEXT, adopted_version TEXT, mode TEXT NOT NULL DEFAULT 'copied', scenario TEXT NOT NULL DEFAULT '', adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE adoption_files (adoption_id TEXT, source_library_id TEXT, source_version TEXT, adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE component_asset_dependencies (component_id TEXT, library_id TEXT, version TEXT);
INSERT INTO components VALUES ('panel', 'react-component-library:Panel', '1.1.0', '', 'components/Panel/component.json');
INSERT INTO component_versions(id, component_id, library_id, version, status, source_path, presence) VALUES
  ('panel-1.0.0', 'panel', 'react-component-library:Panel', '1.0.0', 'released', 'components/Panel/versions/1.0.0/Panel.tsx', 'evicted'),
  ('panel-1.1.0', 'panel', 'react-component-library:Panel', '1.1.0', 'released', 'components/Panel/versions/1.1.0/Panel.tsx', 'materialized');
`)
	require.NoError(t, err)
	report, err := BuildCorpusReport(root, catalog)
	require.NoError(t, err)
	byID := make(map[string]CorpusInvariant, len(report.Invariants))
	for _, invariant := range report.Invariants {
		byID[invariant.ID] = invariant
	}
	require.Equal(t, 3.0, byID["I1"].Value, "all three retention reads succeed on the fixture")
	require.Empty(t, byID["I1"].Status)
	require.Equal(t, 1.0, byID["I2"].Value, "the evicted version without mirror rows is the one unreadable mirror")
	require.Empty(t, byID["I2"].Status)
}
