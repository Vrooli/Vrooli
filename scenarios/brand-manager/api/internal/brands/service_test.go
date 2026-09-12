package brands_test

import (
	"context"
	"errors"
	"testing"

	"brand-manager/internal/brands"
	mocks "brand-manager/internal/brands/mocks"

	"github.com/stretchr/testify/require"
)

func newService(repo *mocks.FakeRepository, versions *mocks.FakeVersionRepository) brands.Service {
	return brands.NewService(repo, versions, nil)
}

func TestService_Create_RejectsEmptyName(t *testing.T) {
	repo := &mocks.FakeRepository{}
	versions := &mocks.FakeVersionRepository{}
	svc := newService(repo, versions)

	_, err := svc.Create(context.Background(), brands.CreateInput{Name: "  "})
	require.Error(t, err)
	var inv brands.ErrInvalidBrand
	require.True(t, errors.As(err, &inv), "expected ErrInvalidBrand, got %T", err)
	require.Equal(t, "name", inv.Field)
	require.Equal(t, int64(0), repo.CreateCalls.Load(), "validation must reject before the repository")
}

func TestService_Create_PersistsAndSnapshotsVersion1(t *testing.T) {
	repo := &mocks.FakeRepository{}
	versions := &mocks.FakeVersionRepository{}
	svc := newService(repo, versions)

	got, err := svc.Create(context.Background(), brands.CreateInput{
		Name:     "  Acme  ",
		Colors:   brands.Colors{Primary: "#fff"},
		Identity: brands.Identity{DisplayName: "Acme Inc"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, got.ID)
	require.Equal(t, "Acme", got.Name, "name is trimmed before persisting")
	require.Equal(t, 1, got.Version)

	recorded := versions.Recorded()
	require.Len(t, recorded, 1, "create must snapshot version 1")
	require.Equal(t, got.ID, recorded[0].BrandID)
	require.Equal(t, 1, recorded[0].Version)
	require.Contains(t, recorded[0].Snapshot, "Acme", "snapshot is the JSON-encoded brand")
}

func TestService_Create_SnapshotFailureDoesNotFailMutation(t *testing.T) {
	repo := &mocks.FakeRepository{}
	versions := &mocks.FakeVersionRepository{CreateErr: errors.New("snapshot down")}
	svc := newService(repo, versions)

	got, err := svc.Create(context.Background(), brands.CreateInput{Name: "Acme"})
	require.NoError(t, err, "a snapshot failure is best-effort and must not fail the create")
	require.NotEmpty(t, got.ID)
}

func TestService_Update_OptimisticLockConflict(t *testing.T) {
	repo := &mocks.FakeRepository{}
	versions := &mocks.FakeVersionRepository{}
	repo.Seed(brands.Brand{ID: "b1", Name: "Acme", Version: 3})
	svc := newService(repo, versions)

	_, err := svc.Update(context.Background(), brands.UpdateInput{ID: "b1", Description: "x", ExpectedVersion: 2})
	require.Error(t, err)
	var conflict brands.ErrVersionConflict
	require.True(t, errors.As(err, &conflict), "expected ErrVersionConflict, got %T", err)
	require.Equal(t, 2, conflict.Expected)
	require.Equal(t, 3, conflict.Actual)
	require.Equal(t, int64(0), repo.UpdateCalls.Load(), "conflict must reject before the write")
}

func TestService_Update_PartialMergePreservesSiblingFacetFields(t *testing.T) {
	repo := &mocks.FakeRepository{}
	versions := &mocks.FakeVersionRepository{}
	repo.Seed(brands.Brand{
		ID:       "b1",
		Name:     "Acme",
		Version:  1,
		Identity: brands.Identity{DisplayName: "Acme Inc", Tagline: "Keep it"},
		Colors:   brands.Colors{Primary: "#111", Secondary: "#222"},
	})
	svc := newService(repo, versions)

	// Only change the tagline + primary; siblings must survive.
	got, err := svc.Update(context.Background(), brands.UpdateInput{
		ID:       "b1",
		Identity: brands.Identity{Tagline: "New tagline"},
		Colors:   brands.Colors{Primary: "#999"},
	})
	require.NoError(t, err)
	require.Equal(t, "New tagline", got.Identity.Tagline)
	require.Equal(t, "Acme Inc", got.Identity.DisplayName, "untouched facet field must be preserved")
	require.Equal(t, "#999", got.Colors.Primary)
	require.Equal(t, "#222", got.Colors.Secondary, "untouched color must be preserved")
	require.Equal(t, 2, got.Version, "update increments the version")

	require.Len(t, versions.Recorded(), 1, "update snapshots the new version")
}

func TestService_Update_ExpectedVersionMatchSucceeds(t *testing.T) {
	repo := &mocks.FakeRepository{}
	versions := &mocks.FakeVersionRepository{}
	repo.Seed(brands.Brand{ID: "b1", Name: "Acme", Version: 4})
	svc := newService(repo, versions)

	got, err := svc.Update(context.Background(), brands.UpdateInput{ID: "b1", Description: "ok", ExpectedVersion: 4})
	require.NoError(t, err)
	require.Equal(t, 5, got.Version)
}

func TestService_Update_NotFound(t *testing.T) {
	repo := &mocks.FakeRepository{}
	versions := &mocks.FakeVersionRepository{}
	svc := newService(repo, versions)

	_, err := svc.Update(context.Background(), brands.UpdateInput{ID: "ghost", Description: "x"})
	require.Error(t, err)
	var nf brands.ErrBrandNotFound
	require.True(t, errors.As(err, &nf))
}

func TestService_Delete_IdempotentWhenMissing(t *testing.T) {
	repo := &mocks.FakeRepository{}
	versions := &mocks.FakeVersionRepository{}
	svc := newService(repo, versions)

	err := svc.Delete(context.Background(), "ghost")
	require.NoError(t, err, "deleting a missing brand is a success (idempotent)")
	require.Equal(t, int64(1), repo.DeleteCalls.Load())
}

func TestService_Delete_RemovesExisting(t *testing.T) {
	repo := &mocks.FakeRepository{}
	versions := &mocks.FakeVersionRepository{}
	repo.Seed(brands.Brand{ID: "b1", Name: "Acme", Version: 1})
	svc := newService(repo, versions)

	require.NoError(t, svc.Delete(context.Background(), "b1"))
	_, err := svc.Get(context.Background(), "b1")
	var nf brands.ErrBrandNotFound
	require.True(t, errors.As(err, &nf))
}

func TestService_List_ClampsLimit(t *testing.T) {
	repo := &mocks.FakeRepository{}
	versions := &mocks.FakeVersionRepository{}

	// Verify clamp via a spy on the repository List filter.
	spy := &listFilterSpy{Repository: repo}
	svc := brands.NewService(spy, versions, nil)

	_, err := svc.List(context.Background(), brands.ListFilter{Limit: 0})
	require.NoError(t, err)
	require.Equal(t, 100, spy.got.Limit, "limit <= 0 substitutes the default (100)")

	_, err = svc.List(context.Background(), brands.ListFilter{Limit: 9999})
	require.NoError(t, err)
	require.Equal(t, 500, spy.got.Limit, "limit is clamped to the max (500)")
}

type listFilterSpy struct {
	brands.Repository
	got brands.ListFilter
}

func (s *listFilterSpy) List(ctx context.Context, filter brands.ListFilter) ([]brands.Brand, error) {
	s.got = filter
	return nil, nil
}
