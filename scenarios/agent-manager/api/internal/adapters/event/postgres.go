// Package event provides event storage and streaming implementations.
//
// This file contains a PostgreSQL implementation of the event Store interface.
package event

import (
	"context"
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

// =============================================================================
// PostgreSQL Event Store
// =============================================================================

// PostgresStore is a PostgreSQL implementation of the event Store interface.
// It persists events to the database and maintains in-memory subscribers for
// real-time streaming.
type PostgresStore struct {
	db          *sqlx.DB
	log         *logrus.Logger
	mu          sync.RWMutex
	subscribers map[uuid.UUID][]chan *domain.RunEvent
}

// NewPostgresStore creates a new PostgreSQL event store.
func NewPostgresStore(db *sqlx.DB, log *logrus.Logger) *PostgresStore {
	return &PostgresStore{
		db:          db,
		log:         log,
		subscribers: make(map[uuid.UUID][]chan *domain.RunEvent),
	}
}

// eventRow is the database row representation for run_events.
type eventRow struct {
	ID        uuid.UUID `db:"id"`
	RunID     uuid.UUID `db:"run_id"`
	Sequence  int64     `db:"sequence"`
	EventType string    `db:"event_type"`
	Timestamp time.Time `db:"timestamp"`
	Data      []byte    `db:"data"`
}

func (e *eventRow) toDomain() *domain.RunEvent {
	evt := &domain.RunEvent{
		ID:        e.ID,
		RunID:     e.RunID,
		Sequence:  e.Sequence,
		EventType: domain.RunEventType(e.EventType),
		Timestamp: e.Timestamp,
	}

	// Parse data as RunEventData first (for backward compatibility)
	var legacy domain.RunEventData
	if err := json.Unmarshal(e.Data, &legacy); err == nil {
		evt.Data = legacy.ToTypedPayload()
	}
	return evt
}

const eventColumns = `id, run_id, sequence, event_type, timestamp, data`

// Append adds events to a run's event stream.
// Events are assigned sequence numbers automatically and persisted to PostgreSQL.
func (s *PostgresStore) Append(ctx context.Context, runID uuid.UUID, events ...*domain.RunEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Use a transaction for consistency
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get the next sequence number
	var maxSeq int64
	query := `SELECT COALESCE(MAX(sequence), -1) FROM run_events WHERE run_id = $1`
	if err := tx.GetContext(ctx, &maxSeq, query, runID); err != nil {
		return fmt.Errorf("failed to get max sequence: %w", err)
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
			return fmt.Errorf("failed to marshal event data: %w", err)
		}

		insertQuery := `INSERT INTO run_events (id, run_id, sequence, event_type, timestamp, data)
			VALUES ($1, $2, $3, $4, $5, $6)`

		if _, err := tx.ExecContext(ctx, insertQuery,
			evt.ID, evt.RunID, evt.Sequence, string(evt.EventType), evt.Timestamp, data); err != nil {
			return fmt.Errorf("failed to insert event: %w", err)
		}

		// Keep a copy for notification
		copy := *evt
		storedEvents = append(storedEvents, &copy)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Notify subscribers after successful commit
	s.notifySubscribers(runID, storedEvents)

	return nil
}

// marshalEventData converts event data to JSON for storage.
func (s *PostgresStore) marshalEventData(evt *domain.RunEvent) ([]byte, error) {
	if evt.Data == nil {
		return []byte("{}"), nil
	}

	// Marshal the typed payload directly
	return json.Marshal(evt.Data)
}

// notifySubscribers sends events to all subscribers for a run.
func (s *PostgresStore) notifySubscribers(runID uuid.UUID, events []*domain.RunEvent) {
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
func (s *PostgresStore) Get(ctx context.Context, runID uuid.UUID, opts GetOptions) ([]*domain.RunEvent, error) {
	// Build the query dynamically based on options
	var conditions []string
	var args []interface{}
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("run_id = $%d", argNum))
	args = append(args, runID)
	argNum++

	// AfterSequence filter
	conditions = append(conditions, fmt.Sprintf("sequence > $%d", argNum))
	args = append(args, opts.AfterSequence)
	argNum++

	// Since timestamp filter
	if opts.Since != nil {
		conditions = append(conditions, fmt.Sprintf("timestamp > $%d", argNum))
		args = append(args, *opts.Since)
		argNum++
	}

	// EventTypes filter
	if len(opts.EventTypes) > 0 {
		placeholders := make([]string, len(opts.EventTypes))
		for i, t := range opts.EventTypes {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, string(t))
			argNum++
		}
		conditions = append(conditions, fmt.Sprintf("event_type IN (%s)", strings.Join(placeholders, ",")))
	}

	query := fmt.Sprintf("SELECT %s FROM run_events WHERE %s ORDER BY sequence ASC",
		eventColumns, strings.Join(conditions, " AND "))

	// Apply offset
	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argNum)
		args = append(args, opts.Offset)
		argNum++
	}

	// Apply limit
	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, opts.Limit)
	}

	var rows []eventRow
	if err := s.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	result := make([]*domain.RunEvent, len(rows))
	for i, row := range rows {
		result[i] = row.toDomain()
	}
	return result, nil
}

// Stream returns a channel that receives events in real-time.
// The channel is closed when the context is cancelled.
func (s *PostgresStore) Stream(ctx context.Context, runID uuid.UUID, opts StreamOptions) (<-chan *domain.RunEvent, error) {
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
func (s *PostgresStore) Count(ctx context.Context, runID uuid.UUID) (int64, error) {
	query := `SELECT COUNT(*) FROM run_events WHERE run_id = $1`
	var count int64
	if err := s.db.GetContext(ctx, &count, query, runID); err != nil {
		return 0, fmt.Errorf("failed to count events: %w", err)
	}
	return count, nil
}

// Delete removes all events for a run.
func (s *PostgresStore) Delete(ctx context.Context, runID uuid.UUID) error {
	query := `DELETE FROM run_events WHERE run_id = $1`
	_, err := s.db.ExecContext(ctx, query, runID)
	if err != nil {
		return fmt.Errorf("failed to delete events: %w", err)
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
var _ Store = (*PostgresStore)(nil)
