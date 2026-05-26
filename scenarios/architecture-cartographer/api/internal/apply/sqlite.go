package apply

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"architecture-cartographer/internal/clock"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock clock.Clock
}

func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

const planTimeFormat = time.RFC3339Nano

type planPayload struct {
	Operations  []Operation `json:"operations,omitempty"`
	ConflictIDs []string    `json:"conflict_ids,omitempty"`
}

const (
	insertPlanSQL  = `INSERT INTO apply_plans (id, scenario, domain, payload, planned_at) VALUES (?, ?, ?, ?, ?)`
	selectPlanSQL  = `SELECT id, scenario, domain, payload, planned_at FROM apply_plans WHERE id = ?`
	listRunsSQL    = `SELECT id, plan_id, scenario, domain, status, build_log, started_at, finished_at FROM apply_runs WHERE scenario = ? ORDER BY started_at DESC, id DESC LIMIT ?`
	getBaselineSQL = `SELECT scenario, green, toolchain, log, captured_at FROM apply_baselines WHERE scenario = ?`
)

func (r *sqliteRepository) SavePlan(ctx context.Context, p Plan) (Plan, error) {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if p.PlannedAt.IsZero() {
		p.PlannedAt = r.clock.Now().UTC()
	}
	payload, err := json.Marshal(planPayload{Operations: p.Operations, ConflictIDs: p.ConflictIDs})
	if err != nil {
		return Plan{}, fmt.Errorf("encode plan: %w", err)
	}
	_, err = r.db.ExecContext(ctx, insertPlanSQL,
		p.ID, p.Scenario, p.Domain, payload, p.PlannedAt.Format(planTimeFormat),
	)
	if err != nil {
		return Plan{}, fmt.Errorf("insert plan %q: %w", p.ID, err)
	}
	return p, nil
}

func (r *sqliteRepository) GetPlan(ctx context.Context, id string) (Plan, error) {
	row := r.db.QueryRowContext(ctx, selectPlanSQL, id)
	var (
		p         Plan
		payload   []byte
		plannedAt string
	)
	if err := row.Scan(&p.ID, &p.Scenario, &p.Domain, &payload, &plannedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Plan{}, fmt.Errorf("apply plan %q not found", id)
		}
		return Plan{}, fmt.Errorf("get plan %q: %w", id, err)
	}
	t, err := time.Parse(planTimeFormat, plannedAt)
	if err != nil {
		return Plan{}, fmt.Errorf("parse planned_at: %w", err)
	}
	p.PlannedAt = t
	if len(payload) > 0 {
		var pp planPayload
		if err := json.Unmarshal(payload, &pp); err != nil {
			return Plan{}, fmt.Errorf("decode plan payload: %w", err)
		}
		p.Operations = pp.Operations
		p.ConflictIDs = pp.ConflictIDs
	}
	return p, nil
}

func (r *sqliteRepository) ListRuns(ctx context.Context, f ListRunsFilter) (RunPage, error) {
	limit := f.PageSize
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, listRunsSQL, f.Scenario, limit)
	if err != nil {
		return RunPage{}, fmt.Errorf("list apply runs: %w", err)
	}
	defer rows.Close()

	var out []ApplyRun
	for rows.Next() {
		var (
			a        ApplyRun
			status   string
			started  string
			finished string
		)
		if err := rows.Scan(&a.ID, &a.PlanID, &a.Scenario, &a.Domain, &status, &a.BuildLog, &started, &finished); err != nil {
			return RunPage{}, err
		}
		a.Status = ApplyStatus(status)
		if t, err := time.Parse(planTimeFormat, started); err == nil {
			a.StartedAt = t
		}
		if finished != "" {
			if t, err := time.Parse(planTimeFormat, finished); err == nil {
				a.FinishedAt = t
			}
		}
		if f.Domain != "" && a.Domain != f.Domain {
			continue
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return RunPage{}, fmt.Errorf("iterate apply runs: %w", err)
	}
	return RunPage{Runs: out}, nil
}

func (r *sqliteRepository) GetBaseline(ctx context.Context, scenario string) (BuildBaseline, error) {
	row := r.db.QueryRowContext(ctx, getBaselineSQL, scenario)
	var (
		b          BuildBaseline
		green      int
		capturedAt string
	)
	if err := row.Scan(&b.Scenario, &green, &b.Toolchain, &b.Log, &capturedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return BuildBaseline{Scenario: scenario}, nil
		}
		return BuildBaseline{}, fmt.Errorf("get baseline: %w", err)
	}
	b.Green = green != 0
	if t, err := time.Parse(planTimeFormat, capturedAt); err == nil {
		b.CapturedAt = t
	}
	return b, nil
}
