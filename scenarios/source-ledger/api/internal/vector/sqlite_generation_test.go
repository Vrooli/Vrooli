package vector_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"source-ledger/internal/facets"
	"source-ledger/internal/forest"
	"source-ledger/internal/journal"
	"source-ledger/internal/policy"
	"source-ledger/internal/recall"
	"source-ledger/internal/vector"

	"github.com/stretchr/testify/require"
	aisearch "github.com/vrooli/ai-go/search"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

type generationRecipes struct {
	current aisearch.EmbeddingRecipe
	byID    map[string]aisearch.EmbeddingRecipe
}

func (p *generationRecipes) ResolveCurrentRecipe(context.Context) (aisearch.EmbeddingRecipe, error) {
	return p.current, nil
}

func (p *generationRecipes) ResolveRecipe(_ context.Context, id string) (aisearch.EmbeddingRecipe, error) {
	recipe, ok := p.byID[id]
	if !ok {
		return aisearch.EmbeddingRecipe{}, fmt.Errorf("recipe %q is not owner-approved", id)
	}
	return recipe, nil
}

type generationEmbedder struct {
	mu         sync.Mutex
	failText   string
	failOnce   bool
	calls      int
	dimensions map[string]int
}

func (e *generationEmbedder) EmbedDocument(_ context.Context, recipeID, text string) ([]float64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calls++
	if e.failOnce && text == e.failText {
		e.failOnce = false
		return nil, errors.New("simulated process interruption while building a page")
	}
	recipeDims := e.dimensions[recipeID]
	if recipeDims <= 0 {
		return nil, errors.New("fake encoder received an unknown recipe identity")
	}
	vector := make([]float64, recipeDims)
	vector[0] = float64(len(text))
	if recipeDims > 1 {
		vector[1] = 1
	}
	return vector, nil
}

// dimensions is kept separate from recipe request data to mirror the owner
// resolver's immutable registration.
func (e *generationEmbedder) setDimensions(values map[string]int) { e.dimensions = values }

type generationDiskAdmission struct {
	mu               sync.Mutex
	receipts         map[string]string
	completed        map[string]int
	failNext         int
	completeCalls    int
	failCompleteCall int
}

func newGenerationDiskAdmission() *generationDiskAdmission {
	return &generationDiskAdmission{receipts: make(map[string]string), completed: make(map[string]int)}
}

