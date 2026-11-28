package adoptions_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scheduletest"
	"react-component-library/internal/adoptions"
	"react-component-library/internal/adoptions/mocks"
)

type removableFiles struct {
	*fakeFiles
	removed []string
	err     error
}

func (f *removableFiles) Remove(_ context.Context, scenario, path string) error {
	if f.err != nil {
		return f.err
	}
	f.removed = append(f.removed, scenario+"::"+path)
	return nil
}

type presenceResult struct {
	componentID string
	apply       bool
	err         error
}

func (p *presenceResult) ReconcilePresence(_ context.Context, componentID string, apply bool) error {
	p.componentID, p.apply = componentID, apply
	return p.err
}

func newDeleteService(repo *mocks.FakeRepository, files *removableFiles, presence *presenceResult) adoptions.Service {
	svc := adoptions.NewService(repo, &fakeLibrary{}, files, scheduletest.New(time.Unix(0, 0)))
	if presence != nil {
		adoptions.SetPresenceReconciler(svc, presence)
	}
	return svc
}

func TestServiceDeleteWithOptionsRequiresConfirmationAndPreservesSharedFiles(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewFakeRepository()
	repo.Seed(adoptions.Adoption{ID: "delete-me", ComponentID: "cmp", Scenario: "target", AdoptedPath: "src/Root.tsx", Files: []adoptions.AdoptionFile{
		{AdoptedPath: "src/Root.tsx"}, {AdoptedPath: "src/shared.ts"},
	}})
	repo.Seed(adoptions.Adoption{ID: "other", ComponentID: "other", Scenario: "target", AdoptedPath: "src/Other.tsx", Files: []adoptions.AdoptionFile{
		{AdoptedPath: "src/shared.ts"},
	}})
	files := &removableFiles{fakeFiles: &fakeFiles{bytes: map[string][]byte{}}}
	svc := newDeleteService(repo, files, nil)

	result, err := svc.DeleteWithOptions(ctx, " delete-me ", false)
	var invalid adoptions.ErrInvalidAdoption
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "confirm_remove_files", invalid.Field)
	require.True(t, result.RequiresConfirmation)
	require.Equal(t, []string{"src/Root.tsx"}, result.RemovableFiles)
	require.Empty(t, files.removed)
	require.Zero(t, repo.DeleteCalls.Load(), "confirmation is required before the row or files are removed")

	result, err = svc.DeleteWithOptions(ctx, "delete-me", true)
	require.NoError(t, err)
	require.Equal(t, []string{"target::src/Root.tsx"}, files.removed)
	require.Equal(t, []string{"src/Root.tsx"}, result.RemovedFiles)
	require.Equal(t, int64(1), repo.DeleteCalls.Load())
	_, err = repo.Get(ctx, "other")
	require.NoError(t, err, "shared file ownership must not cause the other adoption to be deleted")
}

func TestServiceDeleteWithOptionsPropagatesFileAndRepositoryFailures(t *testing.T) {
	ctx := context.Background()
	removeFailure := errors.New("target filesystem is read-only")
	repo := mocks.NewFakeRepository()
	repo.Seed(adoptions.Adoption{ID: "remove-fails", ComponentID: "cmp", Scenario: "target", AdoptedPath: "src/Root.tsx"})
	files := &removableFiles{fakeFiles: &fakeFiles{bytes: map[string][]byte{}}, err: removeFailure}
	svc := newDeleteService(repo, files, nil)
	result, err := svc.DeleteWithOptions(ctx, "remove-fails", true)
	require.ErrorIs(t, err, removeFailure)
	require.Equal(t, []string{"src/Root.tsx"}, result.RemovableFiles)
	require.Empty(t, result.RemovedFiles)
	require.Zero(t, repo.DeleteCalls.Load(), "the registry row remains when removing its source file fails")

	files.err = nil
	repo.DeleteErr = errors.New("registry is unavailable")
	result, err = svc.DeleteWithOptions(ctx, "remove-fails", true)
	require.ErrorIs(t, err, repo.DeleteErr)
	require.Equal(t, []string{"src/Root.tsx"}, result.RemovedFiles)
}

