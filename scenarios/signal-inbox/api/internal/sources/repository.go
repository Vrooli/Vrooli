package sources

import (
	"context"
	"database/sql"
	"time"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type sqliteRepository struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) GetState(ctx context.Context, id string) (State, bool, error) {
	var state State
	var enabled int
	var lastRun string
	err := r.db.QueryRowContext(ctx, "SELECT adapter_id,risk_tier,enabled,last_run_at,last_error,disabled_reason FROM adapter_state WHERE adapter_id=?", id).Scan(&state.AdapterID, &state.RiskTier, &enabled, &lastRun, &state.LastError, &state.DisabledReason)
	if err == sql.ErrNoRows {
		return State{}, false, nil
	}
	if err != nil {
		return State{}, false, err
	}
	state.Enabled = enabled != 0
	if lastRun != "" {
		state.LastRunAt, _ = time.Parse(time.RFC3339Nano, lastRun)
	}
	return state, true, nil
}

func (r *sqliteRepository) PutState(ctx context.Context, state State) error {
	lastRun := ""
	if !state.LastRunAt.IsZero() {
		lastRun = state.LastRunAt.UTC().Format(time.RFC3339Nano)
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO adapter_state(adapter_id,risk_tier,enabled,last_run_at,last_error,disabled_reason) VALUES(?,?,?,?,?,?) ON CONFLICT(adapter_id) DO UPDATE SET risk_tier=excluded.risk_tier,enabled=excluded.enabled,last_run_at=excluded.last_run_at,last_error=excluded.last_error,disabled_reason=excluded.disabled_reason`, state.AdapterID, state.RiskTier, boolInt(state.Enabled), lastRun, state.LastError, state.DisabledReason)
	return err
}

func (r *sqliteRepository) AppendRun(ctx context.Context, result ImportResult, started, finished time.Time) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO import_run(id,adapter_id,created_count,duplicate_count,failed_count,started_at,finished_at) VALUES(?,?,?,?,?,?,?)", result.RunID, result.AdapterID, result.Created, result.Duplicated, result.Failed, started.UTC().Format(time.RFC3339Nano), finished.UTC().Format(time.RFC3339Nano))
	return err
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
