package capacity

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

// TestSchemaMigrationV1ToCurrentPreservesRows proves additive migrations preserve
// live claims, default new opt-in columns to disabled, and re-stamp the schema.
func TestSchemaMigrationV1ToCurrentPreservesRows(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "capacity.db")
	ctx := context.Background()

	// Build a v1 ledger by hand: the v2 schema minus yield_when_idle, stamped at
	// user_version = 1, holding one resident claim.
	db, err := sql.Open("sqlite", buildDSN(dbPath))
	if err != nil {
		t.Fatalf("open v1 db: %v", err)
	}
	const v1Schema = `
CREATE TABLE capacity_claims (
  claim_id TEXT PRIMARY KEY,
  owner_kind TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  instance_id TEXT NOT NULL DEFAULT '',
  resource_kind TEXT NOT NULL,
  gpu_index INTEGER,
  amount_bytes INTEGER NOT NULL DEFAULT 0,
  preferred_bytes INTEGER NOT NULL DEFAULT 0,
  floor_bytes INTEGER NOT NULL DEFAULT 0,
  priority INTEGER NOT NULL DEFAULT 10,
  protected INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL,
  activity_state TEXT NOT NULL DEFAULT 'idle',
  generation INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  last_heartbeat_at TEXT,
  heartbeat_deadline_at TEXT,
  last_active_at TEXT,
  degrade_profile TEXT NOT NULL DEFAULT ''
);
CREATE TABLE capacity_policy (key TEXT PRIMARY KEY, value TEXT NOT NULL, updated_at TEXT NOT NULL);`
	if _, err := db.ExecContext(ctx, v1Schema); err != nil {
		t.Fatalf("apply v1 schema: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO capacity_claims
(claim_id, owner_kind, owner_id, resource_kind, amount_bytes, priority, protected, status, created_at, updated_at)
VALUES ('clm-legacy','resource','whisper','vram',8589934592,30,1,'granted','2026-06-22T00:00:00.000000000Z','2026-06-22T00:00:00.000000000Z')`); err != nil {
		t.Fatalf("insert v1 row: %v", err)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA user_version = 1`); err != nil {
		t.Fatalf("stamp v1: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close v1 db: %v", err)
	}

	// Reopen via the production store: it must migrate 1 -> 2 in place.
	store, err := NewSQLiteStore(ctx, Config{DBPath: dbPath})
	if err != nil {
		t.Fatalf("reopen + migrate: %v", err)
	}
	defer store.Close()

	got, err := store.GetClaim(ctx, "clm-legacy")
	if err != nil {
		t.Fatalf("legacy claim must survive migration: %v", err)
	}
	if got.OwnerID != "whisper" || got.AmountBytes != 8589934592 {
		t.Errorf("legacy claim corrupted by migration: %+v", got)
	}
	if got.YieldWhenIdle {
		t.Error("migrated legacy claim should default yield_when_idle = false")
	}
	if got.IdleUnloadTTL != 0 {
		t.Errorf("migrated legacy claim idle_unload_ttl = %s, want 0", got.IdleUnloadTTL)
	}
	if got.IdleGrace != 0 {
		t.Errorf("migrated legacy claim idle_grace = %s, want 0", got.IdleGrace)
	}

	if v, verr := readSchemaVersion(ctx, store.db); verr != nil {
		t.Fatalf("read version: %v", verr)
	} else if v != SchemaVersion {
		t.Errorf("user_version = %d, want %d after migration", v, SchemaVersion)
	}

	// A new claim can set the new column and it round-trips.
	created, err := store.CreateClaim(ctx, CapacityClaim{OwnerID: "x", ResourceKind: ResourceKindVRAM, YieldWhenIdle: true, IdleGrace: 15 * time.Minute}, 0)
	if err != nil {
		t.Fatalf("create post-migration claim: %v", err)
	}
	reread, err := store.GetClaim(ctx, created.ClaimID)
	if err != nil {
		t.Fatalf("reread post-migration claim: %v", err)
	}
	if !reread.YieldWhenIdle {
		t.Error("post-migration claim must persist yield_when_idle = true")
	}
	if reread.IdleGrace != 15*time.Minute {
		t.Errorf("post-migration claim idle_grace = %s, want 15m", reread.IdleGrace)
	}
}