func (a *generationDiskAdmission) Admit(_ context.Context, operationID, targetID, recipeID string, peak int64) (string, error) {
	if operationID == "" || targetID == "" || recipeID == "" || peak <= 0 {
		return "", errors.New("fake disk owner received an incomplete admission request")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if receipt := a.receipts[operationID]; receipt != "" {
		return receipt, nil
	}
	receipt := "receipt:" + operationID
	a.receipts[operationID] = receipt
	return receipt, nil
}

func (a *generationDiskAdmission) Complete(_ context.Context, operationID, receipt string, success bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.receipts[operationID] != receipt {
		return errors.New("fake disk receipt identity mismatch")
	}
	if success {
		a.completeCalls++
		if a.failCompleteCall > 0 && a.completeCalls == a.failCompleteCall {
			return errors.New("simulated owner release interruption")
		}
		if a.failNext > 0 {
			a.failNext--
			return errors.New("simulated owner release interruption")
		}
		a.completed[operationID]++
	}
	return nil
}

func TestSQLiteGenerationCrashResumeCutoverRollbackAndCollect(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "source-ledger.sqlite")
	db := openGenerationDB(t, path)
	require.NoError(t, ensureGenerationSchemas(ctx, db))
	seedGenerationSources(t, db, "facet-a", "topic alpha", "summary-a", "summary alpha")

	oldRecipe := generationRecipe(t, "model-a", 3)
	newRecipe := generationRecipe(t, "model-b", 4)
	recipes := &generationRecipes{current: oldRecipe, byID: map[string]aisearch.EmbeddingRecipe{oldRecipe.ID(): oldRecipe, newRecipe.ID(): newRecipe}}
	embedder := &generationEmbedder{}
	embedder.setDimensions(map[string]int{oldRecipe.ID(): 3, newRecipe.ID(): 4})
	disk := newGenerationDiskAdmission()
	now := time.Date(2026, 11, 12, 13, 0, 0, 0, time.UTC)
	backend, err := vector.NewSQLiteGenerationBackend(db, recipes, embedder, disk, vector.SQLiteGenerationPeakEstimate{}, vector.SQLiteGenerationOptions{
		Now: func() time.Time { return now }, RollbackWindow: time.Hour, BatchSize: 1,
	})
	require.NoError(t, err)

	embedder.failText, embedder.failOnce = "summary alpha", true
	_, err = backend.Stage(ctx, "stage-old")
	require.Error(t, err)
	var interruptedID string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT generation_id FROM vector_generation_operations WHERE operation_id='stage-old'`).Scan(&interruptedID))
	var pointCount int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM vector_generation_points WHERE generation_id=?`, interruptedID).Scan(&pointCount))
	require.Equal(t, 1, pointCount, "the first committed page should survive a process failure; generation=%q err=%v", interruptedID, err)
	_ = db.Close()
	db = openGenerationDB(t, path)
	require.NoError(t, ensureGenerationSchemas(ctx, db))
	backend, err = vector.NewSQLiteGenerationBackend(db, recipes, embedder, disk, vector.SQLiteGenerationPeakEstimate{}, vector.SQLiteGenerationOptions{
		Now: func() time.Time { return now }, RollbackWindow: time.Hour, BatchSize: 1,
	})
	require.NoError(t, err)
	oldCandidate, err := backend.Stage(ctx, "stage-old")
	require.NoError(t, err)
	require.Equal(t, interruptedID, oldCandidate.ID)
	require.Equal(t, "candidate", oldCandidate.State)
	require.Equal(t, oldRecipe.ID(), oldCandidate.RecipeID)
	oldActive, err := backend.Promote(ctx, "promote-old", oldCandidate.ID)
	require.NoError(t, err)
	require.Equal(t, "active", oldActive.State)

	// Inserts and deletes are trigger-fed. A failed cutover stays atomic until
	// the candidate has consumed the source changes through the current watermark.
	seedGenerationSources(t, db, "facet-b", "topic beta", "", "")
	newCandidate, err := backend.Stage(ctx, "stage-new")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `DELETE FROM summaries WHERE id='summary-a'`)
	require.NoError(t, err)
	seedGenerationSources(t, db, "facet-c", "topic gamma", "", "")
	before, ok, err := backend.Active(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	_, err = backend.Promote(ctx, "promote-too-early", newCandidate.ID)
	require.ErrorIs(t, err, vector.ErrGenerationCatchUpRequired)
	stillActive, ok, err := backend.Active(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, before.ID, stillActive.ID, "a refused cutover changed the active generation")

	recipes.current = newRecipe
	caughtUp, err := backend.CatchUp(ctx, "catch-up-new", newCandidate.ID)
	require.NoError(t, err)
	require.Equal(t, "candidate", caughtUp.State)
	_, err = backend.CatchUp(ctx, "catch-up-new", newCandidate.ID)
	require.NoError(t, err, "same completed catch-up operation must be idempotent")
	disk.mu.Lock()
	disk.failNext = 1
	disk.mu.Unlock()
	_, err = backend.Promote(ctx, "promote-new", newCandidate.ID)
	require.Error(t, err, "pointer change must be durable even if lease completion is interrupted")
	newActive, err := backend.Promote(ctx, "promote-new", newCandidate.ID)
	require.NoError(t, err)
	require.Equal(t, "active", newActive.State)
	require.Equal(t, oldActive.ID, newActive.PredecessorID)

	// Rollback catches the retained recipe up through the same owner feed before
	// switching its pointer, then retains the displaced generation for a window.
	seedGenerationSources(t, db, "facet-d", "topic delta", "", "")
	disk.mu.Lock()
	disk.completeCalls = 0
	disk.failCompleteCall = 2 // catch-up releases first; the pointer-switch release fails second.
	disk.mu.Unlock()
	_, err = backend.Rollback(ctx, "rollback-new")
	require.ErrorContains(t, err, "simulated owner release interruption")
	activeAfterRollback, ok, err := backend.Active(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, oldActive.ID, activeAfterRollback.ID, "the durable pointer switch precedes capacity release")
	rolledBack, err := backend.Rollback(ctx, "rollback-new")
	require.NoError(t, err)
	require.Equal(t, oldActive.ID, rolledBack.ID)
	require.Equal(t, "active", rolledBack.State)
	activeAfterRollback, ok, err = backend.Active(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, oldActive.ID, activeAfterRollback.ID, "retry must not switch back to the displaced generation")
	rollbackOperation, err := backend.Operation(ctx, "rollback-new")
	require.NoError(t, err)
	require.Equal(t, "complete", rollbackOperation.State)
	completedRollback, err := backend.Rollback(ctx, "rollback-new")
	require.NoError(t, err, "completed rollback replay must return the durable result")
	require.Equal(t, oldActive.ID, completedRollback.ID)
	newRetained, err := backend.Load(ctx, newActive.ID)
	require.NoError(t, err)
	require.Equal(t, "retained", newRetained.State)

	var activePoints, staleDeleted int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM vector_generation_points WHERE generation_id=?`, oldActive.ID).Scan(&activePoints))
	require.Equal(t, 4, activePoints, "rollback target should be caught up through the latest feed sequence")
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM vector_generation_points WHERE generation_id=? AND source_id='summary-a'`, oldActive.ID).Scan(&staleDeleted))
	require.Zero(t, staleDeleted, "delete tombstone failed to remove the summary")

	now = now.Add(2 * time.Hour)
	removed, err := backend.Collect(ctx, "collect-expired")
	require.NoError(t, err)
	require.Greater(t, removed, int64(0))
	newRetained, err = backend.Load(ctx, newActive.ID)
	require.NoError(t, err)
	require.Equal(t, "deleted", newRetained.State)
	activeAfterRollback, ok, err = backend.Active(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, oldActive.ID, activeAfterRollback.ID, "cleanup removed the active generation")

	var unresolved int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM vector_generation_census WHERE generation_id='unresolved' AND recipe_id=''`).Scan(&unresolved))
	require.Zero(t, unresolved, "the test corpus had no legacy vectors; candidate recipes must be separate")
}

type recipeDirectionEmbedder struct{ directions map[string][]float64 }

func (e recipeDirectionEmbedder) EmbedDocument(_ context.Context, recipeID, _ string) ([]float64, error) {
	return append([]float64(nil), e.directions[recipeID]...), nil
}

type recipeDirectionQueryEmbedder struct{ directions map[string][]float64 }

func (e recipeDirectionQueryEmbedder) EmbedQuery(context.Context, string) ([]float64, error) {
	return []float64{1, 0}, nil
}

func (e recipeDirectionQueryEmbedder) EmbedQueryForRecipe(_ context.Context, recipeID, _ string) ([]float64, error) {
	return append([]float64(nil), e.directions[recipeID]...), nil
}

func TestServingCallersReadPromotedGenerationAndRollbackRecipe(t *testing.T) {
	ctx := context.Background()
	db := openGenerationDB(t, filepath.Join(t.TempDir(), "source-ledger-serving.sqlite"))
	require.NoError(t, apidb.EnsureSchemas(ctx, db,
		apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(facets.Schema),
		apidb.SchemaProviderFunc(forest.Schema), apidb.SchemaProviderFunc(vector.Schema)))

	entryID := "serving-entry"
	_, err := db.ExecContext(ctx, `INSERT INTO entries(id,scope,body,facet_id,kind,created_at) VALUES(?,?,?,?,?,?)`, entryID, "agent-memory", "serving document", "episode", "note", time.Now().UTC().Format(time.RFC3339Nano))
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO facet_texts(id,entry_id,kind,text) VALUES('serving-facet-text',?,'topic','serving document')`, entryID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO embeddings(id,facet_text_id,vector_json,vector_blob,created_at) VALUES('legacy-serving-vector','serving-facet-text','',?,'2026-11-14')`, vector.Encode([]float64{1, 0}))
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO facet_definitions(id,scope,label,created_at) VALUES('episode','agent-memory','Episode','2026-11-14')`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO facet_policies(facet_id,scope,retention_policy,compaction_eligible) VALUES('episode','agent-memory','episode',1)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO facet_assignments(id,entry_id,facet_id,assigned_at,actor_id) VALUES('serving-assignment',?,'episode','2026-11-14','test')`, entryID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO summaries(id,scope,body,facet_id,vector_json,vector_blob,depth,generation,created_at) VALUES('serving-summary','agent-memory','summary','episode','',? ,1,1,'2026-11-14')`, vector.Encode([]float64{1, 0}))
	require.NoError(t, err)

	oldRecipe, newRecipe := generationRecipe(t, "serving-model-a", 2), generationRecipe(t, "serving-model-b", 2)
	recipes := &generationRecipes{current: oldRecipe, byID: map[string]aisearch.EmbeddingRecipe{oldRecipe.ID(): oldRecipe, newRecipe.ID(): newRecipe}}
	directions := map[string][]float64{oldRecipe.ID(): {1, 0}, newRecipe.ID(): {0, 1}}
	disk := newGenerationDiskAdmission()
	backend, err := vector.NewSQLiteGenerationBackend(db, recipes, recipeDirectionEmbedder{directions: directions}, disk, vector.SQLiteGenerationPeakEstimate{}, vector.SQLiteGenerationOptions{BatchSize: 16})
	require.NoError(t, err)
	oldCandidate, err := backend.Stage(ctx, "serving-stage-old")
	require.NoError(t, err)
	oldActive, err := backend.Promote(ctx, "serving-promote-old", oldCandidate.ID)
	require.NoError(t, err)
	require.Equal(t, "active", oldActive.State)

	serving, err := vector.NewSQLiteGenerationServing(db)
	require.NoError(t, err)
	source := recall.NewSQLiteSource(db, serving)
	queryEmbedder := recipeDirectionQueryEmbedder{directions: directions}
	service := recall.NewService(source, queryEmbedder, recall.Config{})
	assertServingRecipe := func(expectedID string, expected []float64) {
		t.Helper()
		recalled, err := service.Recall(policy.WithScope(ctx, "agent-memory"), "query", 10)
		hits := recalled.Hits
		require.NoError(t, err)
		require.NotEmpty(t, hits)
		var found bool
		for _, hit := range hits {
			if hit.Node.ID == entryID {
				found = true
				require.Equal(t, expected, hit.Node.Vectors[0])
				require.InDelta(t, 1, hit.Score, 1e-9)
			}
		}
		require.True(t, found, "recall must serve the leaf from generation %s", expectedID)
		recipeID, active, err := source.ActiveRecipeID(ctx)
		require.NoError(t, err)
		require.True(t, active)
		require.Equal(t, expectedID, recipeID)
	}
	assertForestRecipe := func(expected []float64) {
		t.Helper()
		forestCandidates, err := forest.NewSQLiteCandidateSource(db, serving).CompactionCandidates(policy.WithScope(ctx, "agent-memory"))
		require.NoError(t, err)
		var found bool
		for _, candidate := range forestCandidates {
			if candidate.ID == entryID {
				found = true
				require.Equal(t, expected, candidate.Vectors[0])
			}
		}
		require.True(t, found, "forest compaction must read the expected active generation")
		forestNodes, err := forest.NewSQLiteRepository(db, serving).Nodes(policy.WithScope(ctx, "agent-memory"), 0)
		require.NoError(t, err)
		for _, node := range forestNodes {
			if node.ID == entryID || node.ID == "serving-summary" {
				require.Equal(t, expected, node.Vectors[0], "forest node projection must read the active generation")
			}
		}
	}
	assertServingRecipe(oldRecipe.ID(), directions[oldRecipe.ID()])
	assertForestRecipe(directions[oldRecipe.ID()])

	recipes.current = newRecipe
	newCandidate, err := backend.Stage(ctx, "serving-stage-new")
	require.NoError(t, err)
	_, err = backend.CatchUp(ctx, "serving-catch-up-new", newCandidate.ID)
	require.NoError(t, err)
	newActive, err := backend.Promote(ctx, "serving-promote-new", newCandidate.ID)
	require.NoError(t, err)
	require.Equal(t, newRecipe.ID(), newActive.RecipeID)
	assertServingRecipe(newRecipe.ID(), directions[newRecipe.ID()])
	assertForestRecipe(directions[newRecipe.ID()])
	assertCensusRecipe := func(expected vector.SQLiteGeneration, expectedCount int64) {
		t.Helper()
		records, err := vector.CensusExistingVectors(ctx, db, time.Now().UTC())
		require.NoError(t, err)
		for _, dataset := range []string{"journal.facets", "forest.summaries"} {
			found := false
			for _, record := range records {
				if record.DatasetID == dataset && record.GenerationID == expected.ID {
					found = true
					require.Equal(t, expected.RecipeID, record.RecipeID)
					require.Equal(t, "recipe_bound", record.IdentityState)
					require.Equal(t, expectedCount, record.VectorCount)
				}
			}
			require.True(t, found, "census must report active generation %s for %s", expected.ID, dataset)
		}
	}
	assertCensusRecipe(newActive, 1)

	rolledBack, err := backend.Rollback(ctx, "serving-rollback-new")
	require.NoError(t, err)
	require.Equal(t, oldRecipe.ID(), rolledBack.RecipeID)
	assertServingRecipe(oldRecipe.ID(), directions[oldRecipe.ID()])
	assertCensusRecipe(rolledBack, 1)
	assertForestRecipe(directions[oldRecipe.ID()])
}

