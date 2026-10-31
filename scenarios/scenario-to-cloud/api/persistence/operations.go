package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
)

// Every write in this file carries the fence (and, where it matters, the
// worker id and the expected state) in the WHERE clause. A stale worker —
// one whose fence was superseded by a successor's acquisition — therefore
// fails on every write with fence_stale instead of clobbering the successor.

// AdmitOperation persists the operation intent before any effect. Semantics
// are those of CreateOperation (P03): the same (deployment, request_key)
// with the same plan digest returns the existing row; a different digest is
// a typed request_key_conflict and nothing is written.
func (r *Repository) AdmitOperation(ctx context.Context, op *domain.CloudOperation) (*domain.CloudOperation, error) {
	return r.CreateOperation(ctx, op)
}

// ListNonTerminal returns every operation that still needs an owner or an
// observer, oldest first. Startup reconciliation walks this list.
func (r *Repository) ListNonTerminal(ctx context.Context) ([]*domain.CloudOperation, error) {
	q := `SELECT ` + cloudOperationColumns + ` FROM cloud_operations
		WHERE state NOT IN ('succeeded', 'failed', 'failed_recovery', 'cancelled')
		ORDER BY created_at ASC, id ASC`
	return r.queryOperations(ctx, q)
}

// ListOperationsByDeployment returns the operations admitted against one
// deployment, newest first.
func (r *Repository) ListOperationsByDeployment(ctx context.Context, deploymentID string) ([]*domain.CloudOperation, error) {
	q := `SELECT ` + cloudOperationColumns + ` FROM cloud_operations WHERE deployment_id = $1 ORDER BY created_at DESC, id DESC`
	return r.queryOperations(ctx, q, deploymentID)
}

func (r *Repository) queryOperations(ctx context.Context, q string, args ...any) ([]*domain.CloudOperation, error) {
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list operations: %w", err)
	}
	defer rows.Close()
	var out []*domain.CloudOperation
	for rows.Next() {
		op, err := scanCloudOperation(rows)
		if err != nil {
			return nil, fmt.Errorf("scan operation: %w", err)
		}
		out = append(out, op)
	}
	return out, rows.Err()
}

// requireOperation loads an operation or returns a typed operation_not_found.
func (r *Repository) requireOperation(ctx context.Context, id string) (*domain.CloudOperation, error) {
	op, err := r.GetOperation(ctx, id)
	if err != nil {
		return nil, err
	}
	if op == nil {
		return nil, apierrors.New(apierrors.CodeOperationNotFound, "Operation not found").WithDetail("operation_id", id)
	}
	return op, nil
}

func fenceStale(op *domain.CloudOperation, fence uint64) error {
	return apierrors.New(apierrors.CodeFenceStale, "The worker's fence has been superseded; this write is refused").
		WithDetail("operation_id", op.ID).
		WithDetail("current_fence", op.Fence).
		WithDetail("presented_fence", fence).
		WithDetail("current_worker", op.WorkerID)
}

// leaseLive reports whether the operation holds a lease that has not expired
// at now.
func leaseLive(op *domain.CloudOperation, now time.Time) bool {
	return op.WorkerID != "" && op.LeaseExpiresAt != nil && op.LeaseExpiresAt.After(now)
}

