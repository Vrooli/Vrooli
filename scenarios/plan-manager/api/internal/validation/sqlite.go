package validation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

var _ ResultStore = (*sqliteResultStore)(nil)
var _ OperationStore = (*sqliteResultStore)(nil)

const (
	insertResultSQL = `
INSERT INTO validation_results (id, plan_id, phase_id, verdict, staleness, commands_run, detail, ran_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO NOTHING`

	lastResultSQL = `
SELECT id, plan_id, phase_id, verdict, staleness, commands_run, detail, ran_at
FROM validation_results
WHERE plan_id = ? AND phase_id = ?
ORDER BY ran_at DESC, id DESC
LIMIT 1`
	resultByIDSQL = `
SELECT id, plan_id, phase_id, verdict, staleness, commands_run, detail, ran_at
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
	if _, err := r.db.ExecContext(ctx, insertResultSQL,
		res.ID, res.PlanID, res.PhaseID, string(res.Verdict), string(res.Staleness), string(cmds), res.Detail, ran,
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
		res       Result
		verdict   string
		staleness string
		commands  string
	)
	err := row.Scan(
		&res.ID, &res.PlanID, &res.PhaseID, &verdict, &staleness, &commands, &res.Detail, &res.RanAt,
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
WHERE plan_id = ? AND phase_id = ? AND idempotency_key = ? LIMIT 1`
	getActiveOperationSQL = `
SELECT payload_json FROM validation_operations
WHERE plan_id = ? AND phase_id = ? AND status <> ?
ORDER BY queued_at, id LIMIT 1`
	listNonTerminalOperationsSQL = `
SELECT payload_json FROM validation_operations WHERE status <> ? ORDER BY queued_at, id`
)

func (r *sqliteResultStore) CreateOperation(ctx context.Context, op ValidationOperation) (ValidationOperation, bool, error) {
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
	stored, found, err := r.activeOperation(ctx, op.PlanID, op.PhaseID)
	if err != nil {
		return ValidationOperation{}, false, err
	}
	if !found && op.IdempotencyKey != "" {
		stored, found, err = r.operationByKey(ctx, op.PlanID, op.PhaseID, op.IdempotencyKey)
	}
	if err != nil {
		return ValidationOperation{}, false, err
	}
	if !found {
		return ValidationOperation{}, false, fmt.Errorf("idempotent validation operation disappeared after conflict")
	}
	return stored, false, nil
}

func (r *sqliteResultStore) activeOperation(ctx context.Context, planID, phaseID string) (ValidationOperation, bool, error) {
	return scanOperation(r.db.QueryRowContext(ctx, getActiveOperationSQL, planID, phaseID, string(OperationTerminal)))
}

func (r *sqliteResultStore) SaveOperation(ctx context.Context, op ValidationOperation) error {
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

func (r *sqliteResultStore) operationByKey(ctx context.Context, planID, phaseID, key string) (ValidationOperation, bool, error) {
	return scanOperation(r.db.QueryRowContext(ctx, getOperationByKeySQL, planID, phaseID, key))
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
	op = migrateOperationForRead(op)
	return op, true, nil
}

// migrateOperationForRead preserves legacy command-only JSON rows without
// mutating user data during a read. New writes persist schema V2 typed checks;
// callers that checkpoint a migrated operation naturally compact it to V2.
func migrateOperationForRead(op ValidationOperation) ValidationOperation {
	if op.SchemaVersion >= CurrentOperationSchemaVersion {
		return op
	}
	for i := range op.Children {
		child := &op.Children[i]
		if child.Check.SemanticKey == "" {
			child.Check = parseKnownValidationCheck(child.Command)
			if child.Check.SemanticKey == "" {
				child.Check = ValidationCheck{Kind: ValidationCheckCustom, SemanticKey: "legacy:" + strings.TrimSpace(child.Command), Command: child.Command, Oracle: child.Oracle}
			}
		}
		if child.Command == "" {
			child.Command = child.Check.Command
		}
		if !child.Oracle {
			child.Oracle = child.Check.Oracle
		}
	}
	op.SchemaVersion = CurrentOperationSchemaVersion
	return op
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
		out = append(out, migrateOperationForRead(op))
	}
	return out, rows.Err()
}