func TestSQLiteGenerationConcurrentSameArmStageIsIdempotent(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, ensureGenerationSchemas(ctx, db))
	seedGenerationSources(t, db, "facet-a", "topic alpha", "summary-a", "summary alpha")
	recipe := generationRecipe(t, "model-a", 3)
	recipes := &generationRecipes{current: recipe, byID: map[string]aisearch.EmbeddingRecipe{recipe.ID(): recipe}}
	embedder := &generationEmbedder{}
	embedder.setDimensions(map[string]int{recipe.ID(): 3})
	backend, err := vector.NewSQLiteGenerationBackend(db, recipes, embedder, newGenerationDiskAdmission(), vector.SQLiteGenerationPeakEstimate{}, vector.SQLiteGenerationOptions{BatchSize: 1})
	require.NoError(t, err)

	results := make([]vector.SQLiteGeneration, 2)
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index], errs[index] = backend.Stage(ctx, "stage-same")
		}(i)
	}
	wg.Wait()
	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	require.Equal(t, results[0].ID, results[1].ID)
	var generations, points int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM vector_generations`).Scan(&generations))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM vector_generation_points WHERE generation_id=?`, results[0].ID).Scan(&points))
	require.Equal(t, 1, generations)
	require.Equal(t, 2, points)
}

func openGenerationDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{TxLock: "immediate"})
	require.NoError(t, err)
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func ensureGenerationSchemas(ctx context.Context, db *sql.DB) error {
	return apidb.EnsureSchemas(ctx, db,
		apidb.SchemaProviderFunc(journal.Schema),
		apidb.SchemaProviderFunc(forest.Schema),
		apidb.SchemaProviderFunc(vector.Schema),
	)
}

func seedGenerationSources(t *testing.T, db *sql.DB, facetID, facetText, summaryID, summaryBody string) {
	t.Helper()
	if facetID != "" {
		entryID := "entry-" + facetID
		_, err := db.Exec(`INSERT INTO entries(id,scope,body,facet_id,kind,created_at) VALUES(?,?,?,?,?,?)`, entryID, "test", facetText, facetID, "note", time.Now().UTC().Format(time.RFC3339Nano))
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO facet_texts(id,entry_id,kind,text) VALUES(?,?,?,?)`, facetID, entryID, "topic", facetText)
		require.NoError(t, err)
	}
	if summaryID != "" {
		_, err := db.Exec(`INSERT INTO summaries(id,scope,body,facet_id,depth,generation,created_at) VALUES(?,?,?,?,?,?,?)`, summaryID, "test", summaryBody, "facet", 0, 1, time.Now().UTC().Format(time.RFC3339Nano))
		require.NoError(t, err)
	}
}

func generationRecipe(t *testing.T, model string, dimensions int) aisearch.EmbeddingRecipe {
	t.Helper()
	recipe, err := aisearch.NewEmbeddingRecipe(aisearch.EmbeddingRecipeInput{
		SchemaVersion: "1", ModelID: model, ModelRevision: "owner-approved:" + model,
		ArtifactSHA256: strings.Repeat("a", 64), TokenizerRevision: "tokenizer-v1",
		Runtime: "source-ledger-test-owner", RuntimeBuild: "test/v1", Precision: "f32",
		QueryInstruction: "query: ", DocumentInstruction: "document: ", Pooling: "mean",
		Normalization: "l2", Dimensions: dimensions, OutputDType: "float32",
		OutputQuantization: "none", Truncation: "reject-over-100-runes",
		ChunkerVersion: "source-ledger-test-chunker/v1", ComposerVersion: "source-ledger-test-composer/v1",
	})
	require.NoError(t, err)
	return recipe
}

// generationFixture is one staged-ready Source Ledger store with a single
// owner recipe, two eligible sources and a controllable clock.
type generationFixture struct {
	ctx      context.Context
	db       *sql.DB
	backend  *vector.SQLiteGenerationBackend
	recipes  *generationRecipes
	embedder *generationEmbedder
	disk     *generationDiskAdmission
	now      time.Time
}

func newGenerationFixture(t *testing.T) *generationFixture {
	t.Helper()
	f := &generationFixture{ctx: context.Background(), disk: newGenerationDiskAdmission(), now: time.Date(2026, 11, 12, 13, 0, 0, 0, time.UTC)}
	f.db = openGenerationDB(t, filepath.Join(t.TempDir(), "source-ledger.sqlite"))
	require.NoError(t, ensureGenerationSchemas(f.ctx, f.db))
	seedGenerationSources(t, f.db, "facet-a", "topic alpha", "summary-a", "summary alpha")
	recipe := generationRecipe(t, "model-a", 3)
	f.recipes = &generationRecipes{current: recipe, byID: map[string]aisearch.EmbeddingRecipe{recipe.ID(): recipe}}
	f.embedder = &generationEmbedder{}
	f.embedder.setDimensions(map[string]int{recipe.ID(): 3})
	var err error
	f.backend, err = vector.NewSQLiteGenerationBackend(f.db, f.recipes, f.embedder, f.disk, vector.SQLiteGenerationPeakEstimate{}, vector.SQLiteGenerationOptions{
		Now: func() time.Time { return f.now }, RollbackWindow: time.Hour, BatchSize: 1,
	})
	require.NoError(t, err)
	return f
}

func (f *generationFixture) completedReleases(operationID string) int {
	f.disk.mu.Lock()
	defer f.disk.mu.Unlock()
	return f.disk.completed[operationID]
}

func TestGenerationOperationIDsCannotBeRetargeted(t *testing.T) {
	f := newGenerationFixture(t)
	staged, err := f.backend.Stage(f.ctx, "op-stage")
	require.NoError(t, err)
	_, err = f.backend.CatchUp(f.ctx, "op-catch-up", staged.ID)
	require.NoError(t, err)
	_, err = f.backend.Collect(f.ctx, "op-collect")
	require.NoError(t, err)

	other := generationRecipe(t, "model-b", 4)
	f.recipes.byID[other.ID()] = other
	f.embedder.setDimensions(map[string]int{f.recipes.current.ID(): 3, other.ID(): 4})
	f.recipes.current = other
	second, err := f.backend.Stage(f.ctx, "op-stage-second")
	require.NoError(t, err)

	retargets := map[string]func() error{
		"stage under another recipe": func() error { _, err := f.backend.Stage(f.ctx, "op-stage"); return err },
		"stage op as catch-up":       func() error { _, err := f.backend.CatchUp(f.ctx, "op-stage", staged.ID); return err },
		"stage op as promote":        func() error { _, err := f.backend.Promote(f.ctx, "op-stage", staged.ID); return err },
		"stage op as collect":        func() error { _, err := f.backend.Collect(f.ctx, "op-stage"); return err },
		"catch-up of another gen":    func() error { _, err := f.backend.CatchUp(f.ctx, "op-catch-up", second.ID); return err },
		"collect op as catch-up":     func() error { _, err := f.backend.CatchUp(f.ctx, "op-collect", staged.ID); return err },
	}
	for name, retarget := range retargets {
		require.ErrorIs(t, retarget(), vector.ErrGenerationOperationConflict, name)
	}
	reloaded, err := f.backend.Load(f.ctx, staged.ID)
	require.NoError(t, err)
	require.Equal(t, staged.RecipeID, reloaded.RecipeID, "a refused replay must not retarget the generation")
}

func TestPromotingTheActiveGenerationAgainChangesNothing(t *testing.T) {
	f := newGenerationFixture(t)
	staged, err := f.backend.Stage(f.ctx, "stage")
	require.NoError(t, err)
	active, err := f.backend.Promote(f.ctx, "promote", staged.ID)
	require.NoError(t, err)

	again, err := f.backend.Promote(f.ctx, "promote-again", staged.ID)
	require.NoError(t, err)
	require.Equal(t, active, again, "re-promoting the serving generation must be a no-op")
	operation, err := f.backend.Operation(f.ctx, "promote-again")
	require.NoError(t, err)
	require.Equal(t, "complete", operation.State)
	require.Equal(t, 1, f.completedReleases("promote-again"))

	removed, err := f.backend.Collect(f.ctx, "collect-nothing-expired")
	require.NoError(t, err)
	require.Zero(t, removed)
	stillActive, ok, err := f.backend.Active(f.ctx)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, active, stillActive)
}

func TestInterruptedCatchUpKeepsItsLeaseAndResumes(t *testing.T) {
	f := newGenerationFixture(t)
	staged, err := f.backend.Stage(f.ctx, "stage")
	require.NoError(t, err)
	seedGenerationSources(t, f.db, "facet-b", "topic beta", "", "")

	f.embedder.failText, f.embedder.failOnce = "topic beta", true
	_, err = f.backend.CatchUp(f.ctx, "catch-up", staged.ID)
	require.Error(t, err)
	operation, err := f.backend.Operation(f.ctx, "catch-up")
	require.NoError(t, err)
	require.Equal(t, "running", operation.State)
	require.NotEmpty(t, operation.ReceiptID)
	require.Zero(t, f.completedReleases("catch-up"), "an interrupted catch-up must keep its disk lease for recovery")
	validation, err := f.backend.Validate(f.ctx, staged.ID)
	require.NoError(t, err)
	require.False(t, validation.ReadyForPromotion)

	resumed, generation, err := f.backend.ResumeOperation(f.ctx, "catch-up")
	require.NoError(t, err)
	require.Equal(t, "complete", resumed.State)
	require.Equal(t, staged.ID, generation.ID)
	require.Equal(t, 1, f.completedReleases("catch-up"))
	validation, err = f.backend.Validate(f.ctx, staged.ID)
	require.NoError(t, err)
	require.True(t, validation.ReadyForPromotion)
	require.EqualValues(t, 3, validation.ActualPoints)
}

func TestCorruptGenerationPointBlocksValidationServingAndPromotion(t *testing.T) {
	f := newGenerationFixture(t)
	staged, err := f.backend.Stage(f.ctx, "stage")
	require.NoError(t, err)
	_, err = f.backend.Promote(f.ctx, "promote", staged.ID)
	require.NoError(t, err)
	// The blob declares one value but carries none.
	_, err = f.db.ExecContext(f.ctx, `UPDATE vector_generation_points SET vector_blob=X'01000000' WHERE generation_id=? AND source_id='facet-a'`, staged.ID)
	require.NoError(t, err)

	validation, err := f.backend.Validate(f.ctx, staged.ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, validation.InvalidPoints)
	require.False(t, validation.ReadyForPromotion)
	serving, err := vector.NewSQLiteGenerationServing(f.db)
	require.NoError(t, err)
	_, err = serving.ReadActiveVectors(f.ctx, "journal.facets")
	require.Error(t, err, "serving must refuse a point that does not match the recipe")
	summaries, err := serving.ReadActiveVectors(f.ctx, "forest.summaries")
	require.NoError(t, err)
	require.Len(t, summaries.Vectors, 1)
	_, err = f.backend.Promote(f.ctx, "promote-corrupt", staged.ID)
	require.ErrorIs(t, err, vector.ErrGenerationCatchUpRequired)

	records, err := vector.CensusExistingVectors(f.ctx, f.db, f.now)
	require.NoError(t, err)
	require.Contains(t, records, vector.CensusRecord{
		StoreID: "source-ledger.sqlite", DatasetID: "journal.facets", GenerationID: staged.ID,
		RecipeID: staged.RecipeID, IdentityState: "recipe_bound", Dimensions: 0, InvalidCount: 1, ObservedAt: f.now,
	}, "census must count the undecodable active point as invalid")
}