// AcquireWorker takes ownership of an operation for workerID. It bumps the
// deployment fence (the monotonic ownership generation shared by every
// operation and every target effect of that deployment), stamps the worker,
// lease and heartbeat and moves an admitted operation to running. It refuses
// with operation_conflict when another worker holds a live lease on this
// operation, when any other non-terminal operation of the same deployment
// holds a live lease (conflicting mutations serialise, whoever the worker
// is), and when the operation is terminal.
func (r *Repository) AcquireWorker(ctx context.Context, opID, workerID string, leaseTTL time.Duration) (uint64, error) {
	if strings.TrimSpace(workerID) == "" {
		return 0, apierrors.New(apierrors.CodeInvalidRequest, "worker id is required")
	}
	if leaseTTL <= 0 {
		return 0, apierrors.New(apierrors.CodeInvalidRequest, "lease ttl must be positive")
	}
	op, err := r.requireOperation(ctx, opID)
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	if op.State.IsTerminal() {
		return 0, apierrors.New(apierrors.CodeOperationConflict, "Operation is terminal and cannot be acquired").
			WithDetail("operation_id", op.ID).WithDetail("state", string(op.State))
	}
	if op.State == domain.OperationWaitingInput {
		return 0, apierrors.New(apierrors.CodeOperationConflict, "Operation is waiting for input; resolve the handoff first").
			WithDetail("operation_id", op.ID).WithDetail("state", string(op.State))
	}
	if leaseLive(op, now) && op.WorkerID != workerID {
		return 0, apierrors.New(apierrors.CodeOperationConflict, "Another worker holds a live lease on this operation").
			WithDetail("operation_id", op.ID).WithDetail("worker_id", op.WorkerID).WithDetail("lease_expires_at", op.LeaseExpiresAt.Format(time.RFC3339Nano))
	}
	siblings, err := r.ListOperationsByDeployment(ctx, op.DeploymentID)
	if err != nil {
		return 0, err
	}
	for _, sibling := range siblings {
		if sibling.ID == op.ID || sibling.State.IsTerminal() {
			continue
		}
		if leaseLive(sibling, now) {
			return 0, apierrors.New(apierrors.CodeOperationConflict, "Another operation on this deployment is being executed; operations serialise").
				WithDetail("operation_id", op.ID).WithDetail("blocking_operation_id", sibling.ID).WithDetail("blocking_state", string(sibling.State)).
				WithNextAction(apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "operation", Reference: sibling.ID, Label: "Wait for the blocking operation"})
		}
	}
	if err := r.checkHostConcurrency(ctx, op, now); err != nil {
		return 0, err
	}
	fence, err := r.BumpFence(ctx, op.DeploymentID)
	if err != nil {
		return 0, err
	}
	expires := now.Add(leaseTTL)
	// The guard repeats the liveness check in SQL so two concurrent
	// acquirers that both passed the Go checks cannot both win: the first
	// UPDATE installs a live lease and the second sees zero rows.
	const q = `
		UPDATE cloud_operations SET
			fence = $2,
			worker_id = $3,
			lease_expires_at = $4,
			heartbeat_at = $5,
			state = CASE WHEN state = 'admitted' THEN 'running' ELSE state END,
			updated_at = $5
		WHERE id = $1
		  AND state NOT IN ('succeeded', 'failed', 'failed_recovery', 'cancelled', 'waiting_input')
		  AND fence < $2
		  AND (worker_id IS NULL OR worker_id = $3 OR lease_expires_at IS NULL OR lease_expires_at < $5)`
	res, err := r.db.ExecContext(ctx, q, op.ID, fence, workerID, expires, now)
	if err != nil {
		return 0, fmt.Errorf("acquire worker: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("acquire worker rows: %w", err)
	}
	if n == 0 {
		return 0, apierrors.New(apierrors.CodeOperationConflict, "Operation was acquired by another worker or changed state").
			WithDetail("operation_id", op.ID)
	}
	return fence, nil
}

// SetHostConcurrencyLimit bounds the live effectful operations across every
// deployment that shares one target (deployments.target_key). Zero means
// unbounded. The production value is phase-23's
// deployment_queue.effectful_operations_per_host_max.
func (r *Repository) SetHostConcurrencyLimit(n int) { r.hostConcurrencyMax = n }

// checkHostConcurrency refuses acquisition when the target already carries
// the maximum number of live leases held by other deployments' operations.
func (r *Repository) checkHostConcurrency(ctx context.Context, op *domain.CloudOperation, now time.Time) error {
	if r.hostConcurrencyMax <= 0 {
		return nil
	}
	const q = `
		SELECT COUNT(*) FROM cloud_operations o
		JOIN deployments d ON d.id = o.deployment_id
		WHERE d.target_key <> ''
		  AND d.target_key = (SELECT target_key FROM deployments WHERE id = $1)
		  AND o.id <> $2
		  AND o.state NOT IN ('succeeded', 'failed', 'failed_recovery', 'cancelled', 'waiting_input')
		  AND o.lease_expires_at IS NOT NULL AND o.lease_expires_at > $3`
	var live int
	if err := r.db.QueryRowContext(ctx, q, op.DeploymentID, op.ID, now).Scan(&live); err != nil {
		return fmt.Errorf("count host operations: %w", err)
	}
	if live < r.hostConcurrencyMax {
		return nil
	}
	return apierrors.New(apierrors.CodeOperationConflict, "The target already runs the maximum number of concurrent operations").
		WithDetail("operation_id", op.ID).WithDetail("live_operations", strconv.Itoa(live)).WithDetail("effectful_operations_per_host_max", strconv.Itoa(r.hostConcurrencyMax)).
		WithNextAction(apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "operation", Reference: op.ID, Label: "Wait for an operation on this target to finish"})
}

