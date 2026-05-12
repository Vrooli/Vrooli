package components_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/components"
	"react-component-library/internal/components/mocks"
)

func TestService_UpsertRejectsBlankLibraryID(t *testing.T) {
	repo := mocks.NewFakeRepository()
	svc := components.NewService(repo)

	_, err := svc.Upsert(context.Background(), components.UpsertInput{LibraryID: "   "})
	var bad components.ErrInvalidHeader
	require.True(t, errors.As(err, &bad))
	require.Equal(t, int64(0), repo.UpsertCalls.Load())
}

func TestService_ListAppliesDefaultLimit(t *testing.T) {
	repo := mocks.NewFakeRepository()
	svc := components.NewService(repo)
	ctx := context.Background()

	_, err := svc.Upsert(ctx, components.UpsertInput{LibraryID: "a"})
	require.NoError(t, err)
	_, err = svc.Upsert(ctx, components.UpsertInput{LibraryID: "b"})
	require.NoError(t, err)

	got, err := svc.List(ctx, components.SearchQuery{Limit: 0})
	require.NoError(t, err)
	require.Len(t, got, 2, "default limit should fetch all seeded rows")
}

func TestService_GetByLibraryIDPropagatesNotFound(t *testing.T) {
	repo := mocks.NewFakeRepository()
	svc := components.NewService(repo)
	_, err := svc.GetByLibraryID(context.Background(), "missing")
	var nf components.ErrComponentNotFound
	require.True(t, errors.As(err, &nf))
}
