package facets

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestEnsureMigrationsBackfillsAssignmentProvenanceWithoutChangingRows(t *testing.T) {
	db, err := sql.Open("sqlite", "file:facet-migrations?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(Schema())
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO facet_assignments(id,entry_id,facet_id,assigned_at,actor_id) VALUES('legacy','entry','episode','2026-08-05T00:00:00Z','')`)
	require.NoError(t, err)

	ctx := context.Background()
	var before int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM facet_assignments`).Scan(&before))
	require.NoError(t, EnsureMigrations(ctx, db))
	require.NoError(t, EnsureMigrations(ctx, db))

	var after int
	var actor string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM facet_assignments`).Scan(&after))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT actor_id FROM facet_assignments WHERE id='legacy'`).Scan(&actor))
	require.Equal(t, before, after)
	require.Equal(t, "migration:legacy-facet-assignment", actor)
}

func TestAssignProvidesFallbackProvenance(t *testing.T) {
	db, err := sql.Open("sqlite", "file:facet-assignment-provenance?mode=memory&cache=shared")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(Schema())
	require.NoError(t, err)
	repo := NewSQLiteRepository(db)
	require.NoError(t, repo.Seed(context.Background()))

	assigned, err := repo.Assign(context.Background(), Assignment{EntryID: "entry", FacetID: "episode"})
	require.NoError(t, err)
	require.Equal(t, "system:facets", assigned.ActorID)
}