// PruneTerminalOperations deletes terminal operation records whose terminal
// time is older than olderThan, except the protected ids (incident or
// evidence references the caller still needs). Non-terminal records are
// never touched. It returns the number of deleted rows.
func (r *Repository) PruneTerminalOperations(ctx context.Context, olderThan time.Time, protectedIDs []string) (int64, error) {
	args := []any{olderThan.UTC()}
	q := `DELETE FROM cloud_operations
		WHERE state IN ('succeeded', 'failed', 'failed_recovery', 'cancelled')
		  AND terminal_at IS NOT NULL AND terminal_at < $1`
	if len(protectedIDs) > 0 {
		placeholders := make([]string, 0, len(protectedIDs))
		for _, id := range protectedIDs {
			args = append(args, id)
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
		}
		q += " AND id NOT IN (" + strings.Join(placeholders, ", ") + ")"
	}
	res, err := r.db.ExecContext(ctx, q, args...)
	if err != nil {
		return 0, fmt.Errorf("prune terminal operations: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("prune terminal operations rows: %w", err)
	}
	return n, nil
}

// Heartbeat extends the lease. It succeeds only for the worker that holds
// the current fence.
func (r *Repository) Heartbeat(ctx context.Context, opID, workerID string, fence uint64, leaseTTL time.Duration) error {
	now := time.Now().UTC()
	const q = `
		UPDATE cloud_operations SET heartbeat_at = $4, lease_expires_at = $5, updated_at = $4
		WHERE id = $1 AND worker_id = $2 AND fence = $3
		  AND state NOT IN ('succeeded', 'failed', 'failed_recovery', 'cancelled')`
	res, err := r.db.ExecContext(ctx, q, opID, workerID, fence, now, now.Add(leaseTTL))
	if err != nil {
		return fmt.Errorf("heartbeat: %w", err)
	}
	return r.refuseIfStale(ctx, res, opID, fence)
}

// refuseIfStale turns a zero-row fenced update into fence_stale (or
// operation_not_found when the row is gone).
func (r *Repository) refuseIfStale(ctx context.Context, res sql.Result, opID string, fence uint64) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 1 {
		return nil
	}
	op, err := r.requireOperation(ctx, opID)
	if err != nil {
		return err
	}
	return fenceStale(op, fence)
}

// SetActiveStep records (or clears, with nil) the step in flight. It is the
// marker reconciliation uses after an owner crash: a step that has an
// active marker and no receipt is an unknown effect until the target
// receipt says otherwise.
func (r *Repository) SetActiveStep(ctx context.Context, opID string, fence uint64, step *domain.ActiveStep) error {
	var raw any
	if step != nil {
		encoded, err := json.Marshal(step)
		if err != nil {
			return err
		}
		raw = encoded
	}
	const q = `UPDATE cloud_operations SET active_step = $3, updated_at = $4 WHERE id = $1 AND fence = $2`
	res, err := r.db.ExecContext(ctx, q, opID, fence, raw, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("set active step: %w", err)
	}
	return r.refuseIfStale(ctx, res, opID, fence)
}

