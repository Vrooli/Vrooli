package recovery

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"tunnel-manager/internal/clock"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface the repository depends on.
// Both *sql.DB (repository tests) and *database.RoutedDB (production)
// satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock clock.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

const (
	eventTimeFormat = time.RFC3339Nano

	eventColumns = `id, trigger, action, outcome, details, attempt, created_at`

	insertEventSQL = `
INSERT INTO recovery_events (id, trigger, action, outcome, details, attempt, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
`
)

func (s *sqliteRepository) PersistEvent(ctx context.Context, e RecoveryEvent) (RecoveryEvent, error) {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = s.clock.Now().UTC()
	}
	if e.Action == "" {
		e.Action = ActionRestart
	}
	_, err := s.db.ExecContext(ctx, insertEventSQL,
		e.ID, e.Trigger, e.Action, string(e.Outcome), e.Details, e.Attempt,
		e.CreatedAt.Format(eventTimeFormat),
	)
	if err != nil {
		return RecoveryEvent{}, fmt.Errorf("insert recovery event %q: %w", e.ID, err)
	}
	return e, nil
}

func (s *sqliteRepository) ListEvents(ctx context.Context, limit int) ([]RecoveryEvent, error) {
	if limit <= 0 {
		limit = DefaultEventLimit
	}
	rows, err := s.db.QueryContext(ctx,
		"SELECT "+eventColumns+" FROM recovery_events ORDER BY created_at DESC, id DESC LIMIT ?", limit)
	if err != nil {
		return nil, fmt.Errorf("list recovery events: %w", err)
	}
	defer rows.Close()

	var events []RecoveryEvent
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("scan recovery event: %w", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recovery events: %w", err)
	}
	return events, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEvent(sc rowScanner) (RecoveryEvent, error) {
	var (
		e          RecoveryEvent
		outcomeRaw string
		createdRaw string
	)
	if err := sc.Scan(&e.ID, &e.Trigger, &e.Action, &outcomeRaw, &e.Details, &e.Attempt, &createdRaw); err != nil {
		return RecoveryEvent{}, err
	}
	e.Outcome = EventOutcome(outcomeRaw)
	created, err := time.Parse(eventTimeFormat, createdRaw)
	if err != nil {
		return RecoveryEvent{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	e.CreatedAt = created
	return e, nil
}
