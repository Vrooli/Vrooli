// Read-side repository for typed-operational events.
//
// Phases write events through the existing event.Store seam (per-run
// Append → SQLiteStore.Append → broadcaster). Stats, health, and the UI
// read events through this Repository so they get strongly-typed payload
// values instead of dispatching on event_type at every consumer.
//
// The repository is a thin SQL layer; it does NOT cache, watermark, or
// aggregate. Those concerns live in internal/stats/ (Phase 3) and
// internal/health/ (Phase 2).

package eventlog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Record pairs a decoded payload with the row metadata stats and health
// consumers always need (run id, sequence, timestamp, schema version).
//
// Payload is the registered Go type from the dispatch table — consumers
// type-assert to the concrete struct.
type Record struct {
	ID            uuid.UUID
	RunID         uuid.UUID
	Rowid         int64
	Sequence      int64
	EventType     domain.RunEventType
	Timestamp     time.Time
	SchemaVersion int
	Payload       Payload
}

// Repository is the typed read interface over the run_events table.
//
// Implementations are required to filter to typed-operational event types
// only (RunEventType.IsTypedOperationalEvent) so consumers cannot
// accidentally pull legacy LogEventData rows through this seam — that
// would be a category error: the Repository's contract is "typed events,
// decoded".
type Repository interface {
	// Since returns typed-operational events with sequence > afterSeq for
	// the given run, ordered ascending. Used by stream-style consumers.
	SinceForRun(ctx context.Context, runID uuid.UUID, afterSeq int64, limit int) ([]Record, error)

	// SinceID returns typed-operational events with id > afterID across
	// all runs, ordered by id ascending. Used by the stats engine
	// watermark walker (Phase 3) — afterID is the engine's last-processed
	// row id, decoupled from per-run sequence.
	SinceID(ctx context.Context, afterID int64, limit int) ([]Record, error)

	// ByEventType returns typed events of the given category since the
	// supplied timestamp. Used by the future fallback-insights endpoint
	// and by Phase 2's health audit reads. The category filter is
	// strongly typed; unknown categories produce no rows (and the HTTP
	// layer is responsible for 400'ing the input — Phase 3).
	ByEventType(ctx context.Context, eventType domain.RunEventType, since time.Time, limit int) ([]Record, error)
}

// SQLiteRepository implements Repository over the existing run_events
// table. Schema initialization happens in database/connection.go; this
// repository assumes the table and schema_version column already exist.
type SQLiteRepository struct {
	db *sqlx.DB
}

