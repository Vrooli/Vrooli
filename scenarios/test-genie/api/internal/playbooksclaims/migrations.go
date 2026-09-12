package playbooksclaims

import (
	"context"
	"fmt"

	"test-genie/internal/dbexec"
)

// Migrate adds typed target identity to databases created before claims were
// generalized. The legacy scenario primary key remains harmless; the target
// unique index is the conflict target used by current acquire operations.
func Migrate(ctx context.Context, db dbexec.Executor) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(playbooks_claims)")
	if err != nil {
		return err
	}
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, typ string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for name, statement := range map[string]string{
		"target_kind": "ALTER TABLE playbooks_claims ADD COLUMN target_kind TEXT NOT NULL DEFAULT 'scenario'",
		"target_id":   "ALTER TABLE playbooks_claims ADD COLUMN target_id TEXT NOT NULL DEFAULT ''",
	} {
		if columns[name] {
			continue
		}
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("add %s: %w", name, err)
		}
	}
	if _, err := db.ExecContext(ctx, `UPDATE playbooks_claims SET target_kind = 'scenario' WHERE TRIM(target_kind) = ''`); err != nil {
		return fmt.Errorf("backfill claim target kinds: %w", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE playbooks_claims SET target_id = scenario_name WHERE TRIM(target_id) = ''`); err != nil {
		return fmt.Errorf("backfill claim target ids: %w", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS playbooks_claims_target_idx ON playbooks_claims(target_kind, target_id)`); err != nil {
		return fmt.Errorf("create target claim index: %w", err)
	}
	return nil
}
