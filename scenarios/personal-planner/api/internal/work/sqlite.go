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
	_, err := s.db.ExecContext(ctx, `INSERT INTO work_items (id,title,description,remaining_minutes,source_label,created_at,updated_at) VALUES (?,?,?,?,?,?,?)`, item.ID, item.Title, item.Description, item.RemainingMinutes, item.SourceLabel, item.CreatedAt.Format(time.RFC3339Nano), item.UpdatedAt.Format(time.RFC3339Nano))
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
	rows, err := s.db.QueryContext(ctx, `SELECT id,title,description,remaining_minutes,source_label,created_at,updated_at FROM work_items ORDER BY created_at DESC,id DESC LIMIT ?`, limit)
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
