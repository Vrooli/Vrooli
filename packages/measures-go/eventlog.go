package measures

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Event is a single append-only state change. It generalizes swarm-manager's
// eventlog.Event: an auto-incrementing ID (the watermark axis), a timestamp, the
// entity it concerns, what happened, who did it, and a typed metadata blob.
// Measures that adopt the CQRS substrate fold these into a read model
// (readmodel.go) for incremental on-read aggregation.
type Event struct {
	ID         int64           `json:"id"`
	Timestamp  time.Time       `json:"timestamp"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	EventType  string          `json:"event_type"`
	ActorType  string          `json:"actor_type"`
	ActorID    string          `json:"actor_id"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

// EventLog is the append-only event store the measures CQRS substrate reads.
// It is the seam between a measure's read model and durable storage; an adopter
// may back it with SQLEventLog, MemoryEventLog, or its own implementation. The
// contract: IDs are monotonically increasing in append order so Since/MaxID can
// drive a watermark.
type EventLog interface {
	// Append inserts an event and returns its assigned ID.
	Append(ctx context.Context, e Event) (int64, error)
	// Since returns events with ID > afterID, ordered by ID ascending, up to
	// limit. limit <= 0 means no bound.
	Since(ctx context.Context, afterID int64, limit int) ([]Event, error)
	// All returns every event ordered by ID ascending.
	All(ctx context.Context) ([]Event, error)
	// MaxID returns the highest event ID, or 0 when empty.
	MaxID(ctx context.Context) (int64, error)
}

// -----------------------------------------------------------------------------
// MemoryEventLog — an in-memory reference/test implementation.
// -----------------------------------------------------------------------------

// MemoryEventLog is a goroutine-safe in-memory EventLog. It is the reference
// implementation and the test double for read-model code; production adopters
// use SQLEventLog or their own store.
type MemoryEventLog struct {
	mu     sync.RWMutex
	events []Event
	nextID int64
}

// NewMemoryEventLog constructs an empty in-memory event log.
func NewMemoryEventLog() *MemoryEventLog { return &MemoryEventLog{} }

// Append assigns the next ID, defaults timestamp/actor, and stores the event.
func (m *MemoryEventLog) Append(_ context.Context, e Event) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	e.ID = m.nextID
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	if e.ActorType == "" {
		e.ActorType = "user"
	}
	m.events = append(m.events, e)
	return e.ID, nil
}

// Since returns events with ID > afterID ordered by ID, up to limit.
func (m *MemoryEventLog) Since(_ context.Context, afterID int64, limit int) ([]Event, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Event, 0)
	for _, e := range m.events {
		if e.ID > afterID {
			out = append(out, e)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

// All returns every event ordered by ID.
func (m *MemoryEventLog) All(_ context.Context) ([]Event, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Event, len(m.events))
	copy(out, m.events)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// MaxID returns the highest event ID, or 0 when empty.
func (m *MemoryEventLog) MaxID(_ context.Context) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.nextID, nil
}

var _ EventLog = (*MemoryEventLog)(nil)

// -----------------------------------------------------------------------------
// SQLEventLog — a database/sql-backed implementation (driver-agnostic).
// -----------------------------------------------------------------------------

// SQLEventLog is an EventLog backed by a *sql.DB with a configurable table name.
// It is driver-agnostic (no driver import) — the adopter supplies the *sql.DB.
// The schema mirrors swarm-manager's events table so an existing event log can
// be adopted in place.
type SQLEventLog struct {
	db    *sql.DB
	table string
}

// NewSQLEventLog constructs a SQL-backed event log. `table` defaults to
// "events" when empty.
func NewSQLEventLog(db *sql.DB, table string) *SQLEventLog {
	if table == "" {
		table = "events"
	}
	return &SQLEventLog{db: db, table: table}
}

// InitSchema creates the events table and its watermark/index columns if absent.
func (s *SQLEventLog) InitSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp TEXT NOT NULL,
			entity_type TEXT NOT NULL,
			entity_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			actor_type TEXT NOT NULL DEFAULT 'user',
			actor_id TEXT NOT NULL DEFAULT '',
			metadata TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_%s_entity ON %s(entity_type, entity_id);
		CREATE INDEX IF NOT EXISTS idx_%s_type ON %s(event_type);
	`, s.table, s.table, s.table, s.table, s.table))
	if err != nil {
		return fmt.Errorf("measures: init event-log schema: %w", err)
	}
	return nil
}

// Append inserts an event and returns its auto-generated ID.
func (s *SQLEventLog) Append(ctx context.Context, e Event) (int64, error) {
	ts := e.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	actorType := e.ActorType
	if actorType == "" {
		actorType = "user"
	}
	var meta *string
	if len(e.Metadata) > 0 {
		str := string(e.Metadata)
		meta = &str
	}
	res, err := s.db.ExecContext(ctx, fmt.Sprintf(
		`INSERT INTO %s (timestamp, entity_type, entity_id, event_type, actor_type, actor_id, metadata)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`, s.table),
		ts.Format(time.RFC3339Nano), e.EntityType, e.EntityID, e.EventType, actorType, e.ActorID, meta)
	if err != nil {
		return 0, fmt.Errorf("measures: event-log append: %w", err)
	}
	return res.LastInsertId()
}

// Since returns events with ID > afterID ordered by ID ascending, up to limit.
func (s *SQLEventLog) Since(ctx context.Context, afterID int64, limit int) ([]Event, error) {
	q := fmt.Sprintf(`SELECT id, timestamp, entity_type, entity_id, event_type, actor_type, actor_id, metadata
		FROM %s WHERE id > ? ORDER BY id ASC`, s.table)
	args := []any{afterID}
	if limit > 0 {
		q += " LIMIT ?"
		args = append(args, limit)
	}
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("measures: event-log since: %w", err)
	}
	defer rows.Close()
	return scanEvents(rows)
}

// All returns every event ordered by ID ascending.
func (s *SQLEventLog) All(ctx context.Context) ([]Event, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(
		`SELECT id, timestamp, entity_type, entity_id, event_type, actor_type, actor_id, metadata
		 FROM %s ORDER BY id ASC`, s.table))
	if err != nil {
		return nil, fmt.Errorf("measures: event-log all: %w", err)
	}
	defer rows.Close()
	return scanEvents(rows)
}

// MaxID returns the highest event ID, or 0 when empty.
func (s *SQLEventLog) MaxID(ctx context.Context) (int64, error) {
	var maxID int64
	err := s.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT COALESCE(MAX(id), 0) FROM %s`, s.table)).Scan(&maxID)
	if err != nil {
		return 0, fmt.Errorf("measures: event-log maxid: %w", err)
	}
	return maxID, nil
}

func scanEvents(rows *sql.Rows) ([]Event, error) {
	var out []Event
	for rows.Next() {
		var (
			e     Event
			tsStr string
			meta  sql.NullString
		)
		if err := rows.Scan(&e.ID, &tsStr, &e.EntityType, &e.EntityID, &e.EventType, &e.ActorType, &e.ActorID, &meta); err != nil {
			return nil, fmt.Errorf("measures: scan event: %w", err)
		}
		if ts, err := time.Parse(time.RFC3339Nano, tsStr); err == nil {
			e.Timestamp = ts
		}
		if meta.Valid && meta.String != "" {
			e.Metadata = json.RawMessage(meta.String)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

var _ EventLog = (*SQLEventLog)(nil)
