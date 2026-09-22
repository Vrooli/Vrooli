package work

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
		db          SQLExecutor
		clock       schedule.Clock
		idGenerator func() string
	}
)

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return NewSQLiteRepositoryWithOptions(db, clock, RepositoryOptions{})
}

type RepositoryOptions struct {
	IDGenerator func() string
}

func NewSQLiteRepositoryWithOptions(db SQLExecutor, clock schedule.Clock, options RepositoryOptions) Repository {
	if options.IDGenerator == nil {
		options.IDGenerator = uuid.NewString
	}
	return &sqliteRepository{db: db, clock: clock, idGenerator: options.IDGenerator}
}

func (s *sqliteRepository) Create(ctx context.Context, item WorkItem) (WorkItem, error) {
	if item.ID == "" {
		item.ID = s.idGenerator()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = s.clock.Now().UTC()
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = item.CreatedAt
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO work_items (id,title,description,remaining_minutes,original_estimate_minutes,status,source_label,created_at,updated_at) VALUES (?,?,?,?,?,'open',?,?,?)`, item.ID, item.Title, item.Description, item.RemainingMinutes, item.RemainingMinutes, item.SourceLabel, item.CreatedAt.Format(time.RFC3339Nano), item.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return WorkItem{}, fmt.Errorf("insert work item %q: %w", item.ID, err)
	}
	return item, nil
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (WorkItem, error) {
	var item WorkItem
	var created, updated string
	err := s.db.QueryRowContext(ctx, `SELECT id,title,description,remaining_minutes,source_label,created_at,updated_at FROM work_items WHERE id = ?`, id).Scan(&item.ID, &item.Title, &item.Description, &item.RemainingMinutes, &item.SourceLabel, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return WorkItem{}, ErrWorkItemNotFound{id}
	}
	if err != nil {
		return WorkItem{}, fmt.Errorf("get work item %q: %w", id, err)
	}
	item.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return WorkItem{}, err
	}
	item.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	return item, err
}

func (s *sqliteRepository) List(ctx context.Context, limit int) ([]WorkItem, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,title,description,remaining_minutes,source_label,created_at,updated_at FROM work_items WHERE (snoozed_until='' OR snoozed_until<=date('now','localtime')) ORDER BY created_at DESC,id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list work items: %w", err)
	}
	defer rows.Close()
	var result []WorkItem
	for rows.Next() {
		var item WorkItem
		var created, updated string
		if err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.RemainingMinutes, &item.SourceLabel, &created, &updated); err != nil {
			return nil, err
		}
		item.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, err
		}
		item.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *sqliteRepository) Snooze(ctx context.Context, id, until, reason string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE work_items SET snoozed_until=?,updated_at=? WHERE id=? AND status<>'complete'`, until, s.clock.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return fmt.Errorf("snooze work item %q: %w", id, err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrWorkItemNotFound{id}
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO work_item_snoozes(id,work_item_id,until_date,reason,created_at) VALUES (?,?,?,?,?)`, s.idGenerator(), id, until, reason, s.clock.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("record snooze for work item %q: %w", id, err)
	}
	return nil
}

func (s *sqliteRepository) UpdateEstimate(ctx context.Context, id string, minutes int, reason string) error {
	var previous int
	var status string
	if err := s.db.QueryRowContext(ctx, `SELECT remaining_minutes,status FROM work_items WHERE id=?`, id).Scan(&previous, &status); errors.Is(err, sql.ErrNoRows) {
		return ErrWorkItemNotFound{id}
	} else if err != nil {
		return fmt.Errorf("read work item %q for estimate update: %w", id, err)
	} else if status == "complete" {
		return ErrInvalidWorkItem{"id", "completed work items cannot be re-estimated"}
	}
	now := s.clock.Now().UTC().Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `UPDATE work_items SET remaining_minutes=?,updated_at=? WHERE id=? AND status<>'complete'`, minutes, now, id); err != nil {
		return fmt.Errorf("update estimate for work item %q: %w", id, err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO work_item_estimation_changes(id,work_item_id,previous_minutes,new_minutes,reason_code,changed_at) VALUES (?,?,?,?,?,?)`, s.idGenerator(), id, previous, minutes, reason, now); err != nil {
		return fmt.Errorf("record estimate change for %q: %w", id, err)
	}
	return nil
}

func (s *sqliteRepository) Complete(ctx context.Context, id string) error {
	var original, remaining int
	var status string
	if err := s.db.QueryRowContext(ctx, `SELECT original_estimate_minutes,remaining_minutes,status FROM work_items WHERE id=?`, id).Scan(&original, &remaining, &status); errors.Is(err, sql.ErrNoRows) {
		return ErrWorkItemNotFound{id}
	} else if err != nil {
		return fmt.Errorf("read work item %q for completion: %w", id, err)
	} else if status == "complete" {
		return nil
	}
	now := s.clock.Now().UTC()
	completedAt := now.Format(time.RFC3339Nano)
	result, err := s.db.ExecContext(ctx, `UPDATE work_items SET status='complete',remaining_minutes=0,completed_at=?,updated_at=? WHERE id=? AND status<>'complete'`, completedAt, completedAt, id)
	if err != nil {
		return fmt.Errorf("complete work item %q: %w", id, err)
	}
	if count, err := result.RowsAffected(); err != nil {
		return err
	} else if count != 1 {
		return ErrWorkItemNotFound{id}
	}
	actual := original - remaining
	if actual < 0 {
		actual = 0
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO work_item_completion_history(id,work_item_id,original_estimate_minutes,final_actual_minutes,variance_minutes,created_date,completed_date) VALUES (?,?,?,?,?,?,?)`, s.idGenerator(), id, original, actual, actual-original, now.Format("2006-01-02"), now.Format("2006-01-02")); err != nil {
		return fmt.Errorf("record completion history for %q: %w", id, err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE milestones SET status='complete',completed_date=?,updated_at=?,revision=revision+1 WHERE id IN (SELECT l.milestone_id FROM milestone_work_links l WHERE l.work_item_id=?) AND status<>'complete' AND NOT EXISTS (SELECT 1 FROM milestone_prerequisites p JOIN milestones prerequisite ON prerequisite.id=p.prerequisite_milestone_id WHERE p.milestone_id=milestones.id AND prerequisite.status<>'complete')`, now.Format("2006-01-02"), now.Unix(), id); err != nil {
		return fmt.Errorf("advance linked milestones for %q: %w", id, err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE goals SET status='complete',progress_basis_points=10000,completed_date=?,updated_at=?,revision=revision+1 WHERE progress_method='milestones' AND id IN (SELECT DISTINCT goal_id FROM milestones WHERE id IN (SELECT milestone_id FROM milestone_work_links WHERE work_item_id=?)) AND NOT EXISTS (SELECT 1 FROM milestones remaining WHERE remaining.goal_id=goals.id AND remaining.status<>'complete')`, now.Format("2006-01-02"), now.Unix(), id); err != nil {
		return fmt.Errorf("advance linked goals for %q: %w", id, err)
	}
	return nil
}
