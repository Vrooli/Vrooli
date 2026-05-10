package scenarioruntime

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

type fixedClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFixedClock(t time.Time) *fixedClock {
	return &fixedClock{now: t.UTC()}
}

func (c *fixedClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fixedClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func TestSQLiteStoreCreatesUpdatesAndQueriesInstance(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)

	created, err := store.CreateInstance(ctx, Instance{
		InstanceID: "inst-alpha-1",
		Scenario:   "alpha",
		Phase:      "api",
		WorkingDir: "/repo/scenarios/alpha",
	})
	if err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}
	if created.Generation != 1 {
		t.Fatalf("created.Generation = %d, want 1", created.Generation)
	}
	if created.Status != StatusStarting {
		t.Fatalf("created.Status = %q, want %q", created.Status, StatusStarting)
	}
	if created.SchemaVersion != SchemaVersion {
		t.Fatalf("created.SchemaVersion = %d, want %d", created.SchemaVersion, SchemaVersion)
	}

	clk.Advance(time.Minute)
	updated, err := store.UpdateInstanceStatus(ctx, created.InstanceID, created.Generation, StatusRunning, "ready")
	if err != nil {
		t.Fatalf("UpdateInstanceStatus() error = %v", err)
	}
	if updated.Status != StatusRunning || updated.Phase != "ready" {
		t.Fatalf("updated = status %q phase %q, want running/ready", updated.Status, updated.Phase)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Fatalf("updated.UpdatedAt = %s, want after %s", updated.UpdatedAt, created.UpdatedAt)
	}

	listed, err := store.ListInstances(ctx, InstanceFilter{
		Scenario: "alpha",
		Statuses: []string{StatusRunning},
	})
	if err != nil {
		t.Fatalf("ListInstances() error = %v", err)
	}
	if len(listed) != 1 || listed[0].InstanceID != created.InstanceID {
		t.Fatalf("ListInstances() = %#v, want created instance", listed)
	}
}

func TestSQLiteStoreRejectsLegacyRegistryForGreenfieldSchemaV3(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	db, err := sql.Open("sqlite", buildDSN(dbPath))
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	_, err = db.ExecContext(ctx, `
CREATE TABLE schema_version (version INTEGER NOT NULL, applied_at TEXT NOT NULL);
INSERT INTO schema_version (version, applied_at) VALUES (1, '2026-05-08T00:00:00Z');
CREATE TABLE runtime_instances (
  instance_id TEXT PRIMARY KEY,
  scenario TEXT NOT NULL,
  generation INTEGER NOT NULL,
  scope_path TEXT NOT NULL DEFAULT '',
  sandbox_id TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  phase TEXT NOT NULL DEFAULT '',
  started_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  last_heartbeat_at TEXT,
  heartbeat_deadline_at TEXT,
  stopped_at TEXT,
  stop_reason TEXT NOT NULL DEFAULT '',
  owner_kind TEXT NOT NULL DEFAULT 'lifecycle',
  owner_pid INTEGER,
  working_dir TEXT NOT NULL DEFAULT '',
  schema_version INTEGER NOT NULL
);
CREATE TABLE runtime_port_claims (
  claim_id TEXT PRIMARY KEY,
  instance_id TEXT NOT NULL,
  scenario TEXT NOT NULL,
  port_name TEXT NOT NULL DEFAULT '',
  env_var TEXT NOT NULL DEFAULT '',
  port INTEGER NOT NULL,
  bind_host TEXT NOT NULL DEFAULT '127.0.0.1',
  url TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  expires_at TEXT,
  last_bound_at TEXT
);
CREATE TABLE runtime_health_snapshots (
  instance_id TEXT PRIMARY KEY,
  scenario TEXT NOT NULL,
  status TEXT NOT NULL,
  readiness INTEGER,
  checked_at TEXT,
  latency_ms INTEGER,
  error TEXT NOT NULL DEFAULT '',
  response_json TEXT NOT NULL DEFAULT '',
  schema_valid INTEGER
);
CREATE TABLE runtime_process_refs (
  ref_id TEXT PRIMARY KEY,
  instance_id TEXT NOT NULL,
  pid INTEGER,
  pgid INTEGER,
  process_id TEXT NOT NULL DEFAULT '',
  step TEXT NOT NULL DEFAULT '',
  command TEXT NOT NULL DEFAULT '',
  log_file TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT '',
  started_at TEXT NOT NULL,
  ended_at TEXT
);
CREATE TABLE runtime_events (
  event_id TEXT PRIMARY KEY,
  instance_id TEXT,
  scenario TEXT NOT NULL DEFAULT '',
  event_type TEXT NOT NULL,
  created_at TEXT NOT NULL,
  details_json TEXT NOT NULL DEFAULT ''
);`)
	if closeErr := db.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		t.Fatalf("create v1 database: %v", err)
	}

	_, err = NewSQLiteStore(ctx, Config{DBPath: dbPath})
	if err == nil {
		t.Fatalf("NewSQLiteStore should reject legacy runtime registry schema")
	}
	if !strings.Contains(err.Error(), "requires greenfield rebuild") {
		t.Fatalf("NewSQLiteStore error = %v, want greenfield rebuild guidance", err)
	}
}

