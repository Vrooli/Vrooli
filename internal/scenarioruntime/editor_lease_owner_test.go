package scenarioruntime

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/testenv"
)

// The owner a launcher records travels with the lease, and a session that
// attaches twice keeps the owner it was born with when the second attach
// carries none.
func TestEditorLeaseRecordsItsOwnerAtLaunch(t *testing.T) {
	ctx := context.Background()
	store, _ := newEditorLeaseStore(t)
	owner := EditorLeaseOwner{Kind: "desktop-app", Ref: "/home/op/.config/web-console-desktop/runtime", EvidenceJSON: `{"app_pid":4100}`}
	if _, err := store.CreateEditorLease(ctx, EditorLease{SessionID: "s1", Harness: "opencode", PID: 4242, Scope: "vrooli-agent-opencode-1", Owner: owner}, time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateEditorLease(ctx, EditorLease{SessionID: "s1", Harness: "opencode", PID: 4242, Scope: "vrooli-agent-opencode-1"}, time.Minute); err != nil {
		t.Fatal(err)
	}
	leases, err := store.ListEditorLeases(ctx, false)
	if err != nil || len(leases) != 1 {
		t.Fatalf("leases = %+v, %v", leases, err)
	}
	got := leases[0].Owner
	if got.Kind != "desktop-app" || got.Ref != owner.Ref || got.EvidenceJSON != owner.EvidenceJSON || got.Source != EditorLeaseOwnerLaunch || got.RecordedAt == nil {
		t.Fatalf("owner = %+v, want the launch-recorded desktop app", got)
	}
}

// A backfill fills only an empty owner; it never replaces a recorded one.
func TestBackfillEditorLeaseOwnerNeverOverwritesARecordedOwner(t *testing.T) {
	ctx := context.Background()
	store, _ := newEditorLeaseStore(t)
	for _, lease := range []EditorLease{
		{SessionID: "bare", PID: 1, Scope: "vrooli-agent-bare"},
		{SessionID: "owned", PID: 2, Scope: "vrooli-agent-owned", Owner: EditorLeaseOwner{Kind: "web-console-session", Ref: "wc-1"}},
	} {
		if _, err := store.CreateEditorLease(ctx, lease, time.Minute); err != nil {
			t.Fatal(err)
		}
	}
	inferred := EditorLeaseOwner{Kind: "agent-manager-run", Ref: "run-9", EvidenceJSON: `{"run_id":"run-9"}`}
	if changed, err := store.BackfillEditorLeaseOwner(ctx, "bare", inferred); err != nil || !changed {
		t.Fatalf("backfill of an empty owner = %v, %v; want changed", changed, err)
	}
	if changed, err := store.BackfillEditorLeaseOwner(ctx, "owned", inferred); err != nil || changed {
		t.Fatalf("backfill over a recorded owner = %v, %v; want unchanged", changed, err)
	}
	byID := leasesByID(t, store)
	if o := byID["bare"].Owner; o.Kind != "agent-manager-run" || o.Ref != "run-9" || o.Source != EditorLeaseOwnerBackfill {
		t.Fatalf("bare owner = %+v, want the backfilled run", o)
	}
	if o := byID["owned"].Owner; o.Kind != "web-console-session" || o.Ref != "wc-1" {
		t.Fatalf("owned owner = %+v, want the launch owner kept", o)
	}
}

// Evidence refresh adds what the owner's authority later reported, and only
// for the same owner.
func TestRefreshEditorLeaseOwnerEvidenceRequiresTheSameOwner(t *testing.T) {
	ctx := context.Background()
	store, _ := newEditorLeaseStore(t)
	if _, err := store.CreateEditorLease(ctx, EditorLease{SessionID: "s", PID: 1, Owner: EditorLeaseOwner{Kind: "web-console-session", Ref: "wc-1", EvidenceJSON: `{}`}}, time.Minute); err != nil {
		t.Fatal(err)
	}
	if changed, _ := store.RefreshEditorLeaseOwnerEvidence(ctx, "s", EditorLeaseOwner{Kind: "web-console-session", Ref: "wc-2", EvidenceJSON: `{"web_console_origin":"programmatic"}`}); changed {
		t.Fatal("refresh for a different owner ref changed the row")
	}
	if changed, err := store.RefreshEditorLeaseOwnerEvidence(ctx, "s", EditorLeaseOwner{Kind: "web-console-session", Ref: "wc-1", EvidenceJSON: `{"web_console_origin":"ui"}`}); err != nil || !changed {
		t.Fatalf("refresh for the same owner = %v, %v", changed, err)
	}
	if got := leasesByID(t, store)["s"].Owner.EvidenceJSON; got != `{"web_console_origin":"ui"}` {
		t.Fatalf("evidence = %s", got)
	}
}

func TestAgentSessionReapAuditRoundTrips(t *testing.T) {
	ctx := context.Background()
	store, clk := newEditorLeaseStore(t)
	if _, err := store.RecordAgentSessionReap(ctx, AgentSessionReap{Scope: "vrooli-agent-a.scope"}); err == nil {
		t.Fatal("a reap without an outcome was recorded")
	}
	written, err := store.RecordAgentSessionReap(ctx, AgentSessionReap{
		SessionID: "s", Scope: "vrooli-agent-a.scope", OwnerKind: "desktop-app", OwnerRef: "/data", OwnerEvidenceJSON: `{"app_pid":7}`,
		LivenessDetail: "desktop app runtime pid 7 has exited", MemoryBytes: 1 << 30, SwapBytes: 2 << 30, Tasks: 12,
		Outcome: AgentSessionReapStopped, Initiator: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	reaps, err := store.ListAgentSessionReaps(ctx, 10)
	if err != nil || len(reaps) != 1 {
		t.Fatalf("reaps = %+v, %v", reaps, err)
	}
	got := reaps[0]
	if got.ReapID != written.ReapID || got.OwnerKind != "desktop-app" || got.MemoryBytes != 1<<30 || got.SwapBytes != 2<<30 || got.Tasks != 12 || got.Outcome != "stopped" || !got.CreatedAt.Equal(clk.Now()) {
		t.Fatalf("reap = %+v", got)
	}
}

// A v13 registry with live leases climbs to v14 with its rows intact and
// gains the owner columns and the reap audit table.
func TestSchemaMigrationAddsEditorLeaseOwnerToExistingDatabase(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	raw, err := sql.Open("sqlite", buildDSN(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := raw.ExecContext(ctx, schemaSQL); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	for _, column := range []string{"owner_kind", "owner_ref", "owner_evidence_json", "owner_source", "owner_recorded_at"} {
		if _, err := raw.ExecContext(ctx, "ALTER TABLE runtime_editor_leases DROP COLUMN "+column); err != nil {
			t.Fatalf("drop %s to simulate v13: %v", column, err)
		}
	}
	if _, err := raw.ExecContext(ctx, "DROP TABLE runtime_agent_session_reaps"); err != nil {
		t.Fatalf("drop reap table: %v", err)
	}
	if _, err := raw.ExecContext(ctx, `INSERT INTO runtime_editor_leases (session_id, harness, agent, pid, scope, status, created_at, last_heartbeat_at, heartbeat_deadline_at)
VALUES ('legacy', 'codex', 'codex', 42, 'vrooli-agent-legacy', 'active', '2026-09-20T10:00:00Z', '2026-09-20T10:00:00Z', '2026-09-20T10:01:00Z')`); err != nil {
		t.Fatalf("insert legacy lease: %v", err)
	}
	if _, err := raw.ExecContext(ctx, "PRAGMA user_version = 13"); err != nil {
		t.Fatal(err)
	}
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}
	clk := testenv.NewClock(time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC))
	store, err := NewSQLiteStore(ctx, Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatalf("NewSQLiteStore on a v13 database: %v", err)
	}
	defer store.Close()
	if version, err := readSchemaVersion(ctx, store.db); err != nil || version != 14 {
		t.Fatalf("schema version = %d, %v; want 14", version, err)
	}
	legacy := leasesByID(t, store)["legacy"]
	if legacy.SessionID != "legacy" || legacy.Owner.Kind != "" {
		t.Fatalf("legacy lease = %+v, want preserved with no owner", legacy)
	}
	if changed, err := store.BackfillEditorLeaseOwner(ctx, "legacy", EditorLeaseOwner{Kind: "terminal", Ref: "/dev/pts/3"}); err != nil || !changed {
		t.Fatalf("backfill after migration = %v, %v", changed, err)
	}
	if _, err := store.RecordAgentSessionReap(ctx, AgentSessionReap{Scope: "vrooli-agent-x.scope", Outcome: AgentSessionReapStopped}); err != nil {
		t.Fatalf("reap audit after migration: %v", err)
	}
}

func leasesByID(t *testing.T, store *SQLiteStore) map[string]EditorLease {
	t.Helper()
	leases, err := store.ListEditorLeases(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]EditorLease{}
	for _, lease := range leases {
		out[lease.SessionID] = lease
	}
	return out
}
