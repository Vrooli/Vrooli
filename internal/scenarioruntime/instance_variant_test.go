package scenarioruntime

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// TestCreateInstancePerVariantGeneration proves the schema-5 invariant: two
// named instances of one scenario coexist, each with its own generation
// counter. Before schema 5 the UNIQUE(scenario, generation) constraint would
// have rejected the shadow's generation 1 because the live instance already
// owned it.
func TestCreateInstancePerVariantGeneration(t *testing.T) {
	store := newTestStore(t, newFixedClock(time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)))

	live1 := mustCreate(t, store, Instance{Scenario: "alpha"})
	if live1.Variant != DefaultVariant {
		t.Fatalf("empty variant normalized to %q, want %q", live1.Variant, DefaultVariant)
	}
	if live1.Generation != 1 {
		t.Fatalf("live generation = %d, want 1", live1.Generation)
	}

	live2 := mustCreate(t, store, Instance{Scenario: "alpha"})
	if live2.Generation != 2 {
		t.Fatalf("second live generation = %d, want 2 (per-variant counter)", live2.Generation)
	}

	shadow1 := mustCreate(t, store, Instance{Scenario: "alpha", Variant: "shadow"})
	if shadow1.Variant != "shadow" {
		t.Fatalf("shadow variant = %q, want shadow", shadow1.Variant)
	}
	if shadow1.Generation != 1 {
		t.Fatalf("shadow generation = %d, want 1 (independent of live)", shadow1.Generation)
	}

	shadow2 := mustCreate(t, store, Instance{Scenario: "alpha", Variant: "shadow"})
	if shadow2.Generation != 2 {
		t.Fatalf("second shadow generation = %d, want 2", shadow2.Generation)
	}
}

// TestCreateInstanceNormalizesVariant confirms the store routes the variant
// through the InstanceKey SSOT so casing/whitespace can never fragment the
// uniqueness key or generation counter.
func TestCreateInstanceNormalizesVariant(t *testing.T) {
	store := newTestStore(t, newFixedClock(time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)))

	a := mustCreate(t, store, Instance{Scenario: "beta", Variant: "  Shadow "})
	if a.Variant != "shadow" {
		t.Fatalf("variant = %q, want normalized shadow", a.Variant)
	}
	// A differently-cased spelling must share the same counter, not start over.
	b := mustCreate(t, store, Instance{Scenario: "beta", Variant: "SHADOW"})
	if b.Generation != 2 {
		t.Fatalf("generation = %d, want 2 (casing must not fork the counter)", b.Generation)
	}
}

// TestListInstancesVariantFilter checks that a variant-scoped query resolves
// only that variant's authoritative instance.
func TestListInstancesVariantFilter(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newFixedClock(time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)))

	mustCreate(t, store, Instance{Scenario: "gamma"})
	mustCreate(t, store, Instance{Scenario: "gamma", Variant: "shadow"})

	all, err := store.ListInstances(ctx, InstanceFilter{Scenario: "gamma"})
	if err != nil {
		t.Fatalf("ListInstances(all) error = %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("unfiltered list = %d instances, want 2", len(all))
	}

	shadow, err := store.ListInstances(ctx, InstanceFilter{Scenario: "gamma", Variant: "shadow"})
	if err != nil {
		t.Fatalf("ListInstances(shadow) error = %v", err)
	}
	if len(shadow) != 1 || shadow[0].Variant != "shadow" {
		t.Fatalf("variant filter = %+v, want exactly the shadow instance", shadow)
	}

	live, err := store.ListInstances(ctx, InstanceFilter{Scenario: "gamma", Variant: DefaultVariant})
	if err != nil {
		t.Fatalf("ListInstances(live) error = %v", err)
	}
	if len(live) != 1 || live[0].Variant != DefaultVariant {
		t.Fatalf("live filter = %+v, want exactly the live instance", live)
	}
}

// TestAcquirePortClaimDenormalizesVariant verifies the claim carries (and
// filters by) its instance's variant.
func TestAcquirePortClaimDenormalizesVariant(t *testing.T) {
	ctx := context.Background()
	store := newTestStore(t, newFixedClock(time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC)))

	live := mustCreate(t, store, Instance{Scenario: "delta"})
	shadow := mustCreate(t, store, Instance{Scenario: "delta", Variant: "shadow"})

	if _, err := store.AcquirePortClaim(ctx, PortClaim{InstanceID: live.InstanceID, Scenario: "delta", Port: 8100}); err != nil {
		t.Fatalf("acquire live claim error = %v", err)
	}
	if _, err := store.AcquirePortClaim(ctx, PortClaim{InstanceID: shadow.InstanceID, Scenario: "delta", Variant: "shadow", Port: 9100}); err != nil {
		t.Fatalf("acquire shadow claim error = %v", err)
	}

	shadowClaims, err := store.ListPortClaims(ctx, PortClaimFilter{Scenario: "delta", Variant: "shadow"})
	if err != nil {
		t.Fatalf("ListPortClaims(shadow) error = %v", err)
	}
	if len(shadowClaims) != 1 || shadowClaims[0].Port != 9100 || shadowClaims[0].Variant != "shadow" {
		t.Fatalf("shadow claims = %+v, want one shadow claim on 9100", shadowClaims)
	}

	liveClaims, err := store.ListPortClaims(ctx, PortClaimFilter{Scenario: "delta", Variant: DefaultVariant})
	if err != nil {
		t.Fatalf("ListPortClaims(live) error = %v", err)
	}
	if len(liveClaims) != 1 || liveClaims[0].Port != 8100 || liveClaims[0].Variant != DefaultVariant {
		t.Fatalf("live claims = %+v, want one live claim on 8100", liveClaims)
	}
}