func TestSQLiteStoreGenerationBlocksStaleWriters(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newFixedClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)))

	first, err := store.CreateInstance(ctx, Instance{InstanceID: "inst-alpha-1", Scenario: "alpha"})
	if err != nil {
		t.Fatalf("CreateInstance(first) error = %v", err)
	}
	second, err := store.CreateInstance(ctx, Instance{InstanceID: "inst-alpha-2", Scenario: "alpha"})
	if err != nil {
		t.Fatalf("CreateInstance(second) error = %v", err)
	}
	if first.Generation != 1 || second.Generation != 2 {
		t.Fatalf("generations = %d, %d; want 1, 2", first.Generation, second.Generation)
	}

	_, err = store.UpdateInstanceStatus(ctx, second.InstanceID, first.Generation, StatusRunning, "ready")
	if !errors.Is(err, ErrStaleGeneration) {
		t.Fatalf("UpdateInstanceStatus(stale generation) error = %v, want ErrStaleGeneration", err)
	}
}

func TestSQLiteStoreActiveClaimUniqueness(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newFixedClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)))
	alpha, err := store.CreateInstance(ctx, Instance{InstanceID: "inst-alpha", Scenario: "alpha"})
	if err != nil {
		t.Fatalf("CreateInstance(alpha) error = %v", err)
	}
	beta, err := store.CreateInstance(ctx, Instance{InstanceID: "inst-beta", Scenario: "beta"})
	if err != nil {
		t.Fatalf("CreateInstance(beta) error = %v", err)
	}

	if _, err := store.AcquirePortClaim(ctx, PortClaim{
		ClaimID:    "claim-alpha-api",
		InstanceID: alpha.InstanceID,
		Scenario:   alpha.Scenario,
		PortName:   "api",
		EnvVar:     "ALPHA_API_PORT",
		Port:       15080,
		BindHost:   "127.0.0.1",
	}); err != nil {
		t.Fatalf("AcquirePortClaim(alpha) error = %v", err)
	}

	_, err = store.AcquirePortClaim(ctx, PortClaim{
		ClaimID:    "claim-beta-api",
		InstanceID: beta.InstanceID,
		Scenario:   beta.Scenario,
		PortName:   "api",
		EnvVar:     "BETA_API_PORT",
		Port:       15080,
		BindHost:   "127.0.0.1",
	})
	if !errors.Is(err, ErrActiveClaimConflict) {
		t.Fatalf("AcquirePortClaim(conflict) error = %v, want ErrActiveClaimConflict", err)
	}

	released, err := store.ReleasePortClaim(ctx, "claim-alpha-api")
	if err != nil {
		t.Fatalf("ReleasePortClaim() error = %v", err)
	}
	if released.Status != ClaimStatusReleased {
		t.Fatalf("released.Status = %q, want %q", released.Status, ClaimStatusReleased)
	}

	if _, err := store.AcquirePortClaim(ctx, PortClaim{
		ClaimID:    "claim-beta-api",
		InstanceID: beta.InstanceID,
		Scenario:   beta.Scenario,
		PortName:   "api",
		EnvVar:     "BETA_API_PORT",
		Port:       15080,
		BindHost:   "127.0.0.1",
	}); err != nil {
		t.Fatalf("AcquirePortClaim(after release) error = %v", err)
	}
}

func TestSQLiteStoreBindsAndReleasesActiveClaimsForInstance(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)
	instance, err := store.CreateInstance(ctx, Instance{InstanceID: "inst-alpha", Scenario: "alpha"})
	if err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}

	if _, err := store.AcquirePortClaim(ctx, PortClaim{
		ClaimID:    "claim-alpha-api",
		InstanceID: instance.InstanceID,
		Scenario:   instance.Scenario,
		PortName:   "api",
		EnvVar:     "API_PORT",
		Port:       15080,
	}); err != nil {
		t.Fatalf("AcquirePortClaim() error = %v", err)
	}

	clk.Advance(time.Second)
	bound, err := store.BindPortClaim(ctx, "claim-alpha-api")
	if err != nil {
		t.Fatalf("BindPortClaim() error = %v", err)
	}
	if bound.Status != ClaimStatusBound {
		t.Fatalf("bound.Status = %q, want %q", bound.Status, ClaimStatusBound)
	}
	if bound.LastBoundAt == nil || !bound.LastBoundAt.Equal(clk.Now()) {
		t.Fatalf("bound.LastBoundAt = %#v, want %s", bound.LastBoundAt, clk.Now())
	}

	clk.Advance(time.Second)
	released, err := store.ReleaseActivePortClaimsForInstance(ctx, instance.InstanceID)
	if err != nil {
		t.Fatalf("ReleaseActivePortClaimsForInstance() error = %v", err)
	}
	if len(released) != 1 || released[0].Status != ClaimStatusReleased {
		t.Fatalf("released = %#v, want one released claim", released)
	}
}

