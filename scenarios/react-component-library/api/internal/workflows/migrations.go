package workflows

import (
	"context"
	"database/sql"
	"fmt"
)

type schemaMigrator interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// EnsureSchemaMigrations preserves existing UI rows while adding the declared
// workflow execution reference used by new assisted operations.
func EnsureSchemaMigrations(ctx context.Context, db schemaMigrator) error {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(assisted_workflows)`)
	if err != nil {
		return fmt.Errorf("inspect assisted_workflows: %w", err)
	}
	defer rows.Close()
	hasExecutionID := false
	hasTable := false
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return fmt.Errorf("scan assisted_workflows column: %w", err)
		}
		hasTable = true
		hasExecutionID = hasExecutionID || name == "agent_manager_execution_id"
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read assisted_workflows columns: %w", err)
	}
	if !hasTable || hasExecutionID {
		return nil
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE assisted_workflows ADD COLUMN agent_manager_execution_id TEXT NOT NULL DEFAULT '';`); err != nil {
		return fmt.Errorf("migrate assisted_workflows.agent_manager_execution_id: %w", err)
	}
	return nil
}
