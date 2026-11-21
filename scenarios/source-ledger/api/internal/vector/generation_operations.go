package vector

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	aisearch "github.com/vrooli/ai-go/search"
)

// GenerationOperation is the durable, public status of one owner operation.
// It exposes the capacity receipt ID, never a lease capability.
type GenerationOperation struct {
	OperationID  string `json:"operation_id"`
	Operation    string `json:"operation"`
	GenerationID string `json:"generation_id"`
	RecipeID     string `json:"recipe_id,omitempty"`
	State        string `json:"state"`
	ReceiptID    string `json:"receipt_id,omitempty"`
	PeakBytes    int64  `json:"peak_bytes"`
	StartedAt    string `json:"started_at"`
	CompletedAt  string `json:"completed_at,omitempty"`
	Error        string `json:"error,omitempty"`
}

// GenerationValidation reports whether a generation still represents every
// eligible source row through the current change-feed watermark. Validation is
// read-only; promotion repeats the watermark fence while holding the writer
// mutex so a source change between validation and cutover cannot slip through.
type GenerationValidation struct {
	GenerationID      string `json:"generation_id"`
	RecipeID          string `json:"recipe_id"`
	State             string `json:"state"`
	Dimensions        int    `json:"dimensions"`
	ExpectedPoints    int64  `json:"expected_points"`
	ActualPoints      int64  `json:"actual_points"`
	MissingPoints     int64  `json:"missing_points"`
	OrphanedPoints    int64  `json:"orphaned_points"`
	InvalidPoints     int64  `json:"invalid_points"`
	PendingChanges    int64  `json:"pending_changes"`
	CatchupWatermark  int64  `json:"catchup_watermark"`
	CurrentWatermark  int64  `json:"current_watermark"`
	ReadyForPromotion bool   `json:"ready_for_promotion"`
}

// Operation returns one durable owner operation for recovery and status APIs.
func (b *SQLiteGenerationBackend) Operation(ctx context.Context, operationID string) (GenerationOperation, error) {
	if b == nil || b.db == nil || anyBlank(operationID) {
		return GenerationOperation{}, errors.New("Source Ledger operation status requires the owner database and operation ID")
	}
	var operation GenerationOperation
	err := b.db.QueryRowContext(ctx, `SELECT operation_id,operation,generation_id,recipe_id,state,receipt_id,peak_bytes,started_at,completed_at,error
FROM vector_generation_operations WHERE operation_id=?`, operationID).Scan(
		&operation.OperationID, &operation.Operation, &operation.GenerationID, &operation.RecipeID,
		&operation.State, &operation.ReceiptID, &operation.PeakBytes, &operation.StartedAt,
		&operation.CompletedAt, &operation.Error,
	)
	if err != nil {
		return GenerationOperation{}, err
	}
	return operation, nil
}

// Validate inspects source coverage and encoded points in one SQLite snapshot.
// It never assigns identity to the legacy vectors; only the persisted recipe of
// this requested generation is checked.
func (b *SQLiteGenerationBackend) Validate(ctx context.Context, generationID string) (GenerationValidation, error) {
	if b == nil || b.db == nil || anyBlank(generationID) {
		return GenerationValidation{}, errors.New("Source Ledger generation validation requires the owner database and generation ID")
	}
	generation, err := b.Load(ctx, generationID)
	if err != nil {
		return GenerationValidation{}, err
	}
	if _, err := b.exactRecipe(ctx, generation.RecipeID, "Source Ledger generation recipe is not currently owner-verified"); err != nil {
		return GenerationValidation{}, err
	}
	out, err := b.measureGeneration(ctx, generation)
	if err != nil {
		return GenerationValidation{}, err
	}
	out.ReadyForPromotion = promotableStates[generation.State] &&
		[...]int64{out.MissingPoints, out.OrphanedPoints, out.InvalidPoints, out.PendingChanges} == [4]int64{} &&
		out.ExpectedPoints == out.ActualPoints && out.CatchupWatermark == out.CurrentWatermark
	return out, nil
}

var promotableStates = map[string]bool{"candidate": true, "active": true, "retained": true}

