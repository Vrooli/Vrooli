package adoptions_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"react-component-library/internal/adoptions"
)

func TestSQLiteRepository_CreateBatchPersistsRowsAndRollsBackFailures(t *testing.T) {
	repo, _ := newAdoptionsDB(t)
	ctx := context.Background()
	inputs := []adoptions.CreateInput{
		{ID: "batch-a", ComponentID: "a", Scenario: "target", AdoptedPath: "src/A.tsx", ForkReason: "local extension", ExtensionPoints: []string{"render"}, Files: []adoptions.AdoptionFile{{LibraryPath: "A.tsx", AdoptedPath: "src/A.tsx"}}},
		{ID: "batch-b", ComponentID: "b", Scenario: "target", AdoptedPath: "src/B.tsx"},
	}
	created, err := repo.(interface {
		CreateBatch(context.Context, []adoptions.CreateInput) ([]adoptions.Adoption, error)
	}).CreateBatch(ctx, inputs)
	require.NoError(t, err)
	require.Len(t, created, 2)
	require.Equal(t, "batch-a", created[0].ID)
	require.Equal(t, []string{"render"}, created[0].ExtensionPoints)
	require.Equal(t, "src/A.tsx", created[0].Files[0].AdoptedPath)

	_, err = repo.(interface {
		CreateBatch(context.Context, []adoptions.CreateInput) ([]adoptions.Adoption, error)
	}).CreateBatch(ctx, []adoptions.CreateInput{
		{ID: "batch-c", ComponentID: "c", Scenario: "target", AdoptedPath: "src/C.tsx"},
		{ID: "batch-a", ComponentID: "duplicate", Scenario: "target", AdoptedPath: "src/D.tsx"},
	})
	require.Error(t, err)
	_, err = repo.Get(ctx, "batch-c")
	var missing adoptions.ErrAdoptionNotFound
	require.ErrorAs(t, err, &missing, "a failed batch must roll back earlier rows")
}

func TestSQLiteRepository_UpdateAppliedUnitAndRebaselineReplaceProvenance(t *testing.T) {
	repo, _ := newAdoptionsDB(t)
	ctx := context.Background()
	a, err := repo.Create(ctx, adoptions.CreateInput{
		ID: "unit", ComponentID: "cmp", Scenario: "target", AdoptedPath: "src/Root.tsx", AdoptedVersion: "1.0.0",
		Files: []adoptions.AdoptionFile{{LibraryPath: "old.tsx", AdoptedPath: "src/old.tsx"}},
	})
	require.NoError(t, err)

	updated, err := repo.UpdateAppliedUnit(ctx, adoptions.AppliedUnitUpdate{
		AppliedSnapshotUpdate: adoptions.AppliedSnapshotUpdate{ID: a.ID, AdoptedVersion: "2.0.0", SourceSHA256: "new-source", AdoptedSnapshotSHA256: "new-snapshot"},
		Files:                 []adoptions.AdoptionFile{{LibraryPath: "Root.tsx", AdoptedPath: "src/Root.tsx", SourceVersion: "2.0.0"}},
	})
	require.NoError(t, err)
	require.Equal(t, "2.0.0", updated.AdoptedVersion)
	require.Equal(t, []string{"src/Root.tsx"}, []string{updated.Files[0].AdoptedPath})

	_, err = repo.Rebaseline(ctx, adoptions.RebaselineInput{ID: a.ID, AdoptedSnapshotSHA256: "healed", Files: []adoptions.AdoptionFile{{LibraryPath: "Root.tsx", AdoptedPath: "src/Root.tsx", AdoptedSnapshotSHA256: "file-healed"}}})
	require.NoError(t, err)
	got, err := repo.Get(ctx, a.ID)
	require.NoError(t, err)
	require.Equal(t, "healed", got.AdoptedSnapshotSHA256)
	require.Equal(t, "file-healed", got.Files[0].AdoptedSnapshotSHA256)
	require.Equal(t, "2.0.0", got.AdoptedVersion, "rebaseline must preserve applied version")

	_, err = repo.Rebaseline(ctx, adoptions.RebaselineInput{ID: "missing"})
	var missing adoptions.ErrAdoptionNotFound
	require.ErrorAs(t, err, &missing)
}

func TestSQLiteRepository_UpdateModeAndLinkedState(t *testing.T) {
	repo, _ := newAdoptionsDB(t)
	ctx := context.Background()
	a, err := repo.Create(ctx, adoptions.CreateInput{ID: "mode", ComponentID: "cmp", Scenario: "target", AdoptedPath: "src/A.tsx", Files: []adoptions.AdoptionFile{{LibraryPath: "A.tsx", AdoptedPath: "src/A.tsx"}}})
	require.NoError(t, err)

	_, err = repo.UpdateMode(ctx, a.ID, adoptions.AdoptionModeEjected, "  intentional fork  ")
	require.NoError(t, err)
	got, err := repo.Get(ctx, a.ID)
	require.NoError(t, err)
	require.Equal(t, adoptions.AdoptionModeEjected, got.Mode)
	require.Equal(t, "intentional fork", got.ForkReason)
	require.Equal(t, adoptions.ForkStatusDeclared, got.ForkStatus)
	_, err = repo.UpdateMode(ctx, a.ID, adoptions.AdoptionModeEjected, " ")
	var invalid adoptions.ErrInvalidAdoption
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "reason", invalid.Field)
	_, err = repo.UpdateLinked(ctx, a.ID, "./A/2.0.0", "2.0.0", "linked-hash")
	require.NoError(t, err)
	got, err = repo.Get(ctx, a.ID)
	require.NoError(t, err)
	require.Equal(t, adoptions.AdoptionModeLinked, got.Mode)
	require.Empty(t, got.Files, "linked adoption should not retain copied-file provenance")
	require.Empty(t, got.ForkReason)
}