func TestSQLiteStoreUpdatesPortClaimListenerEvidence(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)
	instance, err := store.CreateInstance(ctx, Instance{InstanceID: "inst-alpha", Scenario: "alpha"})
	if err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}
	claim, err := store.AcquirePortClaim(ctx, PortClaim{
		ClaimID:    "claim-alpha-api",
		InstanceID: instance.InstanceID,
		Scenario:   instance.Scenario,
		PortName:   "api",
		EnvVar:     "API_PORT",
		Port:       15080,
		Status:     ClaimStatusBound,
	})
	if err != nil {
		t.Fatalf("AcquirePortClaim() error = %v", err)
	}
	if claim.ListenerStatus != ListenerStatusUnknown {
		t.Fatalf("claim.ListenerStatus = %q, want unknown", claim.ListenerStatus)
	}

	clk.Advance(time.Second)
	unbound, err := store.UpdatePortClaimListenerEvidence(ctx, claim.ClaimID, ListenerObservation{
		CheckedAt: clk.Now(),
		Status:    ListenerStatusNotListening,
	})
	if err != nil {
		t.Fatalf("UpdatePortClaimListenerEvidence(not_listening) error = %v", err)
	}
	if unbound.ListenerStatus != ListenerStatusNotListening || unbound.ConsecutiveListenerMisses != 1 {
		t.Fatalf("unbound evidence = %#v, want one not_listening miss", unbound)
	}
	if unbound.FirstUnboundAt == nil || !unbound.FirstUnboundAt.Equal(clk.Now()) {
		t.Fatalf("FirstUnboundAt = %#v, want %s", unbound.FirstUnboundAt, clk.Now())
	}

	clk.Advance(time.Second)
	pid := 4321
	listening, err := store.UpdatePortClaimListenerEvidence(ctx, claim.ClaimID, ListenerObservation{
		CheckedAt:    clk.Now(),
		Status:       ListenerStatusListening,
		PID:          &pid,
		ProcessLabel: "alpha-api",
	})
	if err != nil {
		t.Fatalf("UpdatePortClaimListenerEvidence(listening) error = %v", err)
	}
	if listening.ListenerStatus != ListenerStatusListening || listening.ConsecutiveListenerMisses != 0 {
		t.Fatalf("listening evidence = %#v, want listening without misses", listening)
	}
	if listening.FirstUnboundAt != nil {
		t.Fatalf("FirstUnboundAt = %#v, want nil after listener is seen", listening.FirstUnboundAt)
	}
	if listening.LastListenerSeenAt == nil || !listening.LastListenerSeenAt.Equal(clk.Now()) {
		t.Fatalf("LastListenerSeenAt = %#v, want %s", listening.LastListenerSeenAt, clk.Now())
	}
	if listening.ListenerPID == nil || *listening.ListenerPID != pid || listening.ListenerProcessLabel != "alpha-api" {
		t.Fatalf("listener identity = pid %#v label %q, want %d alpha-api", listening.ListenerPID, listening.ListenerProcessLabel, pid)
	}
}

func TestSQLiteStoreProcessRefsRoundTrip(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)
	instance, err := store.CreateInstance(ctx, Instance{InstanceID: "inst-alpha", Scenario: "alpha"})
	if err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}

	pid := 1234
	pgid := 1200
	ref, err := store.AddProcessRef(ctx, ProcessRef{
		RefID:      "proc-alpha-api",
		InstanceID: instance.InstanceID,
		PID:        &pid,
		PGID:       &pgid,
		ProcessID:  "vrooli.develop.alpha.start-api",
		Step:       "start-api",
		Command:    "api/mock-api",
		LogFile:    "/tmp/alpha.log",
	})
	if err != nil {
		t.Fatalf("AddProcessRef() error = %v", err)
	}
	if ref.Status != "running" {
		t.Fatalf("ref.Status = %q, want running", ref.Status)
	}

	endedAt := clk.Now().Add(time.Minute)
	updated, err := store.UpdateProcessRefStatus(ctx, ref.RefID, "failed", &endedAt)
	if err != nil {
		t.Fatalf("UpdateProcessRefStatus() error = %v", err)
	}
	if updated.Status != "failed" || updated.EndedAt == nil || !updated.EndedAt.Equal(endedAt) {
		t.Fatalf("updated = %#v, want failed with ended_at", updated)
	}

	refs, err := store.ListProcessRefs(ctx, instance.InstanceID)
	if err != nil {
		t.Fatalf("ListProcessRefs() error = %v", err)
	}
	if len(refs) != 1 || refs[0].RefID != ref.RefID {
		t.Fatalf("refs = %#v, want process ref", refs)
	}
}

