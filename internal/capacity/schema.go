package capacity

import (
	"context"
	"database/sql"
	"fmt"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS capacity_claims (
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
  yield_when_idle INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL,
  activity_state TEXT NOT NULL DEFAULT 'idle',
  generation INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  last_heartbeat_at TEXT,
  heartbeat_deadline_at TEXT,
  last_active_at TEXT,
  degrade_profile TEXT NOT NULL DEFAULT '',
  observed_bytes INTEGER NOT NULL DEFAULT 0,
  observed_peak_bytes INTEGER NOT NULL DEFAULT 0,
  observed_at TEXT,
  idle_unload_ttl_seconds INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_capacity_claims_owner ON capacity_claims(owner_kind, owner_id);
CREATE INDEX IF NOT EXISTS idx_capacity_claims_status ON capacity_claims(status);
CREATE INDEX IF NOT EXISTS idx_capacity_claims_resource ON capacity_claims(resource_kind, gpu_index);
CREATE INDEX IF NOT EXISTS idx_capacity_claims_heartbeat_deadline ON capacity_claims(heartbeat_deadline_at);
CREATE INDEX IF NOT EXISTS idx_capacity_claims_activity ON capacity_claims(activity_state);

CREATE TABLE IF NOT EXISTS capacity_policy (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
`

func (s *SQLiteStore) ensureSchema(ctx context.Context) error {
	current, err := readSchemaVersion(ctx, s.db)
	if err != nil {
		return fmt.Errorf("read capacity ledger schema version: %w", err)
	}
	if current > SchemaVersion {
		return fmt.Errorf("capacity ledger schema_version %d > expected %d: binary is older than database", current, SchemaVersion)
	}
	if current == SchemaVersion {
		return nil
	}
	if current != 0 {
		// A stamped older version: apply the additive, row-preserving migrations
		// up to SchemaVersion. Only purely-additive column adds are handled here;
		// anything else is rejected loudly rather than silently dropping claims.
		if err := s.migrateSchema(ctx, current); err != nil {
			return err
		}
		return nil
	}
	existing, err := capacitySchemaExists(ctx, s.db)
	if err != nil {
		return fmt.Errorf("inspect capacity ledger schema: %w", err)
	}
	if existing {
		return fmt.Errorf("capacity ledger schema is unstamped or incompatible with schema_version %d: requires greenfield rebuild or an operator-run temporary conversion script", SchemaVersion)
	}
	if _, err := s.db.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("apply capacity ledger schema: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`PRAGMA user_version = %d`, SchemaVersion)); err != nil {
		return fmt.Errorf("stamp capacity ledger schema version: %w", err)
	}
	return nil
}

// migrateSchema applies the additive, row-preserving migrations from `from` up
// to SchemaVersion, then re-stamps user_version. Each step is a column add (or
// other additive change) that never drops or rewrites a live claim; a version
// gap with no registered migration is an explicit error.
func (s *SQLiteStore) migrateSchema(ctx context.Context, from int) error {
	for v := from; v < SchemaVersion; v++ {
		switch v {
		case 1: // 1 -> 2: idle-yield opt-in column (additive).
			if err := addColumnIfMissing(ctx, s.db, "capacity_claims", "yield_when_idle", "INTEGER NOT NULL DEFAULT 0"); err != nil {
				return fmt.Errorf("migrate capacity ledger 1 -> 2: %w", err)
			}
		case 2: // 2 -> 3: observed-usage sampling + autonomous idle-unload (additive).
			for _, col := range []struct{ name, decl string }{
				{"observed_bytes", "INTEGER NOT NULL DEFAULT 0"},
				{"observed_peak_bytes", "INTEGER NOT NULL DEFAULT 0"},
				{"observed_at", "TEXT"},
				{"idle_unload_ttl_seconds", "INTEGER NOT NULL DEFAULT 0"},
			} {
				if err := addColumnIfMissing(ctx, s.db, "capacity_claims", col.name, col.decl); err != nil {
					return fmt.Errorf("migrate capacity ledger 2 -> 3: %w", err)
				}
			}
		default:
			return fmt.Errorf("capacity ledger schema_version %d -> %d has no additive migration: requires greenfield rebuild or an operator-run conversion script", v, SchemaVersion)
		}
	}
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`PRAGMA user_version = %d`, SchemaVersion)); err != nil {
		return fmt.Errorf("re-stamp capacity ledger schema version: %w", err)
	}
	return nil
}

// addColumnIfMissing adds a column only when it is absent, so the migration is
// idempotent across reruns (PRAGMA table_info is the source of truth).
func addColumnIfMissing(ctx context.Context, db *sql.DB, table, column, decl string) error {
	rows, err := db.QueryContext(ctx, fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return fmt.Errorf("inspect %s columns: %w", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid        int
			name       string
			ctype      string
			notNull    int
			dfltValue  sql.NullString
			primaryKey int
		)
		if scanErr := rows.Scan(&cid, &name, &ctype, &notNull, &dfltValue, &primaryKey); scanErr != nil {
			return fmt.Errorf("scan %s columns: %w", table, scanErr)
		}
		if name == column {
			return rows.Err() // already present — nothing to do
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, column, decl)); err != nil {
		return fmt.Errorf("add column %s.%s: %w", table, column, err)
	}
	return nil
}

func readSchemaVersion(ctx context.Context, db *sql.DB) (int, error) {
	var v int
	if err := db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&v); err != nil {
		return 0, err
	}
	return v, nil
}

func capacitySchemaExists(ctx context.Context, db *sql.DB) (bool, error) {
	rows, err := db.QueryContext(ctx, `
SELECT name
FROM sqlite_master
WHERE type = 'table'
  AND name IN ('capacity_claims', 'capacity_policy')`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	return rows.Next(), rows.Err()
}
