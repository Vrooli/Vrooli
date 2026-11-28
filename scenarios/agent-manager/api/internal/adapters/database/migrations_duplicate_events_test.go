package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

func insertImportedRun(t *testing.T, db *DB) string {
	t.Helper()
	id := uuid.NewString()
	if _, err := db.Exec(`INSERT INTO runs (id, execution_mode, status) VALUES (?, 'imported', 'complete')`, id); err != nil {
		t.Fatal(err)
	}
	return id
}

func insertEventWithID(t *testing.T, db *DB, id, runID string, sequence int, eventType, timestamp, data string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO run_events (id, run_id, sequence, event_type, timestamp, data) VALUES (?, ?, ?, ?, ?, ?)`,
		id, runID, sequence, eventType, timestamp, data); err != nil {
		t.Fatal(err)
	}
}

func insertEvent(t *testing.T, db *DB, runID string, sequence int, eventType, timestamp, data string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO run_events (id, run_id, sequence, event_type, timestamp, data) VALUES (?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), runID, sequence, eventType, timestamp, data); err != nil {
		t.Fatal(err)
	}
}

func TestDuplicateTranscriptEventsAreDroppedOnceKeepingTheFirstCopy(t *testing.T) {
	db := newMigrationTestDB(t)
	ctx := context.Background()
	copied, other := insertImportedRun(t, db), insertImportedRun(t, db)
	call := func(id string) string { return fmt.Sprintf(`{"toolName":"Bash","toolCallId":%q}`, id) }
	// One run read twice: events 0..3 then the same four again (shift 4), then two
	// identical events no shift explains, which are genuine and stay.
	for pass := range 2 {
		base := pass * 4
		insertEvent(t, db, copied, base, "tool_call", "2026-10-29T04:00:01Z", call("t1"))
		insertEvent(t, db, copied, base+1, "tool_result", "2026-10-29T04:00:02Z", `{"toolCallId":"t1","output":"ok"}`)
		insertEvent(t, db, copied, base+2, "metric", "2026-10-29T04:00:03Z", `{"payloadKind":"usage","inputTokens":5}`)
		insertEvent(t, db, copied, base+3, "tool_call", "2026-10-29T04:00:04Z", call("t2"))
	}
	insertEvent(t, db, copied, 8, "log", "2026-10-29T04:00:09Z", `{"message":"same"}`)
	insertEvent(t, db, copied, 9, "log", "2026-10-29T04:00:09Z", `{"message":"same"}`)
	// A tool call id reused later at another time is not a copy.
	insertEvent(t, db, other, 0, "tool_call", "2026-10-29T04:00:01Z", call("t1"))
	insertEvent(t, db, other, 1, "tool_call", "2026-10-29T05:00:01Z", call("t1"))
	if got := countWhere(t, db, `SELECT COUNT(*) FROM schema_migrations_done WHERE name = ?`, migrationDuplicateTranscriptEvents); got != 0 {
		t.Fatal("schema initialisation ran the repair; it must not")
	}

	if err := db.RepairDuplicateTranscriptEvents(ctx); err != nil {
		t.Fatal(err)
	}
	if got := countWhere(t, db, `SELECT COUNT(*) FROM run_events WHERE run_id = ?`, copied); got != 5 {
		t.Fatalf("events left in the re-read run = %d, want 4 first-copy events and 1 log (legacy twins collapse)", got)
	}
	if got := countWhere(t, db, `SELECT COUNT(*) FROM run_events WHERE run_id = ? AND sequence >= 4 AND sequence < 8`, copied); got != 0 {
		t.Fatalf("later copy remains: %d events", got)
	}
	if got := countWhere(t, db, `SELECT COUNT(*) FROM run_events WHERE run_id = ?`, other); got != 2 {
		t.Fatalf("reused tool call id lost events: %d, want 2", got)
	}
	if got := countWhere(t, db, `SELECT COUNT(*) FROM invocation_read_model_pending WHERE run_id = ?`, copied); got != 1 {
		t.Fatalf("repaired run queued for re-projection = %d, want 1", got)
	}

	if got := countWhere(t, db, `SELECT COUNT(*) FROM conversation_search_changes WHERE operation = 'upsert_run' AND source_run_id = ? AND processed_at IS NULL`, copied); got < 1 {
		t.Fatalf("repaired run queued for conversation-search reindex = %d, want at least 1", got)
	}

	// The marker makes a later start a no-op.
	insertEvent(t, db, copied, 10, "tool_call", "2026-10-29T04:00:01Z", call("t1"))
	insertEvent(t, db, copied, 11, "tool_call", "2026-10-29T04:00:01Z", call("t1"))
	if err := db.RepairDuplicateTranscriptEvents(ctx); err != nil {
		t.Fatal(err)
	}
	if got := countWhere(t, db, `SELECT COUNT(*) FROM run_events WHERE run_id = ?`, copied); got != 7 {
		t.Fatalf("migration ran twice: %d events", got)
	}
}

// A one-shot repair must not need a quiet database: while another connection
// holds the write lock the run's transaction backs off, runs it could not
// repair are retried by the next pass, and the marker lands only when every
// run is done.
func TestDuplicateTranscriptRepairOutlastsAContendedDatabase(t *testing.T) {
	db := newMigrationTestDB(t)
	ctx := context.Background()
	prev := [3]any{duplicateRepairBusyAttempts, duplicateRepairBusyBackoff, duplicateRepairPassInterval}
	duplicateRepairBusyAttempts, duplicateRepairBusyBackoff, duplicateRepairPassInterval = 2, 20*time.Millisecond, 100*time.Millisecond
	t.Cleanup(func() {
		duplicateRepairBusyAttempts, duplicateRepairBusyBackoff, duplicateRepairPassInterval = prev[0].(int), prev[1].(time.Duration), prev[2].(time.Duration)
	})
	if _, err := db.Exec(`PRAGMA busy_timeout = 30`); err != nil { // the pool holds one connection
		t.Fatal(err)
	}
	run := insertImportedRun(t, db)
	for pass := range 2 {
		insertEvent(t, db, run, pass*2, "tool_call", "2026-10-29T04:00:01Z", `{"toolName":"Bash","toolCallId":"t1"}`)
		insertEvent(t, db, run, pass*2+1, "tool_result", "2026-10-29T04:00:02Z", `{"toolCallId":"t1","output":"ok"}`)
	}

	dsn, err := SQLiteDSN(logrus.New())
	if err != nil {
		t.Fatal(err)
	}
	holder, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Close()
	if _, err := holder.Exec(`BEGIN IMMEDIATE`); err != nil {
		t.Fatal(err)
	}
	released := make(chan struct{})
	go func() {
		defer close(released)
		time.Sleep(600 * time.Millisecond)
		_, _ = holder.Exec(`COMMIT`)
	}()

	if err := db.RepairDuplicateTranscriptEvents(ctx); err != nil {
		t.Fatal(err)
	}
	<-released
	if got := countWhere(t, db, `SELECT COUNT(*) FROM run_events WHERE run_id = ?`, run); got != 2 {
		t.Fatalf("events left = %d, want the 2 first-copy events", got)
	}
	if got := countWhere(t, db, `SELECT COUNT(*) FROM schema_migrations_done WHERE name = ?`, migrationDuplicateTranscriptEvents); got != 1 {
		t.Fatalf("marker rows = %d, want 1 once every run is repaired", got)
	}
}

// Copies with no repeated tool call are invisible to the shift rule: usage and
// charge rows doubled by a re-read, and v5 rows re-read over legacy v4 ones.
// The later copy goes when either side predates deterministic ids; two v5 rows
// with the same content are distinct source lines and stay.
func TestLegacyTwinEventsAreDroppedButDeterministicTwinsStay(t *testing.T) {
	db := newMigrationTestDB(t)
	run := insertImportedRun(t, db)
	const usage = `{"payloadKind":"usage","inputTokens":9037}`
	const charge = `{"payloadKind":"charge","basis":"unknown"}`
	const at = "2026-10-29T14:19:37.247Z"
	v4 := func() string { return uuid.NewString() }
	v5 := func(name string) string { return uuid.NewSHA1(uuid.Nil, []byte(name)).String() }
	insertEventWithID(t, db, v4(), run, 0, "metric", at, usage)
	insertEventWithID(t, db, v4(), run, 1, "metric", at, charge)
	insertEventWithID(t, db, v4(), run, 2, "metric", at, usage)     // v4 re-read over v4
	insertEventWithID(t, db, v5("a"), run, 3, "metric", at, charge) // v5 re-read over v4
	insertEventWithID(t, db, v5("b"), run, 4, "log", at, `{"message":"twin"}`)
	insertEventWithID(t, db, v5("c"), run, 5, "log", at, `{"message":"twin"}`) // two source lines

	if err := db.RepairDuplicateTranscriptEvents(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := countWhere(t, db, `SELECT COUNT(*) FROM run_events WHERE run_id = ?`, run); got != 4 {
		t.Fatalf("events left = %d, want usage, charge and the two v5 twins", got)
	}
	if got := countWhere(t, db, `SELECT COUNT(*) FROM run_events WHERE run_id = ? AND sequence IN (2, 3)`, run); got != 0 {
		t.Fatalf("legacy twin copies remain: %d", got)
	}
	if got := countWhere(t, db, `SELECT COUNT(*) FROM schema_migrations_done WHERE name = ?`, migrationLegacyTwinEvents); got != 1 {
		t.Fatalf("marker rows = %d, want 1", got)
	}
}
