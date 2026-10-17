package orchestrator

import (
	"context"
	"testing"
	"time"

	"storage-manager/internal/cleanup"
	"storage-manager/internal/policy"

	db "github.com/vrooli/api-core/databasetest"
)

func newStore(t *testing.T) (*SQLiteStore, func()) {
	t.Helper()
	handle := db.NewSQLite(t)
	if _, err := handle.ExecContext(context.Background(), schemaSQL); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	return NewSQLiteStore(handle), func() {}
}

// TestPolicySurvivesRestart is the reason this store exists.
//
// While the policy lived only in process memory, an operator who enabled
// cleanup got it silently reverted to the shipped default on the next restart —
// indistinguishable from cleanup never having been enabled, which is the exact
// failure that let a disk fill while three safeguards reported healthy.
func TestPolicySurvivesRestart(t *testing.T) {
	t.Parallel()

	store, done := newStore(t)
	defer done()
	ctx := context.Background()

	saved := Policy{
		Version: "policy-abc123",
		Profile: policy.ProfileBalanced,
		Providers: map[string]cleanup.ProviderPolicy{
			"tmp": {Enabled: true, MinAge: 72 * time.Hour, ApprovalMode: cleanup.ApprovalModeOperator},
		},
		CreatedAt: time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC),
	}
	if err := store.SavePolicy(ctx, saved); err != nil {
		t.Fatalf("SavePolicy: %v", err)
	}

	// A restart: brand new store value, same database.
	restarted := NewSQLiteStore(store.db)

	got, ok, err := restarted.CurrentPolicy(ctx)
	if err != nil {
		t.Fatalf("CurrentPolicy: %v", err)
	}
	if !ok {
		t.Fatal("no policy after restart; the operator's choice was lost")
	}
	if got.Profile != policy.ProfileBalanced {
		t.Errorf("profile = %q, want %q", got.Profile, policy.ProfileBalanced)
	}
	if got.Version != saved.Version {
		t.Errorf("version = %q, want %q", got.Version, saved.Version)
	}
	tmp, present := got.Providers["tmp"]
	if !present {
		t.Fatalf("per-provider policy lost: %#v", got.Providers)
	}
	if !tmp.Enabled || tmp.MinAge != 72*time.Hour {
		t.Errorf("tmp policy = %+v, want enabled with a 72h min age", tmp)
	}
}

// TestCurrentPolicy_ReportsAbsenceRatherThanEmptyPolicy asserts a fresh
// database is distinguishable from a stored empty policy. Conflating them would
// mean a brand-new install silently ran with every provider disabled instead of
// applying its default profile.
func TestCurrentPolicy_ReportsAbsenceRatherThanEmptyPolicy(t *testing.T) {
	t.Parallel()

	store, done := newStore(t)
	defer done()

	_, ok, err := store.CurrentPolicy(context.Background())
	if err != nil {
		t.Fatalf("CurrentPolicy: %v", err)
	}
	if ok {
		t.Error("an empty database reported a stored policy")
	}
}

