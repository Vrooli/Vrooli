package journal

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	localdb "vrooli-memory/internal/database"
	"vrooli-memory/internal/facets"
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
	var queuedEntryID, reason string
	require.NoError(t, repo.db.QueryRowContext(context.Background(), `SELECT entry_id, reason FROM journal_retry_queue`).Scan(&queuedEntryID, &reason))
	require.Equal(t, entry.ID, queuedEntryID)
	require.Equal(t, "classify", reason)
}

func TestRepositoryExposesNoMutationMethods(t *testing.T) {
	typ := reflect.TypeOf((*Repository)(nil)).Elem()
	for i := 0; i < typ.NumMethod(); i++ {
		name := typ.Method(i).Name
		require.NotContains(t, name, "Update")
		require.NotContains(t, name, "Delete")
	}
}

func TestProcessClassificationRetriesAppendsFacetAssignmentAndAcknowledgesQueue(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(Schema), apidb.SchemaProviderFunc(facets.Schema)))
	repo := NewSQLiteRepository(d)
	fr := facets.NewSQLiteRepository(d)
	require.NoError(t, fr.Seed(context.Background()))
	failing := &mocks.FakeInference{ClassifyErr: errors.New("unavailable"), EmbedOut: []float64{1}}
	entry, err := NewService(repo, failing, facets.NewService(fr)).Append(context.Background(), Entry{Body: "keep this rule", Kind: "memory"})
	require.NoError(t, err)
	require.Equal(t, UnclassifiedFacet, entry.FacetID)

	result, err := NewService(repo, &mocks.FakeInference{ClassifyOut: "standing-rule"}, facets.NewService(fr)).ProcessClassificationRetries(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, 1, result.Processed)
	assignments, err := fr.Assignments(context.Background(), entry.ID)
	require.NoError(t, err)
	require.Len(t, assignments, 1)
	require.Equal(t, "standing-rule", assignments[0].FacetID)
	var queued int
	require.NoError(t, d.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM journal_retry_queue`).Scan(&queued))
	require.Zero(t, queued)
	stored, err := repo.Get(context.Background(), entry.ID)
	require.NoError(t, err)
	require.Equal(t, UnclassifiedFacet, stored.FacetID, "entry rows remain immutable; facet history owns retry correction")
}

func TestProcessEmbeddingRetriesRestoresAllFacetVectorsAndAcknowledgesQueue(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(d)
	failing := &mocks.FakeInference{ClassifyOut: "episode", EmbedErr: errors.New("unavailable")}
	entry, err := NewService(repo, failing).Append(context.Background(), Entry{Body: "recover vectors", Kind: "memory"})
	require.NoError(t, err)
	require.Len(t, entry.FacetTexts, 3)
	result, err := NewService(repo, &mocks.FakeInference{EmbedOut: []float64{0.1, 0.2}}).ProcessEmbeddingRetries(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, 1, result.Processed)
	var vectors, queued int
	require.NoError(t, d.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM embeddings`).Scan(&vectors))
	require.Equal(t, 3, vectors)
	require.NoError(t, d.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM journal_retry_queue WHERE reason='embed'`).Scan(&queued))
	require.Zero(t, queued)
}
