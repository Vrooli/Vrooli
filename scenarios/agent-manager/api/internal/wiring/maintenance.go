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
	var runs []*domain.Run
	// Bound SQL bind parameters independently of the host protocol capacity.
	// All batches finish before the observer takes its one indexed host snapshot;
	// the observer still requires exact identity coverage and unchanged PID/PGID.
	const batchSize = 512
	for offset := 0; offset < len(refs); offset += batchSize {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		batch := refs[offset:min(offset+batchSize, len(refs))]
		args := make([]any, len(batch))
		for i, ref := range batch {
			args[i] = ref.ID
		}
		query := `SELECT id,COALESCE(tag,''),COALESCE(runner_pid,0),COALESCE(runner_pgid,0),started_at,ended_at FROM runs WHERE id IN (` + strings.TrimSuffix(strings.Repeat("?,", len(args)), ",") + `)`
		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			run := new(domain.Run)
			var started, ended database.NullableTime
			if err := rows.Scan(&run.ID, &run.Tag, &run.RunnerPID, &run.RunnerPGID, &started, &ended); err != nil {
				rows.Close()
				return nil, err
			}
			run.StartedAt, run.EndedAt = started.ToPtr(), ended.ToPtr()
			runs = append(runs, run)
		}
		readErr := rows.Err()
		closeErr := rows.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
	}
	return runs, nil
}
