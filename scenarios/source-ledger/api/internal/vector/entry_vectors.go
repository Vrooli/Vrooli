package vector

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type entryVectorReader interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// EntryVectors loads every facet and summary vector of one scope in one
// pass, keyed by entry or summary id: from the active generation when one is
// serving, else from the legacy embedding columns. Recall and compaction both
// read through it, so the two can never disagree about which vectors count.
func EntryVectors(ctx context.Context, db entryVectorReader, generations *SQLiteGenerationServing, scope string) (map[string][][]float64, error) {
	if generations != nil {
		facets, err := generations.ReadActiveVectors(ctx, vectorDatasetFacets)
		if err != nil {
			return nil, err
		}
		if facets.Active {
			summaries, err := generations.ReadActiveVectors(ctx, vectorDatasetSummaries)
			if err != nil {
				return nil, err
			}
			if !summaries.Active || summaries.GenerationID != facets.GenerationID || summaries.RecipeID != facets.RecipeID {
				return nil, fmt.Errorf("Source Ledger active generation changed while reading vectors")
			}
			return generationVectorsByEntry(ctx, db, scope, facets, summaries)
		}
	}
	return legacyEntryVectors(ctx, db, scope)
}

func legacyEntryVectors(ctx context.Context, db entryVectorReader, scope string) (map[string][][]float64, error) {
	// Group by entry in Go. Sorting the UNION would materialize both
	// result sets with their vector blobs in SQLite's temporary B-tree.
	rows, err := db.QueryContext(ctx, `
SELECT ft.entry_id, em.vector_blob, em.vector_json
FROM facet_texts ft JOIN embeddings em ON em.facet_text_id=ft.id JOIN entries e ON e.id=ft.entry_id
WHERE e.scope=?
UNION ALL
SELECT sm.id, sm.vector_blob, sm.vector_json FROM summaries sm WHERE sm.scope=?`, scope, scope)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][][]float64{}
	for rows.Next() {
		var id, raw string
		var blob sql.RawBytes
		if err := rows.Scan(&id, &blob, &raw); err != nil {
			return nil, err
		}
		var vector []float64
		switch {
		case len(blob) > 0:
			vector, err = Decode(blob)
		case raw != "" && raw != "[]":
			err = json.Unmarshal([]byte(raw), &vector)
		default:
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("decode vector for %s: %w", id, err)
		}
		out[id] = append(out[id], vector)
	}
	return out, rows.Err()
}

func generationVectorsByEntry(ctx context.Context, db entryVectorReader, scope string, facets, summaries ActiveGenerationVectors) (map[string][][]float64, error) {
	rows, err := db.QueryContext(ctx, `SELECT ft.id,ft.entry_id FROM facet_texts ft JOIN entries e ON e.id=ft.entry_id WHERE e.scope=? ORDER BY ft.entry_id,ft.id`, scope)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][][]float64{}
	for rows.Next() {
		var sourceID, entryID string
		if err := rows.Scan(&sourceID, &entryID); err != nil {
			return nil, err
		}
		if vector, ok := facets.Vectors[sourceID]; ok {
			out[entryID] = append(out[entryID], vector)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for id, vector := range summaries.Vectors {
		out[id] = append(out[id], vector)
	}
	return out, nil
}
