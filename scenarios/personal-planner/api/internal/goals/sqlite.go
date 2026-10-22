package goals

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/schedule"
)

type (
	SQLExecutor interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
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

func goalOwner(ctx context.Context) (string, error) {
	principal, ok := identity.PrincipalFromContext(ctx)
	if !ok || principal.OwnerSubject() == "" {
		return "", ErrUnverifiedGoalPrincipal{}
	}
	return principal.OwnerSubject(), nil
}

func (r *sqliteRepository) List(ctx context.Context) ([]Goal, error) {
	owner, err := goalOwner(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT g.id,g.title,g.purpose,g.status,g.progress_method,CASE WHEN g.progress_method='milestones' THEN COALESCE((SELECT CAST(SUM(CASE WHEN status='complete' THEN 1 ELSE 0 END) * 10000 / NULLIF(COUNT(*),0) AS INTEGER) FROM milestones WHERE goal_id=g.id),0) ELSE g.progress_basis_points END,g.target_basis_points,g.revision,COALESCE(g.target_date,''),m.unit,m.target_amount_minor,m.currency_code,m.basis,m.recurrence,m.period FROM goals g JOIN goal_owners o ON o.goal_id=g.id AND o.owner_subject=? LEFT JOIN goal_measures m ON m.goal_id=g.id ORDER BY g.updated_at DESC`, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Goal{}
	for rows.Next() {
		var g Goal
		var unit, currency, basis, recurrence, period sql.NullString
		var amount sql.NullInt64
		if err := rows.Scan(&g.ID, &g.Title, &g.Purpose, &g.Status, &g.ProgressMethod, &g.ProgressBasisPoints, &g.TargetBasisPoints, &g.Revision, &g.TargetDate, &unit, &amount, &currency, &basis, &recurrence, &period); err != nil {
			return nil, err
		}
		if unit.Valid {
			g.Measure = &Measure{Unit: unit.String, TargetAmountMinor: amount.Int64, CurrencyCode: currency.String, Basis: basis.String, Recurrence: recurrence.String, Period: period.String}
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) Create(ctx context.Context, g Goal) (Goal, error) {
	owner, err := goalOwner(ctx)
	if err != nil {
		return Goal{}, err
	}
	if g.ID == "" {
		g.ID = r.id()
	}
	now := r.clock.Now().Unix()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Goal{}, err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO goals (id,title,purpose,status,progress_method,progress_basis_points,target_basis_points,target_date,created_at,updated_at,revision) VALUES (?,?,?,?,?,?,?,?,?,?,?)`, g.ID, g.Title, g.Purpose, g.Status, g.ProgressMethod, g.ProgressBasisPoints, g.TargetBasisPoints, nullIfEmpty(g.TargetDate), now, now, g.Revision)
	if err != nil {
		return Goal{}, fmt.Errorf("insert goal: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO goal_owners (goal_id,owner_subject) VALUES (?,?)`, g.ID, owner); err != nil {
		return Goal{}, fmt.Errorf("assign goal owner: %w", err)
	}
	if g.Measure != nil {
		m := g.Measure
		if _, err := tx.ExecContext(ctx, `INSERT INTO goal_measures (goal_id,unit,target_amount_minor,currency_code,basis,recurrence,period) VALUES (?,?,?,?,?,?,?)`, g.ID, m.Unit, m.TargetAmountMinor, m.CurrencyCode, m.Basis, m.Recurrence, m.Period); err != nil {
			return Goal{}, fmt.Errorf("insert goal measure: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Goal{}, err
	}
	return g, nil
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (r *sqliteRepository) SetTargetDate(ctx context.Context, id, targetDate, reason string, expectedRevision int64) error {
	owner, err := goalOwner(ctx)
	if err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var previous sql.NullString
	var revision int64
	if err := tx.QueryRowContext(ctx, `SELECT g.target_date,g.revision FROM goals g JOIN goal_owners o ON o.goal_id=g.id WHERE g.id=? AND o.owner_subject=?`, id, owner).Scan(&previous, &revision); errors.Is(err, sql.ErrNoRows) {
		return ErrGoalNotFound{id}
	} else if err != nil {
		return err
	}
	if revision != expectedRevision {
		return ErrRevisionConflict{id}
	}
	now := r.clock.Now().UTC()
	nextRevision := expectedRevision + 1
	res, err := tx.ExecContext(ctx, `UPDATE goals SET target_date=?,updated_at=?,revision=? WHERE id=? AND revision=? AND EXISTS(SELECT 1 FROM goal_owners o WHERE o.goal_id=goals.id AND o.owner_subject=?)`, targetDate, now.Unix(), nextRevision, id, expectedRevision, owner)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrRevisionConflict{id}
	}
	priorDate := previous.String
	if _, err := tx.ExecContext(ctx, `INSERT INTO goal_target_date_history (goal_id,previous_target_date,target_date,reason,resulting_revision,changed_at) VALUES (?,?,?,?,?,?)`, id, priorDate, targetDate, reason, nextRevision, now.Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("insert target date history: %w", err)
	}
	return tx.Commit()
}

func (r *sqliteRepository) ListTargetDateHistory(ctx context.Context, goalID string) ([]TargetDateHistory, error) {
	owner, err := goalOwner(ctx)
	if err != nil {
		return nil, err
	}
	var exists int
	if err := r.db.QueryRowContext(ctx, `SELECT 1 FROM goals g JOIN goal_owners o ON o.goal_id=g.id WHERE g.id=? AND o.owner_subject=?`, goalID, owner).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrGoalNotFound{goalID}
	} else if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT goal_id,previous_target_date,target_date,reason,resulting_revision,changed_at FROM goal_target_date_history WHERE goal_id=? ORDER BY history_id ASC`, goalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	history := []TargetDateHistory{}
	for rows.Next() {
		var item TargetDateHistory
		if err := rows.Scan(&item.GoalID, &item.PreviousTargetDate, &item.TargetDate, &item.Reason, &item.Revision, &item.ChangedAt); err != nil {
			return nil, err
		}
		history = append(history, item)
	}
	return history, rows.Err()
}

func (r *sqliteRepository) UpdateProgress(ctx context.Context, id string, progress, revision int64) (Goal, error) {
	owner, err := goalOwner(ctx)
	if err != nil {
		return Goal{}, err
	}
	var owned bool
	if err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM goals g JOIN goal_owners o ON o.goal_id=g.id WHERE g.id=? AND o.owner_subject=?)`, id, owner).Scan(&owned); err != nil {
		return Goal{}, err
	}
	if !owned {
		return Goal{}, ErrGoalNotFound{id}
	}
	var measured bool
	if err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM goal_measures m JOIN goal_owners o ON o.goal_id=m.goal_id WHERE m.goal_id=? AND o.owner_subject=?)`, id, owner).Scan(&measured); err != nil {
		return Goal{}, err
	}
	if measured {
		return Goal{}, ErrInvalidGoal{"progress", "measured progress remains unknown until observations are supported"}
	}
	now := r.clock.Now().Unix()
	status := StatusActive
	if progress == 10000 {
		status = "complete"
	}
	res, err := r.db.ExecContext(ctx, `UPDATE goals SET progress_basis_points=?,status=?,updated_at=?,revision=revision+1 WHERE id=? AND revision=? AND EXISTS(SELECT 1 FROM goal_owners o WHERE o.goal_id=goals.id AND o.owner_subject=?)`, progress, status, now, id, revision, owner)
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
	completedDate := ""
	if status == "complete" {
		completedDate = r.clock.Now().Format("2006-01-02")
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE goals SET completed_date=? WHERE id=? AND EXISTS(SELECT 1 FROM goal_owners o WHERE o.goal_id=goals.id AND o.owner_subject=?)`, completedDate, id, owner); err != nil {
		return Goal{}, fmt.Errorf("persist goal completion date: %w", err)
	}
	var g Goal
	err = r.db.QueryRowContext(ctx, `SELECT g.id,g.title,g.purpose,g.status,g.progress_method,g.progress_basis_points,g.target_basis_points,g.revision FROM goals g JOIN goal_owners o ON o.goal_id=g.id WHERE g.id=? AND o.owner_subject=?`, id, owner).Scan(&g.ID, &g.Title, &g.Purpose, &g.Status, &g.ProgressMethod, &g.ProgressBasisPoints, &g.TargetBasisPoints, &g.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		return Goal{}, ErrGoalNotFound{id}
	}
	return g, err
}

