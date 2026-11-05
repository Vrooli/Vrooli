// Responsibility: retire the finding review state and the generic findings the
// recurring-friction publisher wrote.
package database

import (
	"context"

	"agent-manager/internal/domain"

	"github.com/sirupsen/logrus"
)

// retiredRunFindingColumns held operator review state nothing read (decision,
// measured effectiveness) and the candidate fields only the retired publisher
// wrote.
var retiredRunFindingColumns = []string{"operator_decision", "target_measure", "before_value", "after_value", "effectiveness", "proposed_workaround", "acceptance_impact", "owner_hint"}

// genericFindingText is the one recommendation the retired publisher wrote for
// every recurring fingerprint it found (888 rows, none routed anywhere).
const genericFindingText = "Review the recurring deterministic friction fingerprint and its suspected owner."

// retireRunFindingReviewState drops the retired columns and deletes the
// publisher's generic, never-routed rows. Both steps are idempotent: they do
// nothing once the columns are gone and no such row remains.
func (db *DB) retireRunFindingReviewState(ctx context.Context) error {
	columns, err := db.tableColumns(ctx, "run_findings")
	if err != nil {
		return err
	}
	for _, column := range retiredRunFindingColumns {
		if _, ok := columns[column]; !ok {
			continue
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE run_findings DROP COLUMN "+column); err != nil {
			return &domain.DatabaseError{Operation: "schema_migrate", EntityType: "Schema", Cause: err}
		}
	}
	if _, ok := columns["friction_topic"]; !ok {
		return nil // the column is added after this step; no row has a topic yet
	}
	result, err := db.ExecContext(ctx, `DELETE FROM run_findings WHERE severity = 'recurring' AND recommendation_text = ? AND friction_topic = ''`, genericFindingText)
	if err != nil {
		return &domain.DatabaseError{Operation: "schema_migrate", EntityType: "Schema", Cause: err}
	}
	if deleted, _ := result.RowsAffected(); deleted > 0 && db.log != nil {
		db.log.WithFields(logrus.Fields{"table": "run_findings", "rows": deleted}).Info("Deleted generic findings from the retired recurring-friction publisher")
	}
	return nil
}