// TestSavePolicy_ReplacesRatherThanAccumulates asserts there is exactly one
// active policy. Accumulating versions would let a reader pick the wrong one.
func TestSavePolicy_ReplacesRatherThanAccumulates(t *testing.T) {
	t.Parallel()

	store, done := newStore(t)
	defer done()
	ctx := context.Background()

	for _, profile := range []policy.ProfileName{policy.ProfileConservative, policy.ProfileBalanced, policy.ProfileAggressive} {
		if err := store.SavePolicy(ctx, Policy{Version: "v-" + string(profile), Profile: profile}); err != nil {
			t.Fatalf("SavePolicy(%s): %v", profile, err)
		}
	}

	var rows int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM cleanup_policy`).Scan(&rows); err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != 1 {
		t.Errorf("cleanup_policy has %d rows, want exactly 1", rows)
	}

	got, _, err := store.CurrentPolicy(ctx)
	if err != nil {
		t.Fatalf("CurrentPolicy: %v", err)
	}
	if got.Profile != policy.ProfileAggressive {
		t.Errorf("profile = %q, want the last one saved", got.Profile)
	}
}

// TestAuditSurvivesRestartAndDeduplicates asserts the audit trail outlives the
// process that wrote it, and that replaying the same event does not duplicate
// it. Evidence of a deletion is worthless if it vanishes with the deleter.
func TestAuditSurvivesRestartAndDeduplicates(t *testing.T) {
	t.Parallel()

	store, done := newStore(t)
	defer done()
	ctx := context.Background()

	base := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	events := []AuditEvent{
		{ID: "a1", Time: base, Type: "policy.saved", Message: "balanced"},
		{ID: "a2", Time: base.Add(time.Second), Type: "pressure.applied", PlanID: "plan-1", ProviderID: "tmp", Message: "reclaimed at [path]", Redacted: true},
	}
	for _, event := range events {
		if err := store.AddAudit(ctx, event); err != nil {
			t.Fatalf("AddAudit: %v", err)
		}
	}
	// Same event again: must not duplicate.
	if err := store.AddAudit(ctx, events[1]); err != nil {
		t.Fatalf("AddAudit replay: %v", err)
	}

	restarted := NewSQLiteStore(store.db)
	got, err := restarted.ListAudit(ctx)
	if err != nil {
		t.Fatalf("ListAudit: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d audit events after restart, want 2: %#v", len(got), got)
	}
	if got[0].ID != "a1" || got[1].ID != "a2" {
		t.Errorf("audit order = %s,%s; want oldest first", got[0].ID, got[1].ID)
	}
	if !got[1].Redacted || got[1].ProviderID != "tmp" || got[1].PlanID != "plan-1" {
		t.Errorf("audit fields lost across restart: %+v", got[1])
	}
}

// TestPlansAndAppliesStayInMemory documents the deliberate half of the split: a
// plan is a filesystem measurement consumed within seconds and meaningless
// after a restart, and one real plan serialised to 6.9 MB. Persisting those in
// the scenario whose job is reclaiming disk space would be self-defeating.
func TestPlansAndAppliesStayInMemory(t *testing.T) {
	t.Parallel()

	store, done := newStore(t)
	defer done()
	ctx := context.Background()

	if err := store.SavePlan(ctx, Plan{ID: "plan-1", PolicyVersion: "v1"}); err != nil {
		t.Fatalf("SavePlan: %v", err)
	}
	if _, ok, _ := store.GetPlan(ctx, "plan-1"); !ok {
		t.Fatal("plan not retrievable within the same process")
	}

	if _, ok, _ := NewSQLiteStore(store.db).GetPlan(ctx, "plan-1"); ok {
		t.Error("plan survived a restart; plans are intentionally transient")
	}
}

func TestRecoveryLedgerSurvivesRestartAndKeepsByteAccountingTyped(t *testing.T) {
	t.Parallel()
	store, done := newStore(t)
	defer done()
	ctx := context.Background()
	started := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	run := RecoveryRun{ID: "recovery-1", Status: RecoveryComplete, Trigger: "RATE", Partition: "/", TargetFreeBytes: 100, ReclaimedBytes: 250, StartedAt: started, CompletedAt: started.Add(time.Second), StoppedBecause: "target_met"}
	if err := store.SaveRecoveryRun(ctx, run); err != nil {
		t.Fatalf("SaveRecoveryRun: %v", err)
	}
	updated := run
	updated.TargetFreeBytes = 200
	if err := store.SaveRecoveryRun(ctx, updated); err != nil {
		t.Fatalf("SaveRecoveryRun update: %v", err)
	}
	if err := store.SaveRecoveryActionMetrics(ctx, run.ID, "tmp", "R0", 250, 50, 300, 1, 1250*time.Millisecond, cleanup.ApplyResult{AppliedItems: []string{"tmp/item"}}); err != nil {
		t.Fatalf("SaveRecoveryAction: %v", err)
	}

	var reclaimed, freeBefore, freeAfter int64
	if err := NewSQLiteStore(store.db).db.QueryRowContext(ctx, `SELECT reclaimed_bytes, target_free_bytes FROM recovery_runs WHERE id = ?`, run.ID).Scan(&reclaimed, &freeBefore); err != nil {
		t.Fatalf("read recovery run after restart: %v", err)
	}
	if reclaimed != 250 || freeBefore != 200 {
		t.Fatalf("durable run bytes = %d/%d, want 250/200", reclaimed, freeBefore)
	}
	if err := NewSQLiteStore(store.db).db.QueryRowContext(ctx, `SELECT free_before, free_after FROM recovery_actions WHERE run_id = ?`, run.ID).Scan(&freeBefore, &freeAfter); err != nil {
		t.Fatalf("read recovery action after restart: %v", err)
	}
	if freeBefore != 50 || freeAfter != 300 {
		t.Fatalf("durable action free space = %d/%d, want 50/300", freeBefore, freeAfter)
	}
	var filesRemoved, durationMS int
	if err := NewSQLiteStore(store.db).db.QueryRowContext(ctx, `SELECT files_removed, duration_ms FROM recovery_actions WHERE run_id = ?`, run.ID).Scan(&filesRemoved, &durationMS); err != nil {
		t.Fatalf("read recovery action metrics after restart: %v", err)
	}
	if filesRemoved != 1 || durationMS != 1250 {
		t.Fatalf("durable action metrics = %d/%d, want 1/1250", filesRemoved, durationMS)
	}
}

func TestListWriterSnapshotsReturnsNewestObservationPerRoot(t *testing.T) {
	t.Parallel()
	store, done := newStore(t)
	defer done()
	ctx := context.Background()
	old := time.Now().UTC().Add(-3 * time.Hour).Format(time.RFC3339Nano)
	newest := time.Now().UTC().Add(-30 * time.Minute).Format(time.RFC3339Nano)
	if err := store.SaveWriterSnapshot(ctx, "old", old, "/", "/cache", 1, 1, 1, 999, false, true); err != nil {
		t.Fatalf("save old writer snapshot: %v", err)
	}
	if err := store.SaveWriterSnapshot(ctx, "new", newest, "/", "/cache", 2, 1, 1, 10, false, false); err != nil {
		t.Fatalf("save newest writer snapshot: %v", err)
	}
	if err := store.SaveWriterSnapshot(ctx, "other", old, "/", "/other", 3, 1, 1, 20, false, true); err != nil {
		t.Fatalf("save other writer snapshot: %v", err)
	}
	rows, err := store.ListWriterSnapshots(ctx, 10)
	if err != nil {
		t.Fatalf("list writer snapshots: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("writer rows = %d, want only the one fresh root", len(rows))
	}
	for _, row := range rows {
		if row.Root == "/cache" && (row.ID != "new" || row.BytesPerHour != 10 || row.Hot) {
			t.Fatalf("stale cache row was returned: %#v", row)
		}
	}
}

func TestMarkWritersCooledReconcilesAbsentHotRoots(t *testing.T) {
	t.Parallel()
	store, done := newStore(t)
	defer done()
	ctx := context.Background()
	if err := store.SaveWriterSnapshot(ctx, "hot", time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano), "/", "/cache", 42, 42, 1, 999, false, true); err != nil {
		t.Fatalf("save hot snapshot: %v", err)
	}
	if err := store.MarkWritersCooled(ctx, time.Now().UTC().Format(time.RFC3339Nano), map[string]struct{}{}); err != nil {
		t.Fatalf("mark cooled: %v", err)
	}
	rows, err := store.ListWriterSnapshots(ctx, 10)
	if err != nil {
		t.Fatalf("list writer snapshots: %v", err)
	}
	if len(rows) != 1 || rows[0].Root != "/cache" || rows[0].Hot {
		t.Fatalf("cooled writer rows = %#v", rows)
	}
}
