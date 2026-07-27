// Package repository provides database operations for sandboxes.
package repository

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"time"

	"workspace-sandbox/internal/clock"
)

//go:embed schema.sql
var SchemaSQL string

// ExpectedSchemaVersion is the schema generation this binary expects in
// the embedded SQLite store. Bumped whenever the schema changes in a
// way that requires a new forward-only migration step.
//
// Round 4 Phase 9 (2026-04-29): introduced as version 1. Earlier
// databases (no schema_version row) are treated as the same canonical
// state because the prior schema is a strict subset of the current one
// — we only ever add columns/tables, never drop or rename in-place.
//
// Round 5 (2026-04-29): bumped to version 2. Adds the
// sandbox_diff_archives table and its indexes (see schema.sql) for
// durable diff snapshots taken at terminal status transitions. Pure
// additive change — the new table is created via IF NOT EXISTS, no
// existing tables are altered.
const ExpectedSchemaVersion = 3

// EnsureSchema applies the embedded schema and records the schema
// version. It is the single startup entry point for all DDL: legacy
// column migrations (driver_id rename, home_overlay_state add) live
// alongside the CREATE TABLE statements so callers see exactly one
// "make the database match the binary" call.
//
// Behavior:
//
//  1. Apply schema.sql (idempotent — every CREATE is IF NOT EXISTS).
//  2. Run legacy column migrations (idempotent — both probe before
//     mutating).
//  3. Inspect schema_version. If empty, write ExpectedSchemaVersion
//     stamped via clk. If present and < expected, refuse to start.
//     If > expected, refuse to start (binary older than DB).
//
// The version check is the loud-failure guard. Forward-only by design:
// there is no rollback path because we never need one — single-tenant
// local-dev, idempotent DDL, no production cluster fan-out.
//
// Returns an error annotated with the persisted/expected versions when
// drift is detected so operators get an actionable message instead of
// a confusing mid-operation failure later.
func EnsureSchema(ctx context.Context, db *sql.DB, clk clock.Clock) error {
	if db == nil {
		return fmt.Errorf("EnsureSchema: db is nil")
	}
	if clk == nil {
		return fmt.Errorf("EnsureSchema: clock is nil")
	}

	if _, err := db.ExecContext(ctx, SchemaSQL); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	if err := migrateDriverColumn(ctx, db); err != nil {
		return err
	}
	if err := migrateHomeOverlayStateColumn(ctx, db); err != nil {
		return err
	}

	current, err := readSchemaVersion(ctx, db)
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	if current > ExpectedSchemaVersion {
		return fmt.Errorf(
			"schema_version %d > expected %d: binary is older than database — refusing to start",
			current, ExpectedSchemaVersion,
		)
	}

	if current == ExpectedSchemaVersion {
		return nil
	}

	// Walk forward through every required version step. Each migrator
	// is responsible for any DDL its version needs *beyond* what the
	// embedded schema.sql already applies — the embedded schema is
	// always applied first via IF NOT EXISTS, so pure-additive table
	// creates are no-ops here and the migrator only needs to bump the
	// recorded version. The schema_version table holds exactly one
	// row at the latest applied version (matching the existing single-
	// row contract); writeSchemaVersion at the end of the walk records
	// the final value.
	for v := current + 1; v <= ExpectedSchemaVersion; v++ {
		mig, ok := migrations[v]
		if !ok {
			return fmt.Errorf(
				"schema_version %d → %d: forward-only migration missing — refusing to start",
				current, v,
			)
		}
		if err := mig(ctx, db); err != nil {
			return fmt.Errorf("migrate schema to v%d: %w", v, err)
		}
	}

	if err := stampSchemaVersion(ctx, db, ExpectedSchemaVersion, clk.Now()); err != nil {
		return fmt.Errorf("stamp schema_version: %w", err)
	}
	return nil
}

