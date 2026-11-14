package homeintegration

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db SQLExecutor
}

func NewSQLiteRepository(db SQLExecutor) Repository {
	return &sqliteRepository{db: db}
}

var _ Repository = (*sqliteRepository)(nil)

func (r *sqliteRepository) SaveEvent(ctx context.Context, event Event) (Event, error) {
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	if event.PublishStatus == "" {
		event.PublishStatus = "pending"
	}
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO home_events (id, type, summary, occurred_at, publish_status, publish_error)
VALUES (?, ?, ?, ?, ?, ?)
`, event.ID, event.Type, event.Summary, formatTime(event.OccurredAt), event.PublishStatus, event.PublishError); err != nil {
		return Event{}, fmt.Errorf("save home event %q: %w", event.ID, err)
	}
	return event, nil
}

func (r *sqliteRepository) UpdateEventPublish(ctx context.Context, id, status, publishError string) (Event, error) {
	res, err := r.db.ExecContext(ctx, `
UPDATE home_events
SET publish_status = ?, publish_error = ?
WHERE id = ?
`, status, publishError, id)
	if err != nil {
		return Event{}, fmt.Errorf("update home event publish %q: %w", id, err)
	}
	if rows, err := res.RowsAffected(); err == nil && rows == 0 {
		return Event{}, sql.ErrNoRows
	}
	return r.getEvent(ctx, id)
}

func (r *sqliteRepository) ListEvents(ctx context.Context, limit int) ([]Event, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, type, summary, occurred_at, publish_status, publish_error
FROM home_events
ORDER BY occurred_at DESC
LIMIT ?
`, limit)
	if err != nil {
		return nil, fmt.Errorf("list home events: %w", err)
	}
	defer rows.Close()
	var events []Event
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func (r *sqliteRepository) SaveInvocation(ctx context.Context, invocation Invocation) (Invocation, error) {
	if invocation.CreatedAt.IsZero() {
		invocation.CreatedAt = time.Now().UTC()
	}
	paramsJSON, err := encodeParams(invocation.Params)
	if err != nil {
		return Invocation{}, err
	}
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO home_action_invocations (
  id, action_name, status, approved, message, params_json, event_id, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`, invocation.ID, invocation.ActionName, invocation.Status, boolInt(invocation.Approved), invocation.Message, paramsJSON, invocation.EventID, formatTime(invocation.CreatedAt)); err != nil {
		return Invocation{}, fmt.Errorf("save home action invocation %q: %w", invocation.ID, err)
	}
	return invocation, nil
}

func (r *sqliteRepository) getEvent(ctx context.Context, id string) (Event, error) {
	event, err := scanEvent(r.db.QueryRowContext(ctx, `
SELECT id, type, summary, occurred_at, publish_status, publish_error
FROM home_events
WHERE id = ?
`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Event{}, sql.ErrNoRows
	}
	return event, err
}

type eventScanner interface {
	Scan(dest ...any) error
}

func scanEvent(row eventScanner) (Event, error) {
	var event Event
	var occurredAt string
	if err := row.Scan(&event.ID, &event.Type, &event.Summary, &occurredAt, &event.PublishStatus, &event.PublishError); err != nil {
		return Event{}, err
	}
	parsed, err := time.Parse(TimeFormat, occurredAt)
	if err != nil {
		return Event{}, fmt.Errorf("parse home event occurred_at: %w", err)
	}
	event.OccurredAt = parsed
	return event, nil
}

func encodeParams(params map[string]string) (string, error) {
	if params == nil {
		params = map[string]string{}
	}
	b, err := json.Marshal(params)
	if err != nil {
		return "", fmt.Errorf("encode home action params: %w", err)
	}
	return string(b), nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func formatTime(value time.Time) string {
	return value.UTC().Format(TimeFormat)
}
