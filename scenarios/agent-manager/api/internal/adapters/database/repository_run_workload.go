package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func (r *runRepository) UpdateRunWorkload(ctx context.Context, id uuid.UUID, workload domain.WorkloadRef) error {
	if !workload.Kind.IsValid() || strings.TrimSpace(workload.Key) == "" {
		return fmt.Errorf("valid workload kind and non-empty workload key are required")
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	key, instance := strings.TrimSpace(workload.Key), strings.TrimSpace(workload.Instance)
	result, err := tx.ExecContext(ctx, `UPDATE runs SET workload_kind=?, workload_key=?, workload_instance=?, updated_at=? WHERE id=?`, string(workload.Kind), key, instance, SQLiteTime(time.Now()), id)
	if err != nil {
		return wrapDBError("update_workload", "Run", id.String(), err)
	}
	if changed, err := result.RowsAffected(); err != nil {
		return err
	} else if changed == 0 {
		return domain.NewNotFoundError("Run", id)
	}
	// Keep attribution readers aligned with the source run in the same commit.
	if _, err := tx.ExecContext(ctx, `UPDATE invocation_read_model_runs SET workload_kind=?, workload_key=?, workload_instance=? WHERE run_id=?`, string(workload.Kind), key, instance, id.String()); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE run_ledger_facts SET workload_key=? WHERE run_id=?`, key, id.String()); err != nil {
		return err
	}
	return tx.Commit()
}
