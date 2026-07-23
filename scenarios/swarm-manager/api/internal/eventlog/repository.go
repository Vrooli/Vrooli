package eventlog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vrooli/api-core/database"
)

// Repository defines the append-only event log storage interface.
type Repository interface {
	// Append inserts a new event and returns its auto-generated ID.
	Append(ctx context.Context, e Event) (int64, error)
	// Since returns events with ID > afterID, ordered by ID, up to limit.
	Since(ctx context.Context, afterID int64, limit int) ([]Event, error)
	// All returns every event ordered by ID.
	All(ctx context.Context) ([]Event, error)
	// QueryByEntity returns events for one entity after afterID, ordered by ID.
	// Entity timelines use this instead of scanning unrelated event history.
	QueryByEntity(ctx context.Context, entityType EntityType, entityID string, afterID int64, limit int) ([]Event, error)
	// MaxID returns the highest event ID, or 0 if empty.
	MaxID(ctx context.Context) (int64, error)
}

// SQLiteRepository implements Repository backed by a SQLite database.
//
// It holds a *database.RoutedDB rather than a raw *sql.DB so that, under a
// test-genie in-place e2e run, every query routes to the installed test pool
// instead of the live event log. RoutedDB's method surface mirrors *sql.DB, so
// the query bodies below are unchanged.
type SQLiteRepository struct {
	db *database.RoutedDB
}

// NewSQLiteRepository creates a repository using the given routed database
// handle. Callers outside the lifecycle (tests) wrap a raw *sql.DB with
// database.NewFromPrimary.
func NewSQLiteRepository(db *database.RoutedDB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// schemaSQL is the declarative event-log schema. It is the single source of
// truth applied both at boot (via database.EnsureSchemas) and by InitSchema.
const schemaSQL = `
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
		CREATE INDEX IF NOT EXISTS idx_events_entity ON events(entity_type, entity_id, id);
		CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
	`

// Schema returns the declarative event-log schema for database.EnsureSchemas.
func Schema() string { return schemaSQL }

// InitSchema creates the events table and indexes if they don't exist.
func (r *SQLiteRepository) InitSchema(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, schemaSQL)
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

// QueryByEntity returns a bounded, chronological slice of one entity's
// append-only history. Callers supply a cursor rather than relying on wall
// clock timestamps, which keeps concurrent writes deterministic.
func (r *SQLiteRepository) QueryByEntity(ctx context.Context, entityType EntityType, entityID string, afterID int64, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, timestamp, entity_type, entity_id, event_type, actor_type, actor_id, metadata
		 FROM events WHERE entity_type = ? AND entity_id = ? AND id > ? ORDER BY id ASC LIMIT ?`,
		string(entityType), entityID, afterID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("eventlog query entity: %w", err)
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

// CountEventsInRange returns how many events of eventType have a Timestamp in
// the half-open range [from, to) — from inclusive, to exclusive, matching the
// packages/measures-go time-window resolver. It is the substrate the granular
// measures (api/handlers/measures) compute against.
//
// The event_type filter is pushed to SQL (the idx_events_type index), but the
// time-window comparison is applied in Go on parsed time.Time values rather
// than as a SQL string range: timestamps are stored as RFC3339Nano, whose
// variable-width fractional seconds make lexical comparison unsafe at
// sub-second boundaries. Scanning one event-type's rows is bounded and exact.
func (r *SQLiteRepository) CountEventsInRange(ctx context.Context, eventType EventType, from, to time.Time) (int, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT timestamp FROM events WHERE event_type = ?`,
		string(eventType),
	)
	if err != nil {
		return 0, fmt.Errorf("eventlog count %s: %w", eventType, err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var tsStr string
		if err := rows.Scan(&tsStr); err != nil {
			return 0, fmt.Errorf("eventlog count scan: %w", err)
		}
		t, err := time.Parse(time.RFC3339Nano, tsStr)
		if err != nil {
			return 0, fmt.Errorf("eventlog count parse timestamp %q: %w", tsStr, err)
		}
		if inRange(t, from, to) {
			count++
		}
	}
	return count, rows.Err()
}

// CountStatusTransitionsInRange counts events of eventType for which the typed
// StatusChangePayload's `to` equals toStatus and whose Timestamp is in
// [from, to). It backs status-transition measures such as
// "backlog items completed this week" (backlog.status_changed → to=completed),
// which a bare event-type count cannot express. Malformed metadata is skipped
// (never counted), never an error — the measure abstains rather than over-counts.
func (r *SQLiteRepository) CountStatusTransitionsInRange(ctx context.Context, eventType EventType, toStatus string, from, to time.Time) (int, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT timestamp, metadata FROM events WHERE event_type = ?`,
		string(eventType),
	)
	if err != nil {
		return 0, fmt.Errorf("eventlog count transitions %s: %w", eventType, err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var tsStr string
		var metaStr sql.NullString
		if err := rows.Scan(&tsStr, &metaStr); err != nil {
			return 0, fmt.Errorf("eventlog count transitions scan: %w", err)
		}
		t, err := time.Parse(time.RFC3339Nano, tsStr)
		if err != nil {
			return 0, fmt.Errorf("eventlog count transitions parse timestamp %q: %w", tsStr, err)
		}
		if !inRange(t, from, to) || !metaStr.Valid {
			continue
		}
		var p StatusChangePayload
		if err := json.Unmarshal([]byte(metaStr.String), &p); err != nil {
			continue // malformed payload — abstain, do not over-count
		}
		if p.To == toStatus {
			count++
		}
	}
	return count, rows.Err()
}

// inRange reports whether t is in the half-open range [from, to).
func inRange(t, from, to time.Time) bool {
	return !t.Before(from) && t.Before(to)
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
