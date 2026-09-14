package storagehealth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"testing"
)

type fakeWatermark struct {
	w        int64
	set      bool
	released bool
}

func (f *fakeWatermark) HoldWatermark() (int64, func(int64), func()) {
	return f.w, func(w int64) { f.w, f.set = w, true }, func() { f.released = true }
}

func createEventTables(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, statement := range []string{
		`CREATE TABLE run_events (id TEXT PRIMARY KEY, run_id TEXT NOT NULL, data BLOB NOT NULL)`,
		`CREATE TABLE stats_checkpoint (name TEXT PRIMARY KEY, last_rowid INTEGER NOT NULL, updated_at TEXT NOT NULL DEFAULT '')`,
		`CREATE TABLE cohort_watches (watch_id TEXT PRIMARY KEY, cursor_rowid INTEGER NOT NULL, revision INTEGER NOT NULL DEFAULT 1)`,
		`CREATE TABLE event_retention_state (singleton INTEGER PRIMARY KEY, generation INTEGER NOT NULL, floor_rowid INTEGER NOT NULL, updated_at TEXT NOT NULL)`,
		`INSERT INTO event_retention_state VALUES (1, 1, 0, '')`,
	} {
		mustExec(t, db, statement)
	}
}

func eventIDsAfter(t *testing.T, db *sql.DB, rowid int64) []string {
	t.Helper()
	rows, err := db.Query(`SELECT id FROM run_events WHERE rowid > ? ORDER BY rowid`, rowid)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids
}

