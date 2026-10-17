package cooking

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type sqliteRepository struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) Create(ctx context.Context, s Session) (Session, error) {
	if s.ID == "" || s.WorkspaceID == "" || s.RecipeID == "" || s.RecipeRevision < 1 || s.MethodID == "" || s.Scale == "" {
		return Session{}, ErrInvalid
	}
	if s.Status == "" {
		s.Status = "active"
	}
	if s.Status != "active" {
		return Session{}, ErrInvalid
	}
	now := time.Now().UTC()
	if s.StartedAt.IsZero() {
		s.StartedAt = now
	}
	s.UpdatedAt = s.StartedAt
	s.Version = 1
	if s.CompletedSteps == nil {
		s.CompletedSteps = []string{}
	}
	if s.Timers == nil {
		s.Timers = []Timer{}
	}
	completed, _ := json.Marshal(s.CompletedSteps)
	timers, _ := json.Marshal(s.Timers)
	_, err := r.db.ExecContext(ctx, `INSERT INTO cooking_sessions(workspace_id,session_id,recipe_id,recipe_revision,method_id,scale,current_step_index,completed_steps_json,timers_json,status,actual_yield,yield_unit,version,started_at,updated_at,finished_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, s.WorkspaceID, s.ID, s.RecipeID, s.RecipeRevision, s.MethodID, s.Scale, s.CurrentStepIndex, string(completed), string(timers), s.Status, s.ActualYield, s.YieldUnit, s.Version, stamp(s.StartedAt), stamp(s.UpdatedAt), "")
	if err != nil {
		if old, getErr := r.Get(ctx, s.WorkspaceID, s.ID); getErr == nil && samePin(old, s) {
			return old, nil
		}
		return Session{}, fmt.Errorf("create cooking session: %w", err)
	}
	return s, nil
}

func (r *sqliteRepository) Get(ctx context.Context, workspaceID, id string) (Session, error) {
	s, err := scanSession(r.db.QueryRowContext(ctx, `SELECT session_id,workspace_id,recipe_id,recipe_revision,method_id,scale,current_step_index,completed_steps_json,timers_json,status,actual_yield,yield_unit,version,started_at,updated_at,finished_at FROM cooking_sessions WHERE workspace_id=? AND session_id=?`, workspaceID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNotFound{id}
	}
	return s, err
}

func (r *sqliteRepository) List(ctx context.Context, workspaceID string) ([]Session, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT session_id,workspace_id,recipe_id,recipe_revision,method_id,scale,current_step_index,completed_steps_json,timers_json,status,actual_yield,yield_unit,version,started_at,updated_at,finished_at FROM cooking_sessions WHERE workspace_id=? ORDER BY updated_at DESC,session_id`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Session{}
	for rows.Next() {
		s, e := scanSession(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) Save(ctx context.Context, workspaceID, eventID, requestHash string, s Session, expectedVersion int64) (Session, error) {
	if workspaceID == "" || eventID == "" || requestHash == "" || s.ID == "" || s.WorkspaceID != workspaceID || expectedVersion < 1 {
		return Session{}, ErrInvalid
	}
	if s.Status != "active" && s.Status != "finished" {
		return Session{}, ErrInvalid
	}
	completed, _ := json.Marshal(s.CompletedSteps)
	timers, _ := json.Marshal(s.Timers)
	now := time.Now().UTC()
	finished := ""
	if s.Status == "finished" {
		if s.FinishedAt.IsZero() {
			s.FinishedAt = now
		}
		finished = stamp(s.FinishedAt)
	} else {
		s.FinishedAt = time.Time{}
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Session{}, err
	}
	defer tx.Rollback()
	var priorHash string
	var priorVersion int64
	lookupErr := tx.QueryRowContext(ctx, `SELECT request_hash,version FROM cooking_session_events WHERE workspace_id=? AND session_id=? AND event_id=?`, workspaceID, s.ID, eventID).Scan(&priorHash, &priorVersion)
	if lookupErr == nil {
		if priorHash != requestHash {
			return Session{}, ErrConflict{s.ID}
		}
		return sessionTx(ctx, tx, workspaceID, s.ID)
	}
	if !errors.Is(lookupErr, sql.ErrNoRows) {
		return Session{}, lookupErr
	}
	result, err := tx.ExecContext(ctx, `UPDATE cooking_sessions SET current_step_index=?,completed_steps_json=?,timers_json=?,status=?,actual_yield=?,yield_unit=?,version=version+1,updated_at=?,finished_at=? WHERE workspace_id=? AND session_id=? AND version=? AND status='active'`, s.CurrentStepIndex, string(completed), string(timers), s.Status, s.ActualYield, s.YieldUnit, stamp(now), finished, workspaceID, s.ID, expectedVersion)
	if err != nil {
		return Session{}, err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return Session{}, ErrConflict{s.ID}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO cooking_session_events(workspace_id,session_id,event_id,request_hash,version,created_at) VALUES(?,?,?,?,?,?)`, workspaceID, s.ID, eventID, requestHash, expectedVersion+1, stamp(now)); err != nil {
		return Session{}, err
	}
	if err = tx.Commit(); err != nil {
		return Session{}, err
	}
	return r.Get(ctx, workspaceID, s.ID)
}

func sessionTx(ctx context.Context, tx *sql.Tx, workspaceID, id string) (Session, error) {
	return scanSession(tx.QueryRowContext(ctx, `SELECT session_id,workspace_id,recipe_id,recipe_revision,method_id,scale,current_step_index,completed_steps_json,timers_json,status,actual_yield,yield_unit,version,started_at,updated_at,finished_at FROM cooking_sessions WHERE workspace_id=? AND session_id=?`, workspaceID, id))
}

type scanner interface{ Scan(...any) error }

func scanSession(row scanner) (Session, error) {
	var s Session
	var completed, timers, started, updated, finished string
	if err := row.Scan(&s.ID, &s.WorkspaceID, &s.RecipeID, &s.RecipeRevision, &s.MethodID, &s.Scale, &s.CurrentStepIndex, &completed, &timers, &s.Status, &s.ActualYield, &s.YieldUnit, &s.Version, &started, &updated, &finished); err != nil {
		return Session{}, err
	}
	if err := json.Unmarshal([]byte(completed), &s.CompletedSteps); err != nil {
		return Session{}, err
	}
	if err := json.Unmarshal([]byte(timers), &s.Timers); err != nil {
		return Session{}, err
	}
	var err error
	if s.StartedAt, err = time.Parse(time.RFC3339Nano, started); err != nil {
		return Session{}, err
	}
	if s.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated); err != nil {
		return Session{}, err
	}
	if finished != "" {
		s.FinishedAt, err = time.Parse(time.RFC3339Nano, finished)
	}
	return s, err
}
func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }
func samePin(a, b Session) bool {
	return a.WorkspaceID == b.WorkspaceID && a.RecipeID == b.RecipeID && a.RecipeRevision == b.RecipeRevision && a.MethodID == b.MethodID && a.Scale == b.Scale
}