func TestSQLiteRepository_ApplyRefreshUpdatesForkAndBacklogMetadata(t *testing.T) {
	repo, clk := newAdoptionsDB(t)
	ctx := context.Background()
	a, err := repo.Create(ctx, adoptions.CreateInput{ID: "refresh-meta", ComponentID: "cmp", Scenario: "target", AdoptedPath: "src/A.tsx"})
	require.NoError(t, err)
	updatedAt := clk.Now().Add(time.Minute)
	touched, err := repo.ApplyRefresh(ctx, []adoptions.RefreshUpdate{{
		ID: a.ID, LibraryVersionStatus: adoptions.LibraryVersionStatusBehind, LocalStatus: adoptions.LocalStatusModified,
		ForkStatus: adoptions.ForkStatusLocalFork, StatusDetail: "local edits", RefreshedAt: updatedAt, DriftBacklogRef: "drift/A",
	}})
	require.NoError(t, err)
	require.Equal(t, 1, touched)
	got, err := repo.Get(ctx, a.ID)
	require.NoError(t, err)
	require.Equal(t, adoptions.ForkStatusLocalFork, got.ForkStatus)
	require.Equal(t, "drift/A", got.DriftBacklogRef)

	_, err = repo.ApplyRefresh(ctx, []adoptions.RefreshUpdate{{ID: a.ID, LibraryVersionStatus: adoptions.LibraryVersionStatusCurrent, LocalStatus: adoptions.LocalStatusClean, RefreshedAt: updatedAt, DriftBacklogRef: "ignored", ClearDriftBacklogRef: true}})
	require.NoError(t, err)
	got, err = repo.Get(ctx, a.ID)
	require.NoError(t, err)
	require.Empty(t, got.DriftBacklogRef, "explicit clear must win over a simultaneously supplied ref")
	_, err = repo.UpdateAppliedSnapshot(ctx, adoptions.AppliedSnapshotUpdate{ID: "absent"})
	var missing adoptions.ErrAdoptionNotFound
	require.ErrorAs(t, err, &missing)
	_, err = repo.UpdateMode(ctx, "absent", adoptions.AdoptionModeLinked, "")
	require.ErrorAs(t, err, &missing)
	_, err = repo.UpdateLinked(ctx, "absent", "path", "1.0", "hash")
	require.True(t, errors.As(err, &missing))
}

func TestSQLiteRepository_ListEffectiveEmptyAndListNoLimit(t *testing.T) {
	repo, _ := newAdoptionsDB(t)
	effective, err := repo.ListEffective(context.Background(), "nobody", 10)
	require.NoError(t, err)
	require.NotNil(t, effective)
	require.Empty(t, effective)
	rows, err := repo.List(context.Background(), adoptions.ListQuery{ComponentID: "ignored", Limit: 0})
	require.NoError(t, err)
	require.Nil(t, rows)
}

func TestSQLiteRepository_CreateBatchRejectsEmptyAndInvalidEntries(t *testing.T) {
	repo, _ := newAdoptionsDB(t)
	batch := repo.(interface {
		CreateBatch(context.Context, []adoptions.CreateInput) ([]adoptions.Adoption, error)
	})
	_, err := batch.CreateBatch(context.Background(), nil)
	var invalid adoptions.ErrInvalidAdoption
	require.ErrorAs(t, err, &invalid)
	_, err = batch.CreateBatch(context.Background(), []adoptions.CreateInput{{Scenario: "target", AdoptedPath: "src/A.tsx"}})
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "component_id", invalid.Field)
	_, err = batch.CreateBatch(context.Background(), []adoptions.CreateInput{{ComponentID: "cmp", Scenario: "target", AdoptedPath: "src/A.tsx", Mode: "bad"}})
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "mode", invalid.Field)
	_, err = batch.CreateBatch(context.Background(), []adoptions.CreateInput{{ComponentID: "cmp", Scenario: "target", AdoptedPath: "src/A.tsx", ID: "mode-invalid"}, {ComponentID: "cmp", Scenario: "target", AdoptedPath: "src/B.tsx", Mode: "bad"}})
	require.Error(t, err)
	_, err = repo.Get(context.Background(), "mode-invalid")
	var missing adoptions.ErrAdoptionNotFound
	require.ErrorAs(t, err, &missing)
}
