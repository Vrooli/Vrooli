package wiring

import (
	"context"
	"strings"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/maintenance"
)

// NewMaintenanceInventory composes the bounded control-plane read with AM's
// routed SQL and existing nullable timestamp representation. Only identity
// columns are hydrated; this read never builds actions, secrets or run config.
func NewMaintenanceInventory(db *database.DB) *maintenance.InventoryReader {
	return maintenance.NewControlPlaneInventoryBatch(db, func(ctx context.Context, refs []maintenance.ExecutorRef) ([]*domain.Run, error) {
		return readMaintenanceExecutorIdentities(ctx, db, refs)
	})
}

func readMaintenanceExecutorIdentities(ctx context.Context, db *database.DB, refs []maintenance.ExecutorRef) ([]*domain.Run, error) {
	if len(refs) == 0 {
		return nil, nil
	}
	args := make([]any, len(refs))
	for i, ref := range refs {
		args[i] = ref.ID
	}
	query := `SELECT id,COALESCE(tag,''),COALESCE(runner_pid,0),COALESCE(runner_pgid,0),started_at,ended_at FROM runs WHERE id IN (` + strings.TrimSuffix(strings.Repeat("?,", len(args)), ",") + `)`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var runs []*domain.Run
	for rows.Next() {
		run := new(domain.Run)
		var started, ended database.NullableTime
		if err := rows.Scan(&run.ID, &run.Tag, &run.RunnerPID, &run.RunnerPGID, &started, &ended); err != nil {
			return nil, err
		}
		run.StartedAt, run.EndedAt = started.ToPtr(), ended.ToPtr()
		runs = append(runs, run)
	}
	return runs, rows.Err()
}
