package entitlements

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type sqliteRepository struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) Get(ctx context.Context, workspaceID string) (State, error) {
	if workspaceID == "" {
		return State{}, errors.New("workspace_id is required")
	}
	var state State
	var enabled int
	err := r.db.QueryRowContext(ctx, `SELECT version,optional_compute,monthly_limit,used_units FROM workspace_entitlements WHERE workspace_id=?`, workspaceID).Scan(&state.Version, &enabled, &state.MonthlyLimit, &state.Used)
	if err == sql.ErrNoRows {
		state = New(workspaceID)
		if _, err := r.db.ExecContext(ctx, `INSERT INTO workspace_entitlements(workspace_id,version,optional_compute,monthly_limit,used_units,updated_at) VALUES(?,?,?,?,?,?)`, workspaceID, state.Version, boolInt(state.OptionalCompute), state.MonthlyLimit, state.Used, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return State{}, err
		}
		return state, nil
	}
	if err != nil {
		return State{}, err
	}
	state.WorkspaceID, state.OptionalCompute = workspaceID, enabled != 0
	state.AppliedEvents = map[string]int64{}
	return state, nil
}

func (r *sqliteRepository) ReserveOptional(ctx context.Context, workspaceID, operationID string, units int64) (State, error) {
	if operationID == "" || units < 0 {
		return State{}, errors.New("optional reservation requires operation id and non-negative units")
	}
	state, err := r.Get(ctx, workspaceID)
	if err != nil {
		return State{}, err
	}
	var existing int64
	if err := r.db.QueryRowContext(ctx, `SELECT units FROM entitlement_reservations WHERE workspace_id=? AND operation_id=?`, workspaceID, operationID).Scan(&existing); err == nil {
		if existing != units {
			return State{}, errors.New("entitlement operation reused with different units")
		}
		return state, nil
	} else if err != sql.ErrNoRows {
		return State{}, err
	}
	if err := state.CanUseOptional(units); err != nil {
		return State{}, err
	}
	if _, err := r.db.ExecContext(ctx, `INSERT INTO entitlement_reservations(workspace_id,operation_id,units,created_at) VALUES(?,?,?,?)`, workspaceID, operationID, units, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return State{}, err
	}
	state.Used += units
	if _, err := r.db.ExecContext(ctx, `UPDATE workspace_entitlements SET used_units=?,updated_at=? WHERE workspace_id=?`, state.Used, time.Now().UTC().Format(time.RFC3339Nano), workspaceID); err != nil {
		return State{}, err
	}
	return state, nil
}

func (r *sqliteRepository) ApplyEvent(ctx context.Context, workspaceID string, event Event) (State, error) {
	if event.ID == "" {
		return State{}, errors.New("entitlement event id is required")
	}
	state, err := r.Get(ctx, workspaceID)
	if err != nil {
		return State{}, err
	}
	if event.Version <= state.Version {
		return state, nil
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE workspace_entitlements SET version=?,optional_compute=?,monthly_limit=?,updated_at=? WHERE workspace_id=? AND version<?`, event.Version, boolInt(event.OptionalCompute), event.MonthlyLimit, time.Now().UTC().Format(time.RFC3339Nano), workspaceID, event.Version); err != nil {
		return State{}, err
	}
	state.Version, state.OptionalCompute, state.MonthlyLimit = event.Version, event.OptionalCompute, event.MonthlyLimit
	return state, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