func TestSQLiteStoreExpiredClaimsAreQueryableSeparately(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)
	instance, err := store.CreateInstance(ctx, Instance{InstanceID: "inst-alpha", Scenario: "alpha"})
	if err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}
	expiresAt := clk.Now().Add(30 * time.Second)
	if _, err := store.AcquirePortClaim(ctx, PortClaim{
		ClaimID:    "claim-alpha-api",
		InstanceID: instance.InstanceID,
		Scenario:   instance.Scenario,
		PortName:   "api",
		EnvVar:     "ALPHA_API_PORT",
		Port:       15080,
		ExpiresAt:  &expiresAt,
	}); err != nil {
		t.Fatalf("AcquirePortClaim() error = %v", err)
	}

	before, err := store.ListExpiredActivePortClaims(ctx, clk.Now().Add(29*time.Second))
	if err != nil {
		t.Fatalf("ListExpiredActivePortClaims(before) error = %v", err)
	}
	if len(before) != 0 {
		t.Fatalf("expired before deadline = %#v, want none", before)
	}

	after, err := store.ListExpiredActivePortClaims(ctx, clk.Now().Add(31*time.Second))
	if err != nil {
		t.Fatalf("ListExpiredActivePortClaims(after) error = %v", err)
	}
	if len(after) != 1 || after[0].ClaimID != "claim-alpha-api" {
		t.Fatalf("expired after deadline = %#v, want claim-alpha-api", after)
	}
}

func TestSQLiteStoreHealthSnapshotRoundTrip(t *testing.T) {
	ctx := context.Background()
	clk := newFixedClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC))
	store := newTestStore(t, clk)
	instance, err := store.CreateInstance(ctx, Instance{InstanceID: "inst-alpha", Scenario: "alpha"})
	if err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}

	ready := true
	schemaValid := false
	latencyMillis := int64(42)
	checkedAt := clk.Now()
	if _, err := store.UpsertHealthSnapshot(ctx, HealthSnapshot{
		InstanceID:    instance.InstanceID,
		Scenario:      instance.Scenario,
		Status:        HealthStatusDegraded,
		Readiness:     &ready,
		CheckedAt:     &checkedAt,
		LatencyMillis: &latencyMillis,
		Error:         "dependency degraded",
		ResponseJSON:  `{"status":"degraded"}`,
		SchemaValid:   &schemaValid,
	}); err != nil {
		t.Fatalf("UpsertHealthSnapshot() error = %v", err)
	}

	snapshot, err := store.GetHealthSnapshot(ctx, instance.InstanceID)
	if err != nil {
		t.Fatalf("GetHealthSnapshot() error = %v", err)
	}
	if snapshot.Status != HealthStatusDegraded {
		t.Fatalf("snapshot.Status = %q, want %q", snapshot.Status, HealthStatusDegraded)
	}
	if snapshot.Readiness == nil || *snapshot.Readiness != ready {
		t.Fatalf("snapshot.Readiness = %#v, want true", snapshot.Readiness)
	}
	if snapshot.SchemaValid == nil || *snapshot.SchemaValid != schemaValid {
		t.Fatalf("snapshot.SchemaValid = %#v, want false", snapshot.SchemaValid)
	}
	if snapshot.LatencyMillis == nil || *snapshot.LatencyMillis != latencyMillis {
		t.Fatalf("snapshot.LatencyMillis = %#v, want 42", snapshot.LatencyMillis)
	}
}

func TestSQLiteStoreUsesExplicitTempPathOnly(t *testing.T) {
	ctx := context.Background()
	home := t.TempDir()
	path, err := DefaultDBPath(home)
	if err != nil {
		t.Fatalf("DefaultDBPath() error = %v", err)
	}
	want := filepath.Join(home, ".vrooli", "state", "runtime.db")
	if path != want {
		t.Fatalf("DefaultDBPath(%q) = %q, want %q", home, path, want)
	}

	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	store, err := NewSQLiteStore(ctx, Config{
		DBPath: dbPath,
		Clock:  newFixedClock(time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)),
	})
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	defer store.Close()
	if _, err := store.CreateInstance(ctx, Instance{Scenario: "alpha"}); err != nil {
		t.Fatalf("CreateInstance() error = %v", err)
	}
}

func newTestStore(t *testing.T, clk Clock) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(context.Background(), Config{
		DBPath: filepath.Join(t.TempDir(), "runtime.db"),
		Clock:  clk,
	})
	if err != nil {
		t.Fatalf("NewSQLiteStore() error = %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})
	return store
}
