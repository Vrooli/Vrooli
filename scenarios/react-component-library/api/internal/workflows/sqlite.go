package workflows

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"react-component-library/internal/clock"
)

const timestampFormat = time.RFC3339Nano

type sqliteRepository struct {
	db  *sql.DB
	clk clock.Clock
}

func NewSQLiteRepository(db *sql.DB, clk clock.Clock) Repository {
	if clk == nil {
		clk = clock.System{}
	}
	return &sqliteRepository{db: db, clk: clk}
}

func (r *sqliteRepository) Create(ctx context.Context, w Workflow) (Workflow, error) {
	if w.ID == "" {
		w.ID = uuid.NewString()
	}
	if w.CreatedAt.IsZero() {
		w.CreatedAt = r.clk.Now().UTC()
	}
	if w.UpdatedAt.IsZero() {
		w.UpdatedAt = w.CreatedAt
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO assisted_workflows (id,kind,asset_id,source_scenario,target_scenario,source_path,requested_version,agent_manager_task_id,agent_manager_run_id,idempotency_key,status,last_event_sequence,summary,error,created_at,updated_at,completed_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		w.ID, w.Kind, w.AssetID, w.SourceScenario, w.TargetScenario, w.SourcePath, w.RequestedVersion, w.AgentManagerTaskID, w.AgentManagerRunID, w.IdempotencyKey, w.Status, w.LastEventSequence, w.Summary, w.Error, stamp(w.CreatedAt), stamp(w.UpdatedAt), stamp(w.CompletedAt))
	if err != nil {
		return Workflow{}, fmt.Errorf("create workflow: %w", err)
	}
	return w, nil
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (Workflow, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,kind,asset_id,source_scenario,target_scenario,source_path,requested_version,agent_manager_task_id,agent_manager_run_id,idempotency_key,status,last_event_sequence,summary,error,created_at,updated_at,completed_at FROM assisted_workflows WHERE id=?`, id)
	w, err := scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Workflow{}, ErrNotFound
	}
	if err != nil {
		return Workflow{}, fmt.Errorf("get workflow: %w", err)
	}
	return w, nil
}

func (r *sqliteRepository) FindActiveByIdempotency(ctx context.Context, key string) (Workflow, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,kind,asset_id,source_scenario,target_scenario,source_path,requested_version,agent_manager_task_id,agent_manager_run_id,idempotency_key,status,last_event_sequence,summary,error,created_at,updated_at,completed_at FROM assisted_workflows WHERE idempotency_key=? AND status IN ('queued','running','parked')`, key)
	w, err := scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Workflow{}, ErrNotFound
	}
	return w, err
}

func (r *sqliteRepository) List(ctx context.Context, assetID, target string, active bool, limit int) ([]Workflow, error) {
	if limit <= 0 {
		limit = 100
	}
	clauses, args := []string{"1=1"}, []any{}
	if assetID != "" {
		clauses, args = append(clauses, "asset_id=?"), append(args, assetID)
	}
	if target != "" {
		clauses, args = append(clauses, "target_scenario=?"), append(args, target)
	}
	if active {
		clauses = append(clauses, "status IN ('queued','running','parked')")
	}
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, `SELECT id,kind,asset_id,source_scenario,target_scenario,source_path,requested_version,agent_manager_task_id,agent_manager_run_id,idempotency_key,status,last_event_sequence,summary,error,created_at,updated_at,completed_at FROM assisted_workflows WHERE `+strings.Join(clauses, " AND ")+` ORDER BY updated_at DESC LIMIT ?`, args...)
	if err != nil {
		return nil, fmt.Errorf("list workflows: %w", err)
	}
	defer rows.Close()
	var out []Workflow
	for rows.Next() {
		w, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) Update(ctx context.Context, w Workflow) (Workflow, error) {
	w.UpdatedAt = r.clk.Now().UTC()
	if !w.Status.Active() && w.CompletedAt.IsZero() {
		w.CompletedAt = w.UpdatedAt
	}
	res, err := r.db.ExecContext(ctx, `UPDATE assisted_workflows SET agent_manager_task_id=?,agent_manager_run_id=?,status=?,last_event_sequence=?,summary=?,error=?,updated_at=?,completed_at=? WHERE id=?`, w.AgentManagerTaskID, w.AgentManagerRunID, w.Status, w.LastEventSequence, w.Summary, w.Error, stamp(w.UpdatedAt), stamp(w.CompletedAt), w.ID)
	if err != nil {
		return Workflow{}, fmt.Errorf("update workflow: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return Workflow{}, ErrNotFound
	}
	return w, nil
}

type scanner interface{ Scan(...any) error }

func scan(s scanner) (w Workflow, err error) {
	var created, updated, completed string
	err = s.Scan(&w.ID, &w.Kind, &w.AssetID, &w.SourceScenario, &w.TargetScenario, &w.SourcePath, &w.RequestedVersion, &w.AgentManagerTaskID, &w.AgentManagerRunID, &w.IdempotencyKey, &w.Status, &w.LastEventSequence, &w.Summary, &w.Error, &created, &updated, &completed)
	if err != nil {
		return
	}
	w.CreatedAt, _ = time.Parse(timestampFormat, created)
	w.UpdatedAt, _ = time.Parse(timestampFormat, updated)
	if completed != "" {
		w.CompletedAt, _ = time.Parse(timestampFormat, completed)
	}
	return
}
func stamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(timestampFormat)
}
