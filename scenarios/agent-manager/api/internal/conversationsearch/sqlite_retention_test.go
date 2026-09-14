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

func TestPruneSettledGenerationsKeepsOnlyUnpublishedStaging(t *testing.T) {
	t.Parallel()

	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	repository := NewSQLiteRepository(db)
	ctx := context.Background()
	base := time.Date(2026, 9, 9, 6, 0, 0, 0, time.UTC)
	stage := func(id, state string, offset int) {
		created := base.Add(time.Duration(offset) * time.Minute)
		require.NoError(t, repository.SaveGeneration(ctx, Generation{GenerationID: id, State: state, RecipeVersion: DefaultRecipeVersion, CreatedAt: created, UpdatedAt: created}))
		document := testDocument()
		document.DocumentID = "doc-" + id
		document.SourceRunID, document.SourceEventID, document.SourceMessageID = "run-"+id, "event-"+id, "message-"+id
		require.NoError(t, repository.StageDocument(ctx, id, document))
	}
	// gen-1 .. gen-3 activate in order: gen-3 serves and the others retire.
	for index := 1; index <= 3; index++ {
		id := fmt.Sprintf("gen-%d", index)
		stage(id, "building", index)
		require.NoError(t, repository.PublishStagedGeneration(ctx, id, 1))
		require.NoError(t, repository.ActivateGeneration(ctx, id, base.Add(time.Duration(index)*time.Minute)))
	}
	stage("abandoned", "failed", 4)
	stage("resumable", "ready", 5)
	stage("in-flight", "building", 6)
	rowsFor := func(id string) int {
		var count int
		require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM conversation_search_generation_documents WHERE generation_id = ?`, id))
		return count
	}

	purged, err := repository.PruneSettledGenerations(ctx)

	require.NoError(t, err)
	require.Equal(t, []string{"gen-1", "gen-2", "gen-3", "abandoned"}, purged, "active, retired and failed generations never read staging again")
	for _, id := range purged {
		require.Zero(t, rowsFor(id), "%s kept staged rows", id)
	}
	require.Equal(t, 1, rowsFor("resumable"), "restart recovery republishes a ready generation from its staged rows")
	require.Equal(t, 1, rowsFor("in-flight"), "a building generation is still being staged")
	ids, err := repository.ProjectionDocumentIDs(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"doc-gen-3"}, ids, "pruning staging never touches the serving catalog")

	again, err := repository.PruneSettledGenerations(ctx)
	require.NoError(t, err)
	require.Empty(t, again)
}
