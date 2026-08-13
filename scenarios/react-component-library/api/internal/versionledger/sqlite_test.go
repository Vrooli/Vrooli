package versionledger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/databasetest"
)

func TestRetireCandidatesExcludeAllDraftVersions(t *testing.T) {
	database := databasetest.NewSQLite(t)
	_, err := database.ExecContext(context.Background(), `
CREATE TABLE components (
  id TEXT PRIMARY KEY,
  library_id TEXT NOT NULL,
  latest_version TEXT NOT NULL,
  draft_version TEXT NOT NULL
);
CREATE TABLE component_versions (
  component_id TEXT NOT NULL,
  version TEXT NOT NULL,
  status TEXT NOT NULL
);
CREATE TABLE adoption_records (component_id TEXT, adopted_version TEXT);
CREATE TABLE adoption_files (source_library_id TEXT, source_version TEXT);
CREATE TABLE component_asset_dependencies (library_id TEXT, version TEXT);
INSERT INTO components(id, library_id, latest_version, draft_version)
VALUES ('component-1', 'library-1', '3.0.0', '2.0.0');
INSERT INTO component_versions(component_id, version, status) VALUES
  ('component-1', '1.0.0', 'released'),
  ('component-1', '1.1.0-draft.1', 'draft'),
  ('component-1', '1.2.0', 'drafted'),
  ('component-1', '2.0.0', 'draft'),
  ('component-1', '3.0.0', 'released');
`)
	require.NoError(t, err)

	candidates, err := NewRepository(database, t.TempDir()).RetireCandidates(context.Background(), "component-1")
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, "1.0.0", candidates[0].Version)
	require.Equal(t, "released", candidates[0].Status)
}
