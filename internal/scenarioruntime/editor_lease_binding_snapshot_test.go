package scenarioruntime

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

func TestGetEditorLeaseBindingsPreservesStoppedCanonicalAndReadOnlyLookup(t *testing.T) {
	ctx := context.Background()
	store, _ := newEditorLeaseStore(t)
	stored := make(map[string]EditorLease)
	for _, id := range []string{"host-session", "am-run", "expired-session", "unrelated-history"} {
		lease, err := store.CreateEditorLease(ctx, EditorLease{
			SessionID: id, Harness: "claude-code", Agent: "claude", PID: 123,
			HostBootID: "boot", Scope: "vrooli-agent-lease.scope", WorkingDir: "/repo", Claims: []string{"internal/"},
			Owner: EditorLeaseOwner{Kind: "terminal", Ref: "tty:/dev/pts/1", Source: "launch", EvidenceJSON: `{"pid":123}`},
		}, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		stored[id] = lease
	}
	stopped, err := store.StopEditorLease(ctx, "host-session", "session ended")
	if err != nil {
		t.Fatal(err)
	}
	stored["host-session"] = stopped
	if _, err := store.db.ExecContext(ctx, `UPDATE runtime_editor_leases SET status = ? WHERE session_id = ?`, EditorLeaseExpired, "expired-session"); err != nil {
		t.Fatal(err)
	}
	expired := stored["expired-session"]
	expired.Status = EditorLeaseExpired
	stored["expired-session"] = expired
	// An unrelated malformed historical row must not be enumerated or parsed.
	if _, err := store.db.ExecContext(ctx, `UPDATE runtime_editor_leases SET created_at = 'invalid-history-time' WHERE session_id = 'unrelated-history'`); err != nil {
		t.Fatal(err)
	}
	readOnly, err := NewSQLiteStore(ctx, Config{DBPath: store.path, ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer readOnly.Close()
	got, err := readOnly.GetEditorLeaseBindings(ctx, []string{"host-session", "am-run", "expired-session", "truly-missing", " am-run ", "host-session"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("bindings = %+v, want exactly the stopped canonical, active legacy, and expired lease", got)
	}
	if padded, err := readOnly.GetEditorLeaseBindings(ctx, []string{" am-run "}); err != nil || len(padded) != 0 {
		t.Fatalf("exact key was normalized: bindings=%+v err=%v", padded, err)
	}
	byID := make(map[string]EditorLease, len(got))
	for _, lease := range got {
		want, exists := stored[lease.SessionID]
		if !exists || lease.SessionID == "unrelated-history" || !EditorLeaseBindingEqual(lease, want) {
			t.Fatalf("unexpected or changed binding: %+v", lease)
		}
		if lease.WorkingDir != "" || len(lease.Claims) != 0 || !lease.LastHeartbeatAt.IsZero() || !lease.HeartbeatDeadlineAt.IsZero() || lease.StoppedAt != nil || lease.Owner.RecordedAt != nil {
			t.Fatalf("binding-only projection included unrelated editor metadata: %+v", lease)
		}
		byID[lease.SessionID] = lease
	}
	if byID["host-session"].Status != EditorLeaseStopped || byID["am-run"].Status != EditorLeaseActive || byID["expired-session"].Status != EditorLeaseExpired {
		t.Fatalf("all-status snapshot lost state: %+v", byID)
	}
	resolved, err := ResolveAttachedEditorLease(ctx, attachedEditorLeaseLookupFunc(func(_ context.Context, key string) (EditorLease, error) {
		lease, exists := byID[key]
		if !exists {
			return EditorLease{}, ErrNotFound
		}
		return lease, nil
	}), "am-run", "host-session")
	if err != nil || resolved.SessionID != "host-session" || resolved.Status != EditorLeaseStopped {
		t.Fatalf("snapshot borrowed active legacy for stopped canonical: lease=%+v err=%v", resolved, err)
	}
	var rowCount int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM runtime_editor_leases`).Scan(&rowCount); err != nil || rowCount != 4 {
		t.Fatalf("read-only lookup changed stored rows: count=%d err=%v", rowCount, err)
	}
}

func TestGetEditorLeaseBindingsCompletesMultipleBatchesWithoutDuplicates(t *testing.T) {
	ctx := context.Background()
	store, _ := newEditorLeaseStore(t)
	const total = 2*512 + 17
	keys := make([]string, 0, total)
	if err := store.withTx(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `INSERT INTO runtime_editor_leases
(session_id, binding_id, status, created_at, last_heartbeat_at, heartbeat_deadline_at)
VALUES (?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return err
		}
		defer stmt.Close()
		for i := 0; i < total; i++ {
			id := fmt.Sprintf("session-%04d", i)
			keys = append(keys, id)
			status := EditorLeaseActive
			if i%3 == 1 {
				status = EditorLeaseStopped
			} else if i%3 == 2 {
				status = EditorLeaseExpired
			}
			at := "2026-11-09T12:00:00Z"
			if _, err := stmt.ExecContext(ctx, id, "binding-"+id, status, at, at, at); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	// Enforce the promised SQL parameter bound in the actual SQLite engine,
	// so a regression to one large IN query cannot pass on a permissive build.
	conn, err := store.db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sqlite.Limit(conn, sqlite3.SQLITE_LIMIT_VARIABLE_NUMBER, 512); err != nil {
		_ = conn.Close()
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	// Repeat keys on either side of a query boundary, and include real absence.
	keys = append(keys, keys[0], keys[512], keys[len(keys)-1], "absent")
	got, err := store.GetEditorLeaseBindings(ctx, keys)
	if err != nil || len(got) != total {
		t.Fatalf("batch returned %d leases, want %d; err=%v", len(got), total, err)
	}
	byID := make(map[string]EditorLease, len(got))
	for _, lease := range got {
		if _, duplicate := byID[lease.SessionID]; duplicate {
			t.Fatalf("duplicate result for %q", lease.SessionID)
		}
		byID[lease.SessionID] = lease
	}
	for i, key := range keys[:total] {
		lease, exists := byID[key]
		wantStatus := EditorLeaseActive
		if i%3 == 1 {
			wantStatus = EditorLeaseStopped
		} else if i%3 == 2 {
			wantStatus = EditorLeaseExpired
		}
		if !exists || lease.Status != wantStatus || lease.PID != 0 || lease.BindingID != "binding-"+key {
			t.Fatalf("missing or incorrect binding for %q: %+v", key, lease)
		}
	}
	// An error in a later batch must not return a partial snapshot that could
	// make unobserved canonical rows appear absent to a caller.
	if _, err := store.db.ExecContext(ctx, `UPDATE runtime_editor_leases SET created_at = 'invalid-binding-time' WHERE session_id = ?`, keys[total-1]); err != nil {
		t.Fatal(err)
	}
	if got, err := store.GetEditorLeaseBindings(ctx, keys); err == nil || got != nil {
		t.Fatalf("later scan error returned partial snapshot: len=%d err=%v", len(got), err)
	}
}

func TestGetEditorLeaseBindingsBoundsInputsBeforeDeduplication(t *testing.T) {
	ctx := context.Background()
	store, _ := newEditorLeaseStore(t)
	keys := make([]string, 131072)
	for i := range keys {
		keys[i] = "same-missing-key"
	}
	if got, err := store.GetEditorLeaseBindings(ctx, keys); err != nil || len(got) != 0 {
		t.Fatalf("maximum input must be accepted and deduplicated: got=%+v err=%v", got, err)
	}
	keys = append(keys, "same-missing-key")
	if got, err := store.GetEditorLeaseBindings(ctx, keys); err == nil || !strings.Contains(err.Error(), "131072") || got != nil {
		t.Fatalf("excess input must fail before deduplication: got=%+v err=%v", got, err)
	}
	for _, keys := range [][]string{{""}, {"valid", " \t"}} {
		if got, err := store.GetEditorLeaseBindings(ctx, keys); err == nil || got != nil {
			t.Fatalf("blank input accepted: keys=%q got=%+v err=%v", keys, got, err)
		}
	}
	if got, err := store.GetEditorLeaseBindings(ctx, nil); err != nil || !reflect.DeepEqual(got, []EditorLease{}) {
		t.Fatalf("empty input = %+v, %v; want empty binding result", got, err)
	}
}

func TestGetEditorLeaseBindingsHonorsCancellationAndUnavailableStore(t *testing.T) {
	store, _ := newEditorLeaseStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, keys := range [][]string{nil, {"session"}} {
		if got, err := store.GetEditorLeaseBindings(ctx, keys); !errors.Is(err, context.Canceled) || got != nil {
			t.Fatalf("canceled read = %+v, %v; want context cancellation", got, err)
		}
	}
	// Occupy the store's single connection so cancellation occurs inside a
	// pending database read, rather than merely during the method's preflight.
	conn, err := store.db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	deadline, stop := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer stop()
	got, err := store.GetEditorLeaseBindings(deadline, []string{"session"})
	if !errors.Is(err, context.DeadlineExceeded) || got != nil {
		t.Fatalf("blocked read ignored deadline: got=%+v err=%v", got, err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if got, err := store.GetEditorLeaseBindings(context.Background(), []string{"session"}); err == nil || got != nil {
		t.Fatalf("unavailable store looked like an empty snapshot: got=%+v err=%v", got, err)
	}
}

func TestGetEditorLeaseBindingsPreservesAttemptIDReadOnly(t *testing.T) {
	store, _ := newEditorLeaseStore(t)
	ctx := context.Background()
	before := beginProvenance(t, store, "first")
	reader, err := NewSQLiteStore(ctx, Config{DBPath: store.path, ReadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	snapshot, err := reader.GetEditorLeaseBindings(ctx, []string{"session", "session", " session ", "missing"})
	if err != nil || len(snapshot) != 1 || snapshot[0].BindingID != before.BindingID {
		t.Fatalf("snapshot=%+v error=%v", snapshot, err)
	}
	beginProvenance(t, store, "second")
	after, err := reader.GetEditorLeaseBindings(ctx, []string{"session"})
	if err != nil || len(after) != 1 || after[0].BindingID != "second" {
		t.Fatalf("rebound snapshot=%+v error=%v", after, err)
	}
	if EditorLeaseBindingEqual(snapshot[0], after[0]) {
		t.Fatal("bounded read hid same-process attempt change")
	}
}
