package validation

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
)

//go:embed schema.sql
var schemaSQL string

// Schema returns the validation domain's SQL contribution (terminal results and
// durable validation-operation checkpoints). Applied by database.EnsureSchemas.
// Forward-only declarative — re-runs are no-ops.
func Schema() string { return schemaSQL }

// EnsureMigrations upgrades result receipts before EnsureSchemas verifies the
// declarative schema. It is safe to call at every stopped-service startup: each
// column is added only when missing. Legacy results intentionally receive safe
// zero values, so they cannot prove a later execution's validation.
func EnsureMigrations(ctx context.Context, db SQLExecutor) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(validation_results)")
	if err != nil {
		return fmt.Errorf("inspect validation_results columns: %w", err)
	}
	defer rows.Close()

	existing := make(map[string]bool)
	for rows.Next() {
		var (
			cid      int
			name     string
			dataType string
			notNull  int
			defaultV sql.NullString
			primary  int
		)
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultV, &primary); err != nil {
			return fmt.Errorf("scan validation_results columns: %w", err)
		}
		existing[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read validation_results columns: %w", err)
	}
	// The table does not exist on a fresh store; Schema creates its complete
	// shape immediately afterward.
	if len(existing) == 0 {
		return nil
	}

	for _, column := range resultBindingColumns {
		if existing[column.name] {
			continue
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE validation_results ADD COLUMN "+column.definition); err != nil {
			return fmt.Errorf("add validation_results.%s: %w", column.name, err)
		}
	}
	return nil
}

var resultBindingColumns = []struct {
	name       string
	definition string
}{
	{name: "execution_id", definition: "execution_id TEXT NOT NULL DEFAULT ''"},
	{name: "operation_id", definition: "operation_id TEXT NOT NULL DEFAULT ''"},
	{name: "scope_generation", definition: "scope_generation INTEGER NOT NULL DEFAULT 0"},
	{name: "full_inventory", definition: "full_inventory INTEGER NOT NULL DEFAULT 0"},
	{name: "required_members", definition: "required_members TEXT NOT NULL DEFAULT '[]'"},
	{name: "selected_members", definition: "selected_members TEXT NOT NULL DEFAULT '[]'"},
}
