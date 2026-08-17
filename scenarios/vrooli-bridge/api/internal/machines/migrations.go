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
	if err := ensureLocatorClaims(ctx, db); err != nil {
		return err
	}
	if err := addMachineTrustColumns(ctx, db); err != nil {
		return err
	}
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
	if err := reconcileDuplicateCurrentNodes(ctx, db); err != nil {
		return err
	}
	return migrateLegacyCleanupTombstones(ctx, db)
}

// migrateLegacyCleanupTombstones turns pre-operation cleanup records into
// honest terminal history. They never executed a remote effect, so pending
// must not survive startup as if work were still waiting to run.
func migrateLegacyCleanupTombstones(ctx context.Context, db SQLExecutor) error {
	exists, err := migrationTableExists(ctx, db, "machine_cleanup_tombstones")
	if err != nil || !exists {
		return err
	}
	_, err = db.ExecContext(ctx, `UPDATE machine_cleanup_tombstones
SET action='cleanup_record',
    status='not_applicable',
    detail=CASE WHEN trim(detail) <> '' THEN detail || '; legacy record migrated: no remote cleanup was executed' ELSE 'legacy record migrated: no remote cleanup was executed' END
WHERE action='remove_ssh_access' AND status='pending'`)
	if err != nil {
		return fmt.Errorf("migrate legacy cleanup tombstones: %w", err)
	}
	return nil
}

// ensureLocatorClaims creates the active-identity uniqueness boundary for
// databases created before durable Machine creation was made idempotent. When
// legacy active duplicates exist, the oldest active record is retained and
// newer siblings are archived with their history preserved for review.
func ensureLocatorClaims(ctx context.Context, db SQLExecutor) error {
	for _, table := range []string{"machines", "machine_locators"} {
		exists, err := migrationTableExists(ctx, db, table)
		if err != nil {
			return err
		}
		if !exists {
			return nil
		}
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS machine_locator_claims (
        kind TEXT NOT NULL, normalized_value TEXT NOT NULL, machine_id TEXT NOT NULL,
        created_at TEXT NOT NULL, PRIMARY KEY(kind, normalized_value),
        UNIQUE(machine_id, kind, normalized_value))`); err != nil {
		return fmt.Errorf("prepare machine locator claims: %w", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_machine_locator_claims_machine ON machine_locator_claims(machine_id)`); err != nil {
		return fmt.Errorf("index machine locator claims: %w", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS machine_audit_events (
        id TEXT PRIMARY KEY, machine_id TEXT NOT NULL, action TEXT NOT NULL,
        actor TEXT NOT NULL DEFAULT '', detail TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL)`); err != nil {
		return fmt.Errorf("prepare machine locator migration audit: %w", err)
	}
	rows, err := db.QueryContext(ctx, `SELECT l.kind,l.normalized_value FROM machine_locators l JOIN machines m ON m.id=l.machine_id WHERE m.lifecycle='active' GROUP BY l.kind,l.normalized_value HAVING COUNT(*)>1`)
	if err != nil {
		return fmt.Errorf("find duplicate active machine locators: %w", err)
	}
	defer rows.Close()
	type locator struct{ kind, normalized string }
	var duplicates []locator
	for rows.Next() {
		var item locator
		if err := rows.Scan(&item.kind, &item.normalized); err != nil {
			return err
		}
		duplicates = append(duplicates, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	now := time.Now().UTC().Format(machineTimeFormat)
	for _, duplicate := range duplicates {
		var keep string
		if err := db.QueryRowContext(ctx, `SELECT m.id FROM machines m JOIN machine_locators l ON l.machine_id=m.id WHERE m.lifecycle='active' AND l.kind=? AND l.normalized_value=? ORDER BY m.created_at,m.id LIMIT 1`, duplicate.kind, duplicate.normalized).Scan(&keep); err != nil {
			return fmt.Errorf("select retained machine for %s=%s: %w", duplicate.kind, duplicate.normalized, err)
		}
		losers, err := db.QueryContext(ctx, `SELECT m.id FROM machines m JOIN machine_locators l ON l.machine_id=m.id WHERE m.lifecycle='active' AND l.kind=? AND l.normalized_value=? AND m.id<>?`, duplicate.kind, duplicate.normalized, keep)
		if err != nil {
			return fmt.Errorf("select duplicate machines for %s=%s: %w", duplicate.kind, duplicate.normalized, err)
		}
		var ids []string
		for losers.Next() {
			var id string
			if err := losers.Scan(&id); err != nil {
				losers.Close()
				return err
			}
			ids = append(ids, id)
		}
		losers.Close()
		for _, loser := range ids {
			if _, err := db.ExecContext(ctx, "UPDATE machines SET lifecycle='archived',version=version+1,archived_at=?,updated_at=? WHERE id=? AND lifecycle='active'", now, now, loser); err != nil {
				return fmt.Errorf("archive duplicate Machine %s: %w", loser, err)
			}
			if _, err := db.ExecContext(ctx, `INSERT INTO machine_audit_events (id,machine_id,action,actor,detail,created_at) VALUES (?,?,?,?,?,?)`, uuid.NewString(), keep, "migration_merge_duplicate_locator", "system:migration", fmt.Sprintf("archived_machine=%s locator=%s=%s", loser, duplicate.kind, duplicate.normalized), now); err != nil {
				return fmt.Errorf("audit duplicate Machine %s: %w", loser, err)
			}
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT OR IGNORE INTO machine_locator_claims(kind,normalized_value,machine_id,created_at) SELECT l.kind,l.normalized_value,l.machine_id,l.created_at FROM machine_locators l JOIN machines m ON m.id=l.machine_id WHERE m.lifecycle='active'`); err != nil {
		return fmt.Errorf("backfill machine locator claims: %w", err)
	}
	return nil
}

func addMachineTrustColumns(ctx context.Context, db SQLExecutor) error {
	exists, err := migrationTableExists(ctx, db, "machine_trust")
	if err != nil || !exists {
		return err
	}
	columns := []struct {
		name string
		ddl  string
	}{
		{"ssh_user", "ALTER TABLE machine_trust ADD COLUMN ssh_user TEXT NOT NULL DEFAULT ''"},
		{"ssh_port", "ALTER TABLE machine_trust ADD COLUMN ssh_port INTEGER NOT NULL DEFAULT 22"},
		{"connection_state", "ALTER TABLE machine_trust ADD COLUMN connection_state TEXT NOT NULL DEFAULT 'untrusted'"},
	}
	for _, column := range columns {
		var found int
		err := db.QueryRowContext(ctx, "SELECT 1 FROM pragma_table_info('machine_trust') WHERE name=?", column.name).Scan(&found)
		if errors.Is(err, sql.ErrNoRows) {
			if _, err := db.ExecContext(ctx, column.ddl); err != nil {
				return fmt.Errorf("add machine_trust.%s: %w", column.name, err)
			}
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect machine_trust.%s: %w", column.name, err)
		}
	}
	return nil
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