// TestMigrateV4ToV5PreservesRows is the safety-critical test: the registry
// tracks LIVE processes, so the 4→5 upgrade must carry every existing instance
// (and its port claims) forward as variant 'live' rather than dropping them.
// After the migration the new (scenario, variant, generation) uniqueness must
// also admit a shadow instance for the same scenario.
func TestMigrateV4ToV5PreservesRows(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "runtime.db")
	now := formatTime(time.Date(2026, 6, 4, 12, 0, 0, 0, time.UTC))

	seedSchemaV4(t, dbPath, now)

	store, err := NewSQLiteStore(ctx, Config{DBPath: dbPath})
	if err != nil {
		t.Fatalf("NewSQLiteStore() (triggers migration) error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	instances, err := store.ListInstances(ctx, InstanceFilter{Scenario: "legacy"})
	if err != nil {
		t.Fatalf("ListInstances() error = %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("post-migration instances = %d, want 1 (rows must survive)", len(instances))
	}
	got := instances[0]
	if got.Variant != DefaultVariant {
		t.Fatalf("migrated variant = %q, want %q", got.Variant, DefaultVariant)
	}
	if got.Generation != 7 {
		t.Fatalf("migrated generation = %d, want 7 (preserved)", got.Generation)
	}
	if got.InstanceID != "inst-legacy-7" {
		t.Fatalf("migrated instance_id = %q, want inst-legacy-7", got.InstanceID)
	}
	if got.SchemaVersion != SchemaVersion {
		t.Fatalf("migrated schema_version = %d, want %d", got.SchemaVersion, SchemaVersion)
	}

	claims, err := store.ListPortClaims(ctx, PortClaimFilter{Scenario: "legacy"})
	if err != nil {
		t.Fatalf("ListPortClaims() error = %v", err)
	}
	if len(claims) != 1 || claims[0].Variant != DefaultVariant || claims[0].Port != 8200 {
		t.Fatalf("migrated claims = %+v, want one live claim on 8200", claims)
	}

	// The new uniqueness key admits a shadow that the old (scenario, generation)
	// constraint would have blocked at generation 7.
	shadow := mustCreate(t, store, Instance{InstanceID: "inst-legacy-shadow", Scenario: "legacy", Variant: "shadow", Generation: 7})
	if shadow.Generation != 7 {
		t.Fatalf("shadow generation = %d, want 7 (independent namespace)", shadow.Generation)
	}

	// A fresh live instance continues the preserved live counter (7 -> 8).
	nextLive := mustCreate(t, store, Instance{Scenario: "legacy"})
	if nextLive.Generation != 8 {
		t.Fatalf("next live generation = %d, want 8 (continues preserved counter)", nextLive.Generation)
	}
}

func mustCreate(t *testing.T, store *SQLiteStore, in Instance) Instance {
	t.Helper()
	out, err := store.CreateInstance(context.Background(), in)
	if err != nil {
		t.Fatalf("CreateInstance(%+v) error = %v", in, err)
	}
	return out
}

// seedSchemaV4 writes a schema-version-4 runtime registry (no variant column,
// UNIQUE(scenario, generation)) with one instance and one port claim, so the
// store-open path exercises migrateV4ToV5.
func seedSchemaV4(t *testing.T, dbPath, now string) {
	t.Helper()
	db, err := sql.Open("sqlite", buildDSN(dbPath))
	if err != nil {
		t.Fatalf("open v4 seed db error = %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schemaV4SQL); err != nil {
		t.Fatalf("apply v4 schema error = %v", err)
	}
	if _, err := db.Exec(`INSERT INTO runtime_instances
  (instance_id, scenario, generation, status, started_at, updated_at, schema_version)
  VALUES ('inst-legacy-7', 'legacy', 7, 'running', ?, ?, 4)`, now, now); err != nil {
		t.Fatalf("seed v4 instance error = %v", err)
	}
	if _, err := db.Exec(`INSERT INTO runtime_port_claims
  (claim_id, instance_id, scenario, port, status, created_at, updated_at)
  VALUES ('claim-legacy', 'inst-legacy-7', 'legacy', 8200, 'bound', ?, ?)`, now, now); err != nil {
		t.Fatalf("seed v4 port claim error = %v", err)
	}
	if _, err := db.Exec(`PRAGMA user_version = 4`); err != nil {
		t.Fatalf("stamp v4 user_version error = %v", err)
	}
}

// schemaV4SQL is the frozen schema-version-4 shape of the two tables the
// migration rewrites (instances rebuilt, port claims ADD COLUMN). The FK from
// claims to instances is included so the migration's foreign_key_check is
// exercised against a real reference.
const schemaV4SQL = `
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
  host_boot_id TEXT NOT NULL DEFAULT '',
  host_session_id TEXT NOT NULL DEFAULT '',
  supervisor_id TEXT NOT NULL DEFAULT '',
  supervised_at TEXT,
  last_reconciled_at TEXT,
  reconciliation_status TEXT NOT NULL DEFAULT '',
  reconciliation_reason TEXT NOT NULL DEFAULT '',
  supervision_policy TEXT NOT NULL DEFAULT 'managed',
  schema_version INTEGER NOT NULL,
  UNIQUE(scenario, generation)
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
  last_bound_at TEXT,
  last_listener_check_at TEXT,
  last_listener_seen_at TEXT,
  first_unbound_at TEXT,
  consecutive_listener_misses INTEGER NOT NULL DEFAULT 0,
  listener_status TEXT NOT NULL DEFAULT 'unknown',
  listener_pid INTEGER,
  listener_process_label TEXT NOT NULL DEFAULT '',
  FOREIGN KEY(instance_id) REFERENCES runtime_instances(instance_id) ON DELETE CASCADE
);
`
