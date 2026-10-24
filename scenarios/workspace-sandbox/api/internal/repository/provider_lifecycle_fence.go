package repository

import (
	"context"
	"errors"
	"fmt"
)

// ProviderLifecycleFence is the durable provider admission fence. Its row is
// owned by the existing Workspace Sandbox database.
type ProviderLifecycleFence struct {
	AdmissionClosed bool
	OperationID     string
	Revision        int64
}

func (r *SandboxRepository) GetProviderLifecycleFence(ctx context.Context) (ProviderLifecycleFence, error) {
	var row ProviderLifecycleFence
	var closed int
	err := r.db.QueryRowContext(ctx, `SELECT admission_closed, operation_id, revision FROM provider_lifecycle_fence WHERE singleton_id = 1`).Scan(&closed, &row.OperationID, &row.Revision)
	if err != nil {
		return ProviderLifecycleFence{}, fmt.Errorf("read provider lifecycle fence: %w", err)
	}
	row.AdmissionClosed = closed != 0
	return row, nil
}

// CompareAndSwapProviderLifecycleFence persists a transition only when the
// caller still holds the observed revision. Zero affected rows means the
// durable fence changed and the caller must reread it.
func (r *SandboxRepository) CompareAndSwapProviderLifecycleFence(ctx context.Context, expectedRevision int64, next ProviderLifecycleFence) error {
	if next.Revision != expectedRevision+1 {
		return fmt.Errorf("provider lifecycle fence revision must advance once")
	}
	if next.AdmissionClosed != (next.OperationID != "") {
		return fmt.Errorf("closed lifecycle fence must have exactly one operation id")
	}
	closed := 0
	if next.AdmissionClosed {
		closed = 1
	}
	result, err := r.db.ExecContext(ctx, `UPDATE provider_lifecycle_fence SET admission_closed = ?, operation_id = ?, revision = ? WHERE singleton_id = 1 AND revision = ?`, closed, next.OperationID, next.Revision, expectedRevision)
	if err != nil {
		return fmt.Errorf("write provider lifecycle fence: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("confirm provider lifecycle fence write: %w", err)
	}
	if n != 1 {
		return errors.New("provider lifecycle fence changed concurrently")
	}
	return nil
}
