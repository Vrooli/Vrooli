package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ============================================================================
// EventRepository Implementation
// ============================================================================

type eventRepository struct {
	db  *DB
	log *logrus.Logger
}

var _ repository.EventRepository = (*eventRepository)(nil)

// eventRow is the database row representation for run_events.
type eventRow struct {
	ID        uuid.UUID  `db:"id"`
	RunID     uuid.UUID  `db:"run_id"`
	Sequence  int64      `db:"sequence"`
	EventType string     `db:"event_type"`
	Timestamp SQLiteTime `db:"timestamp"`
	Data      []byte     `db:"data"`
}

func (e *eventRow) toDomain() (*domain.RunEvent, error) {
	evt := &domain.RunEvent{
		ID:        e.ID,
		RunID:     e.RunID,
		Sequence:  e.Sequence,
		EventType: domain.RunEventType(e.EventType),
		Timestamp: e.Timestamp.Time(),
	}

	payload, err := domain.DecodeEventPayload(evt.EventType, e.Data)
	if err != nil {
		return nil, &domain.DatabaseError{Operation: "decode_event", EntityType: "RunEvent", EntityID: e.ID.String(), Cause: err}
	}
	evt.Data = payload
	return evt, nil
}

const eventColumns = `id, run_id, sequence, event_type, timestamp, data`

func (r *eventRepository) Append(ctx context.Context, runID uuid.UUID, events ...*domain.RunEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Get the next sequence number
	var maxSeq int64
	query := `SELECT COALESCE(MAX(sequence), -1) FROM run_events WHERE run_id = ?`
	if err := r.db.GetContext(ctx, &maxSeq, query, runID); err != nil {
		return wrapDBError("get_max_sequence", "RunEvent", runID.String(), err)
	}

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

		data, err := json.Marshal(evt.Data)
		if err != nil {
			return wrapDBError("marshal_event", "RunEvent", runID.String(), err)
		}

		schemaVersion := evt.SchemaVersion
		if schemaVersion == 0 {
			schemaVersion = 1
		}

		insertQuery := `INSERT INTO run_events (id, run_id, sequence, event_type, timestamp, schema_version, data)
			VALUES (:id, :run_id, :sequence, :event_type, :timestamp, :schema_version, :data)`

		row := struct {
			ID            uuid.UUID  `db:"id"`
			RunID         uuid.UUID  `db:"run_id"`
			Sequence      int64      `db:"sequence"`
			EventType     string     `db:"event_type"`
			Timestamp     SQLiteTime `db:"timestamp"`
			SchemaVersion int        `db:"schema_version"`
			Data          []byte     `db:"data"`
		}{
			ID:            evt.ID,
			RunID:         evt.RunID,
			Sequence:      evt.Sequence,
			EventType:     string(evt.EventType),
			Timestamp:     SQLiteTime(evt.Timestamp),
			SchemaVersion: schemaVersion,
			Data:          data,
		}

		if _, err := r.db.NamedExecContext(ctx, insertQuery, row); err != nil {
			return wrapDBError("insert_event", "RunEvent", runID.String(), err)
		}
	}

	return nil
}

func (r *eventRepository) Get(ctx context.Context, runID uuid.UUID, afterSequence int64, limit int) ([]*domain.RunEvent, error) {
	base := fmt.Sprintf("SELECT %s FROM run_events WHERE run_id = ? AND sequence > ? ORDER BY sequence ASC", eventColumns)
	queryWithLimit := base
	args := []interface{}{runID, afterSequence}
	if limit > 0 {
		queryWithLimit += " LIMIT ?"
		args = append(args, limit)
	}
	query := queryWithLimit

	var rows []eventRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("get_events", "RunEvent", runID.String(), err)
	}

	result := make([]*domain.RunEvent, len(rows))
	for i, row := range rows {
		evt, err := row.toDomain()
		if err != nil {
			return nil, err
		}
		result[i] = evt
	}
	return result, nil
}

func (r *eventRepository) GetByType(ctx context.Context, runID uuid.UUID, types []domain.RunEventType, limit int) ([]*domain.RunEvent, error) {
	if len(types) == 0 {
		return []*domain.RunEvent{}, nil
	}

	typeStrs := make([]interface{}, len(types))
	placeholders := make([]string, len(types))
	for i, t := range types {
		typeStrs[i] = string(t)
		placeholders[i] = "?"
	}

	base := fmt.Sprintf("SELECT %s FROM run_events WHERE run_id = ? AND event_type IN (%s) ORDER BY sequence ASC",
		eventColumns, strings.Join(placeholders, ","))
	args := append([]interface{}{runID}, typeStrs...)

	if limit > 0 {
		base += " LIMIT ?"
		args = append(args, limit)
	}
	query := base

	var rows []eventRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, wrapDBError("get_events_by_type", "RunEvent", runID.String(), err)
	}

	result := make([]*domain.RunEvent, len(rows))
	for i, row := range rows {
		evt, err := row.toDomain()
		if err != nil {
			return nil, err
		}
		result[i] = evt
	}
	return result, nil
}

func (r *eventRepository) Count(ctx context.Context, runID uuid.UUID) (int64, error) {
	query := `SELECT COUNT(*) FROM run_events WHERE run_id = ?`
	var count int64
	if err := r.db.GetContext(ctx, &count, query, runID); err != nil {
		return 0, wrapDBError("count_events", "RunEvent", runID.String(), err)
	}
	return count, nil
}

func (r *eventRepository) Delete(ctx context.Context, runID uuid.UUID) error {
	query := `DELETE FROM run_events WHERE run_id = ?`
	_, err := r.db.ExecContext(ctx, query, runID)
	if err != nil {
		return wrapDBError("delete_events", "RunEvent", runID.String(), err)
	}
	return nil
}
