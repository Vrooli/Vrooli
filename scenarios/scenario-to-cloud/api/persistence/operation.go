package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
)

// cloudOperationsPostgresDDL is the durable operation ledger. The unique
// (deployment_id, request_key) pair is the idempotency scope: one request key
// admits exactly one plan digest per deployment.
const cloudOperationsPostgresDDL = `
	CREATE TABLE IF NOT EXISTS cloud_operations (
		id UUID PRIMARY KEY,
		deployment_id UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
		request_key TEXT NOT NULL,
		plan_digest TEXT NOT NULL,
		plan JSONB,
		state TEXT NOT NULL DEFAULT 'admitted'
			CHECK (state IN ('admitted', 'waiting_input', 'running', 'verifying', 'reconciling', 'recovering',
							 'cancel_requested', 'succeeded', 'failed', 'failed_recovery', 'cancelled')),
		fence BIGINT NOT NULL DEFAULT 0,
		worker_id TEXT,
		lease_expires_at TIMESTAMPTZ,
		heartbeat_at TIMESTAMPTZ,
		cancel_requested BOOLEAN NOT NULL DEFAULT false,
		unknown_effects JSONB,
		step_receipts JSONB,
		active_step JSONB,
		result JSONB,
		error JSONB,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		terminal_at TIMESTAMPTZ,
		UNIQUE (deployment_id, request_key)
	);
	CREATE INDEX IF NOT EXISTS idx_cloud_operations_deployment_id ON cloud_operations(deployment_id);
	CREATE INDEX IF NOT EXISTS idx_cloud_operations_state ON cloud_operations(state);
`

const cloudOperationsSQLiteDDL = `
CREATE TABLE IF NOT EXISTS cloud_operations (
	 id TEXT PRIMARY KEY,
	 deployment_id TEXT NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
	 request_key TEXT NOT NULL,
	 plan_digest TEXT NOT NULL,
	 plan TEXT,
	 state TEXT NOT NULL DEFAULT 'admitted' CHECK (state IN ('admitted', 'waiting_input', 'running', 'verifying', 'reconciling', 'recovering', 'cancel_requested', 'succeeded', 'failed', 'failed_recovery', 'cancelled')),
	 fence INTEGER NOT NULL DEFAULT 0,
	 worker_id TEXT,
	 lease_expires_at TIMESTAMP,
	 heartbeat_at TIMESTAMP,
	 cancel_requested INTEGER NOT NULL DEFAULT 0,
	 unknown_effects TEXT,
	 step_receipts TEXT,
	 active_step TEXT,
	 result TEXT,
	 error TEXT,
	 created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	 updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	 terminal_at TIMESTAMP,
	 UNIQUE (deployment_id, request_key)
);
CREATE INDEX IF NOT EXISTS idx_cloud_operations_deployment_id ON cloud_operations(deployment_id);
CREATE INDEX IF NOT EXISTS idx_cloud_operations_state ON cloud_operations(state);
`

const cloudOperationColumns = `
		id, deployment_id, request_key, plan_digest, plan, state, fence,
		worker_id, lease_expires_at, heartbeat_at, cancel_requested,
		unknown_effects, step_receipts, active_step, result, error,
		created_at, updated_at, terminal_at`

func scanCloudOperation(row rowScanner) (*domain.CloudOperation, error) {
	op := &domain.CloudOperation{}
	var (
		plan, unknownEffects, stepReceipts, activeStep, result, errJSON domain.NullRawMessage
		workerID                                                        sql.NullString
	)
	if err := row.Scan(
		&op.ID, &op.DeploymentID, &op.RequestKey, &op.PlanDigest, &plan, &op.State, &op.Fence,
		&workerID, &op.LeaseExpiresAt, &op.HeartbeatAt, &op.CancelRequested,
		&unknownEffects, &stepReceipts, &activeStep, &result, &errJSON,
		&op.CreatedAt, &op.UpdatedAt, &op.TerminalAt,
	); err != nil {
		return nil, err
	}
	op.WorkerID = workerID.String
	op.Plan = plan.Data
	op.UnknownEffects = unknownEffects.Data
	op.StepReceipts = stepReceipts.Data
	op.ActiveStep = activeStep.Data
	op.Result = result.Data
	op.Error = errJSON.Data
	return op, nil
}

// GetOperation returns one operation by ID, or nil when absent.
func (r *Repository) GetOperation(ctx context.Context, id string) (*domain.CloudOperation, error) {
	q := `SELECT ` + cloudOperationColumns + ` FROM cloud_operations WHERE id = $1`
	op, err := scanCloudOperation(r.db.QueryRowContext(ctx, q, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get operation: %w", err)
	}
	return op, nil
}

// GetOperationByRequestKey returns the operation admitted for a request key.
func (r *Repository) GetOperationByRequestKey(ctx context.Context, deploymentID, requestKey string) (*domain.CloudOperation, error) {
	q := `SELECT ` + cloudOperationColumns + ` FROM cloud_operations WHERE deployment_id = $1 AND request_key = $2`
	op, err := scanCloudOperation(r.db.QueryRowContext(ctx, q, deploymentID, requestKey))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get operation by request key: %w", err)
	}
	return op, nil
}

// CreateOperation admits an operation under its request key. Concurrent
// retries converge on one row through the unique constraint: the same key
// with the same plan digest returns the existing operation; the same key with
// a different plan digest is a typed request_key_conflict and nothing is
// written.
func (r *Repository) CreateOperation(ctx context.Context, op *domain.CloudOperation) (*domain.CloudOperation, error) {
	if op == nil {
		return nil, fmt.Errorf("operation is required")
	}
	if op.ID == "" || op.DeploymentID == "" || op.RequestKey == "" || op.PlanDigest == "" {
		return nil, apierrors.New(apierrors.CodeInvalidRequest, "Operation id, deployment_id, request_key and plan_digest are required")
	}
	if op.CreatedAt.IsZero() {
		op.CreatedAt = time.Now().UTC()
	}
	op.UpdatedAt = op.CreatedAt
	if op.State == "" {
		op.State = domain.OperationAdmitted
	}
	const q = `
		INSERT INTO cloud_operations (
			id, deployment_id, request_key, plan_digest, plan, state, fence, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		ON CONFLICT (deployment_id, request_key) DO NOTHING`
	if _, err := r.db.ExecContext(ctx, q,
		op.ID, op.DeploymentID, op.RequestKey, op.PlanDigest, nullableJSON(op.Plan), op.State, op.Fence, op.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("create operation: %w", err)
	}
	existing, err := r.GetOperationByRequestKey(ctx, op.DeploymentID, op.RequestKey)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("operation %s vanished after admission", op.ID)
	}
	if existing.PlanDigest != op.PlanDigest {
		return nil, apierrors.New(apierrors.CodeRequestKeyConflict, "Request key was already used with a different plan").
			WithDetail("request_key", op.RequestKey).
			WithDetail("existing_operation_id", existing.ID).
			WithDetail("existing_plan_digest", existing.PlanDigest).
			WithDetail("submitted_plan_digest", op.PlanDigest).
			WithNextAction(apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "operation", Reference: existing.ID, Label: "Inspect the admitted operation or use a new request key"})
	}
	return existing, nil
}

func nullableJSON(raw []byte) any {
	if len(raw) == 0 {
		return nil
	}
	return raw
}
