package event_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const compactionMinBytes = 4096

func setupCompactionDB(t *testing.T) (*sqlx.DB, *event.SQLiteStore) {
	t.Helper()
	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)
	if _, err := db.Exec(`CREATE TABLE conversation_search_changes (
		sequence INTEGER PRIMARY KEY AUTOINCREMENT,
		operation TEXT NOT NULL,
		source_run_id TEXT NOT NULL DEFAULT '',
		source_event_id TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		processed_at TEXT)`); err != nil {
		t.Fatal(err)
	}
	return db, event.NewSQLiteStore(db, newTestLogger())
}

func addRun(t *testing.T, db *sqlx.DB, mode string) uuid.UUID {
	t.Helper()
	runID := uuid.New()
	if _, err := db.Exec(`INSERT INTO runs (id, execution_mode) VALUES (?, ?)`, runID.String(), mode); err != nil {
		t.Fatal(err)
	}
	return runID
}

func appendAt(t *testing.T, store *event.SQLiteStore, runID uuid.UUID, at time.Time, evt *domain.RunEvent) *domain.RunEvent {
	t.Helper()
	evt.Timestamp = at
	if err := store.Append(context.Background(), runID, evt); err != nil {
		t.Fatal(err)
	}
	return evt
}

func storedData(t *testing.T, db *sqlx.DB, id uuid.UUID) string {
	t.Helper()
	var data string
	if err := db.Get(&data, `SELECT data FROM run_events WHERE id = ?`, id.String()); err != nil {
		t.Fatal(err)
	}
	return data
}

// bulky builds text with a recognizable head and tail and a multi-byte rune at
// every boundary a naive byte cut could split.
func bulky(size int) string {
	body := strings.Repeat("é中x", size/6+1)
	return "HEAD-MARK " + body[:size] + " TAIL-MARK"
}

func TestCompactImportedToolPayloadsKeepsHeadTailAndChecksum(t *testing.T) {
	db, store := setupCompactionDB(t)
	imported := addRun(t, db, "imported")
	old := time.Now().Add(-40 * 24 * time.Hour)
	output := bulky(30_000)
	result := appendAt(t, store, imported, old, domain.NewToolResultEvent(imported, "Bash", "call-1", output, nil))
	command := "go test ./..."
	call := appendAt(t, store, imported, old, domain.NewToolCallEvent(imported, "Write", "call-2", map[string]interface{}{
		"command": command, "content": bulky(20_000), "nested": map[string]interface{}{"body": bulky(9_000)},
	}))

	compacted, err := store.CompactImportedToolPayloads(context.Background(), time.Now().Add(-30*24*time.Hour), compactionMinBytes, 1000)
	if err != nil || compacted != 2 {
		t.Fatalf("compacted = %d, %v; want 2", compacted, err)
	}

	payload, err := domain.DecodeEventPayload(domain.EventTypeToolResult, []byte(storedData(t, db, result.ID)))
	if err != nil {
		t.Fatalf("compacted tool_result no longer decodes: %v", err)
	}
	got := payload.(*domain.ToolResultEventData).Output
	if !strings.HasPrefix(got, "HEAD-MARK") || !strings.HasSuffix(got, "TAIL-MARK") {
		t.Fatalf("excerpt lost head or tail: %.80q ... %.80q", got, got[len(got)-80:])
	}
	if want := fmt.Sprintf("%d bytes, sha256:%x", len(output), sha256.Sum256([]byte(output))); !strings.Contains(got, want) {
		t.Fatalf("excerpt does not record original size and checksum %q", want)
	}
	if !utf8.ValidString(got) {
		t.Fatal("excerpt split a multi-byte rune")
	}
	if len(got) > 2*compactionMinBytes/4+256 {
		t.Fatalf("excerpt length %d exceeds the bound", len(got))
	}

	callPayload, err := domain.DecodeEventPayload(domain.EventTypeToolCall, []byte(storedData(t, db, call.ID)))
	if err != nil {
		t.Fatalf("compacted tool_call no longer decodes: %v", err)
	}
	input := callPayload.(*domain.ToolCallEventData).Input
	if input["command"] != command {
		t.Fatalf("small parameter was not kept whole: %v", input["command"])
	}
	for _, key := range []string{"content", "nested"} {
		if text, ok := input[key].(string); !ok || !strings.Contains(text, "agent-manager compacted") {
			t.Fatalf("bulky parameter %q was not compacted: %T", key, input[key])
		}
	}

	var queued []struct {
		Operation string `db:"operation"`
		RunID     string `db:"source_run_id"`
		EventID   string `db:"source_event_id"`
	}
	if err := db.Select(&queued, `SELECT operation, source_run_id, source_event_id FROM conversation_search_changes ORDER BY sequence`); err != nil {
		t.Fatal(err)
	}
	if len(queued) != 2 || queued[0].Operation != "upsert_run" || queued[0].RunID != imported.String() || queued[0].EventID != result.ID.String() {
		t.Fatalf("compaction did not queue conversation-search reindex per rewritten event: %+v", queued)
	}
}

