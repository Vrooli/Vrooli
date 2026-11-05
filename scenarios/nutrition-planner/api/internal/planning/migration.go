package planning

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const legacyPlanMigration = "legacy-plan-json-to-occurrences-v1"

// MigrateLegacyPlans copies the legacy workspace-wide JSON projection into
// normalized plan state and dated occurrence rows. The source is retained as
// an archival copy. Schema creation must run first; all copied rows and the
// migration receipt commit together or none do.
func MigrateLegacyPlans(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var applied int
	err = tx.QueryRowContext(ctx, `SELECT 1 FROM plan_migrations WHERE name=?`, legacyPlanMigration).Scan(&applied)
	if err == nil {
		return tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	var hasLegacy int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_master WHERE type='table' AND name='plans'`).Scan(&hasLegacy); err != nil {
		return err
	}
	if hasLegacy > 0 {
		rows, err := tx.QueryContext(ctx, `SELECT workspace_id,revision,plan_json,updated_at FROM plans ORDER BY workspace_id`)
		if err != nil {
			return fmt.Errorf("read legacy plan table: %w", err)
		}
		type legacyRow struct {
			workspace string
			revision  int64
			raw       string
			updated   string
		}
		legacy := make([]legacyRow, 0)
		for rows.Next() {
			var row legacyRow
			if err := rows.Scan(&row.workspace, &row.revision, &row.raw, &row.updated); err != nil {
				rows.Close()
				return err
			}
			legacy = append(legacy, row)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
		for _, row := range legacy {
			var draft Draft
			if err := json.Unmarshal([]byte(row.raw), &draft); err != nil {
				return fmt.Errorf("decode legacy plan for workspace %q: %w", row.workspace, err)
			}
			applyRecipeRevisions(&draft)
			canonicalizeDraft(&draft)
			metadata := draft
			metadata.Occurrences = nil
			metadata.Unresolved = nil
			metadataJSON, err := json.Marshal(metadata)
			if err != nil {
				return err
			}
			unresolvedJSON, err := json.Marshal(draft.Unresolved)
			if err != nil {
				return err
			}
			updated := row.updated
			if updated == "" {
				updated = time.Now().UTC().Format(time.RFC3339Nano)
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO plan_state(workspace_id,revision,metadata_json,unresolved_json,updated_at) VALUES(?,?,?,?,?) ON CONFLICT(workspace_id) DO UPDATE SET revision=excluded.revision,metadata_json=excluded.metadata_json,unresolved_json=excluded.unresolved_json,updated_at=excluded.updated_at`, row.workspace, row.revision, string(metadataJSON), string(unresolvedJSON), updated); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM plan_occurrences WHERE workspace_id=?`, row.workspace); err != nil {
				return err
			}
			for _, occurrence := range draft.Occurrences {
				if occurrence.SlotName == "" {
					occurrence.SlotName = "meal"
				}
				locked := 0
				if occurrence.Locked {
					locked = 1
				}
				if _, err := tx.ExecContext(ctx, `INSERT INTO plan_occurrences(workspace_id,date,slot_name,mode,quantity,recipe_id,recipe_revision,recipe_name,reason,locked) VALUES(?,?,?,?,?,?,?,?,?,?)`, row.workspace, occurrence.Date, occurrence.SlotName, occurrence.Mode, occurrence.Quantity, occurrence.RecipeID, occurrence.RecipeRevision, occurrence.RecipeName, occurrence.Reason, locked); err != nil {
					return fmt.Errorf("migrate occurrence %s/%s: %w", row.workspace, occurrence.Date, err)
				}
			}
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO plan_migrations(name,applied_at) VALUES(?,?)`, legacyPlanMigration, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return tx.Commit()
}

func applyRecipeRevisions(draft *Draft) {
	revisions := make(map[string]int64)
	for _, reference := range draft.InputReferences {
		parts := strings.Split(reference, ":")
		if len(parts) != 3 || parts[0] != "recipe" {
			continue
		}
		if revision, err := strconv.ParseInt(parts[2], 10, 64); err == nil {
			revisions[parts[1]] = revision
		}
	}
	for i := range draft.Occurrences {
		if draft.Occurrences[i].RecipeRevision == 0 {
			draft.Occurrences[i].RecipeRevision = revisions[draft.Occurrences[i].RecipeID]
		}
	}
}
