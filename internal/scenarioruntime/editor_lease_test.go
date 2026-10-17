package scenarioruntime

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/testenv"
)

func newEditorLeaseStore(t *testing.T) (*SQLiteStore, *testenv.Clock) {
	t.Helper()
	clk := testenv.NewClock(time.Date(2026, 9, 2, 15, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "runtime.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})
	return store, clk
}

// A slow session (deadline elapsed, pid alive on this boot) is never evicted;
// a dead pid or another boot's session is. Liveness is the authority.
func TestEditorLeaseExpiresOnlyOnProofOfDeath(t *testing.T) {
	ctx := context.Background()
	store, clk := newEditorLeaseStore(t)
	alive := map[int]bool{100: true, 200: false}
	guard := StartingLeaseGuard{CurrentBootID: "boot-a", PIDRunning: func(pid int) bool { return alive[pid] }}
	for _, lease := range []EditorLease{
		{SessionID: "slow", Harness: "claude", Agent: "claude", PID: 100, HostBootID: "boot-a", WorkingDir: "/repo", Scope: "vrooli-agent-slow.scope", Claims: []string{"internal/"}},
		{SessionID: "dead", Harness: "codex", Agent: "codex", PID: 200, HostBootID: "boot-a", WorkingDir: "/repo"},
		{SessionID: "rebooted", Harness: "claude", Agent: "claude", PID: 100, HostBootID: "boot-old", WorkingDir: "/repo"},
	} {
		if _, err := store.CreateEditorLease(ctx, lease, time.Minute); err != nil {
			t.Fatalf("CreateEditorLease(%s): %v", lease.SessionID, err)
		}
	}
	clk.Advance(2 * time.Minute)
	all, err := store.ListEditorLeases(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if live := LiveEditorLeases(all, guard); len(live) != 1 || live[0].SessionID != "slow" {
		t.Fatalf("live view before the sweep = %+v, want only the slow session", live)
	}
	expired, err := store.ExpireStaleEditorLeases(ctx, clk.Now(), guard)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, lease := range expired {
		got[lease.SessionID] = lease.StopReason
	}
	if len(got) != 2 || got["dead"] != "stale editor lease: owner_pid_dead" || got["rebooted"] != "stale editor lease: boot_id_mismatch" {
		t.Fatalf("expired = %v, want dead and rebooted only", got)
	}
	active, err := store.ListEditorLeases(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 || active[0].SessionID != "slow" || active[0].Claims[0] != "internal/" || active[0].Scope != "vrooli-agent-slow.scope" {
		t.Fatalf("active leases = %+v, want the slow session intact", active)
	}
	if _, err := store.HeartbeatEditorLease(ctx, "slow", time.Minute); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if _, err := store.HeartbeatEditorLease(ctx, "dead", time.Minute); err == nil {
		t.Fatal("heartbeat of an expired lease must fail")
	}
	stopped, err := store.StopEditorLease(ctx, "slow", "session ended")
	if err != nil || stopped.Status != EditorLeaseStopped || stopped.StoppedAt == nil {
		t.Fatalf("stop = %+v, %v", stopped, err)
	}
	if every, err := store.ListEditorLeases(ctx, true); err != nil || len(every) != 3 {
		t.Fatalf("all leases = %d, %v", len(every), err)
	}
	if _, err := store.CreateEditorLease(ctx, EditorLease{SessionID: "slow", PID: 100, HostBootID: "boot-a"}, time.Minute); err != nil {
		t.Fatalf("re-attach of a stopped session must refresh the row: %v", err)
	}
	active, _ = store.ListEditorLeases(ctx, false)
	if len(active) != 1 || active[0].Status != EditorLeaseActive {
		t.Fatalf("re-attached lease = %+v", active)
	}
}
