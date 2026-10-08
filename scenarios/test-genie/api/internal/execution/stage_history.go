package execution

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"test-genie/internal/orchestrator"

	"github.com/google/uuid"
)

const insertStageHistorySQL = `
INSERT INTO suite_execution_stages (
    execution_id, ordinal, stage_name, parent_stage, subject, status, duration_milliseconds
) VALUES (?, ?, ?, ?, ?, ?, ?)`

// insertStageHistory persists only bounded orchestration timing. Rich provider
// responses remain owned by immutable run evidence.
func insertStageHistory(ctx context.Context, tx *sql.Tx, executionID uuid.UUID, stages []orchestrator.PreparationStage) error {
	for ordinal, stage := range stages {
		name := strings.TrimSpace(stage.Name)
		if name == "" {
			return fmt.Errorf("preparation stage %d has no name", ordinal)
		}
		duration := stage.DurationMilliseconds
		if duration < 0 {
			duration = 0
		}
		if _, err := tx.ExecContext(ctx, insertStageHistorySQL,
			executionID.String(), ordinal, name, strings.TrimSpace(stage.Parent),
			strings.TrimSpace(stage.Subject), strings.TrimSpace(stage.Status), duration); err != nil {
			return fmt.Errorf("insert preparation stage history: %w", err)
		}
	}
	return nil
}

func (r *SuiteExecutionRepository) loadStageHistory(ctx context.Context, executionID uuid.UUID) ([]orchestrator.PreparationStage, error) {
	const q = `
SELECT stage_name, parent_stage, subject, status, duration_milliseconds
FROM suite_execution_stages
WHERE execution_id = ?
ORDER BY ordinal ASC`
	rows, err := r.db.QueryContext(ctx, q, executionID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	stages := make([]orchestrator.PreparationStage, 0)
	for rows.Next() {
		var stage orchestrator.PreparationStage
		if err := rows.Scan(&stage.Name, &stage.Parent, &stage.Subject, &stage.Status, &stage.DurationMilliseconds); err != nil {
			return nil, err
		}
		stages = append(stages, stage)
	}
	return stages, rows.Err()
}
