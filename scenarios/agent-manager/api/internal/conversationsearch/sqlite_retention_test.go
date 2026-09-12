package conversationsearch

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

type observeGenerationDeleteDB struct {
	*sqlx.DB
	once        sync.Once
	afterDelete func()
}

func (d *observeGenerationDeleteDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	result, err := d.DB.ExecContext(ctx, query, args...)
	if err == nil && strings.HasPrefix(query, "DELETE FROM conversation_search_generation_documents") {
		if rows, _ := result.RowsAffected(); rows > 0 {
			d.once.Do(d.afterDelete)
		}
	}
	return result, err
}

func TestAbandonedGenerationCleanupYieldsWriterAndRemainsRecoverable(t *testing.T) {
	db, err := sqlx.Connect("sqlite", "file:"+filepath.Join(t.TempDir(), "writer.db")+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(100)")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	db.SetMaxOpenConns(2)
	applyProjectionSchema(t, db)
	_, err = db.Exec(`CREATE TABLE live_writer(value INTEGER NOT NULL)`)
	require.NoError(t, err)
	repository := NewSQLiteRepository(db)
	now := time.Now().UTC()
	require.NoError(t, repository.SaveGeneration(t.Context(), Generation{GenerationID: "abandoned", State: "building", RecipeVersion: DefaultRecipeVersion, CreatedAt: now, UpdatedAt: now}))

	document := testDocument()
	document.DocumentID = "abandoned-0000"
	require.NoError(t, repository.StageDocument(t.Context(), "abandoned", document))
	_, err = db.Exec(`WITH RECURSIVE copies(n) AS (SELECT 1 UNION ALL SELECT n+1 FROM copies WHERE n<2499)
 INSERT INTO conversation_search_generation_documents (generation_id, ` + projectionDocumentColumns + `)
 SELECT 'abandoned',printf('abandoned-%04d',n),` + strings.TrimPrefix(projectionDocumentColumns, "document_id, ") + `
 FROM conversation_search_generation_documents CROSS JOIN copies WHERE document_id='abandoned-0000'`)
	require.NoError(t, err)

	firstBatch, release := make(chan struct{}), make(chan struct{})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	observed := NewSQLiteRepository(&observeGenerationDeleteDB{DB: db, afterDelete: func() { close(firstBatch); <-release }})
	finished := make(chan error, 1)
	go func() { finished <- observed.RollbackStagedGeneration(ctx, "abandoned", "failed", now) }()
	select {
	case <-firstBatch:
	case <-time.After(2 * time.Second):
		t.Fatal("cleanup made no bounded progress")
	}
	writerCtx, writerCancel := context.WithTimeout(t.Context(), time.Second)
	defer writerCancel()
	_, writeErr := db.ExecContext(writerCtx, `INSERT INTO live_writer VALUES (1)`)
	var remaining int
	countErr := db.GetContext(writerCtx, &remaining, `SELECT COUNT(*) FROM conversation_search_generation_documents WHERE generation_id='abandoned'`)
	cancel()
	close(release)
	cleanupErr := <-finished
	require.NoError(t, writeErr, "ordinary writers must acquire SQLite between cleanup batches")
	require.NoError(t, countErr)
	require.Greater(t, remaining, 0, "one cleanup transaction consumed the whole generation")
	require.Less(t, remaining, 2500, "no cleanup progress was committed")
	require.ErrorIs(t, cleanupErr, context.Canceled)
	generation, err := repository.LoadGeneration(t.Context(), "abandoned")
	require.NoError(t, err)
	require.Equal(t, "building", generation.State, "partial cleanup must remain recoverable")
	require.NoError(t, repository.RollbackStagedGeneration(t.Context(), "abandoned", "failed", now))
	err = db.Get(&remaining, `SELECT COUNT(*) FROM conversation_search_generation_documents WHERE generation_id='abandoned'`)
	require.NoError(t, err)
	require.Zero(t, remaining)
	generation, err = repository.LoadGeneration(t.Context(), "abandoned")
	require.NoError(t, err)
	require.Equal(t, "failed", generation.State)
}

func TestPruneRetiredGenerationsKeepsNewestAndBoundsWork(t *testing.T) {
	t.Parallel()

	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	ctx := context.Background()
	base := time.Date(2026, 9, 9, 6, 0, 0, 0, time.UTC)

	// Five generations activated in order: gen-1 .. gen-4 end up retired,
	// gen-5 is the serving generation. Each stages one document row.
	for index := 1; index <= 5; index++ {
		id := fmt.Sprintf("gen-%d", index)
		created := base.Add(time.Duration(index) * time.Minute)
		require.NoError(t, repository.SaveGeneration(ctx, Generation{GenerationID: id, State: "building", RecipeVersion: DefaultRecipeVersion, CreatedAt: created, UpdatedAt: created}))
		document := testDocument()
		document.DocumentID = "doc-" + id
		document.SourceRunID, document.SourceEventID, document.SourceMessageID = "run-"+id, "event-"+id, "message-"+id
		require.NoError(t, repository.StageDocument(ctx, id, document))
		require.NoError(t, repository.PublishStagedGeneration(ctx, id, 1))
		require.NoError(t, repository.ActivateGeneration(ctx, id, created))
	}
	rowsFor := func(id string) int {
		var count int
		require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM conversation_search_generation_documents WHERE generation_id = ?`, id))
		return count
	}
	stateOf := func(id string) string {
		var state string
		require.NoError(t, db.Get(&state, `SELECT state FROM conversation_search_generations WHERE generation_id = ?`, id))
		return state
	}
	prune := func(keep, limit int) []string {
		purged, err := repository.PruneRetiredGenerations(ctx, keep, limit)
		require.NoError(t, err)
		return purged
	}
	for index := 1; index <= 4; index++ {
		require.Equal(t, "retired", stateOf(fmt.Sprintf("gen-%d", index)))
		require.Equal(t, 1, rowsFor(fmt.Sprintf("gen-%d", index)), "activation alone must not delete rows")
	}

	// keep=2 retains gen-4 and gen-3; limit=1 purges only the oldest.
	require.Equal(t, []string{"gen-2"}, prune(2, 1), "newest-first ordering skips the two kept, then purges the next")
	require.Equal(t, 0, rowsFor("gen-2"))
	require.Equal(t, "retired", stateOf("gen-2"), "the record stays retired; only its rows go")
	require.Equal(t, 1, rowsFor("gen-1"), "bounded by limit")

	require.Equal(t, []string{"gen-1"}, prune(2, 10), "an already-pruned generation is skipped, not counted")
	for _, kept := range []string{"gen-3", "gen-4", "gen-5"} {
		require.Equal(t, 1, rowsFor(kept), "%s must keep its rows", kept)
	}
	require.Equal(t, "active", stateOf("gen-5"))

	require.Empty(t, prune(2, 10), "nothing left beyond keep")
}
