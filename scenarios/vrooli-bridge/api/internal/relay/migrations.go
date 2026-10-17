package relay

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Migrate brings an existing relay journal to the current receipt shape. The
// schema provider handles fresh installs; this guarded migration is required
// because CREATE TABLE IF NOT EXISTS does not add columns to an older table.
func Migrate(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("relay migration database is required")
	}
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(relay_commands)`)
	if err != nil {
		return fmt.Errorf("inspect relay journal schema: %w", err)
	}
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var cid, notNull, pk int
		var name, kind string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &kind, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("read relay journal schema: %w", err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate relay journal schema: %w", err)
	}
	// Missing table is normal on a fresh database; EnsureSchemas creates it.
	if len(columns) == 0 {
		return nil
	}
	for _, column := range []string{
		"route_name TEXT NOT NULL DEFAULT ''",
		"route_cost_units INTEGER NOT NULL DEFAULT 0",
		"route_latency_ms INTEGER NOT NULL DEFAULT 0",
	} {
		name := column[:len(column)-len(" TEXT NOT NULL DEFAULT ''")]
		if name == "route_cost_units" || name == "route_latency_ms" {
			name = column[:len(column)-len(" INTEGER NOT NULL DEFAULT 0")]
		}
		if columns[name] {
			continue
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE relay_commands ADD COLUMN "+column); err != nil {
			// Another Bridge process may have completed this guarded migration
			// between PRAGMA table_info and ALTER TABLE. Treat that race as the
			// same idempotent state as observing the column up front.
			if strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
				columns[name] = true
				continue
			}
			return fmt.Errorf("add relay journal column %s: %w", name, err)
		}
	}
	return nil
}
