package vector_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	aisearch "github.com/vrooli/ai-go/search"
	"source-ledger/internal/vector"
)

func TestGenerationValidationStatusAndExactRecipeRecovery(t *testing.T) {
	ctx := context.Background()
	db := openGenerationDB(t, filepath.Join(t.TempDir(), "source-ledger.sqlite"))
	require.NoError(t, ensureGenerationSchemas(ctx, db))
	seedGenerationSources(t, db, "facet-a", "topic alpha", "summary-a", "summary alpha")

	oldRecipe := generationRecipe(t, "model-a", 3)
	newRecipe := generationRecipe(t, "model-b", 4)
	recipes := &generationRecipes{current: oldRecipe, byID: map[string]aisearch.EmbeddingRecipe{oldRecipe.ID(): oldRecipe, newRecipe.ID(): newRecipe}}
	embedder := &generationEmbedder{}
	embedder.setDimensions(map[string]int{oldRecipe.ID(): 3, newRecipe.ID(): 4})
	embedder.failText, embedder.failOnce = "summary alpha", true
	backend, err := vector.NewSQLiteGenerationBackend(db, recipes, embedder, newGenerationDiskAdmission(), vector.SQLiteGenerationPeakEstimate{}, vector.SQLiteGenerationOptions{BatchSize: 1})
	require.NoError(t, err)

	_, err = backend.Stage(ctx, "stable-stage")
	require.Error(t, err)
	operation, err := backend.Operation(ctx, "stable-stage")
	require.NoError(t, err)
	require.Equal(t, "stage", operation.Operation)
	require.Equal(t, "running", operation.State)
	require.NotEmpty(t, operation.ReceiptID)
	require.Positive(t, operation.PeakBytes)

	// A policy change must not retarget the partial generation. Recovery uses
	// the persisted, owner-resolved recipe identity for the original operation.
	recipes.current = newRecipe
	recoveredOperation, generation, err := backend.ResumeOperation(ctx, "stable-stage")
	require.NoError(t, err)
	require.Equal(t, "complete", recoveredOperation.State)
	require.Equal(t, oldRecipe.ID(), generation.RecipeID)
	require.Equal(t, "candidate", generation.State)

	validation, err := backend.Validate(ctx, generation.ID)
	require.NoError(t, err)
	require.True(t, validation.ReadyForPromotion)
	require.EqualValues(t, 2, validation.ExpectedPoints)
	require.EqualValues(t, validation.ExpectedPoints, validation.ActualPoints)
	require.Zero(t, validation.MissingPoints)
	require.Zero(t, validation.OrphanedPoints)
	require.Zero(t, validation.InvalidPoints)
	require.Zero(t, validation.PendingChanges)

	_, err = db.ExecContext(ctx, `UPDATE facet_texts SET text='topic alpha changed' WHERE id='facet-a'`)
	require.NoError(t, err)
	validation, err = backend.Validate(ctx, generation.ID)
	require.NoError(t, err)
	require.False(t, validation.ReadyForPromotion)
	require.Positive(t, validation.PendingChanges)

	_, err = backend.CatchUp(ctx, "stable-catch-up", generation.ID)
	require.NoError(t, err)
	validation, err = backend.Validate(ctx, generation.ID)
	require.NoError(t, err)
	require.True(t, validation.ReadyForPromotion)

	active, err := backend.Promote(ctx, "stable-promote", generation.ID)
	require.NoError(t, err)
	require.Equal(t, "active", active.State)
	operation, err = backend.Operation(ctx, "stable-promote")
	require.NoError(t, err)
	require.Equal(t, "complete", operation.State)
	_, _, err = backend.ResumeOperation(ctx, "stable-promote")
	require.NoError(t, err, "resuming a completed cutover is an idempotent status read")
}