func (r *sqliteRepository) ListMilestones(ctx context.Context, goalID string) ([]Milestone, error) {
	owner, err := goalOwner(ctx)
	if err != nil {
		return nil, err
	}
	var exists int
	if err := r.db.QueryRowContext(ctx, `SELECT 1 FROM goals g JOIN goal_owners o ON o.goal_id=g.id WHERE g.id=? AND o.owner_subject=?`, goalID, owner).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
		return nil, ErrGoalNotFound{goalID}
	} else if err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT m.id,m.goal_id,m.title,m.criteria,m.due_date,m.status,m.revision,COALESCE(l.work_item_id,'') FROM milestones m JOIN goal_owners o ON o.goal_id=m.goal_id AND o.owner_subject=? LEFT JOIN milestone_work_links l ON l.milestone_id=m.id WHERE m.goal_id=? ORDER BY CASE WHEN m.due_date='' THEN 1 ELSE 0 END,m.due_date,m.updated_at`, owner, goalID)
	if err != nil {
		return nil, err
	}
	out := []Milestone{}
	for rows.Next() {
		var m Milestone
		if err := rows.Scan(&m.ID, &m.GoalID, &m.Title, &m.Criteria, &m.DueDate, &m.Status, &m.Revision, &m.LinkedWorkItemID); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range out {
		if err := r.loadPrerequisites(ctx, &out[i]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (r *sqliteRepository) CreateMilestone(ctx context.Context, m Milestone) (Milestone, error) {
	owner, err := goalOwner(ctx)
	if err != nil {
		return Milestone{}, err
	}
	if m.ID == "" {
		m.ID = r.id()
	}
	now := r.clock.Now().Unix()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Milestone{}, err
	}
	defer tx.Rollback()
	var goalExists int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM goals g JOIN goal_owners o ON o.goal_id=g.id WHERE g.id=? AND o.owner_subject=?`, m.GoalID, owner).Scan(&goalExists); errors.Is(err, sql.ErrNoRows) {
		return Milestone{}, ErrGoalNotFound{m.GoalID}
	} else if err != nil {
		return Milestone{}, err
	}
	for _, prerequisiteID := range m.PrerequisiteMilestoneIDs {
		var valid int
		if err := tx.QueryRowContext(ctx, `SELECT 1 FROM milestones WHERE id=? AND goal_id=?`, prerequisiteID, m.GoalID).Scan(&valid); errors.Is(err, sql.ErrNoRows) {
			return Milestone{}, ErrMilestoneNotFound{prerequisiteID}
		} else if err != nil {
			return Milestone{}, err
		}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO milestones (id,goal_id,title,criteria,due_date,original_due_date,status,created_at,updated_at,revision) VALUES (?,?,?,?,?,?,?,?,?,?)`, m.ID, m.GoalID, m.Title, m.Criteria, m.DueDate, m.DueDate, m.Status, now, now, m.Revision)
	if err != nil {
		return Milestone{}, fmt.Errorf("insert milestone: %w", err)
	}
	if m.DueDate != "" {
		if _, err := tx.ExecContext(ctx, `UPDATE goals SET target_date=COALESCE(NULLIF(target_date,''),?),updated_at=? WHERE id=? AND EXISTS(SELECT 1 FROM goal_owners o WHERE o.goal_id=goals.id AND o.owner_subject=?)`, m.DueDate, now, m.GoalID, owner); err != nil {
			return Milestone{}, fmt.Errorf("persist goal target date: %w", err)
		}
	}
	if m.LinkedWorkItemID != "" {
		if _, err := tx.ExecContext(ctx, `INSERT INTO milestone_work_links (milestone_id,work_item_id,created_at) VALUES (?,?,?)`, m.ID, m.LinkedWorkItemID, now); err != nil {
			return Milestone{}, fmt.Errorf("link milestone work item: %w", err)
		}
	}
	for _, prerequisiteID := range m.PrerequisiteMilestoneIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO milestone_prerequisites (milestone_id,prerequisite_milestone_id,created_at) VALUES (?,?,?)`, m.ID, prerequisiteID, now); err != nil {
			return Milestone{}, fmt.Errorf("link milestone prerequisite: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return Milestone{}, err
	}
	return m, nil
}