// measureGeneration counts coverage, feed lag and bad points in one snapshot.
func (b *SQLiteGenerationBackend) measureGeneration(ctx context.Context, generation SQLiteGeneration) (GenerationValidation, error) {
	tx, err := b.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return GenerationValidation{}, fmt.Errorf("begin Source Ledger generation validation snapshot: %w", err)
	}
	defer tx.Rollback()
	out := GenerationValidation{
		GenerationID: generation.ID, RecipeID: generation.RecipeID,
		State: generation.State, Dimensions: generation.Dimensions,
		CatchupWatermark: generation.CatchupWatermark,
	}
	counts := []struct {
		what  string
		into  *int64
		query string
		args  []any
	}{
		{"eligible sources", &out.ExpectedPoints, `SELECT
 (SELECT COUNT(*) FROM facet_texts WHERE trim(text)<>'') +
 (SELECT COUNT(*) FROM summaries WHERE trim(body)<>'')`, nil},
		{"points", &out.ActualPoints, `SELECT COUNT(*) FROM vector_generation_points WHERE generation_id=?`, []any{generation.ID}},
		{"missing points", &out.MissingPoints, `SELECT
 (SELECT COUNT(*) FROM facet_texts s LEFT JOIN vector_generation_points p
   ON p.generation_id=? AND p.dataset_id='journal.facets' AND p.source_id=s.id
   WHERE trim(s.text)<>'' AND p.source_id IS NULL) +
 (SELECT COUNT(*) FROM summaries s LEFT JOIN vector_generation_points p
   ON p.generation_id=? AND p.dataset_id='forest.summaries' AND p.source_id=s.id
   WHERE trim(s.body)<>'' AND p.source_id IS NULL)`, []any{generation.ID, generation.ID}},
		{"orphaned points", &out.OrphanedPoints, `SELECT COUNT(*) FROM vector_generation_points p
LEFT JOIN facet_texts ft ON p.dataset_id='journal.facets' AND ft.id=p.source_id
LEFT JOIN summaries s ON p.dataset_id='forest.summaries' AND s.id=p.source_id
WHERE p.generation_id=? AND (
 (p.dataset_id='journal.facets' AND (ft.id IS NULL OR trim(ft.text)='')) OR
 (p.dataset_id='forest.summaries' AND (s.id IS NULL OR trim(s.body)=''))
)`, []any{generation.ID}},
		{"pending changes", &out.PendingChanges, `SELECT COUNT(*) FROM vector_generation_changes WHERE sequence>?`, []any{generation.CatchupWatermark}},
		{"change watermark", &out.CurrentWatermark, `SELECT COALESCE(MAX(sequence),0) FROM vector_generation_changes`, nil},
	}
	for _, count := range counts {
		if err := tx.QueryRowContext(ctx, count.query, count.args...).Scan(count.into); err != nil {
			return GenerationValidation{}, fmt.Errorf("count Source Ledger generation %s: %w", count.what, err)
		}
	}
	for _, dataset := range generationDatasets {
		err := visitGenerationPoints(ctx, tx, generation.ID, dataset, func(_ string, blob []byte) error {
			if _, err := decodeGenerationVector(blob, generation.Dimensions); err != nil {
				out.InvalidPoints++
			}
			return nil
		})
		if err != nil {
			return GenerationValidation{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return GenerationValidation{}, err
	}
	return out, nil
}

// ResumeOperation replays an incomplete operation with the same durable ID.
// Stage resolves the exact persisted recipe so a current policy change cannot
// silently retarget a partially-built generation.
func (b *SQLiteGenerationBackend) ResumeOperation(ctx context.Context, operationID string) (GenerationOperation, SQLiteGeneration, error) {
	operation, err := b.Operation(ctx, operationID)
	if err != nil {
		return GenerationOperation{}, SQLiteGeneration{}, err
	}
	if operation.State != "complete" {
		if err := b.replay(ctx, operation); err != nil {
			return operation, SQLiteGeneration{}, fmt.Errorf("resume Source Ledger %s operation %s: %w", operation.Operation, operation.OperationID, err)
		}
		if operation, err = b.Operation(ctx, operationID); err != nil {
			return GenerationOperation{}, SQLiteGeneration{}, err
		}
	}
	if operation.GenerationID == collectTarget {
		return operation, SQLiteGeneration{}, nil
	}
	generation, err := b.Load(ctx, operation.GenerationID)
	return operation, generation, err
}

func (b *SQLiteGenerationBackend) replay(ctx context.Context, operation GenerationOperation) error {
	var err error
	switch operation.Operation {
	case "stage":
		var recipe aisearch.EmbeddingRecipe
		if recipe, err = b.exactRecipe(ctx, operation.RecipeID, "cannot resume Source Ledger stage without its exact persisted recipe"); err == nil {
			_, err = b.stageWithRecipe(ctx, operation.OperationID, recipe)
		}
	case "catch_up":
		_, err = b.CatchUp(ctx, operation.OperationID, operation.GenerationID)
	case "promote":
		_, err = b.Promote(ctx, operation.OperationID, operation.GenerationID)
	case "rollback":
		_, err = b.Rollback(ctx, operation.OperationID)
	case "collect":
		_, err = b.Collect(ctx, operation.OperationID)
	default:
		err = fmt.Errorf("unsupported Source Ledger operation recovery kind %q", operation.Operation)
	}
	return err
}
