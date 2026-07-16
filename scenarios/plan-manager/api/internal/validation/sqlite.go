package validation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"plan-manager/internal/clock"
	planmodel "plan-manager/internal/planmodel"
)

// resultTimeFormat matches the rest of the scenario (RFC3339Nano sorts
// lexicographically in time order for a fixed zone).
const resultTimeFormat = time.RFC3339Nano

// SQLExecutor is the narrow database surface the result store depends on.
// Declared at the consumer per seam-discovery: both *sql.DB (tests) and
// *database.RoutedDB (production) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteResultStore struct {
	db    SQLExecutor
	clock clock.Clock
}

// NewSQLiteResultStore constructs the production validation ResultStore over the
// shared home-store DB.
func NewSQLiteResultStore(db SQLExecutor, clk clock.Clock) *sqliteResultStore {
	return &sqliteResultStore{db: db, clock: clk}
}

var (
	_ ResultStore    = (*sqliteResultStore)(nil)
	_ OperationStore = (*sqliteResultStore)(nil)
)

const (
	insertResultSQL = `
INSERT INTO validation_results (
  id, plan_id, phase_id, verdict, staleness, commands_run, detail, ran_at,
  execution_id, operation_id, scope_generation, full_inventory, required_members, selected_members
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO NOTHING`

	lastResultSQL = `
SELECT id, plan_id, phase_id, verdict, staleness, commands_run, detail, ran_at,
       execution_id, operation_id, scope_generation, full_inventory, required_members, selected_members
FROM validation_results
WHERE plan_id = ? AND phase_id = ?
ORDER BY ran_at DESC, id DESC
LIMIT 1`
	resultByIDSQL = `
SELECT id, plan_id, phase_id, verdict, staleness, commands_run, detail, ran_at,
       execution_id, operation_id, scope_generation, full_inventory, required_members, selected_members
FROM validation_results WHERE id = ?`
)

func (r *sqliteResultStore) SaveResult(ctx context.Context, res Result) error {
	ran := res.RanAt
	if ran == "" {
		ran = r.now()
	}
	cmds, err := json.Marshal(res.CommandsRun)
	if err != nil {
		return fmt.Errorf("marshal commands_run for result %q: %w", res.ID, err)
	}
	requiredMembers, err := json.Marshal(res.RequiredMembers)
	if err != nil {
		return fmt.Errorf("marshal required_members for result %q: %w", res.ID, err)
	}
	selectedMembers, err := json.Marshal(res.SelectedMembers)
	if err != nil {
		return fmt.Errorf("marshal selected_members for result %q: %w", res.ID, err)
	}
	if _, err := r.db.ExecContext(ctx, insertResultSQL,
		res.ID, res.PlanID, res.PhaseID, string(res.Verdict), string(res.Staleness), string(cmds), res.Detail, ran,
		res.ExecutionID, res.OperationID, res.ScopeGeneration, res.FullInventory, string(requiredMembers), string(selectedMembers),
	); err != nil {
		return fmt.Errorf("insert validation result %q: %w", res.ID, err)
	}
	return nil
}

func (r *sqliteResultStore) LastResult(ctx context.Context, planID, phaseID string) (Result, bool, error) {
	return scanResult(r.db.QueryRowContext(ctx, lastResultSQL, planID, phaseID), fmt.Sprintf("latest validation result for %q/%q", planID, phaseID))
}

func (r *sqliteResultStore) GetResult(ctx context.Context, id string) (Result, bool, error) {
	return scanResult(r.db.QueryRowContext(ctx, resultByIDSQL, id), fmt.Sprintf("validation result %q", id))
}

