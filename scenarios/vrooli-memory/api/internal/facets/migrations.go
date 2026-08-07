package facets

import (
	"context"
	"database/sql"
	"fmt"
)

func EnsureMigrations(ctx context.Context, db *sql.DB) error {
	var exists int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='facet_policies'`).Scan(&exists); err != nil {
		return fmt.Errorf("inspect facet policies: %w", err)
	}
	if exists == 0 {
		return nil
	}
	// Brownfield databases predate assignment provenance. Preserve every
	// historical assignment while making its migration provenance explicit.
	if _, err := db.ExecContext(ctx, `UPDATE facet_assignments SET actor_id='migration:legacy-facet-assignment' WHERE actor_id=''`); err != nil {
		return fmt.Errorf("backfill facet assignment provenance: %w", err)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('facet_policies') WHERE name='resident_budget'`).Scan(&count); err != nil {
		return fmt.Errorf("inspect resident budget: %w", err)
	}
	if count == 0 {
		if _, err := db.ExecContext(ctx, `ALTER TABLE facet_policies ADD COLUMN resident_budget INTEGER NOT NULL DEFAULT 0`); err != nil {
			return fmt.Errorf("add resident budget: %w", err)
		}
	}
	if err := ensureColumn(ctx, db, "facet_definitions", "classification_guidance", `ALTER TABLE facet_definitions ADD COLUMN classification_guidance TEXT NOT NULL DEFAULT ''`); err != nil {
		return err
	}
	var proposalExists int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='merge_proposals'`).Scan(&proposalExists); err != nil {
		return fmt.Errorf("inspect merge proposals: %w", err)
	}
	if proposalExists != 0 {
		if err := ensureColumn(ctx, db, "merge_proposals", "scope", `ALTER TABLE merge_proposals ADD COLUMN scope TEXT NOT NULL DEFAULT 'agent-memory'`); err != nil {
			return err
		}
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('merge_proposals') WHERE name='entry_ids_json'`).Scan(&count); err != nil {
			return fmt.Errorf("inspect proposal entries: %w", err)
		}
		if count == 0 {
			if _, err := db.ExecContext(ctx, `ALTER TABLE merge_proposals ADD COLUMN entry_ids_json TEXT NOT NULL DEFAULT '[]'`); err != nil {
				return fmt.Errorf("add proposal entries: %w", err)
			}
		}
	}
	if pinsExists, err := tableExists(ctx, db, "pins"); err != nil {
		return err
	} else if pinsExists {
		for name, statement := range map[string]string{
			"review_at": `ALTER TABLE pins ADD COLUMN review_at TEXT`,
			"actor_id":  `ALTER TABLE pins ADD COLUMN actor_id TEXT NOT NULL DEFAULT ''`,
		} {
			if err := ensureColumn(ctx, db, "pins", name, statement); err != nil {
				return err
			}
		}
	}
	return nil
}

func ensureColumn(ctx context.Context, db *sql.DB, table, name, statement string) error {
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info(?) WHERE name=?`, table, name).Scan(&count); err != nil {
		return fmt.Errorf("inspect %s.%s: %w", table, name, err)
	}
	if count == 0 {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("add %s.%s: %w", table, name, err)
		}
	}
	return nil
}

func tableExists(ctx context.Context, db *sql.DB, table string) (bool, error) {
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count); err != nil {
		return false, fmt.Errorf("inspect %s table: %w", table, err)
	}
	return count > 0, nil
}
