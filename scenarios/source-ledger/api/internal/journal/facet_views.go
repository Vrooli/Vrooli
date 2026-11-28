package journal

import (
	"context"
	"database/sql"
	"fmt"
)

// TopicFacetKind is the one derived text an entry is embedded as: its body.
// Entries written before the single-view change also carry "rule" and
// "entities" rows whose text is the body behind a fixed prefix, embedded to
// vectors that measured 0.97-0.99 cosine to the topic vector.
const TopicFacetKind = "topic"

// RedundantFacetViews describes the legacy prefix-only facet rows that a
// prune removes. Every count is measured before anything is deleted.
type RedundantFacetViews struct {
	Entries        int   `json:"entries"`
	FacetTexts     int   `json:"facet_texts"`
	Embeddings     int   `json:"embeddings"`
	EmbeddingBytes int64 `json:"embedding_bytes,omitempty"`
}

const redundantFacetTextsWhere = `ft.kind<>? AND EXISTS (SELECT 1 FROM facet_texts topic WHERE topic.entry_id=ft.entry_id AND topic.kind=?)`

// MeasureRedundantFacetViews reports what PruneRedundantFacetViews would
// remove without changing anything.
func MeasureRedundantFacetViews(ctx context.Context, db *sql.DB) (RedundantFacetViews, error) {
	var out RedundantFacetViews
	err := db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT ft.entry_id), COUNT(*) FROM facet_texts ft WHERE `+redundantFacetTextsWhere, TopicFacetKind, TopicFacetKind).Scan(&out.Entries, &out.FacetTexts)
	if err != nil {
		return out, fmt.Errorf("measure redundant facet texts: %w", err)
	}
	err = db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(length(em.vector_blob)+length(em.vector_json)),0) FROM embeddings em JOIN facet_texts ft ON ft.id=em.facet_text_id WHERE `+redundantFacetTextsWhere, TopicFacetKind, TopicFacetKind).Scan(&out.Embeddings, &out.EmbeddingBytes)
	if err != nil {
		return out, fmt.Errorf("measure redundant embeddings: %w", err)
	}
	return out, nil
}

// PruneRedundantFacetViews deletes the legacy rule and entities facet rows and
// their embeddings. The topic view of the same entry is never touched, so an
// entry keeps its body, its topic text and its topic vector: retrieval reads
// exactly the vectors it preferred before. The rows to remove are selected once,
// then deleted by id in bounded transactions so a live writer is never blocked
// for the whole pass. It is idempotent.
func PruneRedundantFacetViews(ctx context.Context, db *sql.DB, batch int) (RedundantFacetViews, error) {
	if batch <= 0 {
		batch = 500
	}
	rows, err := db.QueryContext(ctx, `SELECT ft.id FROM facet_texts ft WHERE `+redundantFacetTextsWhere+` ORDER BY ft.id`, TopicFacetKind, TopicFacetKind)
	if err != nil {
		return RedundantFacetViews{}, fmt.Errorf("select redundant facet texts: %w", err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return RedundantFacetViews{}, err
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return RedundantFacetViews{}, err
	}
	if err := rows.Err(); err != nil {
		return RedundantFacetViews{}, err
	}
	var removed RedundantFacetViews
	for start := 0; start < len(ids); start += batch {
		end := min(start+batch, len(ids))
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return removed, err
		}
		for _, id := range ids[start:end] {
			emb, err := tx.ExecContext(ctx, `DELETE FROM embeddings WHERE facet_text_id=?`, id)
			if err != nil {
				_ = tx.Rollback()
				return removed, fmt.Errorf("delete redundant embedding: %w", err)
			}
			fts, err := tx.ExecContext(ctx, `DELETE FROM facet_texts WHERE id=?`, id)
			if err != nil {
				_ = tx.Rollback()
				return removed, fmt.Errorf("delete redundant facet text: %w", err)
			}
			e, _ := emb.RowsAffected()
			f, _ := fts.RowsAffected()
			removed.Embeddings += int(e)
			removed.FacetTexts += int(f)
		}
		if err := tx.Commit(); err != nil {
			return removed, err
		}
	}
	return removed, nil
}