func scanResult(row *sql.Row, label string) (Result, bool, error) {
	var (
		res             Result
		verdict         string
		staleness       string
		commands        string
		requiredMembers string
		selectedMembers string
	)
	err := row.Scan(
		&res.ID, &res.PlanID, &res.PhaseID, &verdict, &staleness, &commands, &res.Detail, &res.RanAt,
		&res.ExecutionID, &res.OperationID, &res.ScopeGeneration, &res.FullInventory, &requiredMembers, &selectedMembers,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Result{}, false, nil
	}
	if err != nil {
		return Result{}, false, fmt.Errorf("get %s: %w", label, err)
	}
	res.Verdict = Verdict(verdict)
	res.Staleness = planmodel.StalenessTier(staleness)
	if commands != "" {
		_ = json.Unmarshal([]byte(commands), &res.CommandsRun)
	}
	if requiredMembers != "" {
		_ = json.Unmarshal([]byte(requiredMembers), &res.RequiredMembers)
	}
	if selectedMembers != "" {
		_ = json.Unmarshal([]byte(selectedMembers), &res.SelectedMembers)
	}
	return res, true, nil
}

func (r *sqliteResultStore) now() string { return r.clock.Now().UTC().Format(resultTimeFormat) }

const (
	insertOperationSQL = `
INSERT OR IGNORE INTO validation_operations
  (id, plan_id, phase_id, idempotency_key, status, queued_at, updated_at, payload_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	updateOperationSQL = `
UPDATE validation_operations SET status = ?, updated_at = ?, payload_json = ? WHERE id = ?`
	getOperationSQL      = `SELECT payload_json FROM validation_operations WHERE id = ?`
	getOperationByKeySQL = `
SELECT payload_json FROM validation_operations
WHERE plan_id = ? AND phase_id = ?
  AND COALESCE(json_extract(payload_json, '$.ExecutionID'), '') = ?
  AND COALESCE(CAST(json_extract(payload_json, '$.ScopeGeneration') AS INTEGER), 0) = ?
  AND idempotency_key = ?
LIMIT 1`
	getActiveOperationSQL = `
SELECT payload_json FROM validation_operations
WHERE plan_id = ? AND phase_id = ?
  AND COALESCE(json_extract(payload_json, '$.ExecutionID'), '') = ?
  AND COALESCE(CAST(json_extract(payload_json, '$.ScopeGeneration') AS INTEGER), 0) = ?
  AND status <> ? AND idempotency_key = ''
ORDER BY queued_at, id LIMIT 1`
	listNonTerminalOperationsSQL = `
SELECT payload_json FROM validation_operations WHERE status <> ? ORDER BY queued_at, id`
)

func (r *sqliteResultStore) CreateOperation(ctx context.Context, op ValidationOperation) (ValidationOperation, bool, error) {
	if err := validateStoredOperation(op); err != nil {
		return ValidationOperation{}, false, err
	}
	payload, err := json.Marshal(op)
	if err != nil {
		return ValidationOperation{}, false, fmt.Errorf("marshal validation operation %q: %w", op.ID, err)
	}
	updated := r.now()
	result, err := r.db.ExecContext(ctx, insertOperationSQL,
		op.ID, op.PlanID, op.PhaseID, op.IdempotencyKey, string(op.Status), op.QueuedAt, updated, string(payload),
	)
	if err != nil {
		return ValidationOperation{}, false, fmt.Errorf("create validation operation %q: %w", op.ID, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return ValidationOperation{}, false, fmt.Errorf("inspect validation operation insert %q: %w", op.ID, err)
	}
	if rows == 1 {
		return op, true, nil
	}
	var stored ValidationOperation
	var found bool
	if op.IdempotencyKey != "" {
		stored, found, err = r.operationByKey(ctx, op.PlanID, op.PhaseID, op.ExecutionID, op.ScopeGeneration, op.IdempotencyKey)
	} else {
		stored, found, err = r.activeOperation(ctx, op.PlanID, op.PhaseID, op.ExecutionID, op.ScopeGeneration)
	}
	if err != nil {
		return ValidationOperation{}, false, err
	}
	if !found {
		return ValidationOperation{}, false, fmt.Errorf("idempotent validation operation disappeared after conflict")
	}
	return stored, false, nil
}

func (r *sqliteResultStore) activeOperation(ctx context.Context, planID, phaseID, executionID string, scopeGeneration int) (ValidationOperation, bool, error) {
	return scanOperation(r.db.QueryRowContext(ctx, getActiveOperationSQL, planID, phaseID, executionID, scopeGeneration, string(OperationTerminal)))
}

func (r *sqliteResultStore) SaveOperation(ctx context.Context, op ValidationOperation) error {
	if err := validateStoredOperation(op); err != nil {
		return err
	}
	payload, err := json.Marshal(op)
	if err != nil {
		return fmt.Errorf("marshal validation operation %q: %w", op.ID, err)
	}
	result, err := r.db.ExecContext(ctx, updateOperationSQL, string(op.Status), r.now(), string(payload), op.ID)
	if err != nil {
		return fmt.Errorf("save validation operation %q: %w", op.ID, err)
	}
	if rows, _ := result.RowsAffected(); rows != 1 {
		return fmt.Errorf("validation operation %q not found", op.ID)
	}
	return nil
}

func (r *sqliteResultStore) GetOperation(ctx context.Context, id string) (ValidationOperation, bool, error) {
	return scanOperation(r.db.QueryRowContext(ctx, getOperationSQL, id))
}

func (r *sqliteResultStore) operationByKey(ctx context.Context, planID, phaseID, executionID string, scopeGeneration int, key string) (ValidationOperation, bool, error) {
	return scanOperation(r.db.QueryRowContext(ctx, getOperationByKeySQL, planID, phaseID, executionID, scopeGeneration, key))
}

func scanOperation(row *sql.Row) (ValidationOperation, bool, error) {
	var payload string
	if err := row.Scan(&payload); errors.Is(err, sql.ErrNoRows) {
		return ValidationOperation{}, false, nil
	} else if err != nil {
		return ValidationOperation{}, false, fmt.Errorf("read validation operation: %w", err)
	}
	var op ValidationOperation
	if err := json.Unmarshal([]byte(payload), &op); err != nil {
		return ValidationOperation{}, false, fmt.Errorf("decode validation operation: %w", err)
	}
	if err := validateStoredOperation(op); err != nil {
		return ValidationOperation{}, false, fmt.Errorf("read validation operation: %w", err)
	}
	return op, true, nil
}

// validateStoredOperation keeps persisted validation operations forward-only.
// Local data is upgraded by a one-shot, stopped-service migration; reads never
// perform an implicit compatibility transform.
func validateStoredOperation(op ValidationOperation) error {
	if op.SchemaVersion != CurrentOperationSchemaVersion {
		return fmt.Errorf("validation operation %q uses storage schema v%d; expected v%d (run the documented one-shot storage migration before starting plan-manager)", op.ID, op.SchemaVersion, CurrentOperationSchemaVersion)
	}
	for _, child := range op.Children {
		if child.Check.SemanticKey == "" {
			return fmt.Errorf("validation operation %q child %q is missing its typed check identity (run the documented one-shot storage migration before starting plan-manager)", op.ID, child.ID)
		}
	}
	return nil
}

func (r *sqliteResultStore) ListNonTerminalOperations(ctx context.Context) ([]ValidationOperation, error) {
	rows, err := r.db.QueryContext(ctx, listNonTerminalOperationsSQL, string(OperationTerminal))
	if err != nil {
		return nil, fmt.Errorf("list pending validation operations: %w", err)
	}
	defer rows.Close()
	var out []ValidationOperation
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, fmt.Errorf("scan pending validation operation: %w", err)
		}
		var op ValidationOperation
		if err := json.Unmarshal([]byte(payload), &op); err != nil {
			return nil, fmt.Errorf("decode pending validation operation: %w", err)
		}
		if err := validateStoredOperation(op); err != nil {
			return nil, fmt.Errorf("read pending validation operation: %w", err)
		}
		out = append(out, op)
	}
	return out, rows.Err()
}
