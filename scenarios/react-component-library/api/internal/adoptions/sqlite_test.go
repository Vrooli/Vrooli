package adoptions_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	"react-component-library/internal/adoptions"
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

func TestSQLiteRepository_ForkMetadataRoundTrips(t *testing.T) {
	repo, _ := newAdoptionsDB(t)
	a, err := repo.Create(context.Background(), adoptions.CreateInput{
		ComponentID: "cmp-fork", Scenario: "target", AdoptedPath: "ui/src/components/Panel.tsx",
		ForkReason:      "target owns a domain-specific interaction model",
		ExtensionPoints: []string{"interaction", "aria-label"},
	})
	require.NoError(t, err)
	require.Equal(t, adoptions.ForkStatusDeclared, a.ForkStatus)
	require.Equal(t, "target owns a domain-specific interaction model", a.ForkReason)
	require.Equal(t, []string{"interaction", "aria-label"}, a.ExtensionPoints)

	got, err := repo.Get(context.Background(), a.ID)
	require.NoError(t, err)
	require.Equal(t, a.ForkStatus, got.ForkStatus)
	require.Equal(t, a.ForkReason, got.ForkReason)
	require.Equal(t, a.ExtensionPoints, got.ExtensionPoints)
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

func TestSQLiteRepository_DeleteRemovesOwnedFileRows(t *testing.T) {
	repo, _ := newAdoptionsDB(t)
	ctx := context.Background()
	input := adoptions.CreateInput{
		ID:             "replaceable-adoption",
		ComponentID:    "cmp-delete",
		LibraryID:      "react-component-library:Card",
		Scenario:       "consumer",
		AdoptedPath:    "ui/src/Card.tsx",
		AdoptedVersion: "1.0.0",
		Files: []adoptions.AdoptionFile{{
			LibraryPath: "Card.tsx", AdoptedPath: "ui/src/Card.tsx",
			SourceLibraryID: "react-component-library:Card", SourceVersion: "1.0.0",
		}},
	}
	_, err := repo.Create(ctx, input)
	require.NoError(t, err)
	require.NoError(t, repo.Delete(ctx, input.ID))

	// Reusing the same adoption id and path exercises the child-row primary
	// key. It fails if Delete leaves the old adoption_files row behind.
	_, err = repo.Create(ctx, input)
	require.NoError(t, err)
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
