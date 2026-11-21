package conversationsearch

import (
	"context"
	"crypto/sha256"
	"fmt"
)

// catalogDigestPageSize bounds each keyset read of the serving catalog so the
// comparison never holds SQLite's single connection across the whole corpus.
const catalogDigestPageSize = 5000

// setDigest summarizes a document set independent of traversal order: the
// source walks events by time while the catalog is read through an index, so
// an ordered hash could never compare the two.
type setDigest struct {
	count uint64
	sum   [sha256.Size]byte
}

func (d *setDigest) add(documentID, contentHash string) {
	h := sha256.Sum256([]byte(documentID + "\x00" + contentHash))
	for k := range d.sum {
		d.sum[k] ^= h[k]
	}
	d.count++
}

func (d setDigest) equal(other setDigest) bool {
	return d.count == other.count && d.sum == other.sum
}

// checkDrift compares the authoritative source with the serving catalog using
// reads only. A full rebuild runs only when they disagree while no change was
// queued across the comparison, or when the serving projection is degraded.
// It replaced an unconditional full rebuild every repair tick, which rewrote
// ~986k staged documents and their indexes every 15 minutes (~3 TB/day of
// disk writes measured 2026-09-14) even with nothing to repair.
func (i *Indexer) checkDrift(ctx context.Context) {
	if i.busy() {
		return
	}
	pending, err := i.repository.PendingChanges(ctx, 1)
	if err != nil {
		return
	}
	if len(pending) > 0 {
		i.launchChanges(ctx)
		return
	}
	degraded, err := i.repository.ServingDegraded(ctx)
	if err != nil {
		return
	}
	if degraded {
		i.launchRepair(ctx)
		return
	}
	before, err := i.repository.LatestChangeSequence(ctx)
	if err != nil {
		return
	}
	var source setDigest
	if _, _, _, err := i.scan(ctx, 0, func(document Document) error {
		source.add(document.DocumentID, document.ContentHash)
		return nil
	}); err != nil {
		return
	}
	catalog, err := i.repository.VisibleCatalogDigest(ctx)
	if err != nil {
		return
	}
	// A change queued or applied during the walk makes the two sides describe
	// different moments; the next check compares them again.
	after, err := i.repository.LatestChangeSequence(ctx)
	if err != nil || after != before || i.busy() || source.equal(catalog) {
		return
	}
	i.launchRepair(ctx)
}

func (i *Indexer) busy() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.running || i.recoveryPending
}

// VisibleCatalogDigest reads every served (document_id, content_hash) pair in
// bounded id-keyed pages. The comparison runs about once a day, so it reads the
// table rather than paying for a dedicated covering index on every write.
func (r *SQLiteRepository) VisibleCatalogDigest(ctx context.Context) (setDigest, error) {
	var digest setDigest
	var lastID int64
	for {
		var rows []struct {
			ID          int64  `db:"id"`
			DocumentID  string `db:"document_id"`
			ContentHash string `db:"content_hash"`
			Visible     int    `db:"visible"`
		}
		if err := r.db.SelectContext(ctx, &rows, `SELECT id, document_id, content_hash, visible FROM conversation_search_catalog
WHERE id > ? ORDER BY id LIMIT ?`, lastID, catalogDigestPageSize); err != nil {
			return setDigest{}, fmt.Errorf("digest conversation search catalog: %w", err)
		}
		for _, row := range rows {
			if row.Visible == 1 {
				digest.add(row.DocumentID, row.ContentHash)
			}
		}
		if len(rows) < catalogDigestPageSize {
			return digest, nil
		}
		lastID = rows[len(rows)-1].ID
	}
}

// LatestChangeSequence is the newest queued change, processed or not. It moves
// whenever a canonical mutation is recorded, so equal readings around a walk
// prove no change landed during it.
func (r *SQLiteRepository) LatestChangeSequence(ctx context.Context) (int64, error) {
	var sequence int64
	err := r.db.GetContext(ctx, &sequence, `SELECT COALESCE(MAX(sequence), 0) FROM conversation_search_changes`)
	return sequence, err
}

// ServingDegraded reports whether search is serving without its optional
// semantic leg: an active generation promoted lexically after a semantic
// failure, or a newest incremental pass whose semantic leg failed. Only a full
// rebuild restores that leg.
func (r *SQLiteRepository) ServingDegraded(ctx context.Context) (bool, error) {
	var degraded bool
	err := r.db.GetContext(ctx, &degraded, `SELECT
EXISTS(SELECT 1 FROM conversation_search_generations WHERE state = 'active' AND failed_documents > 0)
OR EXISTS(SELECT 1 FROM (SELECT state, failed_documents FROM conversation_search_generations
  ORDER BY created_at DESC LIMIT 1) WHERE state = 'failed' AND failed_documents > 0)`)
	return degraded, err
}

// ActiveGenerationRecipe returns the recipe the serving generation was built
// with, or "" when nothing serves.
func (r *SQLiteRepository) ActiveGenerationRecipe(ctx context.Context) (string, error) {
	var recipes []string
	if err := r.db.SelectContext(ctx, &recipes, `SELECT recipe_version FROM conversation_search_generations WHERE state = 'active' LIMIT 1`); err != nil {
		return "", err
	}
	if len(recipes) == 0 {
		return "", nil
	}
	return recipes[0], nil
}
