package rewrite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"go-code-graph/internal/clock"
)

// SQLiteStore is the production PlanStore implementation (REQ-P1-002).
// It persists plans in the `rewrite_plans` table and records per-op
// apply outcomes in `rewrite_operation_log`. Both tables are created
// by Schema() via EnsureSchemas at scenario startup.
//
// The store also satisfies the OperationLog seam, so the Service can
// hand the same instance to both fields without juggling two handles.
type SQLiteStore struct {
	db    *sql.DB
	clock clock.Clock
}

// NewSQLiteStore wires the production store against db. The clock seam
// supplies created_at / applied_at timestamps; production wires
// clock.System{} from main.go, tests inject a FakeClock.
func NewSQLiteStore(db *sql.DB, c clock.Clock) *SQLiteStore {
	if c == nil {
		c = clock.System{}
	}
	return &SQLiteStore{db: db, clock: c}
}

// Save persists plan keyed by plan.ID. Re-saving an existing plan is a
// no-op overwrite (matches MemoryStore semantics) — Plan derivation is
// deterministic so identical inputs intentionally collide.
func (s *SQLiteStore) Save(ctx context.Context, plan Plan) error {
	ops, err := encodeOperations(plan.Operations)
	if err != nil {
		return fmt.Errorf("encode operations: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
        INSERT INTO rewrite_plans(id, module_path, operations, created_at)
        VALUES (?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
            module_path = excluded.module_path,
            operations    = excluded.operations,
            created_at    = excluded.created_at
    `, string(plan.ID), plan.ModulePath, ops, s.clock.Now().Unix())
	if err != nil {
		return fmt.Errorf("save plan: %w", err)
	}
	return nil
}

// Load returns the stored plan or RewriteError{Kind: PlanNotFound}.
func (s *SQLiteStore) Load(ctx context.Context, id PlanID) (Plan, error) {
	row := s.db.QueryRowContext(ctx, `
        SELECT module_path, operations
        FROM rewrite_plans
        WHERE id = ?
    `, string(id))
	var modulePath, opsJSON string
	if err := row.Scan(&modulePath, &opsJSON); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Plan{}, RewriteError{
				Kind:    RewriteErrorPlanNotFound,
				Message: "unknown plan_id " + string(id),
			}
		}
		return Plan{}, fmt.Errorf("load plan: %w", err)
	}
	ops, err := decodeOperations(opsJSON)
	if err != nil {
		return Plan{}, fmt.Errorf("decode operations: %w", err)
	}
	return Plan{ID: id, ModulePath: modulePath, Operations: ops}, nil
}

// RecordApply appends per-op results to the operation log. Returns the
// first encountered error; partial writes are accepted (the rows that
// did persist are still useful). Dry-run callers MUST NOT invoke this.
func (s *SQLiteStore) RecordApply(ctx context.Context, planID PlanID, results []OperationResult) error {
	if len(results) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin oplog tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO rewrite_operation_log(plan_id, op_index, kind, status, message, applied_at)
        VALUES (?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		return fmt.Errorf("prepare oplog insert: %w", err)
	}
	defer stmt.Close()
	appliedAt := s.clock.Now().Unix()
	for i, r := range results {
		kind := ""
		if r.Operation != nil {
			kind = string(r.Operation.Kind())
		}
		if _, err := stmt.ExecContext(ctx, string(planID), i, kind, int(r.Status), r.Message, appliedAt); err != nil {
			return fmt.Errorf("oplog insert row %d: %w", i, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit oplog tx: %w", err)
	}
	return nil
}

// Compile-time assertions: SQLiteStore satisfies both seams.
var (
	_ PlanStore    = (*SQLiteStore)(nil)
	_ OperationLog = (*SQLiteStore)(nil)
)

// encodeOperations / decodeOperations serialize the typed Operation
// sum to/from the canonical JSON shape stored in rewrite_plans. The
// shape matches the one derivePlanID hashes, so a hand-decoded plan
// round-trips through derivePlanID without changing its id.
func encodeOperations(ops []Operation) (string, error) {
	out := make([]map[string]string, 0, len(ops))
	for _, op := range ops {
		switch o := op.(type) {
		case FileMove:
			out = append(out, map[string]string{
				"kind": string(OperationKindFileMove),
				"from": o.From,
				"to":   o.To,
			})
		case ImportRewrite:
			out = append(out, map[string]string{
				"kind": string(OperationKindImportRewrite),
				"old":  o.Old,
				"new":  o.New,
			})
		default:
			return "", fmt.Errorf("unknown operation kind %T", op)
		}
	}
	buf, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func decodeOperations(s string) ([]Operation, error) {
	var raw []map[string]string
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		return nil, err
	}
	out := make([]Operation, 0, len(raw))
	for _, r := range raw {
		switch OperationKind(r["kind"]) {
		case OperationKindFileMove:
			out = append(out, FileMove{From: r["from"], To: r["to"]})
		case OperationKindImportRewrite:
			out = append(out, ImportRewrite{Old: r["old"], New: r["new"]})
		default:
			return nil, fmt.Errorf("unknown operation kind %q", r["kind"])
		}
	}
	return out, nil
}