func TestServiceDeleteWithOptionsReportsPresenceFailureAfterDelete(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewFakeRepository()
	repo.Seed(adoptions.Adoption{ID: "presence-fails", ComponentID: "cmp", LibraryID: "rcl:Panel", Scenario: "target", AdoptedPath: "src/Panel.tsx"})
	presence := &presenceResult{err: errors.New("ledger unavailable")}
	svc := newDeleteService(repo, &removableFiles{fakeFiles: &fakeFiles{bytes: map[string][]byte{}}}, presence)
	_, err := svc.DeleteWithOptions(ctx, "presence-fails", true)
	require.ErrorContains(t, err, "reconcile presence after deleting adoption rcl:Panel@")
	require.ErrorContains(t, err, "ledger unavailable")
	require.Equal(t, "cmp", presence.componentID)
	require.True(t, presence.apply)
	require.Equal(t, int64(1), repo.DeleteCalls.Load())
}

func TestServiceDeleteWithOptionsPropagatesLookupAndListFailures(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewFakeRepository()
	svc := newDeleteService(repo, &removableFiles{fakeFiles: &fakeFiles{bytes: map[string][]byte{}}}, nil)
	repo.GetErr = errors.New("lookup failed")
	_, err := svc.DeleteWithOptions(ctx, "id", false)
	require.ErrorIs(t, err, repo.GetErr)
	repo.GetErr = nil
	repo.Seed(adoptions.Adoption{ID: "list-fails", ComponentID: "cmp", Scenario: "target", AdoptedPath: "src/A.tsx"})
	repo.ListErr = errors.New("list failed")
	_, err = svc.DeleteWithOptions(ctx, "list-fails", true)
	require.ErrorIs(t, err, repo.ListErr)
	require.Zero(t, repo.DeleteCalls.Load())
}

func TestServiceDeleteDelegatesToSafeDeleteFlow(t *testing.T) {
	repo := mocks.NewFakeRepository()
	repo.Seed(adoptions.Adoption{ID: "safe-delete", ComponentID: "cmp", Scenario: "target", AdoptedPath: "src/A.tsx"})
	svc := adoptions.NewService(repo, &fakeLibrary{}, &fakeFiles{bytes: map[string][]byte{}}, scheduletest.New(time.Unix(0, 0)))
	err := svc.Delete(context.Background(), "safe-delete")
	var invalid adoptions.ErrInvalidAdoption
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "confirm_remove_files", invalid.Field)
	require.Zero(t, repo.DeleteCalls.Load())
}

func TestServiceListEffectiveValidatesAndDefaultsLimit(t *testing.T) {
	repo := mocks.NewFakeRepository()
	repo.Seed(adoptions.Adoption{ID: "effective", ComponentID: "root", Scenario: "target", AdoptedPath: "src/Root.tsx", Files: []adoptions.AdoptionFile{{SourceAssetID: "dep", SourceLibraryID: "rcl:Dep", SourceVersion: "1.0.0"}}})
	svc := adoptions.NewService(repo, &fakeLibrary{}, &fakeFiles{bytes: map[string][]byte{}}, scheduletest.New(time.Unix(0, 0)))
	_, err := svc.ListEffective(context.Background(), " ", 10)
	var invalid adoptions.ErrInvalidAdoption
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "component_id", invalid.Field)
	rows, err := svc.ListEffective(context.Background(), "dep", 0)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.True(t, rows[0].Mediated)
}

func TestServiceForkReportDelegatesDryRunAndApply(t *testing.T) {
	ctx := context.Background()
	repo := mocks.NewFakeRepository()
	repo.Seed(adoptions.Adoption{ID: "reported", ComponentID: "cmp", Scenario: "target", AdoptedPath: "src/A.tsx"})
	files := &fakeFiles{bytes: map[string][]byte{"target::src/A.tsx": []byte("original")}}
	svc := adoptions.NewService(repo, &fakeLibrary{body: map[string]string{}}, files, scheduletest.New(time.Unix(0, 0)))
	rows, summary, err := svc.ForkReport(ctx, " cmp ", false)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, 1, summary.LibraryMissing)
	require.Zero(t, repo.RefreshCalls.Load(), "dry run must not persist classifications")
	_, _, err = svc.ForkReport(ctx, "cmp", true)
	require.NoError(t, err)
	require.Equal(t, int64(1), repo.RefreshCalls.Load())
}
