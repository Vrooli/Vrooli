package vector

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"
)

//go:embed generation.sql
var generationSchema string

// SQLCensusDB is the minimal SQLite owner surface needed for read-only census
// plus its durable observation record.
type SQLCensusDB interface {
	sqlQueryer
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type CensusRecord struct {
	StoreID       string    `json:"store_id"`
	DatasetID     string    `json:"dataset_id"`
	GenerationID  string    `json:"generation_id"`
	RecipeID      string    `json:"recipe_id,omitempty"`
	IdentityState string    `json:"identity_state"`
	Dimensions    int       `json:"dimensions"`
	VectorCount   int64     `json:"vector_count"`
	MissingCount  int64     `json:"missing_count"`
	InvalidCount  int64     `json:"invalid_count"`
	ObservedAt    time.Time `json:"observed_at"`
}

type dimensionCounts struct {
	vectors int64
	missing int64
	invalid int64
}

// Schema provides Source Ledger's durable vector-generation census table.
// The table deliberately supports an unresolved sentinel: the existing vector
// rows do not carry model, recipe, or generation identity.
func Schema() string { return generationSchema }

// CensusExistingVectors observes the two Source Ledger datasets without
// assigning a recipe to historical vectors. It records per-dimension counts,
// missing/corrupt rows, and unresolved generation sentinels in the same SQLite
// transaction. Only an explicit caller of this API writes census evidence.
func CensusExistingVectors(ctx context.Context, db SQLCensusDB, observedAt time.Time) ([]CensusRecord, error) {
	if db == nil {
		return nil, errors.New("Source Ledger vector census requires its owner database")
	}
	if observedAt.IsZero() {
		return nil, errors.New("Source Ledger vector census requires an observation time")
	}
	if _, err := db.ExecContext(ctx, Schema()); err != nil {
		return nil, fmt.Errorf("initialize Source Ledger vector census: %w", err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin Source Ledger census snapshot: %w", err)
	}
	defer tx.Rollback()
	active, err := activeGeneration(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("read active Source Ledger generation: %w", err)
	}
	records := make([]CensusRecord, 0)
	for _, dataset := range censusDatasets {
		observed, err := censusDataset(ctx, tx, dataset.name, dataset.query, active, observedAt.UTC())
		if err != nil {
			return nil, err
		}
		records = append(records, observed...)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit Source Ledger vector census: %w", err)
	}
	return records, nil
}

var censusDatasets = []struct{ name, query string }{
	{vectorDatasetFacets, `SELECT vector_blob, vector_json FROM embeddings`},
	{vectorDatasetSummaries, `SELECT vector_blob, vector_json FROM summaries`},
}

// censusDataset records legacy vectors as an unresolved sentinel, plus the
// active generation's points under its bound recipe.
func censusDataset(ctx context.Context, tx *sql.Tx, dataset, legacyQuery string, active ActiveGenerationVectors, observedAt time.Time) ([]CensusRecord, error) {
	counts, err := censusDimensions(ctx, tx, legacyQuery)
	if err != nil {
		return nil, fmt.Errorf("census Source Ledger dataset %s: %w", dataset, err)
	}
	records := appendCensus(nil, CensusRecord{
		StoreID: "source-ledger.sqlite", DatasetID: dataset, GenerationID: "unresolved",
		IdentityState: "identity_unknown", ObservedAt: observedAt,
	}, counts)
	if active.Active {
		counts, err := censusGenerationDimensions(ctx, tx, dataset, active.GenerationID)
		if err != nil {
			return nil, fmt.Errorf("census active Source Ledger generation %s/%s: %w", dataset, active.GenerationID, err)
		}
		records = appendCensus(records, CensusRecord{
			StoreID: "source-ledger.sqlite", DatasetID: dataset, GenerationID: active.GenerationID,
			RecipeID: active.RecipeID, IdentityState: "recipe_bound", ObservedAt: observedAt,
		}, counts)
	}
	for _, record := range records {
		if _, err := tx.ExecContext(ctx, `INSERT INTO vector_generation_census
(store_id,dataset_id,generation_id,recipe_id,identity_state,dimensions,vector_count,missing_count,invalid_count,observed_at)
VALUES(?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(store_id,dataset_id,generation_id,dimensions) DO UPDATE SET
 recipe_id=excluded.recipe_id, identity_state=excluded.identity_state, vector_count=excluded.vector_count,
 missing_count=excluded.missing_count, invalid_count=excluded.invalid_count,
			observed_at=excluded.observed_at`, record.StoreID, record.DatasetID, record.GenerationID, record.RecipeID,
			record.IdentityState, record.Dimensions, record.VectorCount, record.MissingCount,
			record.InvalidCount, record.ObservedAt.Format(time.RFC3339Nano)); err != nil {
			return nil, fmt.Errorf("persist Source Ledger census for %s: %w", record.DatasetID, err)
		}
	}
	return records, nil
}

// appendCensus appends one record per dimension; an empty observation still
// records a zero-dimension row.
func appendCensus(records []CensusRecord, template CensusRecord, counts map[int]dimensionCounts) []CensusRecord {
	if len(counts) == 0 {
		counts = map[int]dimensionCounts{0: {}}
	}
	for _, dim := range slices.Sorted(maps.Keys(counts)) {
		record := template
		record.Dimensions = dim
		record.VectorCount, record.MissingCount, record.InvalidCount = counts[dim].vectors, counts[dim].missing, counts[dim].invalid
		records = append(records, record)
	}
	return records
}

func censusGenerationDimensions(ctx context.Context, tx *sql.Tx, dataset, generationID string) (map[int]dimensionCounts, error) {
	out := map[int]dimensionCounts{}
	err := visitGenerationPoints(ctx, tx, generationID, dataset, func(_ string, blob []byte) error {
		vector, err := Decode(blob)
		if err != nil {
			counts := out[0]
			counts.invalid++
			out[0] = counts
			return nil
		}
		counts := out[len(vector)]
		counts.vectors++
		out[len(vector)] = counts
		return nil
	})
	return out, err
}

func censusDimensions(ctx context.Context, tx *sql.Tx, query string) (map[int]dimensionCounts, error) {
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int]dimensionCounts{}
	for rows.Next() {
		var blob []byte
		var raw string
		if err := rows.Scan(&blob, &raw); err != nil {
			return nil, err
		}
		vector, err := decodeStoredVector(blob, raw)
		if err != nil {
			counts := out[0]
			counts.invalid++
			out[0] = counts
			continue
		}
		if len(vector) == 0 {
			counts := out[0]
			counts.missing++
			out[0] = counts
			continue
		}
		counts := out[len(vector)]
		counts.vectors++
		out[len(vector)] = counts
	}
	return out, rows.Err()
}

func decodeStoredVector(blob []byte, raw string) ([]float64, error) {
	if len(blob) > 0 {
		return Decode(blob)
	}
	if raw == "" || raw == "[]" {
		return nil, nil
	}
	var vector []float64
	if err := json.Unmarshal([]byte(raw), &vector); err != nil {
		return nil, err
	}
	return vector, nil
}
