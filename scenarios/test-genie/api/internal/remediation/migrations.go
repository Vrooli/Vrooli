package remediation

import (
	"context"
	"encoding/json"
	"fmt"

	"test-genie/internal/dbexec"
)

// Migrate evolves durable remediation rows without recreating the table. A
// selected requirement is part of the immutable operator intent and must
// survive restart just like a selected finding.
func Migrate(ctx context.Context, db dbexec.Executor) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(remediation_jobs)")
	if err != nil {
		return err
	}
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue any
		var primaryKey int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for name, statement := range map[string]string{
		"selected_requirement_ids_json": "ALTER TABLE remediation_jobs ADD COLUMN selected_requirement_ids_json TEXT NOT NULL DEFAULT '[]'",
		"source_hash":                   "ALTER TABLE remediation_jobs ADD COLUMN source_hash TEXT NOT NULL DEFAULT ''",
		"selection_hash":                "ALTER TABLE remediation_jobs ADD COLUMN selection_hash TEXT NOT NULL DEFAULT ''",
		"launch_attempt":                "ALTER TABLE remediation_jobs ADD COLUMN launch_attempt INTEGER NOT NULL DEFAULT 0",
	} {
		if columns[name] {
			continue
		}
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("add %s: %w", name, err)
		}
	}
	// CREATE INDEX IF NOT EXISTS cannot change the partial predicate on an
	// existing database. Rebuild it so a pending launch retains the one-active
	//-job invariant after an upgrade.
	if _, err := db.ExecContext(ctx, `DROP INDEX IF EXISTS remediation_jobs_one_active_per_scenario`); err != nil {
		return fmt.Errorf("replace active-job index: %w", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE UNIQUE INDEX remediation_jobs_one_active_per_scenario ON remediation_jobs(scenario_name) WHERE status IN ('created', 'launch_pending', 'running', 'agent_completed', 'verification_running')`); err != nil {
		return fmt.Errorf("create active-job index: %w", err)
	}

	return backfillImmutableHashes(ctx, db)
}

func backfillImmutableHashes(ctx context.Context, db dbexec.Executor) error {
	rows, err := db.QueryContext(ctx, `SELECT id, source_json, selected_finding_ids_json, selected_requirement_ids_json, source_hash, selection_hash FROM remediation_jobs`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type row struct {
		id, sourceJSON, findingJSON, requirementJSON, sourceHash, selectionHash string
	}
	var pending []row
	for rows.Next() {
		var id, sourceJSON, findingJSON, requirementJSON, storedSourceHash, storedSelectionHash string
		if err := rows.Scan(&id, &sourceJSON, &findingJSON, &requirementJSON, &storedSourceHash, &storedSelectionHash); err != nil {
			return err
		}
		if storedSourceHash != "" && storedSelectionHash != "" {
			continue
		}
		pending = append(pending, row{id: id, sourceJSON: sourceJSON, findingJSON: findingJSON, requirementJSON: requirementJSON, sourceHash: storedSourceHash, selectionHash: storedSelectionHash})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, stored := range pending {
		var source Plan
		var findings, requirements []string
		if err := json.Unmarshal([]byte(stored.sourceJSON), &source); err != nil {
			return fmt.Errorf("decode source for job %s: %w", stored.id, err)
		}
		if err := json.Unmarshal([]byte(stored.findingJSON), &findings); err != nil {
			return fmt.Errorf("decode findings for job %s: %w", stored.id, err)
		}
		if err := json.Unmarshal([]byte(stored.requirementJSON), &requirements); err != nil {
			return fmt.Errorf("decode requirements for job %s: %w", stored.id, err)
		}
		if _, err := db.ExecContext(ctx, `UPDATE remediation_jobs SET source_hash = ?, selection_hash = ? WHERE id = ?`, sourceHash(source), selectionHash(findings, requirements), stored.id); err != nil {
			return fmt.Errorf("backfill immutable hashes for job %s: %w", stored.id, err)
		}
	}
	return nil
}
