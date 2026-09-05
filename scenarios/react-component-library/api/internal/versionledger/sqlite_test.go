package versionledger

import (
	"context"
	"encoding/json"
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
CREATE TABLE adoption_records (id TEXT, component_id TEXT, library_id TEXT, adopted_version TEXT, mode TEXT NOT NULL DEFAULT 'copied', scenario TEXT NOT NULL DEFAULT '', adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE adoption_files (adoption_id TEXT, source_library_id TEXT, source_version TEXT, adopted_path TEXT NOT NULL DEFAULT '');
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
CREATE TABLE component_versions (id TEXT PRIMARY KEY, component_id TEXT NOT NULL, library_id TEXT NOT NULL, version TEXT NOT NULL, status TEXT NOT NULL, source_path TEXT NOT NULL, created_at TEXT NOT NULL, released_at TEXT NOT NULL, presence TEXT NOT NULL DEFAULT 'materialized');
CREATE TABLE adoption_records (id TEXT, component_id TEXT, library_id TEXT, adopted_version TEXT, mode TEXT NOT NULL DEFAULT 'copied', scenario TEXT NOT NULL DEFAULT '', adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE adoption_files (adoption_id TEXT, source_library_id TEXT, source_version TEXT, adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE component_asset_dependencies (library_id TEXT, version TEXT);
CREATE TABLE version_ledger (library_id TEXT NOT NULL, version TEXT NOT NULL, retired_at TEXT NOT NULL DEFAULT '', lifecycle_state TEXT NOT NULL DEFAULT '', PRIMARY KEY (library_id, version));
INSERT INTO components VALUES ('component-1', 'library-1', '2.0.0', '', 'components/button/component.json');
INSERT INTO component_versions VALUES ('version-old', 'component-1', 'library-1', '1.0.0', 'deprecated', 'components/button/versions/1.0.0/button.tsx', '2025-01-01T00:00:00Z', '2025-01-02T00:00:00Z', 'materialized');
INSERT INTO component_versions VALUES ('version-latest', 'component-1', 'library-1', '2.0.0', 'released', 'components/button/versions/2.0.0/button.tsx', '2026-01-01T00:00:00Z', '2026-01-02T00:00:00Z', 'materialized');
INSERT INTO version_ledger(library_id, version, lifecycle_state) VALUES ('library-1', '1.0.0', 'deprecated'), ('library-1', '2.0.0', 'released');
`)
	require.NoError(t, err)
	manifest := filepath.Join(root, "components/button/component.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(manifest), 0o755))
	require.NoError(t, os.WriteFile(manifest, []byte(`{"deprecatedVersions":[]}`), 0o644))
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
	manifestData, err := os.ReadFile(manifest)
	require.NoError(t, err)
	var manifestDoc struct {
		DeprecatedVersions []string `json:"deprecatedVersions"`
	}
	require.NoError(t, json.Unmarshal(manifestData, &manifestDoc))
	require.Equal(t, []string{"1.0.0"}, manifestDoc.DeprecatedVersions)
	items, _, err = repo.PlanCleanup(context.Background(), CleanupScope{ComponentID: "library-1"})
	require.NoError(t, err)
	for _, item := range items {
		require.NotEqual(t, "1.0.0", item.Candidate.Version, "retired versions must not reappear")
	}
}

func TestPlanCleanupIgnoresEjectedAdoptions(t *testing.T) {
	database := databasetest.NewSQLite(t)
	_, err := database.ExecContext(context.Background(), `
CREATE TABLE components (id TEXT PRIMARY KEY, library_id TEXT NOT NULL, latest_version TEXT NOT NULL, draft_version TEXT NOT NULL);
CREATE TABLE component_versions (component_id TEXT NOT NULL, version TEXT NOT NULL, status TEXT NOT NULL, created_at TEXT NOT NULL, released_at TEXT NOT NULL);
CREATE TABLE adoption_records (id TEXT, component_id TEXT, library_id TEXT, adopted_version TEXT, mode TEXT NOT NULL DEFAULT 'copied');
CREATE TABLE adoption_files (adoption_id TEXT, source_library_id TEXT, source_version TEXT);
CREATE TABLE component_asset_dependencies (library_id TEXT, version TEXT);
INSERT INTO components VALUES ('component-1', 'library-1', '2.0.0', '');
INSERT INTO component_versions VALUES ('component-1', '1.0.0', 'released', '2025-01-01T00:00:00Z', '2025-01-02T00:00:00Z');
INSERT INTO component_versions VALUES ('component-1', '2.0.0', 'released', '2026-01-01T00:00:00Z', '2026-01-02T00:00:00Z');
INSERT INTO adoption_records VALUES ('adoption-1', 'consumer-1', 'library-1', '1.0.0', 'ejected');
INSERT INTO adoption_files VALUES ('adoption-1', 'library-1', '1.0.0');
`)
	require.NoError(t, err)

	items, _, err := NewRepository(database, t.TempDir()).PlanCleanup(context.Background(), CleanupScope{ComponentID: "library-1"})
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.True(t, items[0].Eligible)
	require.Equal(t, "safe to retire", items[0].Reason)
}

func TestTransitionRetireCanBeRestoredByArchive(t *testing.T) {
	database := databasetest.NewSQLite(t)
	root := t.TempDir()
	oldDir := filepath.Join(root, "components/button/versions/1.0.0")
	oldSource := filepath.Join(oldDir, "button.tsx")
	manifest := filepath.Join(root, "components/button/component.json")
	require.NoError(t, os.MkdirAll(oldDir, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Dir(manifest), 0o755))
	require.NoError(t, os.WriteFile(oldSource, []byte("export const oldButton = true;"), 0o644))
	require.NoError(t, os.WriteFile(manifest, []byte(`{"deprecatedVersions":[]}`), 0o644))
	_, err := database.ExecContext(context.Background(), `
CREATE TABLE components (id TEXT PRIMARY KEY, library_id TEXT NOT NULL, latest_version TEXT NOT NULL, draft_version TEXT NOT NULL, manifest_path TEXT NOT NULL);
CREATE TABLE component_versions (component_id TEXT NOT NULL, library_id TEXT NOT NULL, version TEXT NOT NULL, status TEXT NOT NULL, source_path TEXT NOT NULL, created_at TEXT NOT NULL, released_at TEXT NOT NULL, presence TEXT NOT NULL DEFAULT 'materialized');
CREATE TABLE adoption_records (id TEXT, component_id TEXT, library_id TEXT, adopted_version TEXT, mode TEXT NOT NULL DEFAULT 'copied', scenario TEXT NOT NULL DEFAULT '', adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE adoption_files (adoption_id TEXT, source_library_id TEXT, source_version TEXT, adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE component_asset_dependencies (library_id TEXT, version TEXT);
CREATE TABLE version_ledger (library_id TEXT NOT NULL, version TEXT NOT NULL, retired_at TEXT NOT NULL DEFAULT '', lifecycle_state TEXT NOT NULL DEFAULT '', PRIMARY KEY (library_id, version));
INSERT INTO components VALUES ('component-1', 'library-1', '2.0.0', '', 'components/button/component.json');
INSERT INTO component_versions VALUES ('component-1', 'library-1', '1.0.0', 'released', 'components/button/versions/1.0.0/button.tsx', '2025-01-01T00:00:00Z', '2025-01-02T00:00:00Z', 'materialized');
INSERT INTO component_versions VALUES ('component-1', 'library-1', '2.0.0', 'released', 'components/button/versions/2.0.0/button.tsx', '2026-01-01T00:00:00Z', '2026-01-02T00:00:00Z', 'materialized');
INSERT INTO version_ledger(library_id, version, lifecycle_state) VALUES ('library-1', '1.0.0', 'released'), ('library-1', '2.0.0', 'released');
`)
	require.NoError(t, err)
	repo := NewRepository(database, root)
	_, err = repo.Transition(context.Background(), "component-1", "1.0.0", "retired", true)
	require.NoError(t, err)
	require.NoDirExists(t, oldDir)
	require.DirExists(t, repo.retiredSourcePath("library-1", "1.0.0"))

	_, err = repo.Transition(context.Background(), "component-1", "1.0.0", "archived", true)
	require.NoError(t, err)
	require.FileExists(t, oldSource)
	var status, lifecycle string
	require.NoError(t, database.QueryRowContext(context.Background(), `SELECT status FROM component_versions WHERE version='1.0.0'`).Scan(&status))
	require.NoError(t, database.QueryRowContext(context.Background(), `SELECT lifecycle_state FROM version_ledger WHERE version='1.0.0'`).Scan(&lifecycle))
	require.Equal(t, "archived", status)
	require.Equal(t, "archived", lifecycle)
}

func TestTransitionRetiresReplacementBackedAssetAfterHistoricalVersions(t *testing.T) {
	database := databasetest.NewSQLite(t)
	root := t.TempDir()
	versionDir := filepath.Join(root, "components", "OldDrawer", "versions", "1.0.0")
	manifest := filepath.Join(root, "components", "OldDrawer", "component.json")
	require.NoError(t, os.MkdirAll(versionDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(versionDir, "OldDrawer.tsx"), []byte("export const OldDrawer = true;"), 0o644))
	require.NoError(t, os.WriteFile(manifest, []byte(`{"libraryId":"react-component-library:OldDrawer","latest":"1.0.0","deprecatedVersions":[],"replacedBy":["overlays.full-page-drawer"]}`), 0o644))
	_, err := database.ExecContext(context.Background(), `
CREATE TABLE components (id TEXT PRIMARY KEY, library_id TEXT NOT NULL, latest_version TEXT NOT NULL, draft_version TEXT NOT NULL, manifest_path TEXT NOT NULL);
CREATE TABLE component_versions (component_id TEXT NOT NULL, library_id TEXT NOT NULL, version TEXT NOT NULL, status TEXT NOT NULL, source_path TEXT NOT NULL, created_at TEXT NOT NULL, released_at TEXT NOT NULL);
CREATE TABLE adoption_records (id TEXT, component_id TEXT, library_id TEXT, adopted_version TEXT, mode TEXT NOT NULL DEFAULT 'copied', scenario TEXT NOT NULL DEFAULT '', adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE adoption_files (adoption_id TEXT, source_library_id TEXT, source_version TEXT, adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE component_asset_dependencies (library_id TEXT, version TEXT);
CREATE TABLE version_ledger (library_id TEXT NOT NULL, version TEXT NOT NULL, retired_at TEXT NOT NULL DEFAULT '', lifecycle_state TEXT NOT NULL DEFAULT '', PRIMARY KEY (library_id, version));
INSERT INTO components VALUES ('old-drawer', 'react-component-library:OldDrawer', '1.0.0', '', 'components/OldDrawer/component.json');
INSERT INTO component_versions VALUES ('old-drawer', 'react-component-library:OldDrawer', '1.0.0', 'released', 'components/OldDrawer/versions/1.0.0/OldDrawer.tsx', '2026-01-01T00:00:00Z', '2026-01-02T00:00:00Z');
INSERT INTO version_ledger(library_id, version, lifecycle_state) VALUES ('react-component-library:OldDrawer', '1.0.0', 'released');
`)
	require.NoError(t, err)
	repo := NewRepository(database, root)
	_, err = repo.Transition(context.Background(), "old-drawer", "1.0.0", "retired", true)
	require.NoError(t, err)
	require.NoDirExists(t, filepath.Join(root, "components", "OldDrawer"))
	require.DirExists(t, repo.retiredSourcePath("react-component-library:OldDrawer", "1.0.0"))
	var status string
	require.NoError(t, database.QueryRowContext(context.Background(), `SELECT status FROM component_versions WHERE component_id='old-drawer'`).Scan(&status))
	require.Equal(t, "retired", status)
}

func TestTransitionRejectsLatestVersionWithoutAssetReplacement(t *testing.T) {
	database := databasetest.NewSQLite(t)
	root := t.TempDir()
	versionDir := filepath.Join(root, "components", "Current", "versions", "1.0.0")
	require.NoError(t, os.MkdirAll(versionDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(versionDir, "Current.tsx"), []byte("export const Current = true;"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "components", "Current", "component.json"), []byte(`{"latest":"1.0.0","deprecatedVersions":[]}`), 0o644))
	_, err := database.ExecContext(context.Background(), `
CREATE TABLE components (id TEXT PRIMARY KEY, library_id TEXT NOT NULL, latest_version TEXT NOT NULL, draft_version TEXT NOT NULL, manifest_path TEXT NOT NULL);
CREATE TABLE component_versions (component_id TEXT NOT NULL, library_id TEXT NOT NULL, version TEXT NOT NULL, status TEXT NOT NULL, source_path TEXT NOT NULL, created_at TEXT NOT NULL, released_at TEXT NOT NULL);
CREATE TABLE adoption_records (id TEXT, component_id TEXT, library_id TEXT, adopted_version TEXT, mode TEXT NOT NULL DEFAULT 'copied', scenario TEXT NOT NULL DEFAULT '', adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE adoption_files (adoption_id TEXT, source_library_id TEXT, source_version TEXT, adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE component_asset_dependencies (library_id TEXT, version TEXT);
INSERT INTO components VALUES ('current', 'react-component-library:Current', '1.0.0', '', 'components/Current/component.json');
INSERT INTO component_versions VALUES ('current', 'react-component-library:Current', '1.0.0', 'released', 'components/Current/versions/1.0.0/Current.tsx', '2026-01-01T00:00:00Z', '2026-01-02T00:00:00Z');
`)
	require.NoError(t, err)
	_, err = NewRepository(database, root).Transition(context.Background(), "current", "1.0.0", "retired", true)
	require.ErrorContains(t, err, "without replacedBy metadata")
	require.DirExists(t, filepath.Join(root, "components", "Current"))
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
CREATE TABLE adoption_records (id TEXT, component_id TEXT, library_id TEXT, adopted_version TEXT, mode TEXT NOT NULL DEFAULT 'copied', scenario TEXT NOT NULL DEFAULT '', adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE adoption_files (adoption_id TEXT, source_library_id TEXT, source_version TEXT, adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE component_asset_dependencies (library_id TEXT, version TEXT);
INSERT INTO components VALUES ('morphing-icon', 'react-component-library:MorphingIcon', '2.0.7', '', 'primitives/MorphingIcon/component.json');
INSERT INTO component_versions(component_id, library_id, version, status, source_path, created_at, released_at) VALUES ('morphing-icon', 'react-component-library:MorphingIcon', '2.0.0', 'released', 'primitives/MorphingIcon/versions/2.0.0/geometry.ts', '2025-01-01T00:00:00Z', '2025-01-02T00:00:00Z');
INSERT INTO component_versions(component_id, library_id, version, status, source_path, created_at, released_at) VALUES ('morphing-icon', 'react-component-library:MorphingIcon', '2.0.7', 'released', 'primitives/MorphingIcon/versions/2.0.7/geometry.ts', '2026-01-01T00:00:00Z', '2026-01-02T00:00:00Z');
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

func TestPlanCleanupIgnoresReferencesFromUnreachableHistoricalOwners(t *testing.T) {
	database := databasetest.NewSQLite(t)
	root := t.TempDir()
	targetOld := filepath.Join(root, "components/Target/versions/1.0.0")
	targetLatest := filepath.Join(root, "components/Target/versions/1.0.1")
	ownerOld := filepath.Join(root, "components/Owner/versions/1.0.0")
	ownerLatest := filepath.Join(root, "components/Owner/versions/1.0.1")
	for _, dir := range []string{targetOld, targetLatest, ownerOld, ownerLatest} {
		require.NoError(t, os.MkdirAll(dir, 0o755))
	}
	require.NoError(t, os.WriteFile(filepath.Join(targetOld, "Target.tsx"), []byte("export const oldTarget = true;"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(targetLatest, "Target.tsx"), []byte("export const latestTarget = true;"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(ownerOld, "Owner.tsx"), []byte(`import { oldTarget } from "../../../Target/versions/1.0.0/Target"; export { oldTarget };`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(ownerLatest, "Owner.tsx"), []byte("export const latestOwner = true;"), 0o644))
	_, err := database.ExecContext(context.Background(), `
CREATE TABLE components (id TEXT PRIMARY KEY, library_id TEXT NOT NULL, latest_version TEXT NOT NULL, draft_version TEXT NOT NULL, manifest_path TEXT NOT NULL);
CREATE TABLE component_versions (component_id TEXT NOT NULL, library_id TEXT NOT NULL, version TEXT NOT NULL, status TEXT NOT NULL, source_path TEXT NOT NULL, presence TEXT NOT NULL DEFAULT 'materialized', id TEXT NOT NULL, created_at TEXT NOT NULL, released_at TEXT NOT NULL);
CREATE TABLE adoption_records (id TEXT, component_id TEXT, library_id TEXT, adopted_version TEXT, mode TEXT NOT NULL DEFAULT 'copied', scenario TEXT NOT NULL DEFAULT '', adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE adoption_files (adoption_id TEXT, source_library_id TEXT, source_version TEXT, adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE component_asset_dependencies (library_id TEXT, version TEXT);
INSERT INTO components VALUES
  ('target', 'react-component-library:Target', '1.0.1', '', 'components/Target/component.json'),
  ('owner', 'react-component-library:Owner', '1.0.1', '', 'components/Owner/component.json');
INSERT INTO component_versions VALUES
  ('target', 'react-component-library:Target', '1.0.0', 'released', 'components/Target/versions/1.0.0/Target.tsx', 'materialized', 'target-old', '2025-01-01T00:00:00Z', '2025-01-02T00:00:00Z'),
  ('target', 'react-component-library:Target', '1.0.1', 'released', 'components/Target/versions/1.0.1/Target.tsx', 'materialized', 'target-latest', '2026-01-01T00:00:00Z', '2026-01-02T00:00:00Z'),
  ('owner', 'react-component-library:Owner', '1.0.0', 'released', 'components/Owner/versions/1.0.0/Owner.tsx', 'materialized', 'owner-old', '2025-01-01T00:00:00Z', '2025-01-02T00:00:00Z'),
  ('owner', 'react-component-library:Owner', '1.0.1', 'released', 'components/Owner/versions/1.0.1/Owner.tsx', 'materialized', 'owner-latest', '2026-01-01T00:00:00Z', '2026-01-02T00:00:00Z');
`)
	require.NoError(t, err)

	items, _, err := NewRepository(database, root).PlanCleanup(context.Background(), CleanupScope{LibraryID: "react-component-library:Target"})
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.True(t, items[0].Eligible)
	require.Equal(t, "safe to retire", items[0].Reason)
	require.Len(t, items[0].References, 1)
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
CREATE TABLE adoption_records (id TEXT, component_id TEXT, library_id TEXT, adopted_version TEXT, mode TEXT NOT NULL DEFAULT 'copied', scenario TEXT NOT NULL DEFAULT '', adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE adoption_files (adoption_id TEXT, source_library_id TEXT, source_version TEXT, adopted_path TEXT NOT NULL DEFAULT '');
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

func TestRetireCandidatesProtectLinkedAdopterPinnedVersion(t *testing.T) {
	database := databasetest.NewSQLite(t)
	_, err := database.ExecContext(context.Background(), `
CREATE TABLE components (
  id TEXT PRIMARY KEY,
  library_id TEXT NOT NULL,
  latest_version TEXT NOT NULL,
  draft_version TEXT NOT NULL
);
CREATE TABLE component_versions (component_id TEXT NOT NULL, version TEXT NOT NULL, status TEXT NOT NULL);
CREATE TABLE adoption_records (id TEXT, component_id TEXT, library_id TEXT, adopted_version TEXT, mode TEXT NOT NULL DEFAULT 'copied', scenario TEXT NOT NULL DEFAULT '', adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE adoption_files (adoption_id TEXT, source_library_id TEXT, source_version TEXT, adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE component_asset_dependencies (library_id TEXT, version TEXT);
INSERT INTO components(id, library_id, latest_version, draft_version)
VALUES ('component-1', 'react-component-library:Button', '2.0.0', '');
INSERT INTO component_versions(component_id, version, status) VALUES
  ('component-1', '1.2.0', 'released'),
  ('component-1', '2.0.0', 'released');
INSERT INTO adoption_records(id, component_id, library_id, adopted_version, mode)
VALUES ('adoption-1', 'component-1', 'react-component-library:Button', '1.2.0', 'linked');
INSERT INTO adoption_files(adoption_id, source_library_id, source_version)
VALUES ('adoption-1', 'react-component-library:Button', '1.2.0');
`)
	require.NoError(t, err)

	candidates, err := NewRepository(database, t.TempDir()).RetireCandidates(context.Background(), "react-component-library:Button")
	require.NoError(t, err)
	require.Empty(t, candidates, "a linked adopter pin must keep the version out of retirement candidates")
}

func TestRetireCandidatesIgnoreDeclaredEjectedForkProvenance(t *testing.T) {
	database := databasetest.NewSQLite(t)
	_, err := database.ExecContext(context.Background(), `
CREATE TABLE components (
  id TEXT PRIMARY KEY,
  library_id TEXT NOT NULL,
  latest_version TEXT NOT NULL,
  draft_version TEXT NOT NULL
);
CREATE TABLE component_versions (component_id TEXT NOT NULL, version TEXT NOT NULL, status TEXT NOT NULL);
CREATE TABLE adoption_records (id TEXT, component_id TEXT, library_id TEXT, adopted_version TEXT, mode TEXT NOT NULL DEFAULT 'copied', scenario TEXT NOT NULL DEFAULT '', adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE adoption_files (adoption_id TEXT, source_library_id TEXT, source_version TEXT, adopted_path TEXT NOT NULL DEFAULT '');
CREATE TABLE component_asset_dependencies (library_id TEXT, version TEXT);
INSERT INTO components(id, library_id, latest_version, draft_version)
VALUES ('component-1', 'react-component-library:Input', '1.1.2', '');
INSERT INTO component_versions(component_id, version, status) VALUES
  ('component-1', '1.1.0', 'released'),
  ('component-1', '1.1.2', 'released');
INSERT INTO adoption_records(id, component_id, library_id, adopted_version, mode, scenario, adopted_path)
VALUES ('fork-1', 'component-1', 'react-component-library:Input', '1.1.0', 'ejected', 'consumer', 'ui/src/Input.tsx');
INSERT INTO adoption_files(adoption_id, source_library_id, source_version, adopted_path)
VALUES ('fork-1', 'react-component-library:Input', '1.1.0', 'ui/src/Input.tsx');
`)
	require.NoError(t, err)

	candidates, err := NewRepository(database, t.TempDir()).RetireCandidates(context.Background(), "react-component-library:Input")
	require.NoError(t, err)
	require.Len(t, candidates, 1, "an explicitly ejected fork owns its bytes and must not pin the upstream materialization")
	require.Equal(t, "1.1.0", candidates[0].Version)
}