// stampSchemaVersion replaces the schema_version table contents with
// a single row pinned to version. Idempotent: re-stamping the same
// version is a no-op (the existing row's applied_at remains, since
// we only write when the table is empty or the version differs).
func stampSchemaVersion(ctx context.Context, db *sql.DB, version int, t time.Time) error {
	current, err := readSchemaVersion(ctx, db)
	if err != nil {
		return err
	}
	if current == version {
		return nil
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM schema_version`); err != nil {
		return fmt.Errorf("clear schema_version: %w", err)
	}
	return writeSchemaVersion(ctx, db, version, t)
}

// migrations registers each forward-only step keyed by the target
// version. EnsureSchema invokes them in order from current+1 up to
// ExpectedSchemaVersion. Each migrator runs *after* the embedded
// schema.sql has already been applied, so it should only contain the
// extra DDL that schema.sql can't express idempotently (renames,
// data backfills, etc.). For pure additive changes the migrator may
// be a no-op — the version row itself records the bump.
var migrations = map[int]func(context.Context, *sql.DB) error{
	1: migrateToV1,
	2: migrateToV2,
	3: migrateToV3,
}

// migrateToV1 marks an empty database as v1. The legacy column
// migrations (migrateDriverColumn, migrateHomeOverlayStateColumn)
// already ran before this point in EnsureSchema, so the schema is
// guaranteed up-to-date.
func migrateToV1(_ context.Context, _ *sql.DB) error {
	return nil
}

// migrateToV2 brings the DB up to schema version 2, which adds the
// sandbox_diff_archives table for durable diff snapshots taken at
// terminal status transitions. The table is purely additive and is
// created by the embedded schema.sql via IF NOT EXISTS, so this
// migrator is a no-op — but it must exist so the version walker
// recognizes v1 → v2 as a registered step rather than an unknown bump.
func migrateToV2(_ context.Context, _ *sql.DB) error {
	return nil
}

func migrateToV3(ctx context.Context, db *sql.DB) error {
	var exists int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('sandboxes') WHERE name='auxiliary_roots'`).Scan(&exists); err != nil {
		return fmt.Errorf("probe auxiliary_roots column: %w", err)
	}
	if exists > 0 {
		return nil
	}
	_, err := db.ExecContext(ctx, `ALTER TABLE sandboxes ADD COLUMN auxiliary_roots TEXT NOT NULL DEFAULT '[]'`)
	return err
}

// readSchemaVersion returns the highest persisted schema_version row,
// or 0 when the table is empty. A NULL result (empty table) and a
// missing-row condition both map to 0 so the caller can fall through
// to the "first init" branch uniformly.
func readSchemaVersion(ctx context.Context, db *sql.DB) (int, error) {
	var v sql.NullInt64
	err := db.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_version`).Scan(&v)
	if err != nil {
		return 0, err
	}
	if !v.Valid {
		return 0, nil
	}
	return int(v.Int64), nil
}

// writeSchemaVersion records the version stamped at t. Uses INSERT OR
// IGNORE so a concurrent boot (theoretically not possible — SQLite
// serializes writes — but defensive) doesn't trip a UNIQUE violation.
func writeSchemaVersion(ctx context.Context, db *sql.DB, version int, t time.Time) error {
	_, err := db.ExecContext(ctx,
		`INSERT OR IGNORE INTO schema_version (version, applied_at) VALUES (?, ?)`,
		version, t.UTC().Format(time.RFC3339Nano),
	)
	return err
}

// migrateDriverColumn renames the legacy `driver` column to `driver_id`
// when an older DB is encountered, and backfills `overlayfs` →
// `overlayfs-userns`. Idempotent: both steps are no-ops on fresh DBs
// where the schema landed `driver_id` directly. Greenfield: no rollback
// path; the new ID space is the canonical truth.
func migrateDriverColumn(ctx context.Context, db *sql.DB) error {
	var oldColumnExists int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pragma_table_info('sandboxes') WHERE name='driver'`,
	).Scan(&oldColumnExists)
	if err != nil {
		return fmt.Errorf("probe driver column: %w", err)
	}
	if oldColumnExists > 0 {
		if _, err := db.ExecContext(ctx, `ALTER TABLE sandboxes RENAME COLUMN driver TO driver_id`); err != nil {
			return fmt.Errorf("rename driver column: %w", err)
		}
	}
	if _, err := db.ExecContext(ctx,
		`UPDATE sandboxes SET driver_id = 'overlayfs-userns' WHERE driver_id = 'overlayfs'`,
	); err != nil {
		return fmt.Errorf("backfill driver_id: %w", err)
	}
	return nil
}

// migrateHomeOverlayStateColumn idempotently adds the home_overlay_state
// column to older sandboxes tables. Fresh databases land the column via
// CREATE TABLE; this function only fires on pre-2026-04-29 databases
// that predate the home-overlay refactor.
func migrateHomeOverlayStateColumn(ctx context.Context, db *sql.DB) error {
	var exists int
	err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pragma_table_info('sandboxes') WHERE name='home_overlay_state'`,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("probe home_overlay_state column: %w", err)
	}
	if exists > 0 {
		return nil
	}
	if _, err := db.ExecContext(ctx,
		`ALTER TABLE sandboxes ADD COLUMN home_overlay_state TEXT NOT NULL DEFAULT 'absent'`,
	); err != nil {
		return fmt.Errorf("add home_overlay_state column: %w", err)
	}
	return nil
}
