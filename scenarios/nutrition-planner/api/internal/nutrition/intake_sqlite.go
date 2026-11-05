package nutrition

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"nutrition-planner/internal/decimalx"
)

type intakeSQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type intakeClock interface{ Now() time.Time }

type wallClock struct{}

func (wallClock) Now() time.Time { return time.Now() }

type intakeSQLiteRepository struct {
	db    intakeSQLExecutor
	clock intakeClock
}

func NewSQLiteIntakeRepository(db intakeSQLExecutor, clocks ...intakeClock) IntakeRepository {
	clock := intakeClock(wallClock{})
	if len(clocks) > 0 && clocks[0] != nil {
		clock = clocks[0]
	}
	return &intakeSQLiteRepository{db: db, clock: clock}
}

func (r *intakeSQLiteRepository) Append(ctx context.Context, workspaceID string, event IntakeEvent) error {
	if workspaceID == "" || event.ID == "" || event.Date == "" || event.NutrientID == "" || event.Amount.IsUnknown() || event.Amount.IsZero() || event.Unit == "" {
		return errors.New("intake event requires workspace, id, date, nutrient, known amount, and unit")
	}
	if event.timeRecorded.IsZero() {
		event.timeRecorded = r.clock.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO nutrition_intake_events(workspace_id,event_id,event_date,recipe_id,recipe_revision,nutrient_id,amount,unit,reason,correction_of,recorded_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, workspaceID, event.ID, event.Date, event.RecipeID, event.RecipeRevision, event.NutrientID, event.Amount.String(), event.Unit, event.Reason, event.CorrectionOf, event.timeRecorded.UTC().Format(time.RFC3339Nano))
	if err == nil {
		return nil
	}
	var oldAmount, oldNutrient, oldUnit string
	rowErr := r.db.QueryRowContext(ctx, `SELECT nutrient_id,amount,unit FROM nutrition_intake_events WHERE workspace_id=? AND event_id=?`, workspaceID, event.ID).Scan(&oldNutrient, &oldAmount, &oldUnit)
	if rowErr != nil {
		return fmt.Errorf("append intake event: %w", err)
	}
	if oldNutrient != event.NutrientID || oldUnit != event.Unit || oldAmount != event.Amount.String() {
		return fmt.Errorf("idempotency key %q reused with different payload", event.ID)
	}
	return nil
}

func (r *intakeSQLiteRepository) List(ctx context.Context, workspaceID string) ([]IntakeEvent, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT event_id,event_date,recipe_id,recipe_revision,nutrient_id,amount,unit,reason,correction_of,recorded_at FROM nutrition_intake_events WHERE workspace_id=? ORDER BY event_date,recorded_at,event_id`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []IntakeEvent
	for rows.Next() {
		var event IntakeEvent
		var amount, recorded string
		if err := rows.Scan(&event.ID, &event.Date, &event.RecipeID, &event.RecipeRevision, &event.NutrientID, &amount, &event.Unit, &event.Reason, &event.CorrectionOf, &recorded); err != nil {
			return nil, err
		}
		if event.Amount, err = decimalx.Parse(amount); err != nil {
			return nil, err
		}
		if event.timeRecorded, err = time.Parse(time.RFC3339Nano, recorded); err != nil {
			return nil, err
		}
		event.WorkspaceID = workspaceID
		out = append(out, event)
	}
	return out, rows.Err()
}
