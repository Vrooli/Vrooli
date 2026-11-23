// Schema migrations for the scenario's SQLite database.
//
// The domain `schema.sql` files remain the declarative, forward-only shape
// (CREATE TABLE IF NOT EXISTS) applied by api-core's EnsureSchemas. Migrations
// exist for changes that declarative schemas cannot express — dropping a table,
// altering a column, backfilling data — and run BETWEEN schema application and
// the post-apply drift check.
//
// Ordering is lexical by filename. Each file must be idempotent-safe to re-run
// after a crash between execution and the version record (use IF EXISTS /
// IF NOT EXISTS where possible). Applied versions are recorded in
// `schema_migrations`; never edit or delete an applied migration — add a new one.
//
// Boot wiring: api/main.go calls ApplySchemas → Migrate → ReconcileDeclaredColumns
// (additive columns are added automatically; only non-additive changes need a migration).
package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

const (
	migrationsDir  = "migrations"
	migrationTable = "schema_migrations"
)

// MigrationNames returns the embedded migration filenames in apply order.
func MigrationNames() ([]string, error) { return migrationNames(migrationFS, migrationsDir) }

// Migrate applies every embedded migration that is not yet recorded as applied.
// It is safe to call on every boot and on a database that has none of the
// affected tables.
func Migrate(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("database: migrate: nil db")
	}
	return MigrateFS(ctx, db, migrationFS, migrationsDir)
}

// MigrateFS applies migrations from an explicit filesystem (the test seam).
func MigrateFS(ctx context.Context, db *sql.DB, fsys fs.FS, dir string) error {
	names, err := migrationNames(fsys, dir)
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS `+migrationTable+` (version TEXT PRIMARY KEY, applied_at_unix_ms INTEGER NOT NULL)`,
	); err != nil {
		return fmt.Errorf("database: ensure %s: %w", migrationTable, err)
	}
	for _, name := range names {
		applied, err := migrationApplied(ctx, db, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		body, err := fs.ReadFile(fsys, dir+"/"+name)
		if err != nil {
			return fmt.Errorf("database: read migration %s: %w", name, err)
		}
		if err := applyMigration(ctx, db, name, string(body)); err != nil {
			return err
		}
	}
	return nil
}

func migrationNames(fsys fs.FS, dir string) ([]string, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("database: list migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names, nil
}

func migrationApplied(ctx context.Context, db *sql.DB, name string) (bool, error) {
	var n int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM `+migrationTable+` WHERE version = ?`, name,
	).Scan(&n); err != nil {
		return false, fmt.Errorf("database: check migration %s: %w", name, err)
	}
	return n > 0, nil
}

// applyMigration runs one migration and records it in the same transaction, so a
// failure leaves neither partial schema changes nor a version claim.
func applyMigration(ctx context.Context, db *sql.DB, name, body string) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("database: begin migration %s: %w", name, err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, body); err != nil {
		return fmt.Errorf("database: apply migration %s: %w", name, err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO `+migrationTable+` (version, applied_at_unix_ms) VALUES (?, ?)`,
		name, time.Now().UnixMilli(),
	); err != nil {
		return fmt.Errorf("database: record migration %s: %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("database: commit migration %s: %w", name, err)
	}
	return nil
}
