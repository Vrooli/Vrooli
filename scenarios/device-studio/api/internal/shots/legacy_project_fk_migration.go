package shots

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// RemoveLegacyProjectForeignKey migrates the original shots table in place.
// Shot and project IDs remain stored, but their lifecycle is owned by separate
// domains and must not be coupled by SQLite cascade behavior.
func RemoveLegacyProjectForeignKey(ctx context.Context, db *sql.DB) (retErr error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("open shot schema migration connection: %w", err)
	}
	defer conn.Close()

	rows, err := conn.QueryContext(ctx, `PRAGMA foreign_key_list('shots')`)
	if err != nil {
		return fmt.Errorf("inspect shots foreign keys: %w", err)
	}
	hasProjectForeignKey := false
	for rows.Next() {
		var id, seq int
		var table, from, to, onUpdate, onDelete, match string
		if err := rows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			rows.Close()
			return fmt.Errorf("read shots foreign key: %w", err)
		}
		if table == "projects" && from == "project_id" {
			hasProjectForeignKey = true
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("finish inspecting shots foreign keys: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close shots foreign key inspection: %w", err)
	}
	if !hasProjectForeignKey {
		return nil
	}

	// SQLite only permits changing this pragma outside a transaction. The
	// service opens one primary connection during startup, so the setting is
	// scoped to the connection that owns this atomic rebuild.
	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys=OFF`); err != nil {
		return fmt.Errorf("disable foreign keys for shot table rebuild: %w", err)
	}
	foreignKeysDisabled := true
	restoreForeignKeys := func() error {
		if !foreignKeysDisabled {
			return nil
		}
		restoreCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
		defer cancel()
		if _, err := conn.ExecContext(restoreCtx, `PRAGMA foreign_keys=ON`); err != nil {
			return fmt.Errorf("restore SQLite foreign key enforcement: %w", err)
		}
		foreignKeysDisabled = false
		return nil
	}
	defer func() {
		retErr = errors.Join(retErr, restoreForeignKeys())
	}()

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin shot table rebuild: %w", err)
	}
	defer tx.Rollback()

	const replacement = `CREATE TABLE shots_project_fk_migration (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		name TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		head_revision INTEGER NOT NULL CHECK (head_revision > 0)
	)`
	if _, err := tx.ExecContext(ctx, replacement); err != nil {
		return fmt.Errorf("create replacement shots table: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO shots_project_fk_migration
		(id, project_id, name, created_at, updated_at, head_revision)
		SELECT id, project_id, name, created_at, updated_at, head_revision FROM shots`); err != nil {
		return fmt.Errorf("copy existing shots: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DROP TABLE shots`); err != nil {
		return fmt.Errorf("replace old shots table: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `ALTER TABLE shots_project_fk_migration RENAME TO shots`); err != nil {
		return fmt.Errorf("install replacement shots table: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `CREATE INDEX idx_shots_project_created_at_id
		ON shots(project_id, created_at DESC, id DESC)`); err != nil {
		return fmt.Errorf("restore shots lookup index: %w", err)
	}
	checks, err := tx.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return fmt.Errorf("check foreign keys after shot rebuild: %w", err)
	}
	if checks.Next() {
		var table string
		var rowID sql.NullInt64
		var parent string
		var foreignKeyID int
		if scanErr := checks.Scan(&table, &rowID, &parent, &foreignKeyID); scanErr != nil {
			checks.Close()
			return fmt.Errorf("read post-migration foreign key violation: %w", scanErr)
		}
		checks.Close()
		return fmt.Errorf("shot schema migration would leave a foreign key violation in %s", table)
	}
	if err := checks.Err(); err != nil {
		checks.Close()
		return fmt.Errorf("finish post-migration foreign key check: %w", err)
	}
	if err := checks.Close(); err != nil {
		return fmt.Errorf("close post-migration foreign key check: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit shot table rebuild: %w", err)
	}
	return nil
}