// NewSQLiteRepository constructs a typed-event repository over the given
// SQLite handle.
func NewSQLiteRepository(db *sqlx.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// scanRow holds the raw shape we read out of run_events. Kept private
// so the public surface stays Record-only.
type scanRow struct {
	Rowid         int64     `db:"rowid"`
	ID            uuid.UUID `db:"id"`
	RunID         uuid.UUID `db:"run_id"`
	Sequence      int64     `db:"sequence"`
	EventType     string    `db:"event_type"`
	TimestampStr  string    `db:"timestamp"`
	SchemaVersion int       `db:"schema_version"`
	Data          []byte    `db:"data"`
}

const recordColumns = `rowid, id, run_id, sequence, event_type, timestamp, schema_version, data`

func (r *SQLiteRepository) SinceForRun(ctx context.Context, runID uuid.UUID, afterSeq int64, limit int) ([]Record, error) {
	query := fmt.Sprintf(`SELECT %s FROM run_events
		WHERE run_id = ? AND sequence > ? AND event_type IN (%s)
		ORDER BY sequence ASC`, recordColumns, typedEventInClause())
	args := []any{runID, afterSeq}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	return r.query(ctx, query, args...)
}

func (r *SQLiteRepository) SinceID(ctx context.Context, afterID int64, limit int) ([]Record, error) {
	// run_events.id is a TEXT uuid, not a sequential integer. The stats
	// engine watermark uses (timestamp, run_id, sequence) ordering instead
	// of a single integer id. Phase 3 builds on this; for Phase 1 we
	// expose the simpler "since-rowid" form via SQLite's hidden ROWID so
	// callers have a stable progression cursor that does not require
	// joining on uuid.
	query := fmt.Sprintf(`SELECT %s FROM run_events
		WHERE rowid > ? AND event_type IN (%s)
		ORDER BY rowid ASC`, recordColumns, typedEventInClause())
	args := []any{afterID}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	return r.query(ctx, query, args...)
}

func (r *SQLiteRepository) ByEventType(ctx context.Context, eventType domain.RunEventType, since time.Time, limit int) ([]Record, error) {
	if !eventType.IsTypedOperationalEvent() {
		return nil, fmt.Errorf("eventlog: %s is not a typed-operational event", eventType)
	}
	query := fmt.Sprintf(`SELECT %s FROM run_events
		WHERE event_type = ? AND timestamp >= ?
		ORDER BY timestamp ASC`, recordColumns)
	args := []any{string(eventType), since.UTC().Format(time.RFC3339Nano)}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	return r.query(ctx, query, args...)
}

func (r *SQLiteRepository) query(ctx context.Context, q string, args ...any) ([]Record, error) {
	rows, err := r.db.QueryxContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("eventlog query: %w", err)
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var row scanRow
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("eventlog scan: %w", err)
		}
		ts, err := parseTimestamp(row.TimestampStr)
		if err != nil {
			return nil, fmt.Errorf("eventlog parse timestamp %q: %w", row.TimestampStr, err)
		}
		eventType := domain.RunEventType(row.EventType)
		schemaVersion := row.SchemaVersion
		if schemaVersion == 0 {
			schemaVersion = 1
		}
		payload, err := Decode(eventType, schemaVersion, json.RawMessage(row.Data))
		if err != nil {
			return nil, err
		}
		records = append(records, Record{
			ID:            row.ID,
			RunID:         row.RunID,
			Rowid:         row.Rowid,
			Sequence:      row.Sequence,
			EventType:     eventType,
			Timestamp:     ts,
			SchemaVersion: schemaVersion,
			Payload:       payload,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("eventlog rows: %w", err)
	}
	return records, nil
}

// typedEventInClause expands to a parenthesised list of single-quoted
// event type strings — built once at package init for the IN clauses
// above so we don't have to bind ten args on every typed-event query.
func typedEventInClause() string {
	if cachedTypedInClause != "" {
		return cachedTypedInClause
	}
	parts := make([]string, 0, len(typedEventTypes))
	for _, t := range typedEventTypes {
		parts = append(parts, "'"+string(t)+"'")
	}
	cachedTypedInClause = strings.Join(parts, ", ")
	return cachedTypedInClause
}

var (
	typedEventTypes = []domain.RunEventType{
		domain.EventTypeRunnerFallbackAttempted,
		domain.EventTypeRunnerFallbackExhausted,
		domain.EventTypeModelFallbackAttempted,
		domain.EventTypeModelFallbackExhausted,
		domain.EventTypeModelHealthTransition,
		domain.EventTypeRunnerHealthTransition,
		domain.EventTypeSandboxOperation,
		domain.EventTypeHeartbeatMiss,
		domain.EventTypeCheckpointFailure,
		domain.EventTypeRetryAttempt,
	}
	cachedTypedInClause string
)

// parseTimestamp accepts the RFC3339Nano shape SQLiteStore writes, plus
// the broader fallbacks event.SQLiteStore tolerates so this repository
// stays compatible with rows written by both code paths.
func parseTimestamp(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}
	for _, layout := range formats {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised timestamp")
}

// ensure SQLiteRepository satisfies Repository.
var _ Repository = (*SQLiteRepository)(nil)

// ErrNoSuchEvent is returned when a caller asks for an event id that does
// not exist in run_events. Provided here so consumers can sentinel-check
// without importing database/sql.
var ErrNoSuchEvent = fmt.Errorf("eventlog: no such event")

// rowidFromSQL is exported for the rare test that needs to inspect the
// hidden rowid pinned by SinceID. Production code does not consume this.
func rowidFromSQL(db *sqlx.DB, ctx context.Context, eventID uuid.UUID) (int64, error) {
	var id int64
	err := db.GetContext(ctx, &id, `SELECT rowid FROM run_events WHERE id = ?`, eventID)
	if err == sql.ErrNoRows {
		return 0, ErrNoSuchEvent
	}
	if err != nil {
		return 0, err
	}
	return id, nil
}
