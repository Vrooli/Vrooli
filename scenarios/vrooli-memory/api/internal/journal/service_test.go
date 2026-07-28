package journal

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	localdb "vrooli-memory/internal/database"
	"vrooli-memory/internal/testutil/db"
	"vrooli-memory/internal/testutil/mocks"
)

func journalDB(t *testing.T) *SQLiteRepository {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(Schema)))
	return NewSQLiteRepository(d)
}

func TestAppendPreservesWriteOrderAndFacetEmbeddings(t *testing.T) {
	repo := journalDB(t)
	client := &mocks.FakeInference{ClassifyOut: "project", EmbedOut: []float64{0.1, 0.2}}
	svc := NewService(repo, client)
	for _, body := range []string{"first", "second", "third"} {
		_, err := svc.Append(context.Background(), Entry{Body: body, Kind: "memory"})
		require.NoError(t, err)
	}
	entries, err := repo.List(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, entries, 3)
	for i, body := range []string{"first", "second", "third"} {
		require.Equal(t, body, entries[i].Body)
		require.NotEmpty(t, entries[i].ID)
		require.Len(t, entries[i].FacetTexts, 3)
	}
}

func TestClassifierFailureStillAppendsUnclassifiedEntry(t *testing.T) {
	repo := journalDB(t)
	client := &mocks.FakeInference{ClassifyErr: errors.New("gateway unavailable"), EmbedOut: []float64{1}}
	entry, err := NewService(repo, client).Append(context.Background(), Entry{Body: "never lose this", Kind: "memory"})
	require.NoError(t, err)
	require.Equal(t, UnclassifiedFacet, entry.FacetID)
	stored, err := repo.Get(context.Background(), entry.ID)
	require.NoError(t, err)
	require.Equal(t, UnclassifiedFacet, stored.FacetID)
}

func TestRepositoryExposesNoMutationMethods(t *testing.T) {
	typ := reflect.TypeOf((*Repository)(nil)).Elem()
	for i := 0; i < typ.NumMethod(); i++ {
		name := typ.Method(i).Name
		require.NotContains(t, name, "Update")
		require.NotContains(t, name, "Delete")
	}
}
