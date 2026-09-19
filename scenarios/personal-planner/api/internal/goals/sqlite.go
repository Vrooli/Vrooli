package goals

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

type (
	SQLExecutor interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}
	sqliteRepository struct {
		db    SQLExecutor
		clock schedule.Clock
		id    func() string
	}
)

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clock, id: uuid.NewString}
}

func (r *sqliteRepository) List(ctx context.Context) ([]Goal, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,title,purpose,status,progress_method,CASE WHEN progress_method='milestones' THEN COALESCE((SELECT CAST(SUM(CASE WHEN status='complete' THEN 1 ELSE 0 END) * 10000 / NULLIF(COUNT(*),0) AS INTEGER) FROM milestones WHERE goal_id=goals.id),0) ELSE progress_basis_points END,target_basis_points,revision FROM goals ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Goal{}
	for rows.Next() {
		var g Goal
		if err := rows.Scan(&g.ID, &g.Title, &g.Purpose, &g.Status, &g.ProgressMethod, &g.ProgressBasisPoints, &g.TargetBasisPoints, &g.Revision); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) Create(ctx context.Context, g Goal) (Goal, error) {
	if g.ID == "" {
		g.ID = r.id()
	}
	now := r.clock.Now().Unix()
	_, err := r.db.ExecContext(ctx, `INSERT INTO goals (id,title,purpose,status,progress_method,progress_basis_points,target_basis_points,created_at,updated_at,revision) VALUES (?,?,?,?,?,?,?,?,?,?)`, g.ID, g.Title, g.Purpose, g.Status, g.ProgressMethod, g.ProgressBasisPoints, g.TargetBasisPoints, now, now, g.Revision)
	if err != nil {
		return Goal{}, fmt.Errorf("insert goal: %w", err)
	}
	return g, nil
}

func (r *sqliteRepository) UpdateProgress(ctx context.Context, id string, progress, revision int64) (Goal, error) {
	now := r.clock.Now().Unix()
	res, err := r.db.ExecContext(ctx, `UPDATE goals SET progress_basis_points=?,updated_at=?,revision=revision+1 WHERE id=? AND revision=?`, progress, now, id, revision)
	if err != nil {
		return Goal{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return Goal{}, err
	}
	if n != 1 {
		return Goal{}, ErrRevisionConflict{id}
	}
	var g Goal
	err = r.db.QueryRowContext(ctx, `SELECT id,title,purpose,status,progress_method,progress_basis_points,target_basis_points,revision FROM goals WHERE id=?`, id).Scan(&g.ID, &g.Title, &g.Purpose, &g.Status, &g.ProgressMethod, &g.ProgressBasisPoints, &g.TargetBasisPoints, &g.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		return Goal{}, ErrGoalNotFound{id}
	}
	return g, err
}

func (r *sqliteRepository) ListMilestones(ctx context.Context, goalID string) ([]Milestone, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT m.id,m.goal_id,m.title,m.criteria,m.due_date,m.status,m.revision,COALESCE(l.work_item_id,'') FROM milestones m LEFT JOIN milestone_work_links l ON l.milestone_id=m.id WHERE m.goal_id=? ORDER BY CASE WHEN m.due_date='' THEN 1 ELSE 0 END,m.due_date,m.updated_at`, goalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Milestone{}
	for rows.Next() {
		var m Milestone
		if err := rows.Scan(&m.ID, &m.GoalID, &m.Title, &m.Criteria, &m.DueDate, &m.Status, &m.Revision, &m.LinkedWorkItemID); err != nil {
			return nil, err
		}
		if err := r.loadPrerequisites(ctx, &m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) CreateMilestone(ctx context.Context, m Milestone) (Milestone, error) {
	if m.ID == "" {
		m.ID = r.id()
	}
	now := r.clock.Now().Unix()
	_, err := r.db.ExecContext(ctx, `INSERT INTO milestones (id,goal_id,title,criteria,due_date,status,created_at,updated_at,revision) VALUES (?,?,?,?,?,?,?,?,?)`, m.ID, m.GoalID, m.Title, m.Criteria, m.DueDate, m.Status, now, now, m.Revision)
	if err != nil {
		return Milestone{}, fmt.Errorf("insert milestone: %w", err)
	}
	if m.LinkedWorkItemID != "" {
		if _, err := r.db.ExecContext(ctx, `INSERT INTO milestone_work_links (milestone_id,work_item_id,created_at) VALUES (?,?,?)`, m.ID, m.LinkedWorkItemID, now); err != nil {
			return Milestone{}, fmt.Errorf("link milestone work item: %w", err)
		}
	}
	for _, prerequisiteID := range m.PrerequisiteMilestoneIDs {
		if _, err := r.db.ExecContext(ctx, `INSERT INTO milestone_prerequisites (milestone_id,prerequisite_milestone_id,created_at) VALUES (?,?,?)`, m.ID, prerequisiteID, now); err != nil {
			return Milestone{}, fmt.Errorf("link milestone prerequisite: %w", err)
		}
	}
	return m, nil
}

func (r *sqliteRepository) UpdateMilestoneStatus(ctx context.Context, id, status string, revision int64) (Milestone, error) {
	now := r.clock.Now().Unix()
	if status == MilestoneDone {
		var blocked bool
		if err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM milestone_prerequisites p JOIN milestones prerequisite ON prerequisite.id=p.prerequisite_milestone_id WHERE p.milestone_id=? AND prerequisite.status<>?)`, id, MilestoneDone).Scan(&blocked); err != nil {
			return Milestone{}, err
		}
		if blocked {
			return Milestone{}, ErrPrerequisitesIncomplete{id}
		}
	}
	res, err := r.db.ExecContext(ctx, `UPDATE milestones SET status=?,updated_at=?,revision=revision+1 WHERE id=? AND revision=?`, status, now, id, revision)
	if err != nil {
		return Milestone{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return Milestone{}, err
	}
	if n != 1 {
		return Milestone{}, ErrRevisionConflict{id}
	}
	var m Milestone
	err = r.db.QueryRowContext(ctx, `SELECT m.id,m.goal_id,m.title,m.criteria,m.due_date,m.status,m.revision,COALESCE(l.work_item_id,'') FROM milestones m LEFT JOIN milestone_work_links l ON l.milestone_id=m.id WHERE m.id=?`, id).Scan(&m.ID, &m.GoalID, &m.Title, &m.Criteria, &m.DueDate, &m.Status, &m.Revision, &m.LinkedWorkItemID)
	if errors.Is(err, sql.ErrNoRows) {
		return Milestone{}, ErrMilestoneNotFound{id}
	}
	if err != nil {
		return m, err
	}
	if err := r.loadPrerequisites(ctx, &m); err != nil {
		return Milestone{}, err
	}
	return m, nil
}

func (r *sqliteRepository) WorkItemExists(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM work_items WHERE id=?)`, id).Scan(&exists)
	return exists, err
}

func (r *sqliteRepository) MilestonesExistForGoal(ctx context.Context, goalID string, ids []string) (bool, error) {
	if len(ids) == 0 {
		return true, nil
	}
	query := `SELECT COUNT(*) FROM milestones WHERE goal_id=? AND id IN (` + placeholders(len(ids)) + `)`
	args := make([]any, 0, len(ids)+1)
	args = append(args, goalID)
	for _, id := range ids {
		args = append(args, id)
	}
	var count int
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return false, err
	}
	return count == len(ids), nil
}

func (r *sqliteRepository) loadPrerequisites(ctx context.Context, m *Milestone) error {
	rows, err := r.db.QueryContext(ctx, `SELECT prerequisite_milestone_id FROM milestone_prerequisites WHERE milestone_id=? ORDER BY prerequisite_milestone_id`, m.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	m.PrerequisiteMilestoneIDs = nil
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		m.PrerequisiteMilestoneIDs = append(m.PrerequisiteMilestoneIDs, id)
	}
	return rows.Err()
}

func placeholders(n int) string {
	out := "?"
	for i := 1; i < n; i++ {
		out += ",?"
	}
	return out
}

var _ = time.Time{}
