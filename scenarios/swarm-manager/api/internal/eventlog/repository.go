package eventlog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Repository defines the append-only event log storage interface.
type Repository interface {
	// Append inserts a new event and returns its auto-generated ID.
	Append(ctx context.Context, e Event) (int64, error)
	// Since returns events with ID > afterID, ordered by ID, up to limit.
	Since(ctx context.Context, afterID int64, limit int) ([]Event, error)
	// All returns every event ordered by ID.
	All(ctx context.Context) ([]Event, error)
	// MaxID returns the highest event ID, or 0 if empty.
	MaxID(ctx context.Context) (int64, error)
}

// SQLiteRepository implements Repository backed by a SQLite database.
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a repository using the given database connection.
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// InitSchema creates the events table and indexes if they don't exist.
func (r *SQLiteRepository) InitSchema(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp TEXT NOT NULL,
			entity_type TEXT NOT NULL,
			entity_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			actor_type TEXT NOT NULL DEFAULT 'user',
			actor_id TEXT NOT NULL DEFAULT '',
			metadata TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_events_entity ON events(entity_type, entity_id);
		CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
	`)
	return err
}

// Append inserts a new event and returns its auto-generated ID.
func (r *SQLiteRepository) Append(ctx context.Context, e Event) (int64, error) {
	ts := e.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	var metaStr *string
	if len(e.Metadata) > 0 {
		s := string(e.Metadata)
		metaStr = &s
	}

	actorType := e.ActorType
	if actorType == "" {
		actorType = "user"
	}

	result, err := r.db.ExecContext(ctx,
		`INSERT INTO events (timestamp, entity_type, entity_id, event_type, actor_type, actor_id, metadata)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		ts.Format(time.RFC3339Nano),
		string(e.EntityType),
		e.EntityID,
		string(e.EventType),
		actorType,
		e.ActorID,
		metaStr,
	)
	if err != nil {
		return 0, fmt.Errorf("eventlog append: %w", err)
	}
	return result.LastInsertId()
}

// Since returns events with ID > afterID, ordered by ID ascending, up to limit.
func (r *SQLiteRepository) Since(ctx context.Context, afterID int64, limit int) ([]Event, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, timestamp, entity_type, entity_id, event_type, actor_type, actor_id, metadata
		 FROM events WHERE id > ? ORDER BY id ASC LIMIT ?`,
		afterID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("eventlog since: %w", err)
	}
	defer rows.Close()
	return scanEvents(rows)
}

// All returns every event ordered by ID ascending.
func (r *SQLiteRepository) All(ctx context.Context) ([]Event, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, timestamp, entity_type, entity_id, event_type, actor_type, actor_id, metadata
		 FROM events ORDER BY id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("eventlog all: %w", err)
	}
	defer rows.Close()
	return scanEvents(rows)
}

// MaxID returns the highest event ID, or 0 if the table is empty.
func (r *SQLiteRepository) MaxID(ctx context.Context) (int64, error) {
	var maxID int64
	err := r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0) FROM events`).Scan(&maxID)
	if err != nil {
		return 0, fmt.Errorf("eventlog maxid: %w", err)
	}
	return maxID, nil
}

func scanEvents(rows *sql.Rows) ([]Event, error) {
	var events []Event
	for rows.Next() {
		var e Event
		var tsStr string
		var metaStr sql.NullString
		err := rows.Scan(
			&e.ID,
			&tsStr,
			&e.EntityType,
			&e.EntityID,
			&e.EventType,
			&e.ActorType,
			&e.ActorID,
			&metaStr,
		)
		if err != nil {
			return nil, fmt.Errorf("eventlog scan: %w", err)
		}
		t, err := time.Parse(time.RFC3339Nano, tsStr)
		if err != nil {
			return nil, fmt.Errorf("eventlog parse timestamp %q: %w", tsStr, err)
		}
		e.Timestamp = t
		if metaStr.Valid {
			e.Metadata = json.RawMessage(metaStr.String)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
