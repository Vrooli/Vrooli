package validation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"plan-manager/internal/clock"
	internalplans "plan-manager/internal/plans"
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
func NewSQLiteResultStore(db SQLExecutor, clk clock.Clock) ResultStore {
	return &sqliteResultStore{db: db, clock: clk}
}

var _ ResultStore = (*sqliteResultStore)(nil)

const (
	insertResultSQL = `
INSERT INTO validation_results (id, plan_id, phase_id, verdict, staleness, commands_run, detail, ran_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	lastResultSQL = `
SELECT id, plan_id, phase_id, verdict, staleness, commands_run, detail, ran_at
FROM validation_results
WHERE plan_id = ? AND phase_id = ?
ORDER BY ran_at DESC, id DESC
LIMIT 1`
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
	var (
		res       Result
		verdict   string
		staleness string
		commands  string
	)
	err := r.db.QueryRowContext(ctx, lastResultSQL, planID, phaseID).Scan(
		&res.ID, &res.PlanID, &res.PhaseID, &verdict, &staleness, &commands, &res.Detail, &res.RanAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Result{}, false, nil
	}
	if err != nil {
		return Result{}, false, fmt.Errorf("get last validation result for %q/%q: %w", planID, phaseID, err)
	}
	res.Verdict = Verdict(verdict)
	res.Staleness = internalplans.StalenessTier(staleness)
	if commands != "" {
		_ = json.Unmarshal([]byte(commands), &res.CommandsRun)
	}
	return res, true, nil
}

func (r *sqliteResultStore) now() string { return r.clock.Now().UTC().Format(resultTimeFormat) }