// CommitStep appends a step receipt and clears the active-step marker. It
// refuses when the presented fence is not the operation's current fence.
func (r *Repository) CommitStep(ctx context.Context, opID string, fence uint64, receipt domain.StepReceipt) error {
	if strings.TrimSpace(receipt.Step) == "" {
		return apierrors.New(apierrors.CodeInvalidRequest, "step receipt needs a step id")
	}
	op, err := r.requireOperation(ctx, opID)
	if err != nil {
		return err
	}
	if op.Fence != fence {
		return fenceStale(op, fence)
	}
	receipts, err := op.Receipts()
	if err != nil {
		return err
	}
	receipt.Fence = fence
	if receipt.CompletedAt == "" {
		receipt.CompletedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	receipts = append(receipts, receipt)
	encoded, err := json.Marshal(receipts)
	if err != nil {
		return err
	}
	const q = `UPDATE cloud_operations SET step_receipts = $3, active_step = NULL, updated_at = $4 WHERE id = $1 AND fence = $2`
	res, err := r.db.ExecContext(ctx, q, opID, fence, encoded, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("commit step: %w", err)
	}
	return r.refuseIfStale(ctx, res, opID, fence)
}

// RecordUnknownEffect appends an unknown-effect record (fenced).
func (r *Repository) RecordUnknownEffect(ctx context.Context, opID string, fence uint64, effect domain.UnknownEffect) error {
	op, err := r.requireOperation(ctx, opID)
	if err != nil {
		return err
	}
	if op.Fence != fence {
		return fenceStale(op, fence)
	}
	effects, err := op.UnknownEffectList()
	if err != nil {
		return err
	}
	effect.Fence = fence
	if effect.RecordedAt == "" {
		effect.RecordedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	effects = append(effects, effect)
	encoded, err := json.Marshal(effects)
	if err != nil {
		return err
	}
	const q = `UPDATE cloud_operations SET unknown_effects = $3, updated_at = $4 WHERE id = $1 AND fence = $2`
	res, err := r.db.ExecContext(ctx, q, opID, fence, encoded, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("record unknown effect: %w", err)
	}
	return r.refuseIfStale(ctx, res, opID, fence)
}

// StatePatch carries the optional payload of a state change.
type StatePatch struct {
	Result *domain.OperationResult
	Error  *apierrors.Error
	// ReleaseLease clears worker/lease so a successor can acquire without
	// waiting for expiry (terminal states and waiting_input always release).
	ReleaseLease bool
}

// SetState moves the operation from → to under the fence. The legality of
// the transition is decided by operations.Transition; this method enforces
// atomicity (state = from AND fence = current in the WHERE clause) and the
// terminal bookkeeping.
func (r *Repository) SetState(ctx context.Context, opID string, fence uint64, from, to domain.OperationState, patch StatePatch) error {
	now := time.Now().UTC()
	var resultRaw, errorRaw any
	if patch.Result != nil {
		encoded, err := json.Marshal(patch.Result)
		if err != nil {
			return err
		}
		resultRaw = encoded
	}
	if patch.Error != nil {
		encoded, err := json.Marshal(patch.Error)
		if err != nil {
			return err
		}
		errorRaw = encoded
	}
	release := patch.ReleaseLease || to.IsTerminal() || to == domain.OperationWaitingInput
	var terminalAt any
	if to.IsTerminal() {
		terminalAt = now
	}
	q := `
		UPDATE cloud_operations SET
			state = $4,
			result = COALESCE($5, result),
			error = COALESCE($6, error),
			terminal_at = COALESCE($7, terminal_at),
			updated_at = $8`
	if release {
		q += `, worker_id = NULL, lease_expires_at = NULL, active_step = NULL`
	}
	q += ` WHERE id = $1 AND fence = $2 AND state = $3`
	res, err := r.db.ExecContext(ctx, q, opID, fence, from, to, resultRaw, errorRaw, terminalAt, now)
	if err != nil {
		return fmt.Errorf("set state %s→%s: %w", from, to, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 1 {
		return nil
	}
	op, err := r.requireOperation(ctx, opID)
	if err != nil {
		return err
	}
	if op.Fence != fence {
		return fenceStale(op, fence)
	}
	return apierrors.New(apierrors.CodeOperationConflict, fmt.Sprintf("operation is %s, not %s", op.State, from)).
		WithDetail("operation_id", opID).WithDetail("state", string(op.State)).WithDetail("expected", string(from))
}

// RequestCancel marks cancellation. An operation that has no worker yet
// (admitted, waiting_input) is cancelled immediately; a running or
// verifying one becomes cancel_requested and the worker honours it at the
// next declared cancel point. Terminal operations are refused. The call is
// not fenced: cancellation is a client intent, not a worker write.
func (r *Repository) RequestCancel(ctx context.Context, opID string) (*domain.CloudOperation, error) {
	op, err := r.requireOperation(ctx, opID)
	if err != nil {
		return nil, err
	}
	if op.State.IsTerminal() {
		return nil, apierrors.New(apierrors.CodeOperationConflict, "Operation is terminal and cannot be cancelled").
			WithDetail("operation_id", opID).WithDetail("state", string(op.State))
	}
	now := time.Now().UTC()
	switch op.State {
	case domain.OperationAdmitted, domain.OperationWaitingInput:
		const q = `
			UPDATE cloud_operations SET cancel_requested = $2, state = 'cancelled', terminal_at = $3, updated_at = $3,
				worker_id = NULL, lease_expires_at = NULL, active_step = NULL,
				result = $4
			WHERE id = $1 AND state IN ('admitted', 'waiting_input')`
		result, _ := json.Marshal(domain.OperationResult{Outcome: string(domain.OperationCancelled), Message: "cancelled before any effect"})
		if _, err := r.db.ExecContext(ctx, q, opID, true, now, result); err != nil {
			return nil, fmt.Errorf("cancel admitted operation: %w", err)
		}
	case domain.OperationRunning, domain.OperationVerifying:
		const q = `
			UPDATE cloud_operations SET cancel_requested = $2, state = 'cancel_requested', updated_at = $3
			WHERE id = $1 AND state IN ('running', 'verifying')`
		if _, err := r.db.ExecContext(ctx, q, opID, true, now); err != nil {
			return nil, fmt.Errorf("request cancel: %w", err)
		}
	default:
		// reconciling / recovering / cancel_requested: record the intent; the
		// owner decides at its next safe boundary.
		const q = `UPDATE cloud_operations SET cancel_requested = $2, updated_at = $3 WHERE id = $1`
		if _, err := r.db.ExecContext(ctx, q, opID, true, now); err != nil {
			return nil, fmt.Errorf("record cancel intent: %w", err)
		}
	}
	return r.requireOperation(ctx, opID)
}
