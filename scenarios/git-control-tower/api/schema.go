package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"git-control-tower/internal/dbschema"

	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
)

func ensureAuditSchema(db *sql.DB) error {
	return ensureDatabaseSchema(db)
}

func ensureRepoSchema(db *sql.DB) error {
	if err := ensureDatabaseSchema(db); err != nil {
		return err
	}
	if db == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return ensurePrecommitHookColumns(ctx, db)
}

func ensureDatabaseSchema(db *sql.DB) error {
	if db == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := coredb.EnsureSchemas(ctx, db, coredb.SchemaProviderFunc(dbschema.Schema)); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}

func ensurePrecommitHookColumns(ctx context.Context, db *sql.DB) error {
	cols, err := tableColumns(ctx, db, "git_repo_precommit")
	if err != nil {
		return err
	}
	additions := []struct {
		name string
		ddl  string
	}{
		{"hook_install_status", "ALTER TABLE git_repo_precommit ADD COLUMN hook_install_status TEXT"},
		{"hook_install_reason", "ALTER TABLE git_repo_precommit ADD COLUMN hook_install_reason TEXT"},
		{"hook_existing_kind", "ALTER TABLE git_repo_precommit ADD COLUMN hook_existing_kind TEXT"},
		{"hook_installed_at", "ALTER TABLE git_repo_precommit ADD COLUMN hook_installed_at TEXT"},
	}
	for _, addition := range additions {
		if cols[addition.name] {
			continue
		}
		if _, err := db.ExecContext(ctx, addition.ddl); err != nil {
			return fmt.Errorf("add column %s: %w", addition.name, err)
		}
	}
	return nil
}

func tableColumns(ctx context.Context, db *sql.DB, table string) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return nil, fmt.Errorf("inspect table %s: %w", table, err)
	}
	defer rows.Close()
	cols := map[string]bool{}
	for rows.Next() {
		var (
			cid     int
			name    string
			ctype   string
			notnull int
			dflt    sql.NullString
			pk      int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, fmt.Errorf("scan table_info(%s): %w", table, err)
		}
		cols[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate table_info(%s): %w", table, err)
	}
	return cols, nil
}

// sqliteDSN returns the DSN for git-control-tower's own database.
//
// The tuning deviates from the fleet default in two typed ways, both suited to
// a repository index that is read far more than it is written: a 4 KiB page
// size, and a 256 MiB memory map so reads bypass the page cache.
func sqliteDSN() (string, error) {
	return storage.SQLiteDSN(storage.SQLiteConfig{
		Scenario: "git-control-tower",
		Tuning: storage.SQLiteTuning{
			PageSizeBytes: 4096,
			MMapSizeBytes: 268435456,
		},
	})
}
