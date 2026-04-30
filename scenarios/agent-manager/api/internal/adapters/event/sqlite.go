// Package event provides event storage and streaming implementations.
//
// This file contains a SQLite implementation of the event Store interface.
package event

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// sqliteTime is a time type that handles SQLite TEXT↔time.Time scanning.
type sqliteTime time.Time

// Common time formats to try when parsing SQLite timestamp strings.
var sqliteTimeFormats = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
}

func (t *sqliteTime) Scan(src interface{}) error {
	if src == nil {
		*t = sqliteTime(time.Time{})
		return nil
	}
	switch v := src.(type) {
	case time.Time:
		*t = sqliteTime(v)
		return nil
	case string:
		for _, layout := range sqliteTimeFormats {
			if parsed, err := time.Parse(layout, v); err == nil {
				*t = sqliteTime(parsed)
				return nil
			}
		}
		return fmt.Errorf("sqliteTime: cannot parse %q", v)
	case []byte:
		return (*sqliteTime).Scan(t, string(v))
	default:
		return fmt.Errorf("sqliteTime: unsupported type %T", src)
	}
}

func (t sqliteTime) Value() (driver.Value, error) {
	tt := time.Time(t)
	if tt.IsZero() {
		return nil, nil
	}
	return tt.UTC().Format(time.RFC3339Nano), nil
}

func (t sqliteTime) Time() time.Time {
	return time.Time(t)
}

// =============================================================================
// SQLite Event Store
// =============================================================================

// SQLiteStore is a SQLite implementation of the event Store interface.
// It persists events to the database and maintains in-memory subscribers for
// real-time streaming.
type SQLiteStore struct {
	db          *sqlx.DB
	log         *logrus.Logger
	mu          sync.RWMutex
	subscribers map[uuid.UUID][]chan *domain.RunEvent
}

// NewSQLiteStore creates a new SQLite event store.
func NewSQLiteStore(db *sqlx.DB, log *logrus.Logger) *SQLiteStore {
	return &SQLiteStore{
		db:          db,
		log:         log,
		subscribers: make(map[uuid.UUID][]chan *domain.RunEvent),
	}
}

// eventRow is the database row representation for run_events.
type eventRow struct {
	ID        uuid.UUID  `db:"id"`
	RunID     uuid.UUID  `db:"run_id"`
	Sequence  int64      `db:"sequence"`
	EventType string     `db:"event_type"`
	Timestamp sqliteTime `db:"timestamp"`
	Data      []byte     `db:"data"`
}

func (e *eventRow) toDomain() *domain.RunEvent {
	evt := &domain.RunEvent{
		ID:        e.ID,
		RunID:     e.RunID,
		Sequence:  e.Sequence,
		EventType: domain.RunEventType(e.EventType),
		Timestamp: e.Timestamp.Time(),
	}

	// Unmarshal based on event type
	switch domain.RunEventType(e.EventType) {
	case domain.EventTypeLog:
		var data domain.LogEventData
		if err := json.Unmarshal(e.Data, &data); err == nil {
			evt.Data = &data
		}
	case domain.EventTypeMessage:
		var data domain.MessageEventData
		if err := json.Unmarshal(e.Data, &data); err == nil {
			evt.Data = &data
		}
	case domain.EventTypeMessageDeleted:
		var data domain.MessageDeletedEventData
		if err := json.Unmarshal(e.Data, &data); err == nil {
			evt.Data = &data
		}
	case domain.EventTypeToolCall:
		var data domain.ToolCallEventData
		if err := json.Unmarshal(e.Data, &data); err == nil {
			evt.Data = &data
		}
	case domain.EventTypeToolResult:
		var data domain.ToolResultEventData
		if err := json.Unmarshal(e.Data, &data); err == nil {
			evt.Data = &data
		}
	case domain.EventTypeStatus:
		var data domain.StatusEventData
		if err := json.Unmarshal(e.Data, &data); err == nil {
			evt.Data = &data
		}
	case domain.EventTypeMetric:
		// Try CostEventData first (common for token usage)
		var costData domain.CostEventData
		if err := json.Unmarshal(e.Data, &costData); err == nil && costData.TotalCostUSD > 0 {
			evt.Data = &costData
		} else {
			// Fall back to generic MetricEventData
			var metricData domain.MetricEventData
			if err := json.Unmarshal(e.Data, &metricData); err == nil {
				evt.Data = &metricData
			}
		}
	case domain.EventTypeArtifact:
		var data domain.ArtifactEventData
		if err := json.Unmarshal(e.Data, &data); err == nil {
			evt.Data = &data
		}
	case domain.EventTypeError:
		var data domain.ErrorEventData
		if err := json.Unmarshal(e.Data, &data); err == nil {
			evt.Data = &data
		}
	case domain.EventTypeLifecycle:
		var data domain.LifecycleEventData
		if err := json.Unmarshal(e.Data, &data); err == nil {
			evt.Data = &data
		}
	default:
		// For unknown types, try legacy format
		var legacy domain.RunEventData
		if err := json.Unmarshal(e.Data, &legacy); err == nil {
			evt.Data = legacy.ToTypedPayload()
		}
	}
	return evt
}

