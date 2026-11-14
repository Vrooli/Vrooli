package analysisoperations

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SQLExecutor is implemented by the routed database and isolated unit pools.
type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type SQLRepository struct{ db SQLExecutor }

func NewRepository(db SQLExecutor) *SQLRepository { return &SQLRepository{db: db} }

// Submit with nil scenarios only replays or binds to existing active work.
// ErrNotFound asks the manager to obtain a source snapshot before resubmitting.
func (s *SQLRepository) Submit(ctx context.Context, callerID string, spec Spec, scenarios []string, at time.Time) (Operation, bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Operation{}, false, err
	}
	defer tx.Rollback()
	var id, signature string
	err = tx.QueryRowContext(ctx, `SELECT operation_id, signature FROM analysis_operation_requests WHERE caller_id=?`, callerID).Scan(&id, &signature)
	if err == nil {
		if signature != spec.signature() {
			return Operation{}, false, ErrConflict
		}
		if err := tx.Commit(); err != nil {
			return Operation{}, false, err
		}
		op, err := s.Get(ctx, id)
		return op, true, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Operation{}, false, err
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM analysis_operation_requests`).Scan(&count); err != nil {
		return Operation{}, false, err
	}
	if count >= MaxReceiptKeys {
		return Operation{}, false, ErrCapacity
	}
	err = tx.QueryRowContext(ctx, `SELECT id FROM analysis_operations WHERE signature=? AND state IN ('queued','running')`, spec.signature()).Scan(&id)
	reused := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Operation{}, false, err
	}
	if reused {
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM analysis_operation_requests WHERE operation_id=?`, id).Scan(&count); err != nil {
			return Operation{}, false, err
		}
		if count >= MaxCallerIDs {
			return Operation{}, false, ErrCapacity
		}
	} else {
		if scenarios == nil {
			return Operation{}, false, ErrNotFound
		}
		id = uuid.NewString()
		if _, err := tx.ExecContext(ctx, `INSERT INTO analysis_operations(id,kind,apply_graph,signature,state,created_at,total) VALUES(?,?,?,?,?,?,?)`, id, spec.Kind, spec.ApplyGraph, spec.signature(), Queued, stamp(at), len(scenarios)); err != nil {
			return Operation{}, false, err
		}
		for _, scenario := range scenarios {
			if _, err := tx.ExecContext(ctx, `INSERT INTO analysis_operation_scenarios(operation_id,scenario,state) VALUES(?,?,?)`, id, scenario, Queued); err != nil {
				return Operation{}, false, err
			}
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO analysis_operation_requests(caller_id,operation_id,signature) VALUES(?,?,?)`, callerID, id, spec.signature()); err != nil {
		return Operation{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return Operation{}, false, err
	}
	op, err := s.Get(ctx, id)
	return op, reused, err
}

const operationSelect = `SELECT id,kind,apply_graph,state,created_at,started_at,completed_at,total,completed,failures,error,graph_json,details_pruned FROM analysis_operations`

func (s *SQLRepository) Get(ctx context.Context, id string) (Operation, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return Operation{}, err
	}
	defer tx.Rollback()
	op, err := loadOperation(ctx, tx, id)
	if err != nil {
		return Operation{}, err
	}
	if err := tx.Commit(); err != nil {
		return Operation{}, err
	}
	return op, nil
}

type operationReader interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func loadOperation(ctx context.Context, reader operationReader, id string) (Operation, error) {
	var op Operation
	var created string
	var started, completed, graph sql.NullString
	err := reader.QueryRowContext(ctx, operationSelect+` WHERE id=?`, id).Scan(&op.ID, &op.Spec.Kind, &op.Spec.ApplyGraph, &op.State, &created, &started, &completed, &op.Total, &op.Completed, &op.Failures, &op.Error, &graph, &op.DetailsPruned)
	if errors.Is(err, sql.ErrNoRows) {
		return Operation{}, ErrNotFound
	}
	if err != nil {
		return Operation{}, err
	}
	if op.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
		return Operation{}, err
	}
	if started.Valid {
		t, err := time.Parse(time.RFC3339Nano, started.String)
		if err != nil {
			return Operation{}, err
		}
		op.StartedAt = &t
	}
	if completed.Valid {
		t, err := time.Parse(time.RFC3339Nano, completed.String)
		if err != nil {
			return Operation{}, err
		}
		op.CompletedAt = &t
	}
	if graph.Valid {
		if err := json.Unmarshal([]byte(graph.String), &op.Graph); err != nil {
			return Operation{}, err
		}
	}
	rows, err := reader.QueryContext(ctx, `SELECT scenario,state,detected_resources,scenario_dependencies,deployment_dependencies,duration_ns,error FROM analysis_operation_scenarios WHERE operation_id=? ORDER BY scenario`, id)
	if err != nil {
		return Operation{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var item ScenarioOutcome
		if err := rows.Scan(&item.Scenario, &item.State, &item.DetectedResources, &item.ScenarioDependencies, &item.DeploymentDependencies, &item.Duration, &item.Error); err != nil {
			return Operation{}, err
		}
		op.Scenarios = append(op.Scenarios, item)
	}
	return op, rows.Err()
}

func (s *SQLRepository) ClaimNext(ctx context.Context, at time.Time) (Operation, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Operation{}, err
	}
	defer tx.Rollback()
	var id string
	err = tx.QueryRowContext(ctx, `UPDATE analysis_operations SET state='running',started_at=?
WHERE id=(SELECT id FROM analysis_operations WHERE state='queued' ORDER BY rowid LIMIT 1) AND state='queued' RETURNING id`, stamp(at)).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return Operation{}, ErrNotFound
	}
	if err != nil {
		return Operation{}, err
	}
	op, err := loadOperation(ctx, tx, id)
	if err != nil {
		return Operation{}, err
	}
	if err := tx.Commit(); err != nil {
		return Operation{}, err
	}
	return op, nil
}

func requireChanged(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrState
	}
	return nil
}

func (s *SQLRepository) StartScenario(ctx context.Context, id, scenario string) error {
	return requireChanged(s.db.ExecContext(ctx, `UPDATE analysis_operation_scenarios SET state='running' WHERE operation_id=? AND scenario=? AND state='queued'
AND EXISTS(SELECT 1 FROM analysis_operations WHERE id=? AND state='running')`, id, scenario, id))
}

func (s *SQLRepository) CompleteScenario(ctx context.Context, id string, item ScenarioOutcome) error {
	if (item.State != Succeeded && item.State != Failed) || item.DetectedResources < 0 || item.ScenarioDependencies < 0 || item.DeploymentDependencies < 0 || item.Duration < 0 {
		return ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := requireChanged(tx.ExecContext(ctx, `UPDATE analysis_operation_scenarios SET state=?,detected_resources=?,scenario_dependencies=?,deployment_dependencies=?,duration_ns=?,error=?
WHERE operation_id=? AND scenario=? AND state='running' AND EXISTS(SELECT 1 FROM analysis_operations WHERE id=? AND state='running')`, item.State, item.DetectedResources, item.ScenarioDependencies, item.DeploymentDependencies, item.Duration, bounded(item.Error), id, item.Scenario, id)); err != nil {
		return err
	}
	failure := 0
	if item.State == Failed {
		failure = 1
	}
	if err := requireChanged(tx.ExecContext(ctx, `UPDATE analysis_operations SET completed=completed+1,failures=failures+? WHERE id=? AND state='running'`, failure, id)); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLRepository) Finish(ctx context.Context, id string, state State, message string, graph *GraphOutcome, at time.Time) error {
	if !state.Terminal() {
		return fmt.Errorf("terminal state required: %w", ErrInvalid)
	}
	var graphJSON any
	if graph != nil {
		data, err := json.Marshal(graph)
		if err != nil {
			return err
		}
		graphJSON = string(data)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := requireChanged(tx.ExecContext(ctx, `UPDATE analysis_operations SET state=?,completed_at=?,error=?,graph_json=? WHERE id=? AND state='running'
AND (? != 'succeeded' OR (completed=total AND failures=0))`, state, stamp(at), bounded(message), graphJSON, id, state)); err != nil {
		return err
	}
	// Items not completed when execution stops remain explicitly unexecuted.
	if _, err := tx.ExecContext(ctx, `UPDATE analysis_operation_scenarios SET state='interrupted',error=? WHERE operation_id=? AND state IN ('queued','running')`, bounded(message), id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLRepository) Recover(ctx context.Context, at time.Time) ([]string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT id FROM analysis_operations WHERE state='running'`)
	if err != nil {
		return nil, err
	}
	var interrupted []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		interrupted = append(interrupted, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	const message = "server stopped before operation completion; this receipt will not execute again"
	if _, err := tx.ExecContext(ctx, `UPDATE analysis_operation_scenarios SET state='interrupted',error=? WHERE state IN ('queued','running') AND operation_id IN(SELECT id FROM analysis_operations WHERE state='running')`, message); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE analysis_operations SET state='interrupted',completed_at=?,error=? WHERE state='running'`, stamp(at), message); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return interrupted, nil
}

func (s *SQLRepository) Prune(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT id FROM analysis_operations WHERE details_pruned=0 AND state NOT IN ('queued','running') AND id NOT IN(
SELECT id FROM analysis_operations WHERE state NOT IN ('queued','running') ORDER BY rowid DESC LIMIT ?)`, RetainedDetails)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, `DELETE FROM analysis_operation_scenarios WHERE operation_id=?`, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE analysis_operations SET details_pruned=1 WHERE id=?`, id); err != nil {
			return err
		}
	}
	// Receipt headers and request bindings are retained, so a replay of an old
	// key never becomes new execution after detail retention runs.
	return tx.Commit()
}

func stamp(at time.Time) string { return at.UTC().Format(time.RFC3339Nano) }