// TestCompactCarriesRowidWatermarksAcrossVacuum is the regression for VACUUM
// renumbering run_events' implicit rowids: after compaction every consumer
// position must select exactly the events it had not yet processed.
func TestCompactCarriesRowidWatermarksAcrossVacuum(t *testing.T) {
	db, path := openFileDB(t, false)
	createEventTables(t, db)
	body := make([]byte, 1024)
	for i := 0; i < 3000; i++ {
		mustExec(t, db, `INSERT INTO run_events (id, run_id, data) VALUES (?, 'run', ?)`, fmt.Sprintf("evt-%05d", i), body)
	}
	// Gaps at the head and throughout, as retention leaves them.
	mustExec(t, db, `DELETE FROM run_events WHERE rowid <= 900 OR rowid % 3 = 0`)
	oldRowid := map[string]int64{}
	rows, err := db.Query(`SELECT id, rowid FROM run_events`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id string
		var rowid int64
		if err := rows.Scan(&id, &rowid); err != nil {
			t.Fatal(err)
		}
		oldRowid[id] = rowid
	}
	rows.Close()
	const statsW, watchW, memoryW = 1801, 2400, 2999 // 2400 was deleted
	mustExec(t, db, `INSERT INTO stats_checkpoint (name, last_rowid) VALUES ('operational', ?)`, statsW)
	mustExec(t, db, `INSERT INTO cohort_watches (watch_id, cursor_rowid) VALUES ('watch-1', ?)`, watchW)
	expected := map[string][]string{"stats": eventIDsAfter(t, db, statsW), "watch": eventIDsAfter(t, db, watchW), "memory": eventIDsAfter(t, db, memoryW)}
	memory := &fakeWatermark{w: memoryW}
	svc := newService(t, db, path, &drainedFence{state: FenceState{Closed: true, Drained: true}}, nil)
	svc.opts.Watermarks = []NamedWatermark{{Name: "stats_engine.watermark", Holder: memory}}

	receipt, err := svc.Compact(t.Context(), "owner", "rowid regression")
	if err != nil {
		t.Fatal(err)
	}

	renumbered := 0
	rows, err = db.Query(`SELECT id, rowid FROM run_events`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id string
		var rowid int64
		if err := rows.Scan(&id, &rowid); err != nil {
			t.Fatal(err)
		}
		if oldRowid[id] != rowid {
			renumbered++
		}
	}
	rows.Close()
	t.Logf("VACUUM renumbered %d of %d run_events rowids; remapped %+v", renumbered, len(oldRowid), receipt.RemappedWatermarks)

	var statsNew, watchNew, floor, minRowid int64
	if err := db.QueryRow(`SELECT last_rowid FROM stats_checkpoint`).Scan(&statsNew); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT cursor_rowid FROM cohort_watches`).Scan(&watchNew); err != nil {
		t.Fatal(err)
	}
	for name, position := range map[string]int64{"stats": statsNew, "watch": watchNew, "memory": memory.w} {
		if got := eventIDsAfter(t, db, position); !slices.Equal(got, expected[name]) {
			t.Fatalf("%s position %d selects %d events, want the %d it had not processed", name, position, len(got), len(expected[name]))
		}
	}
	if !memory.set || !memory.released {
		t.Fatalf("in-memory watermark not rewritten and released: %+v", memory)
	}
	if err := db.QueryRow(`SELECT floor_rowid, (SELECT MIN(rowid) FROM run_events) FROM event_retention_state`).Scan(&floor, &minRowid); err != nil || floor != minRowid {
		t.Fatalf("retention floor=%d min=%d err=%v", floor, minRowid, err)
	}
	if len(receipt.RemappedWatermarks) != 4 {
		t.Fatalf("remapped=%+v, want stats, watch, memory, floor", receipt.RemappedWatermarks)
	}
}

// TestCompactRemapsPositionsWhenVacuumRenumbers forces the documented hazard:
// the modernc build keeps rowids through VACUUM, so the hook renumbers
// run_events densely in rowid order, as a renumbering VACUUM would.
func TestCompactRemapsPositionsWhenVacuumRenumbers(t *testing.T) {
	db, path := openFileDB(t, false)
	createEventTables(t, db)
	for i := 0; i < 2000; i++ {
		mustExec(t, db, `INSERT INTO run_events (id, run_id, data) VALUES (?, 'run', 'x')`, fmt.Sprintf("evt-%05d", i))
	}
	mustExec(t, db, `DELETE FROM run_events WHERE rowid <= 700 OR rowid % 4 = 0`)
	const statsW, watchW, memoryW = 1203, 1600, 1999 // 1600 was deleted
	mustExec(t, db, `INSERT INTO stats_checkpoint (name, last_rowid) VALUES ('operational', ?)`, statsW)
	mustExec(t, db, `INSERT INTO cohort_watches (watch_id, cursor_rowid) VALUES ('watch-1', ?)`, watchW)
	expected := map[string][]string{"stats": eventIDsAfter(t, db, statsW), "watch": eventIDsAfter(t, db, watchW), "memory": eventIDsAfter(t, db, memoryW)}
	memory := &fakeWatermark{w: memoryW}
	svc := newService(t, db, path, &drainedFence{state: FenceState{Closed: true, Drained: true}}, nil)
	svc.opts.Watermarks = []NamedWatermark{{Name: "stats_engine.watermark", Holder: memory}}
	svc.afterVacuum = func(ctx context.Context, conn *sql.Conn) error {
		for _, statement := range []string{
			`CREATE TEMP TABLE renumber AS SELECT id, run_id, data FROM run_events ORDER BY rowid`,
			`DELETE FROM run_events`,
			`INSERT INTO run_events (id, run_id, data) SELECT id, run_id, data FROM renumber ORDER BY rowid`,
			`DROP TABLE renumber`,
		} {
			if _, err := conn.ExecContext(ctx, statement); err != nil {
				return err
			}
		}
		return nil
	}

	receipt, err := svc.Compact(t.Context(), "owner", "renumbering regression")
	if err != nil {
		t.Fatal(err)
	}

	var maxRowid, count int64
	if err := db.QueryRow(`SELECT MAX(rowid), COUNT(*) FROM run_events`).Scan(&maxRowid, &count); err != nil || maxRowid != count {
		t.Fatalf("fixture did not renumber densely: max=%d count=%d err=%v", maxRowid, count, err)
	}
	var statsNew, watchNew int64
	if err := db.QueryRow(`SELECT last_rowid FROM stats_checkpoint`).Scan(&statsNew); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT cursor_rowid FROM cohort_watches`).Scan(&watchNew); err != nil {
		t.Fatal(err)
	}
	for name, position := range map[string]int64{"stats": statsNew, "watch": watchNew, "memory": memory.w} {
		if got := eventIDsAfter(t, db, position); !slices.Equal(got, expected[name]) {
			t.Fatalf("%s position %d selects %d events after renumbering, want %d", name, position, len(got), len(expected[name]))
		}
	}
	// Without the remap the old positions would skip unprocessed events.
	if stale := eventIDsAfter(t, db, statsW); len(stale) >= len(expected["stats"]) {
		t.Fatalf("fixture too weak: the stale stats position still selects %d of %d", len(stale), len(expected["stats"]))
	}
	t.Logf("remapped %+v", receipt.RemappedWatermarks)
}

func TestCompactRefusesWhileTheLegacyCatalogExists(t *testing.T) {
	db, path := openFileDB(t, false)
	fill(t, db, 20, 2)
	mustExec(t, db, `CREATE TABLE conversation_search_documents (document_id TEXT PRIMARY KEY, content TEXT NOT NULL)`)
	pauses := &pauseRecorder{}
	svc := newService(t, db, path, &drainedFence{state: FenceState{Closed: true, Drained: true}}, pauses.pause)

	_, err := svc.Compact(t.Context(), "owner", "test")

	if !errors.Is(err, ErrLegacyProjection) {
		t.Fatalf("err=%v, want ErrLegacyProjection", err)
	}
	if st, _ := svc.Stats(t.Context()); st.AutoVacuum != "none" || len(pauses.events) != 0 {
		t.Fatalf("refused compaction changed state: %s %v", st.AutoVacuum, pauses.events)
	}
}

