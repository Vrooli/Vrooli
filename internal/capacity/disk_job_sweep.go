package capacity

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// A disk lease is a promise renewed by heartbeat. When the heartbeat deadline
// passes the reservation stops being capacity anyone holds: admission does not
// count it and the next settle closes it. Before this rule an overdue lease
// stayed active until an operator reconciled it, so a finished or crashed job
// kept withholding its whole declared peak (26 of 28 active rows were past
// their deadline, the oldest eleven days, 52.8 GB against 276 GB free; later a
// 23.6 GB hold refused 2 GB jobs at 197 GB free).
//
// Settling never claims cleanup happened. A job is closed as
// status=reconciled with cleanup_verified=false, a legacy reservation as
// status=expired. Whatever the owner already wrote shows in the measured free
// bytes of the filesystem, so admission stays correct; only the peak it had
// reserved but not yet written is released. Verified reconciliation
// (ReconcileDiskJob, disk-reconcile) is unchanged for owners that can attest
// cleanup, and a live owner that still holds its credential may complete a
// settled job (CompleteDiskJob) because that call attests cleanup itself.
//
// An owner that is alive but stopped heartbeating is treated like any other
// lapsed owner: a lease is liveness of the heartbeat, not of the process. An
// owner whose process is gone before the deadline keeps its lease until the
// deadline, because detached children of the producer may still be writing.

// settleLapsedDiskLeasesTx closes every active disk job and legacy reservation
// whose heartbeat deadline has passed, inside the caller's write transaction.
// It returns the settled job lease ids and the count of legacy reservations.
func settleLapsedDiskLeasesTx(ctx context.Context, tx *sql.Tx, now time.Time) ([]string, int64, error) {
	at := formatTime(now.UTC())
	rows, err := tx.QueryContext(ctx, `SELECT job_lease_id FROM capacity_disk_jobs WHERE status='active' AND heartbeat_deadline_at <= ?`, at)
	if err != nil {
		return nil, 0, fmt.Errorf("list lapsed disk jobs: %w", err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return nil, 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return nil, 0, err
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var settled []string
	for _, id := range ids {
		result, err := tx.ExecContext(ctx, `UPDATE capacity_disk_jobs SET status='reconciled', cleanup_verified=0, updated_at=? WHERE job_lease_id=? AND status='active' AND heartbeat_deadline_at <= ?`, at, id, at)
		if err != nil {
			return nil, 0, fmt.Errorf("settle lapsed disk job %s: %w", id, err)
		}
		if n, raErr := result.RowsAffected(); raErr == nil && n == 1 {
			settled = append(settled, id)
		}
	}
	legacy, err := tx.ExecContext(ctx, `UPDATE capacity_disk_reservations SET status='expired', updated_at=? WHERE status='active' AND heartbeat_deadline_at <= ?`, at, at)
	if err != nil {
		return nil, 0, fmt.Errorf("settle lapsed disk reservations: %w", err)
	}
	expired, _ := legacy.RowsAffected()
	return settled, expired, nil
}

// SettleLapsedDiskLeases settles every lapsed disk lease and returns the
// receipts of the jobs it closed. It is the explicit form of the settle that
// every admission performs before it sums reservations.
func (s *SQLiteStore) SettleLapsedDiskLeases(ctx context.Context) ([]DiskJobReceipt, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("capacity store is unavailable")
	}
	var ids []string
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		var err error
		ids, _, err = settleLapsedDiskLeasesTx(ctx, tx, s.now())
		return err
	})
	if err != nil {
		return nil, err
	}
	receipts := make([]DiskJobReceipt, 0, len(ids))
	for _, id := range ids {
		receipt, err := s.GetDiskJob(ctx, id)
		if err != nil {
			return receipts, err
		}
		receipts = append(receipts, receipt)
	}
	return receipts, nil
}

// diskJobLeaseState classifies a job row for an owner that wants to keep using
// it: live, lapsed (overdue or settled by a lapse), or closed for another reason.
type diskJobLeaseState int

const (
	diskJobLeaseLive diskJobLeaseState = iota
	diskJobLeaseLapsed
	diskJobLeaseClosed
)

func classifyDiskJobLease(status string, cleanupVerified bool, deadline, now time.Time) diskJobLeaseState {
	switch {
	case status == "active" && deadline.After(now):
		return diskJobLeaseLive
	case status == "active", status == "reconciled" && !cleanupVerified:
		return diskJobLeaseLapsed
	default:
		return diskJobLeaseClosed
	}
}