const eventColumns = `id, run_id, sequence, event_type, timestamp, data`

// Append adds events to a run's event stream.
// Events are assigned sequence numbers automatically and persisted to SQLite.
func (s *SQLiteStore) Append(ctx context.Context, runID uuid.UUID, events ...*domain.RunEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Use a transaction for consistency
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return dbError("begin_transaction", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Get the next sequence number
	var maxSeq int64
	query := `SELECT COALESCE(MAX(sequence), -1) FROM run_events WHERE run_id = ?`
	if err := tx.GetContext(ctx, &maxSeq, query, runID); err != nil {
		return dbError("get_max_sequence", err)
	}

	storedEvents := make([]*domain.RunEvent, 0, len(events))

	for _, evt := range events {
		maxSeq++
		evt.RunID = runID
		evt.Sequence = maxSeq
		if evt.ID == uuid.Nil {
			evt.ID = uuid.New()
		}
		if evt.Timestamp.IsZero() {
			evt.Timestamp = time.Now()
		}

		// Marshal event data to JSON
		data, err := s.marshalEventData(evt)
		if err != nil {
			return dbError("marshal_event", err)
		}

		insertQuery := `INSERT INTO run_events (id, run_id, sequence, event_type, timestamp, data)
			VALUES (?, ?, ?, ?, ?, ?)`

		if _, err := tx.ExecContext(ctx, insertQuery,
			evt.ID, evt.RunID, evt.Sequence, string(evt.EventType), sqliteTime(evt.Timestamp), data); err != nil {
			return dbError("insert_event", err)
		}

		// Keep a copy for notification
		copy := *evt
		storedEvents = append(storedEvents, &copy)
	}

	if err := tx.Commit(); err != nil {
		return dbError("commit_transaction", err)
	}

	// Notify subscribers after successful commit
	s.notifySubscribers(runID, storedEvents)

	return nil
}

// marshalEventData converts event data to JSON for storage.
func (s *SQLiteStore) marshalEventData(evt *domain.RunEvent) ([]byte, error) {
	if evt.Data == nil {
		return []byte("{}"), nil
	}

	// Marshal the typed payload directly
	return json.Marshal(evt.Data)
}

// notifySubscribers sends events to all subscribers for a run.
func (s *SQLiteStore) notifySubscribers(runID uuid.UUID, events []*domain.RunEvent) {
	s.mu.RLock()
	subs := s.subscribers[runID]
	s.mu.RUnlock()

	for _, evt := range events {
		for _, ch := range subs {
			select {
			case ch <- evt:
			default:
				// Channel full, skip (non-blocking)
			}
		}
	}
}

// Get retrieves events for a run with optional filtering.
func (s *SQLiteStore) Get(ctx context.Context, runID uuid.UUID, opts GetOptions) ([]*domain.RunEvent, error) {
	// Build the query dynamically based on options
	var conditions []string
	var args []interface{}

	conditions = append(conditions, "run_id = ?")
	args = append(args, runID)

	// AfterSequence filter
	conditions = append(conditions, "sequence > ?")
	args = append(args, opts.AfterSequence)

	// Since timestamp filter
	if opts.Since != nil {
		conditions = append(conditions, "timestamp > ?")
		args = append(args, sqliteTime(*opts.Since))
	}

	// EventTypes filter
	if len(opts.EventTypes) > 0 {
		placeholders := make([]string, len(opts.EventTypes))
		for i, t := range opts.EventTypes {
			placeholders[i] = "?"
			args = append(args, string(t))
		}
		conditions = append(conditions, fmt.Sprintf("event_type IN (%s)", strings.Join(placeholders, ",")))
	}

	query := fmt.Sprintf("SELECT %s FROM run_events WHERE %s ORDER BY sequence ASC",
		eventColumns, strings.Join(conditions, " AND "))

	// Apply limit first (required before OFFSET in standard SQL)
	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}

	// Apply offset (must come after LIMIT)
	if opts.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, opts.Offset)
	}

	var rows []eventRow
	if err := s.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, dbError("get_events", err)
	}

	result := make([]*domain.RunEvent, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

// Stream returns a channel that receives events in real-time.
// The channel is closed when the context is cancelled.
func (s *SQLiteStore) Stream(ctx context.Context, runID uuid.UUID, opts StreamOptions) (<-chan *domain.RunEvent, error) {
	bufSize := opts.BufferSize
	if bufSize <= 0 {
		bufSize = 100
	}

	ch := make(chan *domain.RunEvent, bufSize)

	// Register subscriber
	s.mu.Lock()
	s.subscribers[runID] = append(s.subscribers[runID], ch)
	s.mu.Unlock()

	// Send existing events from the database in a goroutine
	go func() {
		// Build type filter if needed
		var typeFilter []domain.RunEventType
		if len(opts.EventTypes) > 0 {
			typeFilter = opts.EventTypes
		}

		// Query events from the starting sequence
		events, err := s.Get(ctx, runID, GetOptions{
			AfterSequence: opts.FromSequence - 1, // Get events >= FromSequence
			EventTypes:    typeFilter,
		})
		if err != nil {
			s.log.WithError(err).Error("Failed to get historical events for stream")
			return
		}

		for _, evt := range events {
			select {
			case ch <- evt:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Clean up on context cancellation
	go func() {
		<-ctx.Done()
		s.mu.Lock()
		defer s.mu.Unlock()

		// Remove from subscribers
		subs := s.subscribers[runID]
		for i, sub := range subs {
			if sub == ch {
				s.subscribers[runID] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
		close(ch)
	}()

	return ch, nil
}

// Count returns the number of events for a run.
func (s *SQLiteStore) Count(ctx context.Context, runID uuid.UUID) (int64, error) {
	query := `SELECT COUNT(*) FROM run_events WHERE run_id = ?`
	var count int64
	if err := s.db.GetContext(ctx, &count, query, runID); err != nil {
		return 0, dbError("count_events", err)
	}
	return count, nil
}

// Delete removes all events for a run.
func (s *SQLiteStore) Delete(ctx context.Context, runID uuid.UUID) error {
	query := `DELETE FROM run_events WHERE run_id = ?`
	_, err := s.db.ExecContext(ctx, query, runID)
	if err != nil {
		return dbError("delete_events", err)
	}

	// Clean up subscribers
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ch := range s.subscribers[runID] {
		close(ch)
	}
	delete(s.subscribers, runID)

	return nil
}

// Verify interface compliance
var _ Store = (*SQLiteStore)(nil)

func dbError(operation string, err error) error {
	return &domain.DatabaseError{
		Operation:  operation,
		EntityType: "RunEvent",
		Cause:      err,
	}
}
