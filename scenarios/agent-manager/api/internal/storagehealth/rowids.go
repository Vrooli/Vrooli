package storagehealth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// SQLite documents that VACUUM may renumber the implicit rowid of any table
// without an INTEGER PRIMARY KEY; incremental_vacuum only moves pages and never
// changes a rowid. With the driver Agent Manager ships (modernc sqlite
// v1.50.1) neither VACUUM nor VACUUM INTO renumbered the production run_events
// shape or a CHECK-constrained table on 2026-09-14, but that is an
// implementation detail, not a guarantee, so every dependent is handled here.
// Two things in Agent Manager depend on such rowids:
//
//   - The legacy conversation_search_documents catalog fed an FTS index keyed by
//     its implicit rowid. A VACUUM INTO copy pointed 585,642 of 986,209 FTS
//     entries at the wrong document (2026-09-14). Compaction refuses while that
//     table exists; its replacement is keyed by an explicit id.
//   - run_events is keyed by a TEXT id, yet consumers persist positions over its
//     rowid (stats_checkpoint, cohort_watches) or hold one in memory (the stats
//     engine). Compaction holds every consumer, anchors each position to stable
//     event ids before VACUUM, and rewrites it from those ids afterwards.
//
// Positions that need nothing: the invocation read model keys its watermark by
// event id; the conversation source snapshot and the imported-payload
// compaction cursor live only in memory inside work that compaction pauses
// (the indexer refuses while a generation builds; the payload cursor wraps to
// zero once it passes the end of the table).

const (
	legacyProjectionTable = "conversation_search_documents"
	searchCatalogTable    = "conversation_search_catalog"
	searchIndexTable      = "conversation_search_catalog_fts"
	// anchorDepth event ids are kept per position so one row deleted between
	// the capture and VACUUM cannot strand it.
	anchorDepth = 16
)

var (
	ErrLegacyProjection   = errors.New("storage compaction is refused while the legacy conversation_search_documents catalog exists: VACUUM may renumber its implicit rowids and corrupt the FTS index keyed by them; finish the conversation-search projection upgrade first")
	ErrSearchInconsistent = errors.New("conversation search index is inconsistent with its catalog")
)

// WatermarkHolder is an in-memory consumer position over run_events.rowid.
// HoldWatermark blocks the consumer and returns its position, a setter, and
// the release that unblocks it.
type WatermarkHolder interface {
	HoldWatermark() (current int64, set func(int64), release func())
}

// NamedWatermark labels a holder in the compaction receipt.
type NamedWatermark struct {
	Name   string
	Holder WatermarkHolder
}

// RemappedWatermark records one position carried across a VACUUM.
type RemappedWatermark struct {
	Name   string `json:"name"`
	Before int64  `json:"before"`
	After  int64  `json:"after"`
}

// SearchCheck is the conversation-search consistency evidence around a VACUUM.
type SearchCheck struct {
	Checked          bool   `json:"checked"`
	CatalogDocuments int64  `json:"catalogDocuments"`
	IndexedDocuments int64  `json:"indexedDocuments"`
	Integrity        string `json:"integrity"`
}

func (c SearchCheck) consistent() bool {
	return !c.Checked || (c.Integrity == "ok" && c.CatalogDocuments == c.IndexedDocuments)
}

type queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func tableExists(ctx context.Context, db queryer, name string) (bool, error) {
	var count int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type IN ('table','view') AND name = ?`, name).Scan(&count)
	return count > 0, err
}

// persistedRowidWatermarks are the durable positions over run_events.rowid.
var persistedRowidWatermarks = []struct{ table, key, column string }{
	{"stats_checkpoint", "name", "last_rowid"},
	{"cohort_watches", "watch_id", "cursor_rowid"},
}

type rowidAnchor struct {
	name              string
	old               int64
	ids               []string
	table, key, value string
	column            string
	set               func(int64)
}

func anchorIDs(ctx context.Context, conn *sql.Conn, rowid int64) ([]string, error) {
	if rowid <= 0 {
		return nil, nil
	}
	rows, err := conn.QueryContext(ctx, `SELECT id FROM run_events WHERE rowid <= ? ORDER BY rowid DESC LIMIT ?`, rowid, anchorDepth)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

type heldWatermark struct {
	name    string
	current int64
	set     func(int64)
}

// captureAnchors runs after every consumer is held and before VACUUM.
func captureAnchors(ctx context.Context, conn *sql.Conn, held []heldWatermark) ([]rowidAnchor, error) {
	if ok, err := tableExists(ctx, conn, "run_events"); err != nil || !ok {
		return nil, err
	}
	var anchors []rowidAnchor
	for _, spec := range persistedRowidWatermarks {
		ok, err := tableExists(ctx, conn, spec.table)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		rows, err := conn.QueryContext(ctx, fmt.Sprintf(`SELECT %s, %s FROM %s`, spec.key, spec.column, spec.table))
		if err != nil {
			return nil, fmt.Errorf("read %s.%s: %w", spec.table, spec.column, err)
		}
		type position struct {
			key string
			old int64
		}
		var positions []position
		for rows.Next() {
			var p position
			if err := rows.Scan(&p.key, &p.old); err != nil {
				rows.Close()
				return nil, err
			}
			positions = append(positions, p)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
		for _, p := range positions {
			ids, err := anchorIDs(ctx, conn, p.old)
			if err != nil {
				return nil, err
			}
			anchors = append(anchors, rowidAnchor{name: fmt.Sprintf("%s.%s[%s]", spec.table, spec.column, p.key), old: p.old, ids: ids, table: spec.table, key: spec.key, value: p.key, column: spec.column})
		}
	}
	for _, h := range held {
		ids, err := anchorIDs(ctx, conn, h.current)
		if err != nil {
			return nil, err
		}
		anchors = append(anchors, rowidAnchor{name: h.name, old: h.current, ids: ids, set: h.set})
	}
	return anchors, nil
}

// remapAnchors runs right after VACUUM, before any consumer is released. A
// position becomes the new rowid of its newest surviving anchor, which is
// exact because VACUUM keeps rows in rowid order. No anchor at all means no
// event at or below the old position existed, so the new position is zero.
func remapAnchors(ctx context.Context, conn *sql.Conn, anchors []rowidAnchor) ([]RemappedWatermark, error) {
	if ok, err := tableExists(ctx, conn, "run_events"); err != nil || !ok {
		return nil, err
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	remapped := make([]RemappedWatermark, 0, len(anchors)+1)
	var setters []func()
	for _, anchor := range anchors {
		next, found := int64(0), len(anchor.ids) == 0
		for _, id := range anchor.ids {
			err := tx.QueryRowContext(ctx, `SELECT rowid FROM run_events WHERE id = ?`, id).Scan(&next)
			if err == nil {
				found = true
				break
			}
			if !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}
		}
		if !found {
			return nil, fmt.Errorf("every anchor event for %s was deleted during compaction", anchor.name)
		}
		if anchor.set != nil {
			set := anchor.set
			value := next
			setters = append(setters, func() { set(value) })
		} else if _, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET %s = ? WHERE %s = ?`, anchor.table, anchor.column, anchor.key), next, anchor.value); err != nil {
			return nil, fmt.Errorf("rewrite %s: %w", anchor.name, err)
		}
		remapped = append(remapped, RemappedWatermark{Name: anchor.name, Before: anchor.old, After: next})
	}
	if ok, err := tableExists(ctx, tx, "event_retention_state"); err != nil {
		return nil, err
	} else if ok {
		var before, after int64
		_ = tx.QueryRowContext(ctx, `SELECT floor_rowid FROM event_retention_state WHERE singleton = 1`).Scan(&before)
		if _, err := tx.ExecContext(ctx, `UPDATE event_retention_state SET floor_rowid = COALESCE((SELECT MIN(rowid) FROM run_events), 0) WHERE singleton = 1`); err != nil {
			return nil, fmt.Errorf("recompute event retention floor: %w", err)
		}
		_ = tx.QueryRowContext(ctx, `SELECT floor_rowid FROM event_retention_state WHERE singleton = 1`).Scan(&after)
		remapped = append(remapped, RemappedWatermark{Name: "event_retention_state.floor_rowid", Before: before, After: after})
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	for _, set := range setters {
		set()
	}
	return remapped, nil
}

// searchConsistency checks the conversation-search index against its catalog:
// the FTS5 integrity-check plus row parity (the index mirrors every row).
func searchConsistency(ctx context.Context, conn *sql.Conn) (SearchCheck, error) {
	ok, err := tableExists(ctx, conn, searchIndexTable)
	if err != nil || !ok {
		return SearchCheck{}, err
	}
	check := SearchCheck{Checked: true}
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+searchCatalogTable).Scan(&check.CatalogDocuments); err != nil {
		return check, fmt.Errorf("count search catalog: %w", err)
	}
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+searchIndexTable+`_docsize`).Scan(&check.IndexedDocuments); err != nil {
		return check, fmt.Errorf("count search index: %w", err)
	}
	check.Integrity = "ok"
	if _, err := conn.ExecContext(ctx, `INSERT INTO `+searchIndexTable+`(`+searchIndexTable+`) VALUES('integrity-check')`); err != nil {
		check.Integrity = err.Error()
	}
	return check, nil
}