func TestCompactImportedToolPayloadsLeavesEverythingElseAlone(t *testing.T) {
	db, store := setupCompactionDB(t)
	imported, native := addRun(t, db, "imported"), addRun(t, db, "codec_pipe")
	old, fresh := time.Now().Add(-40*24*time.Hour), time.Now().Add(-time.Hour)
	untouched := []*domain.RunEvent{
		appendAt(t, store, native, old, domain.NewToolResultEvent(native, "Bash", "native", bulky(30_000), nil)),
		appendAt(t, store, imported, fresh, domain.NewToolResultEvent(imported, "Bash", "fresh", bulky(30_000), nil)),
		appendAt(t, store, imported, old, domain.NewToolResultEvent(imported, "Bash", "small", "ok", nil)),
		appendAt(t, store, imported, old, domain.NewMessageEvent(imported, "assistant", bulky(30_000))),
	}
	before := make([]string, len(untouched))
	for k, evt := range untouched {
		before[k] = storedData(t, db, evt.ID)
	}

	compacted, err := store.CompactImportedToolPayloads(context.Background(), time.Now().Add(-30*24*time.Hour), compactionMinBytes, 1000)
	if err != nil || compacted != 0 {
		t.Fatalf("compacted = %d, %v; want 0", compacted, err)
	}
	for k, evt := range untouched {
		if storedData(t, db, evt.ID) != before[k] {
			t.Fatalf("event %d (%s) changed", k, evt.EventType)
		}
	}
}

func TestCompactImportedToolPayloadsIsIdempotentAndPaced(t *testing.T) {
	db, store := setupCompactionDB(t)
	imported := addRun(t, db, "imported")
	old := time.Now().Add(-40 * 24 * time.Hour)
	for k := range 3 {
		appendAt(t, store, imported, old, domain.NewToolResultEvent(imported, "Bash", fmt.Sprintf("call-%d", k), bulky(10_000), nil))
	}
	cutoff := time.Now().Add(-30 * 24 * time.Hour)

	first, err := store.CompactImportedToolPayloads(context.Background(), cutoff, compactionMinBytes, 2)
	if err != nil || first != 2 {
		t.Fatalf("first bounded batch = %d, %v; want 2", first, err)
	}
	second, err := store.CompactImportedToolPayloads(context.Background(), cutoff, compactionMinBytes, 2)
	if err != nil || second != 1 {
		t.Fatalf("second batch = %d, %v; want the remaining 1", second, err)
	}
	// The pass reached the end of history, so the next pass waits.
	paused, err := store.CompactImportedToolPayloads(context.Background(), cutoff, compactionMinBytes, 2)
	if err != nil || paused != 0 {
		t.Fatalf("paused pass = %d, %v; want 0", paused, err)
	}
	rerun := event.NewSQLiteStore(db, newTestLogger())
	again, err := rerun.CompactImportedToolPayloads(context.Background(), cutoff, compactionMinBytes, 10)
	if err != nil || again != 0 {
		t.Fatalf("a fresh pass recompacted excerpts: %d, %v", again, err)
	}
}

func TestSQLiteStore_DeleteBeforeRemovesExpiredOrphansWithoutProjection(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ensureProjectionWatermarksTable(t, db)
	store := event.NewSQLiteStore(db, newTestLogger())
	ctx := context.Background()
	expiredOrphan, freshOrphan := uuid.New(), uuid.New()
	appendAt(t, store, expiredOrphan, time.Now().Add(-48*time.Hour), domain.NewLogEvent(expiredOrphan, "info", "run row deleted"))
	appendAt(t, store, freshOrphan, time.Now(), domain.NewLogEvent(freshOrphan, "info", "not yet expired"))

	deleted, err := store.DeleteBefore(ctx, time.Now().Add(-24*time.Hour), 10)
	if err != nil || deleted != 1 {
		t.Fatalf("deleted = %d, %v; want the expired orphan", deleted, err)
	}
	if count, _ := store.Count(ctx, expiredOrphan); count != 0 {
		t.Fatalf("expired orphan events = %d, want 0", count)
	}
	if count, _ := store.Count(ctx, freshOrphan); count != 1 {
		t.Fatalf("fresh orphan events = %d, want 1", count)
	}
}
