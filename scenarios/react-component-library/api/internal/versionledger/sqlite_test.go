package versionledger

import (
	"context"
	"os"
	"path/filepath"
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

func TestPlanAndApplyCleanupRequiresExactPlanAndPreservesLedger(t *testing.T) {
	database := databasetest.NewSQLite(t)
	root := t.TempDir()
	oldDir := filepath.Join(root, "components", "button", "versions", "1.0.0")
	require.NoError(t, os.MkdirAll(oldDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(oldDir, "button.tsx"), []byte("old"), 0o644))
	_, err := database.ExecContext(context.Background(), `
CREATE TABLE components (id TEXT PRIMARY KEY, library_id TEXT NOT NULL, latest_version TEXT NOT NULL, draft_version TEXT NOT NULL, manifest_path TEXT NOT NULL);
CREATE TABLE component_versions (id TEXT PRIMARY KEY, component_id TEXT NOT NULL, library_id TEXT NOT NULL, version TEXT NOT NULL, status TEXT NOT NULL, source_path TEXT NOT NULL, created_at TEXT NOT NULL, released_at TEXT NOT NULL);
CREATE TABLE adoption_records (component_id TEXT, library_id TEXT, adopted_version TEXT);
CREATE TABLE adoption_files (source_library_id TEXT, source_version TEXT);
CREATE TABLE component_asset_dependencies (library_id TEXT, version TEXT);
CREATE TABLE version_ledger (library_id TEXT NOT NULL, version TEXT NOT NULL, retired_at TEXT NOT NULL DEFAULT '', lifecycle_state TEXT NOT NULL DEFAULT '', PRIMARY KEY (library_id, version));
INSERT INTO components VALUES ('component-1', 'library-1', '2.0.0', '', 'components/button/component.json');
INSERT INTO component_versions VALUES ('version-old', 'component-1', 'library-1', '1.0.0', 'deprecated', 'components/button/versions/1.0.0/button.tsx', '2025-01-01T00:00:00Z', '2025-01-02T00:00:00Z');
INSERT INTO component_versions VALUES ('version-latest', 'component-1', 'library-1', '2.0.0', 'released', 'components/button/versions/2.0.0/button.tsx', '2026-01-01T00:00:00Z', '2026-01-02T00:00:00Z');
INSERT INTO version_ledger(library_id, version, lifecycle_state) VALUES ('library-1', '1.0.0', 'deprecated'), ('library-1', '2.0.0', 'released');
`)
	require.NoError(t, err)
	manifest := filepath.Join(root, "components/button/component.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(manifest), 0o755))
	require.NoError(t, os.WriteFile(manifest, []byte(`{"deprecatedVersions":["1.0.0"]}`), 0o644))
	repo := NewRepository(database, root)
	items, planHash, err := repo.PlanCleanup(context.Background(), CleanupScope{ComponentID: "library-1"})
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.True(t, items[0].Eligible)
	require.False(t, items[1].Eligible)
	require.Equal(t, "latest version", items[1].Reason)

	_, _, retired, err := repo.CleanupVersions(context.Background(), CleanupScope{ComponentID: "library-1"}, "wrong", true)
	require.Error(t, err)
	require.Zero(t, retired)
	_, _, retired, err = repo.CleanupVersions(context.Background(), CleanupScope{ComponentID: "library-1"}, planHash, false)
	require.NoError(t, err)
	require.Zero(t, retired)
	require.DirExists(t, oldDir)
	_, _, retired, err = repo.CleanupVersions(context.Background(), CleanupScope{ComponentID: "library-1"}, planHash, true)
	require.NoError(t, err)
	require.Equal(t, 1, retired)
	require.NoDirExists(t, oldDir)
	var state string
	require.NoError(t, database.QueryRowContext(context.Background(), `SELECT lifecycle_state FROM version_ledger WHERE library_id='library-1' AND version='1.0.0'`).Scan(&state))
	require.Equal(t, "retired", state)
	items, _, err = repo.PlanCleanup(context.Background(), CleanupScope{ComponentID: "library-1"})
	require.NoError(t, err)
	for _, item := range items {
		require.NotEqual(t, "1.0.0", item.Candidate.Version, "retired versions must not reappear")
	}
}

func TestPlanCleanupProtectsVersionsReferencedBySurvivingSource(t *testing.T) {
	database := databasetest.NewSQLite(t)
	root := t.TempDir()
	oldDir := filepath.Join(root, "primitives/MorphingIcon/versions/2.0.0")
	newDir := filepath.Join(root, "primitives/MorphingIcon/versions/2.0.7")
	require.NoError(t, os.MkdirAll(oldDir, 0o755))
	require.NoError(t, os.MkdirAll(newDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(oldDir, "geometry.ts"), []byte("export const oldGeometry = true;"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(newDir, "geometry.ts"), []byte(`import { oldGeometry } from "../2.0.0/geometry"; export { oldGeometry };`), 0o644))
	_, err := database.ExecContext(context.Background(), `
CREATE TABLE components (id TEXT PRIMARY KEY, library_id TEXT NOT NULL, latest_version TEXT NOT NULL, draft_version TEXT NOT NULL, manifest_path TEXT NOT NULL);
CREATE TABLE component_versions (component_id TEXT NOT NULL, library_id TEXT NOT NULL, version TEXT NOT NULL, status TEXT NOT NULL, source_path TEXT NOT NULL, created_at TEXT NOT NULL, released_at TEXT NOT NULL);
CREATE TABLE adoption_records (component_id TEXT, library_id TEXT, adopted_version TEXT);
CREATE TABLE adoption_files (source_library_id TEXT, source_version TEXT);
CREATE TABLE component_asset_dependencies (library_id TEXT, version TEXT);
INSERT INTO components VALUES ('morphing-icon', 'react-component-library:MorphingIcon', '2.0.7', '', 'primitives/MorphingIcon/component.json');
INSERT INTO component_versions VALUES ('morphing-icon', 'react-component-library:MorphingIcon', '2.0.0', 'released', 'primitives/MorphingIcon/versions/2.0.0/MorphingIcon.tsx', '2025-01-01T00:00:00Z', '2025-01-02T00:00:00Z');
INSERT INTO component_versions VALUES ('morphing-icon', 'react-component-library:MorphingIcon', '2.0.7', 'released', 'primitives/MorphingIcon/versions/2.0.7/MorphingIcon.tsx', '2026-01-01T00:00:00Z', '2026-01-02T00:00:00Z');
`)
	require.NoError(t, err)

	items, _, err := NewRepository(database, root).PlanCleanup(context.Background(), CleanupScope{LibraryID: "react-component-library:MorphingIcon"})
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.False(t, items[0].Eligible)
	require.Equal(t, "referenced by source import", items[0].Reason)
	require.Len(t, items[0].References, 1)
	require.Equal(t, "2.0.7", items[0].References[0].OwnerVersion)
	require.Equal(t, "../2.0.0/geometry", items[0].References[0].ImportSpecifier)
	require.Contains(t, items[0].References[0].Evidence, "surviving version")
}

func TestCleanupDraftRequiresAgeAndClearsManifest(t *testing.T) {
	database := databasetest.NewSQLite(t)
	root := t.TempDir()
	draftDir := filepath.Join(root, "components", "button", "versions", "2.1.0-draft.1")
	require.NoError(t, os.MkdirAll(draftDir, 0o755))
	manifest := filepath.Join(root, "components/button/component.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(manifest), 0o755))
	require.NoError(t, os.WriteFile(manifest, []byte(`{"draft":"2.1.0-draft.1"}`), 0o644))
	_, err := database.ExecContext(context.Background(), `
CREATE TABLE components (id TEXT PRIMARY KEY, library_id TEXT NOT NULL, draft_version TEXT NOT NULL, manifest_path TEXT NOT NULL);
CREATE TABLE component_versions (component_id TEXT NOT NULL, library_id TEXT NOT NULL, version TEXT NOT NULL, status TEXT NOT NULL, source_path TEXT NOT NULL, created_at TEXT NOT NULL);
INSERT INTO components VALUES ('component-1', 'library-1', '2.1.0-draft.1', 'components/button/component.json');
INSERT INTO component_versions VALUES ('component-1', 'library-1', '2.1.0-draft.1', 'draft', 'components/button/versions/2.1.0-draft.1/button.tsx', '2025-01-01T00:00:00Z');
`)
	require.NoError(t, err)
	repo := NewRepository(database, root)
	item, err := repo.CleanupDraft(context.Background(), "library-1", 99999, true)
	require.NoError(t, err)
	require.False(t, item.Eligible)
	require.DirExists(t, draftDir)
	item, err = repo.CleanupDraft(context.Background(), "library-1", 1, true)
	require.NoError(t, err)
	require.True(t, item.Eligible)
	require.NoDirExists(t, draftDir)
	data, err := os.ReadFile(manifest)
	require.NoError(t, err)
	require.Contains(t, string(data), `"draft": ""`)
}

func TestRetireCandidatesAcceptLibraryID(t *testing.T) {
	database := databasetest.NewSQLite(t)
	_, err := database.ExecContext(context.Background(), `
CREATE TABLE components (
  id TEXT PRIMARY KEY,
  library_id TEXT NOT NULL,
  latest_version TEXT NOT NULL,
  draft_version TEXT NOT NULL
);
CREATE TABLE component_versions (component_id TEXT NOT NULL, version TEXT NOT NULL, status TEXT NOT NULL);
CREATE TABLE adoption_records (component_id TEXT, adopted_version TEXT);
CREATE TABLE adoption_files (source_library_id TEXT, source_version TEXT);
CREATE TABLE component_asset_dependencies (library_id TEXT, version TEXT);
INSERT INTO components(id, library_id, latest_version, draft_version) VALUES ('component-1', 'library-1', '2.0.0', '');
INSERT INTO component_versions(component_id, version, status) VALUES ('component-1', '1.0.0', 'deprecated'), ('component-1', '2.0.0', 'released');
`)
	require.NoError(t, err)

	candidates, err := NewRepository(database, t.TempDir()).RetireCandidates(context.Background(), "library-1")
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, "1.0.0", candidates[0].Version)
}
