//go:build notes_fleet_postgres

package notes

import (
	"context"
	"database/sql"
	"net"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// This integration test is opt-in and refuses every database name except the
// dedicated, disposable test prefix. Schema cleanup leaves database-level
// extensions intact, so the caller must remove the dedicated DB afterward. It
// never falls back to the Notes service database.
func TestFleetPostgresBackfillFencesConcurrentWriter(t *testing.T) {
	db := openNotesFleetPostgresTestDB(t)
	db.SetMaxOpenConns(8)

	schema := "notes_fleet_it_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	admin, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.ExecContext(context.Background(), "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = admin.Close()
		_, _ = db.ExecContext(context.Background(), "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
		_ = db.Close()
	})

	setup := scopedNotesFleetConn(t, db, schema)
	defer setup.Close()
	if _, err := setup.ExecContext(context.Background(), Schema()); err != nil {
		t.Fatalf("initial empty Notes schema: %v", err)
	}
	if _, err := setup.ExecContext(context.Background(), `
		DROP TRIGGER notes_fleet_change_trigger ON notes;
		TRUNCATE notes_fleet_changes;
		UPDATE notes_fleet_sequence SET current_sequence = 0 WHERE singleton = TRUE;
		DELETE FROM notes_fleet_backfill WHERE singleton = TRUE;
	`); err != nil {
		t.Fatalf("prepare isolated pre-migration state: %v", err)
	}
	var defaultUser string
	if err := setup.QueryRowContext(context.Background(), `SELECT id::text FROM users WHERE username = 'default'`).Scan(&defaultUser); err != nil {
		t.Fatal(err)
	}
	const firstID = "00000000-0000-0000-0000-000000000010"
	const secondID = "00000000-0000-0000-0000-000000000020"
	if _, err := setup.ExecContext(context.Background(), `
		INSERT INTO notes(id, user_id, title, content) VALUES
		($1, $3, 'preexisting one', 'legacy body one'),
		($2, $3, 'preexisting two', 'legacy body two')`, firstID, secondID, defaultUser); err != nil {
		t.Fatalf("seed isolated legacy rows without fleet trigger: %v", err)
	}
	setup.Close()

	blocker := scopedNotesFleetConn(t, db, schema)
	defer blocker.Close()
	blockTx, err := blocker.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer blockTx.Rollback()
	if _, err := blockTx.ExecContext(context.Background(), `LOCK TABLE notes IN ACCESS EXCLUSIVE MODE`); err != nil {
		t.Fatal(err)
	}

	migrator := scopedNotesFleetConn(t, db, schema)
	defer migrator.Close()
	var migrationPID int
	if err := migrator.QueryRowContext(context.Background(), `SELECT pg_backend_pid()`).Scan(&migrationPID); err != nil {
		t.Fatal(err)
	}
	migrationDone := make(chan error, 1)
	go func() {
		_, err := migrator.ExecContext(context.Background(), Schema())
		migrationDone <- err
	}()
	if !waitForRelationLockState(t, db, migrationPID, schema+".notes", true) {
		t.Fatal("schema migration did not queue on the isolated Notes table lock")
	}

	writer := scopedNotesFleetConn(t, db, schema)
	defer writer.Close()
	var writerPID int
	if err := writer.QueryRowContext(context.Background(), `SELECT pg_backend_pid()`).Scan(&writerPID); err != nil {
		t.Fatal(err)
	}
	writerDone := make(chan error, 1)
	const writerID = "00000000-0000-0000-0000-000000000030"
	go func() {
		_, err := writer.ExecContext(context.Background(), `INSERT INTO notes(id, user_id, title, content) VALUES ($1, $2, 'concurrent writer', 'after baseline')`, writerID, defaultUser)
		writerDone <- err
	}()
	if !waitForRelationLockState(t, db, writerPID, schema+".notes", true) {
		t.Fatal("concurrent writer did not wait behind the queued schema migration")
	}
	if err := blockTx.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-migrationDone:
		if err != nil {
			t.Fatalf("schema backfill migration: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("schema backfill migration timed out")
	}
	select {
	case err := <-writerDone:
		if err != nil {
			t.Fatalf("post-backfill writer: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("concurrent writer did not resume after backfill")
	}

	assertNotesFleetBaseline(t, migrator, firstID, secondID, writerID)
	if _, err := migrator.ExecContext(context.Background(), Schema()); err != nil {
		t.Fatalf("idempotent schema/backfill replay: %v", err)
	}
	assertNotesFleetBaseline(t, migrator, firstID, secondID, writerID)

	sourceDB := openNotesFleetScopedSourceDB(t, os.Getenv("NOTES_FLEET_TEST_DATABASE_URL"), schema)
	defer sourceDB.Close()
	source := FleetSource{DB: sourceDB, Embedder: &recordingEmbedder{vector: []float32{1, 2}, dimensions: 2}}
	snapshot, err := source.PinFleetSnapshot(context.Background(), "postgres-integration-snapshot")
	if err != nil || snapshot.Watermark != "3" {
		t.Fatalf("pin existing-row snapshot=%#v err=%v", snapshot, err)
	}
	points, cursor, done, err := source.ReadFleetSnapshotPage(context.Background(), "recipe-test", snapshot, "", 10)
	if err != nil || !done || len(points) != 3 || cursor == "" {
		t.Fatalf("read PostgreSQL baseline snapshot points=%d cursor=%q done=%v err=%v", len(points), cursor, done, err)
	}
	seenSnapshotNotes := map[string]bool{}
	for _, point := range points {
		id, _ := point.Payload["note_id"].(string)
		seenSnapshotNotes[id] = point.ID == point.Payload["source_id"]
	}
	if !seenSnapshotNotes[firstID] || !seenSnapshotNotes[secondID] || !seenSnapshotNotes[writerID] {
		t.Fatalf("snapshot omitted legacy/concurrent rows or lost canonical source identity: %#v", seenSnapshotNotes)
	}

	newUser := uuid.NewString()
	if _, err := migrator.ExecContext(context.Background(), `INSERT INTO users(id, username, email, password_hash) VALUES ($1, $2, $3, 'test')`, newUser, "fleet-move-"+newUser[:8], newUser+"@test.local"); err != nil {
		t.Fatal(err)
	}
	if _, err := migrator.ExecContext(context.Background(), `UPDATE notes SET user_id = $2 WHERE id = $1`, firstID, newUser); err != nil {
		t.Fatalf("tenant reassignment through PostgreSQL trigger: %v", err)
	}
	var oldSequence, newSequence int64
	var oldOwner, newOwner, deleteGroup, upsertGroup, payloadOwner string
	if err := migrator.QueryRowContext(context.Background(), `SELECT sequence, user_id::text, move_group::text FROM notes_fleet_changes WHERE note_id = $1 AND event_type = 'delete' ORDER BY sequence DESC LIMIT 1`, firstID).Scan(&oldSequence, &oldOwner, &deleteGroup); err != nil {
		t.Fatal(err)
	}
	if err := migrator.QueryRowContext(context.Background(), `SELECT sequence, user_id::text, move_group::text, payload->>'user_id' FROM notes_fleet_changes WHERE note_id = $1 AND event_type = 'upsert' ORDER BY sequence DESC LIMIT 1`, firstID).Scan(&newSequence, &newOwner, &upsertGroup, &payloadOwner); err != nil {
		t.Fatal(err)
	}
	if newSequence != oldSequence+1 || oldOwner != defaultUser || newOwner != newUser || payloadOwner != newUser || deleteGroup == "" || deleteGroup != upsertGroup {
		t.Fatalf("tenant move is not a contiguous, owner-auditable group: seq %d/%d owner %q/%q payload owner %q group %q/%q", oldSequence, newSequence, oldOwner, newOwner, payloadOwner, deleteGroup, upsertGroup)
	}
	changes, next, err := source.ReadFleetChanges(context.Background(), "recipe-test", snapshot.Watermark, snapshot.Watermark, 1)
	if err != nil || len(changes) != 1 || next != "5" || changes[0].Sequence != "5" || len(changes[0].Deletes) != 1 || len(changes[0].Upserts) != 1 {
		t.Fatalf("read PostgreSQL tenant-move group=%#v next=%q err=%v", changes, next, err)
	}
	if changes[0].Deletes[0] != fleetSourceID(defaultUser, firstID) || changes[0].Upserts[0].ID != fleetSourceID(newUser, firstID) || changes[0].DeleteUserID != defaultUser || changes[0].Upserts[0].Payload["user_id"] != newUser {
		t.Fatalf("PostgreSQL tenant-move change lost old/new ownership or canonical identity: %#v", changes[0])
	}
}

func openNotesFleetPostgresTestDB(t *testing.T) *sql.DB {
	t.Helper()
	connectionURL := strings.TrimSpace(os.Getenv("NOTES_FLEET_TEST_DATABASE_URL"))
	if connectionURL == "" {
		t.Skip("set NOTES_FLEET_TEST_DATABASE_URL to a dedicated notes_fleet_test_* database")
	}
	parsed, err := url.Parse(connectionURL)
	if err != nil {
		t.Fatalf("parse Notes fleet test DB URL: %v", err)
	}
	name := strings.TrimPrefix(parsed.Path, "/")
	if !strings.HasPrefix(name, "notes_fleet_test_") {
		t.Fatalf("refusing PostgreSQL integration test against database %q; require notes_fleet_test_*", name)
	}
	host := parsed.Hostname()
	if host != "localhost" {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			t.Fatalf("refusing non-local PostgreSQL test host %q", host)
		}
	}
	for key := range parsed.Query() {
		if key != "sslmode" {
			t.Fatalf("refusing PostgreSQL test URL override %q", key)
		}
	}
	db, err := sql.Open("postgres", connectionURL)
	if err != nil {
		t.Fatal(err)
	}
	db.SetConnMaxLifetime(30 * time.Second)
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("connect to dedicated Notes fleet test database: %v", err)
	}
	return db
}

func scopedNotesFleetConn(t *testing.T, db *sql.DB, schema string) *sql.Conn {
	t.Helper()
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(context.Background(), "SET search_path TO "+schema+", public"); err != nil {
		conn.Close()
		t.Fatal(err)
	}
	return conn
}

func openNotesFleetScopedSourceDB(t *testing.T, connectionURL, schema string) *sql.DB {
	t.Helper()
	parsed, err := url.Parse(connectionURL)
	if err != nil {
		t.Fatal(err)
	}
	// This options value is constructed from our UUID-derived schema after the
	// incoming URL has passed the strict local/disposable database guard.
	query := parsed.Query()
	query.Set("options", "-csearch_path="+schema+",public")
	parsed.RawQuery = query.Encode()
	db, err := sql.Open("postgres", parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(4)
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("connect Notes fleet source to isolated schema: %v", err)
	}
	return db
}

func waitForRelationLockState(t *testing.T, db *sql.DB, pid int, relation string, waiting bool) bool {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var got bool
		err := db.QueryRowContext(context.Background(), `SELECT EXISTS (
			SELECT 1 FROM pg_locks WHERE pid = $1 AND relation = to_regclass($2) AND NOT granted
		)`, pid, relation).Scan(&got)
		if err != nil {
			t.Fatalf("observe isolated PostgreSQL table lock: %v", err)
		}
		if got == waiting {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}

func assertNotesFleetBaseline(t *testing.T, conn *sql.Conn, firstID, secondID, writerID string) {
	t.Helper()
	var seeded, baseline, current, changes int64
	if err := conn.QueryRowContext(context.Background(), `SELECT seeded_notes, baseline_watermark FROM notes_fleet_backfill WHERE singleton = TRUE`).Scan(&seeded, &baseline); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRowContext(context.Background(), `SELECT current_sequence FROM notes_fleet_sequence WHERE singleton = TRUE`).Scan(&current); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM notes_fleet_changes`).Scan(&changes); err != nil {
		t.Fatal(err)
	}
	if seeded != 2 || baseline != 2 || current != 3 || changes != 3 {
		t.Fatalf("baseline marker/ledger mismatch: seeded=%d baseline=%d current=%d events=%d", seeded, baseline, current, changes)
	}
	var firstSequence, secondSequence, writerSequence int64
	if err := conn.QueryRowContext(context.Background(), `SELECT sequence FROM notes_fleet_changes WHERE note_id = $1`, firstID).Scan(&firstSequence); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRowContext(context.Background(), `SELECT sequence FROM notes_fleet_changes WHERE note_id = $1`, secondID).Scan(&secondSequence); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRowContext(context.Background(), `SELECT sequence FROM notes_fleet_changes WHERE note_id = $1`, writerID).Scan(&writerSequence); err != nil {
		t.Fatal(err)
	}
	if firstSequence != 1 || secondSequence != 2 || writerSequence != 3 || writerSequence <= baseline {
		t.Fatalf("concurrent writer was not sequenced after the complete baseline: seeded sequences=%d/%d writer=%d baseline=%d", firstSequence, secondSequence, writerSequence, baseline)
	}
}