func (r *sqliteRepository) UpdateMilestoneStatus(ctx context.Context, id, status string, revision int64) (Milestone, error) {
	owner, err := goalOwner(ctx)
	if err != nil {
		return Milestone{}, err
	}
	now := r.clock.Now().Unix()
	var goalID string
	if err := r.db.QueryRowContext(ctx, `SELECT m.goal_id FROM milestones m JOIN goal_owners o ON o.goal_id=m.goal_id WHERE m.id=? AND o.owner_subject=?`, id, owner).Scan(&goalID); errors.Is(err, sql.ErrNoRows) {
		return Milestone{}, ErrMilestoneNotFound{id}
	} else if err != nil {
		return Milestone{}, err
	}
	if status == MilestoneDone {
		var blocked bool
		if err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM milestone_prerequisites p JOIN milestones prerequisite ON prerequisite.id=p.prerequisite_milestone_id WHERE p.milestone_id=? AND prerequisite.goal_id=? AND prerequisite.status<>?)`, id, goalID, MilestoneDone).Scan(&blocked); err != nil {
			return Milestone{}, err
		}
		if blocked {
			return Milestone{}, ErrPrerequisitesIncomplete{id}
		}
	}
	completedDate := ""
	if status == MilestoneDone {
		completedDate = r.clock.Now().Format("2006-01-02")
	}
	res, err := r.db.ExecContext(ctx, `UPDATE milestones SET status=?,completed_date=?,updated_at=?,revision=revision+1 WHERE id=? AND revision=? AND EXISTS(SELECT 1 FROM goal_owners o WHERE o.goal_id=milestones.goal_id AND o.owner_subject=?)`, status, completedDate, now, id, revision, owner)
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
	err = r.db.QueryRowContext(ctx, `SELECT m.id,m.goal_id,m.title,m.criteria,m.due_date,m.status,m.revision,COALESCE(l.work_item_id,'') FROM milestones m JOIN goal_owners o ON o.goal_id=m.goal_id AND o.owner_subject=? LEFT JOIN milestone_work_links l ON l.milestone_id=m.id WHERE m.id=?`, owner, id).Scan(&m.ID, &m.GoalID, &m.Title, &m.Criteria, &m.DueDate, &m.Status, &m.Revision, &m.LinkedWorkItemID)
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
	owner, err := goalOwner(ctx)
	if err != nil {
		return false, err
	}
	if len(ids) == 0 {
		var owned bool
		err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM goals g JOIN goal_owners o ON o.goal_id=g.id WHERE g.id=? AND o.owner_subject=?)`, goalID, owner).Scan(&owned)
		return owned, err
	}
	query := `SELECT COUNT(*) FROM milestones m JOIN goal_owners o ON o.goal_id=m.goal_id WHERE m.goal_id=? AND o.owner_subject=? AND m.id IN (` + placeholders(len(ids)) + `)`
	args := make([]any, 0, len(ids)+1)
	args = append(args, goalID)
	args = append(args, owner)
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
	rows, err := r.db.QueryContext(ctx, `SELECT p.prerequisite_milestone_id FROM milestone_prerequisites p JOIN milestones child ON child.id=p.milestone_id JOIN milestones prerequisite ON prerequisite.id=p.prerequisite_milestone_id AND prerequisite.goal_id=child.goal_id WHERE p.milestone_id=? ORDER BY p.prerequisite_milestone_id`, m.ID)
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
