package targets

import (
	"context"
	"fmt"
)

// EnsureColumns applies additive target migrations to existing catalogs. The
// critical classification was added after the original target table shipped;
// old rows remain non-critical until an operator explicitly approves them.
func EnsureColumns(ctx context.Context, db SQLExecutor) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(targets)")
	if err != nil {
		return fmt.Errorf("inspect targets columns: %w", err)
	}
	defer rows.Close()
	seen := map[string]bool{}
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var def any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &def, &pk); err != nil {
			return fmt.Errorf("scan targets columns: %w", err)
		}
		seen[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate targets columns: %w", err)
	}
	if !seen["critical"] {
		if _, err := db.ExecContext(ctx, "ALTER TABLE targets ADD COLUMN critical INTEGER NOT NULL DEFAULT 0"); err != nil {
			return fmt.Errorf("add targets critical column: %w", err)
		}
	}
	return nil
}