func createSearchIndex(t *testing.T, db *sql.DB, rows int) {
	t.Helper()
	for _, statement := range []string{
		`CREATE TABLE conversation_search_catalog (id INTEGER PRIMARY KEY, document_id TEXT NOT NULL UNIQUE, content TEXT NOT NULL, visible INTEGER NOT NULL DEFAULT 1)`,
		`CREATE VIRTUAL TABLE conversation_search_catalog_fts USING fts5(document_id UNINDEXED, content, content = 'conversation_search_catalog', content_rowid = 'id')`,
		`CREATE TRIGGER conversation_search_catalog_ai AFTER INSERT ON conversation_search_catalog BEGIN
			INSERT INTO conversation_search_catalog_fts(rowid, document_id, content) VALUES (new.id, new.document_id, new.content); END`,
		`CREATE TRIGGER conversation_search_catalog_ad AFTER DELETE ON conversation_search_catalog BEGIN
			INSERT INTO conversation_search_catalog_fts(conversation_search_catalog_fts, rowid, document_id, content) VALUES ('delete', old.id, old.document_id, old.content); END`,
	} {
		mustExec(t, db, statement)
	}
	for i := 0; i < rows; i++ {
		mustExec(t, db, `INSERT INTO conversation_search_catalog (document_id, content) VALUES (?, ?)`, fmt.Sprintf("doc-%04d", i), fmt.Sprintf("transcript chunk %d mentions token%d", i, i))
	}
	mustExec(t, db, `DELETE FROM conversation_search_catalog WHERE id % 4 = 0`)
}

func TestCompactKeepsTheExplicitIDSearchIndexConsistent(t *testing.T) {
	db, path := openFileDB(t, false)
	createSearchIndex(t, db, 400)
	svc := newService(t, db, path, &drainedFence{state: FenceState{Closed: true, Drained: true}}, nil)

	receipt, err := svc.Compact(t.Context(), "owner", "search regression")
	if err != nil {
		t.Fatal(err)
	}

	if !receipt.SearchBefore.Checked || receipt.SearchAfter.Integrity != "ok" || receipt.SearchAfter.CatalogDocuments != 300 || receipt.SearchAfter.IndexedDocuments != 300 {
		t.Fatalf("search evidence before=%+v after=%+v", receipt.SearchBefore, receipt.SearchAfter)
	}
	var id string
	if err := db.QueryRow(`SELECT c.document_id FROM conversation_search_catalog_fts f JOIN conversation_search_catalog c ON c.id = f.rowid WHERE conversation_search_catalog_fts MATCH 'token122'`).Scan(&id); err != nil || id != "doc-0122" {
		t.Fatalf("search after compaction found %q err=%v", id, err)
	}
}

func TestCompactRefusesAnAlreadyInconsistentSearchIndex(t *testing.T) {
	db, path := openFileDB(t, false)
	createSearchIndex(t, db, 40)
	mustExec(t, db, `DROP TRIGGER conversation_search_catalog_ai`)
	mustExec(t, db, `INSERT INTO conversation_search_catalog (document_id, content) VALUES ('unindexed', 'never reached the index')`)
	svc := newService(t, db, path, &drainedFence{state: FenceState{Closed: true, Drained: true}}, nil)

	_, err := svc.Compact(t.Context(), "owner", "test")

	if !errors.Is(err, ErrSearchInconsistent) {
		t.Fatalf("err=%v, want ErrSearchInconsistent", err)
	}
	if st, _ := svc.Stats(t.Context()); st.AutoVacuum != "none" {
		t.Fatal("compaction ran over an inconsistent index")
	}
}

// TestIncrementalVacuumNeverRenumbersRowids pins why the online reclaim path
// needs no rowid handling: incremental_vacuum moves pages, not rows.
func TestIncrementalVacuumNeverRenumbersRowids(t *testing.T) {
	db, path := openFileDB(t, true)
	createEventTables(t, db)
	body := make([]byte, 2048)
	for i := 0; i < 1500; i++ {
		mustExec(t, db, `INSERT INTO run_events (id, run_id, data) VALUES (?, 'run', ?)`, fmt.Sprintf("evt-%05d", i), body)
	}
	mustExec(t, db, `DELETE FROM run_events WHERE rowid % 2 = 0`)
	before := map[string]int64{}
	rows, _ := db.Query(`SELECT id, rowid FROM run_events`)
	for rows.Next() {
		var id string
		var rowid int64
		_ = rows.Scan(&id, &rowid)
		before[id] = rowid
	}
	rows.Close()
	svc := newService(t, db, path, nil, nil)

	if _, err := svc.Reclaim(context.Background(), false); err != nil {
		t.Fatal(err)
	}

	rows, _ = db.Query(`SELECT id, rowid FROM run_events`)
	defer rows.Close()
	for rows.Next() {
		var id string
		var rowid int64
		_ = rows.Scan(&id, &rowid)
		if before[id] != rowid {
			t.Fatalf("incremental vacuum moved %s from rowid %d to %d", id, before[id], rowid)
		}
	}
}
