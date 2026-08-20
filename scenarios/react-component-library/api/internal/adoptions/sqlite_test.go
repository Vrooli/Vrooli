package adoptions_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	"react-component-library/internal/adoptions"
	"react-component-library/internal/components"
	localdb "react-component-library/internal/database"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"
)

func newAdoptionsDB(t *testing.T) (adoptions.Repository, *scheduletest.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(adoptions.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 5, 12, 10, 0, 0, 0, time.UTC))
	return adoptions.NewSQLiteRepository(d, clk), clk
}

func TestSQLiteRepository_CreateAndGet(t *testing.T) {
	repo, _ := newAdoptionsDB(t)
	ctx := context.Background()

	a, err := repo.Create(ctx, adoptions.CreateInput{
		ID:             "adopt-fixed",
		ComponentID:    "cmp-1",
		LibraryID:      "react-component-library:Button",
		Scenario:       "swarm-manager",
		AdoptedPath:    "ui/src/components/Button.tsx",
		AdoptedVersion: "1.0.0",
		Files: []adoptions.AdoptionFile{
			{LibraryPath: "Button.tsx", AdoptedPath: "ui/src/components/Button.tsx", SourceSHA256: "entry", AdoptedSnapshotSHA256: "entry-snapshot"},
			{LibraryPath: "button-state.ts", AdoptedPath: "ui/src/components/button-state.ts", SourceSHA256: "helper", AdoptedSnapshotSHA256: "helper-snapshot"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "adopt-fixed", a.ID)
	require.Equal(t, "cmp-1", a.ComponentID)
	require.Equal(t, adoptions.LibraryVersionStatusCurrent, a.LibraryVersionStatus)
	require.Equal(t, adoptions.LocalStatusClean, a.LocalStatus)
	require.False(t, a.CreatedAt.IsZero())
	require.True(t, a.RefreshedAt.IsZero())
	require.Len(t, a.Files, 2)

	got, err := repo.Get(ctx, a.ID)
	require.NoError(t, err)
	require.Equal(t, a.ID, got.ID)
	require.Equal(t, "swarm-manager", got.Scenario)
	require.Equal(t, a.Files, got.Files)
}

func TestSQLiteRepository_UpdateAppliedSnapshot(t *testing.T) {
	repo, clk := newAdoptionsDB(t)
	ctx := context.Background()
	a, err := repo.Create(ctx, adoptions.CreateInput{
		ComponentID: "c", Scenario: "s", AdoptedPath: "p", AdoptedVersion: "1.0.0",
		SourceSHA256: "old-source", AdoptedSnapshotSHA256: "old-snapshot",
	})
	require.NoError(t, err)

	clk.Advance(2 * time.Second)
	got, err := repo.UpdateAppliedSnapshot(ctx, adoptions.AppliedSnapshotUpdate{
		ID:                    a.ID,
		AdoptedVersion:        "1.1.0",
		SourceSHA256:          "new-source",
		AdoptedSnapshotSHA256: "new-snapshot",
		AppliedAt:             clk.Now(),
	})
	require.NoError(t, err)
	require.Equal(t, "1.1.0", got.AdoptedVersion)
	require.Equal(t, "new-source", got.SourceSHA256)
	require.Equal(t, "new-snapshot", got.AdoptedSnapshotSHA256)
	require.Equal(t, adoptions.LibraryVersionStatusCurrent, got.LibraryVersionStatus)
	require.Equal(t, adoptions.LocalStatusClean, got.LocalStatus)
	require.Equal(t, got.AppliedAt, got.RefreshedAt)
}

func TestSQLiteRepository_CreateRejectsMissingFields(t *testing.T) {
	repo, _ := newAdoptionsDB(t)
	ctx := context.Background()

	for _, tc := range []struct {
		name string
		in   adoptions.CreateInput
		want string
	}{
		{"no component_id", adoptions.CreateInput{Scenario: "x", AdoptedPath: "p"}, "component_id"},
		{"no scenario", adoptions.CreateInput{ComponentID: "c", AdoptedPath: "p"}, "scenario"},
		{"no adopted_path", adoptions.CreateInput{ComponentID: "c", Scenario: "x"}, "adopted_path"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := repo.Create(ctx, tc.in)
			var inv adoptions.ErrInvalidAdoption
			require.True(t, errors.As(err, &inv))
			require.Equal(t, tc.want, inv.Field)
		})
	}
}

func TestSQLiteRepository_GetNotFound(t *testing.T) {
	repo, _ := newAdoptionsDB(t)
	_, err := repo.Get(context.Background(), "missing")
	var nf adoptions.ErrAdoptionNotFound
	require.True(t, errors.As(err, &nf))
}

func TestSQLiteRepository_ListFiltersAndOrders(t *testing.T) {
	repo, clk := newAdoptionsDB(t)
	ctx := context.Background()

	// Seed 3 rows across two scenarios + two components, advancing the
	// clock so created_at DESC ordering is observable.
	mk := func(cmp, scen, path string) adoptions.Adoption {
		a, err := repo.Create(ctx, adoptions.CreateInput{
			ComponentID: cmp, Scenario: scen, AdoptedPath: path, AdoptedVersion: "1.0.0",
			LibraryID: "react-component-library:" + cmp,
		})
		require.NoError(t, err)
		clk.Advance(time.Second)
		return a
	}
	a1 := mk("cmp-a", "swarm-manager", "a.tsx")
	a2 := mk("cmp-b", "swarm-manager", "b.tsx")
	a3 := mk("cmp-a", "flow-verifier", "c.tsx")

	got, err := repo.List(ctx, adoptions.ListQuery{Limit: 100})
	require.NoError(t, err)
	require.Len(t, got, 3)
	require.Equal(t, []string{a3.ID, a2.ID, a1.ID}, []string{got[0].ID, got[1].ID, got[2].ID})

	byScenario, err := repo.List(ctx, adoptions.ListQuery{Scenario: "swarm-manager", Limit: 100})
	require.NoError(t, err)
	require.Len(t, byScenario, 2)

	byComponent, err := repo.List(ctx, adoptions.ListQuery{ComponentID: "cmp-a", Limit: 100})
	require.NoError(t, err)
	require.Len(t, byComponent, 2)
}

func TestSQLiteRepository_ListEffectiveReturnsMediatedParent(t *testing.T) {
	repo, _ := newAdoptionsDB(t)
	created, err := repo.Create(context.Background(), adoptions.CreateInput{
		ID: "drawer-adoption", ComponentID: "drawer", LibraryID: "rcl:DrawerShell", Scenario: "target", AdoptedPath: "ui/src/components/DrawerShell.tsx", AdoptedVersion: "2.0.0",
		Files: []adoptions.AdoptionFile{
			{LibraryPath: "DrawerShell.tsx", AdoptedPath: "ui/src/components/DrawerShell.tsx", SourceAssetID: "drawer", SourceLibraryID: "rcl:DrawerShell", SourceVersion: "2.0.0"},
			{LibraryPath: "useFocusTrap.ts", AdoptedPath: "ui/src/hooks/useFocusTrap.ts", SourceAssetID: "focus-hook", SourceLibraryID: "rcl:useFocusTrap", SourceVersion: "1.0.0"},
		},
	})
	require.NoError(t, err)

	effective, err := repo.ListEffective(context.Background(), "focus-hook", 10)
	require.NoError(t, err)
	require.Len(t, effective, 1)
	require.True(t, effective[0].Mediated)
	require.Equal(t, "rcl:useFocusTrap", effective[0].SourceLibraryID)
	require.Equal(t, "1.0.0", effective[0].SourceVersion)
	require.Equal(t, created.ID, effective[0].ParentAdoption.ID)
	require.Equal(t, "drawer", effective[0].ParentAdoption.ComponentID)
}

func TestEnsureSchemaMigrations_BackfillsUniqueDependencyFileAttribution(t *testing.T) {
	d := db.NewSQLite(t)
	ctx := context.Background()
	require.NoError(t, apidb.EnsureSchemas(ctx, d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(components.Schema),
		apidb.SchemaProviderFunc(adoptions.Schema),
	))
	_, err := d.ExecContext(ctx, `
INSERT INTO components (id, library_id, slug, source_path, indexed_at, updated_at) VALUES
  ('drawer', 'rcl:DrawerShell', 'drawer-shell', 'DrawerShell.tsx', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z'),
  ('focus', 'rcl:useFocusTrap', 'use-focus-trap', 'useFocusTrap.ts', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
INSERT INTO component_versions (id, component_id, library_id, version, status, source_path, content_sha256, indexed_at) VALUES
  ('drawer-v1', 'drawer', 'rcl:DrawerShell', '1.0.0', 'released', 'DrawerShell.tsx', 'drawer-sha', '2026-01-01T00:00:00Z'),
  ('focus-v1', 'focus', 'rcl:useFocusTrap', '1.0.0', 'released', 'useFocusTrap.ts', 'focus-sha', '2026-01-01T00:00:00Z');
INSERT INTO component_version_files (version_id, path, content_sha256, is_entry) VALUES
  ('drawer-v1', 'DrawerShell.tsx', 'drawer-sha', 1),
  ('focus-v1', 'useFocusTrap.ts', 'focus-sha', 1);
INSERT INTO component_asset_dependencies (component_id, library_id, version) VALUES ('drawer', 'rcl:useFocusTrap', '1.0.0');
INSERT INTO adoption_records (id, component_id, library_id, scenario, adopted_path, adopted_version, created_at) VALUES
  ('drawer-adoption', 'drawer', 'rcl:DrawerShell', 'target', 'ui/src/components/DrawerShell.tsx', '1.0.0', '2026-01-01T00:00:00Z');
INSERT INTO adoption_files (adoption_id, library_path, adopted_path) VALUES
  ('drawer-adoption', 'DrawerShell.tsx', 'ui/src/components/DrawerShell.tsx'),
  ('drawer-adoption', 'useFocusTrap.ts', 'ui/src/hooks/useFocusTrap.ts');`)
	require.NoError(t, err)

	require.NoError(t, adoptions.EnsureSchemaMigrations(ctx, d))
	repo := adoptions.NewSQLiteRepository(d, scheduletest.New(time.Now()))
	effective, err := repo.ListEffective(ctx, "focus", 10)
	require.NoError(t, err)
	require.Len(t, effective, 1)
	require.True(t, effective[0].Mediated)
	require.Equal(t, "rcl:useFocusTrap", effective[0].SourceLibraryID)
}

func TestEnsureSchemaMigrations_RenamesRetiredScenarioInPlace(t *testing.T) {
	d := db.NewSQLite(t)
	ctx := context.Background()
	require.NoError(t, apidb.EnsureSchemas(ctx, d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(components.Schema),
		apidb.SchemaProviderFunc(adoptions.Schema),
	))
	_, err := d.ExecContext(ctx, `
INSERT INTO adoption_records
  (id, component_id, library_id, scenario, adopted_path, adopted_version, created_at)
VALUES ('retired-scenario-adoption', 'cmp', 'react-component-library:Button', 'cleanup-manager', 'ui/src/components/ui/button.tsx', '1.0.0', '2026-01-01T00:00:00Z')`)
	require.NoError(t, err)

	require.NoError(t, adoptions.EnsureSchemaMigrations(ctx, d))
	require.NoError(t, adoptions.EnsureSchemaMigrations(ctx, d), "scenario remap must be boot-idempotent")

	repo := adoptions.NewSQLiteRepository(d, scheduletest.New(time.Now()))
	current, err := repo.List(ctx, adoptions.ListQuery{Scenario: "storage-manager", Limit: 10})
	require.NoError(t, err)
	require.Len(t, current, 1)
	require.Equal(t, "retired-scenario-adoption", current[0].ID)
	retired, err := repo.List(ctx, adoptions.ListQuery{Scenario: "cleanup-manager", Limit: 10})
	require.NoError(t, err)
	require.Empty(t, retired)
}

func TestSQLiteRepository_DeleteAndApplyRefresh(t *testing.T) {
	repo, clk := newAdoptionsDB(t)
	ctx := context.Background()
	a, err := repo.Create(ctx, adoptions.CreateInput{
		ComponentID: "c", Scenario: "s", AdoptedPath: "p", AdoptedVersion: "1.0.0",
	})
	require.NoError(t, err)

	clk.Advance(2 * time.Second)
	touched, err := repo.ApplyRefresh(ctx, []adoptions.RefreshUpdate{{
		ID:                   a.ID,
		LibraryVersionStatus: adoptions.LibraryVersionStatusBehind,
		LocalStatus:          adoptions.LocalStatusClean,
		StatusDetail:         "library at 1.1.0",
		RefreshedAt:          clk.Now(),
	}})
	require.NoError(t, err)
	require.Equal(t, 1, touched)

	got, err := repo.Get(ctx, a.ID)
	require.NoError(t, err)
	require.Equal(t, adoptions.LibraryVersionStatusBehind, got.LibraryVersionStatus)
	require.Equal(t, adoptions.LocalStatusClean, got.LocalStatus)
	require.Equal(t, "library at 1.1.0", got.StatusDetail)
	require.False(t, got.RefreshedAt.IsZero())

	require.NoError(t, repo.Delete(ctx, a.ID))
	require.ErrorAs(t, repo.Delete(ctx, a.ID), new(adoptions.ErrAdoptionNotFound))
}
