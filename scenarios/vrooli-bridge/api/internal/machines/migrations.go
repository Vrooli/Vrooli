package machines

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Migrate brings an existing Machine schema to the declared shape before
// EnsureSchemas runs its drift check. A fresh database has no review table yet,
// so it remains a no-op and EnsureSchemas creates the complete schema.
func Migrate(ctx context.Context, db SQLExecutor) error {
	exists, err := migrationTableExists(ctx, db, "machine_migration_reviews")
	if err != nil {
		return err
	}
	if exists {
		var found int
		err = db.QueryRowContext(ctx, "SELECT 1 FROM pragma_table_info('machine_migration_reviews') WHERE name='confidence'").Scan(&found)
		if errors.Is(err, sql.ErrNoRows) {
			if _, err = db.ExecContext(ctx, "ALTER TABLE machine_migration_reviews ADD COLUMN confidence TEXT NOT NULL DEFAULT 'ambiguous'"); err != nil {
				return fmt.Errorf("add machine_migration_reviews.confidence: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("inspect machine_migration_reviews.confidence: %w", err)
		}
	}
	return reconcileDuplicateCurrentNodes(ctx, db)
}

// reconcileDuplicateCurrentNodes is deliberately run before the global
// partial unique index is created by schema.sql. Legacy Bridge versions only
// enforced one current node per machine, so one node could be current in many
// machines. The newest lineage remains authoritative; every superseded row is
// retained and gets an audit event.
func reconcileDuplicateCurrentNodes(ctx context.Context, db SQLExecutor) error {
	if ok, err := migrationTableExists(ctx, db, "machine_node_lineage"); err != nil || !ok {
		return err
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS machine_audit_events (
        id TEXT PRIMARY KEY, machine_id TEXT NOT NULL, action TEXT NOT NULL,
        actor TEXT NOT NULL DEFAULT '', detail TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL)`); err != nil {
		return fmt.Errorf("prepare machine migration audit: %w", err)
	}
	rows, err := db.QueryContext(ctx, `SELECT node_id FROM machine_node_lineage WHERE is_current=1 GROUP BY node_id HAVING COUNT(*) > 1`)
	if err != nil {
		return fmt.Errorf("find duplicate current nodes: %w", err)
	}
	defer rows.Close()
	var nodeIDs []string
	for rows.Next() {
		var nodeID string
		if err := rows.Scan(&nodeID); err != nil {
			return err
		}
		nodeIDs = append(nodeIDs, nodeID)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	now := time.Now().UTC().Format(machineTimeFormat)
	for _, nodeID := range nodeIDs {
		var keepMachine, keepLinked string
		if err := db.QueryRowContext(ctx, `SELECT machine_id, linked_at FROM machine_node_lineage WHERE node_id=? AND is_current=1 ORDER BY linked_at DESC, id DESC LIMIT 1`, nodeID).Scan(&keepMachine, &keepLinked); err != nil {
			return fmt.Errorf("select current lineage for node %q: %w", nodeID, err)
		}
		dupes, err := db.QueryContext(ctx, `SELECT id,machine_id FROM machine_node_lineage WHERE node_id=? AND is_current=1 AND NOT (machine_id=? AND linked_at=?)`, nodeID, keepMachine, keepLinked)
		if err != nil {
			return fmt.Errorf("select duplicate lineage for node %q: %w", nodeID, err)
		}
		defer dupes.Close()
		type duplicate struct{ id, machine string }
		var items []duplicate
		for dupes.Next() {
			var item duplicate
			if err := dupes.Scan(&item.id, &item.machine); err != nil {
				dupes.Close()
				return err
			}
			items = append(items, item)
		}
		if err := dupes.Close(); err != nil {
			return err
		}
		for _, item := range items {
			if _, err := db.ExecContext(ctx, `UPDATE machine_node_lineage SET is_current=0,superseded_at=? WHERE id=?`, now, item.id); err != nil {
				return fmt.Errorf("supersede duplicate node %q: %w", nodeID, err)
			}
			detail := fmt.Sprintf("node_id=%s superseded_machine=%s retained_machine=%s", nodeID, item.machine, keepMachine)
			if _, err := db.ExecContext(ctx, `INSERT INTO machine_audit_events (id,machine_id,action,actor,detail,created_at) VALUES (?,?,?,?,?,?)`, uuid.NewString(), item.machine, "migration_supersede_duplicate_node", "system:migration", detail, now); err != nil {
				return fmt.Errorf("audit duplicate node %q: %w", nodeID, err)
			}
		}
	}
	return nil
}

// BackfillLegacy records legacy Registry Nodes and onboarding operations as
// reviewable evidence. Neither legacy record contains the durable pairing
// correlation needed to prove which pre-contact Machine it belongs to, so this
// deliberately creates no Machine and never rewrites historic records.
//
// It is safe on every boot: the unique source/id key makes each evidence record
// immutable and idempotent. Call it after EnsureSchemas, when this domain's
// review table is guaranteed to exist.
func BackfillLegacy(ctx context.Context, db SQLExecutor) error {
	now := time.Now().UTC().Format(machineTimeFormat)
	for _, source := range []struct {
		table  string
		reason string
	}{
		{"nodes", "registry node has no durable Machine enrollment correlation"},
		{"onboarding_ops", "legacy onboarding operation has no durable Machine enrollment correlation"},
	} {
		exists, err := migrationTableExists(ctx, db, source.table)
		if err != nil {
			return fmt.Errorf("inspect legacy %s: %w", source.table, err)
		}
		if !exists {
			continue
		}
		query := fmt.Sprintf(`INSERT OR IGNORE INTO machine_migration_reviews
            (id,legacy_source,legacy_id,status,confidence,reason,created_at)
            SELECT 'legacy:' || ? || ':' || id, ?, id, 'needs_review', 'ambiguous', ?, ? FROM %s`, source.table)
		if _, err := db.ExecContext(ctx, query, source.table, source.table, source.reason, now); err != nil {
			return fmt.Errorf("preserve legacy %s for review: %w", source.table, err)
		}
	}
	return nil
}

func migrationTableExists(ctx context.Context, db SQLExecutor, table string) (bool, error) {
	var one int
	err := db.QueryRowContext(ctx,
		"SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&one)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}
	return false, err
}
