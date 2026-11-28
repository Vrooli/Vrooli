package vector_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	"source-ledger/internal/forest"
	"source-ledger/internal/journal"
	"source-ledger/internal/vector"
)

func TestCensusRegistersUnresolvedSentinelsWithoutInventingRecipe(t *testing.T) {
	db := dbtest.NewSQLite(t)
	ctx := context.Background()
	require.NoError(t, apidb.EnsureSchemas(ctx, db,
		apidb.SchemaProviderFunc(journal.Schema),
		apidb.SchemaProviderFunc(forest.Schema),
		apidb.SchemaProviderFunc(vector.Schema),
	))
	now := time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)
	_, err := db.ExecContext(ctx, `INSERT INTO entries(id,scope,body,facet_id,kind,created_at) VALUES
('entry-1','test','body','facet-1','note','2026-11-12'),
('entry-2','test','body','facet-2','note','2026-11-12'),
('entry-3','test','body','facet-3','note','2026-11-12')`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO facet_texts(id,entry_id,kind,text) VALUES
('facet-1','entry-1','topic','one'),('facet-2','entry-2','topic','two'),('facet-3','entry-3','topic','three')`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO embeddings(id,facet_text_id,vector_json,vector_blob,created_at) VALUES
		('good','facet-1','',?,'2026-11-12'),
('missing','facet-2','[]',X'','2026-11-12'),
		('invalid','facet-3','not-json',X'','2026-11-12')`, vector.Encode([]float64{0.25, -0.5}))
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO summaries(id,body,facet_id,vector_json,vector_blob,depth,generation,created_at)
VALUES('summary-1','body','facet-1','[0.1,0.2,0.3]','',0,1,'2026-11-12')`)
	require.NoError(t, err)

	first, err := vector.CensusExistingVectors(ctx, db, now)
	require.NoError(t, err)
	require.Len(t, first, 3)
	require.Equal(t, vector.CensusRecord{
		StoreID: "source-ledger.sqlite", DatasetID: "journal.facets", GenerationID: "unresolved",
		IdentityState: "identity_unknown", Dimensions: 0, MissingCount: 1, InvalidCount: 1, ObservedAt: now,
	}, first[0])
	require.Equal(t, int64(1), first[1].VectorCount)
	require.Equal(t, 2, first[1].Dimensions)
	require.Equal(t, "", first[1].RecipeID)
	require.Equal(t, "identity_unknown", first[1].IdentityState)
	require.Equal(t, "forest.summaries", first[2].DatasetID)
	require.Equal(t, 3, first[2].Dimensions)

	var rows int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM vector_generation_census WHERE generation_id='unresolved' AND recipe_id=''`).Scan(&rows))
	require.Equal(t, 3, rows)

	secondAt := now.Add(time.Minute)
	second, err := vector.CensusExistingVectors(ctx, db, secondAt)
	require.NoError(t, err)
	require.Len(t, second, 3)
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM vector_generation_census`).Scan(&rows))
	require.Equal(t, 3, rows, "repeat census must update stable sentinels rather than invent generations")
	require.NoError(t, db.QueryRowContext(ctx, `SELECT observed_at FROM vector_generation_census LIMIT 1`).Scan(new(string)))
}

func TestCensusRequiresOwnerDBAndObservationTime(t *testing.T) {
	_, err := vector.CensusExistingVectors(context.Background(), nil, time.Now())
	require.Error(t, err)
	_, err = vector.CensusExistingVectors(context.Background(), dbtest.NewSQLite(t), time.Time{})
	require.Error(t, err)
}
