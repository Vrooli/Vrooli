// Package event provides event storage and streaming implementations.
//
// This file contains a SQLite implementation of the event Store interface.
package event

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/sqlcompat"

	"github.com/google/uuid"
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
	db          sqlcompat.DB
	log         *logrus.Logger
	mu          sync.RWMutex
	subscribers map[uuid.UUID][]*eventSubscriber
	compaction  compactionCursor
}

// NewSQLiteStore creates a new SQLite event store.
func NewSQLiteStore(db sqlcompat.DB, log *logrus.Logger) *SQLiteStore {
	return &SQLiteStore{
		db:          db,
		log:         log,
		subscribers: make(map[uuid.UUID][]*eventSubscriber),
	}
}

type eventSubscriber struct {
	ch          chan *domain.RunEvent
	done        chan struct{}
	historyDone chan struct{}
	once        sync.Once
}

func (s *eventSubscriber) stop() { s.once.Do(func() { close(s.done) }) }

// eventRow is the database row representation for run_events.
type eventRow struct {
	ID            uuid.UUID  `db:"id"`
	RunID         uuid.UUID  `db:"run_id"`
	Sequence      int64      `db:"sequence"`
	EventType     string     `db:"event_type"`
	Timestamp     sqliteTime `db:"timestamp"`
	SchemaVersion int        `db:"schema_version"`
	Data          []byte     `db:"data"`
}

func (e *eventRow) toDomain() (*domain.RunEvent, error) {
	evt := &domain.RunEvent{
		ID:            e.ID,
		RunID:         e.RunID,
		Sequence:      e.Sequence,
		EventType:     domain.RunEventType(e.EventType),
		Timestamp:     e.Timestamp.Time(),
		SchemaVersion: e.SchemaVersion,
	}

	payload, err := domain.DecodeEventPayload(evt.EventType, e.Data)
	if err != nil {
		return nil, &domain.DatabaseError{Operation: "decode_event", EntityType: "RunEvent", EntityID: e.ID.String(), Cause: err}
	}
	evt.Data = payload
	return evt, nil
}

const eventColumns = `id, run_id, sequence, event_type, timestamp, schema_version, data`

// Append adds events to a run's event stream.
// Events are assigned sequence numbers automatically and persisted to SQLite.
func (s *SQLiteStore) Append(ctx context.Context, runID uuid.UUID, events ...*domain.RunEvent) error {
	return s.append(ctx, runID, false, events...)
}

// AppendIfEmpty fails closed if another writer has retained events.
func (s *SQLiteStore) AppendIfEmpty(ctx context.Context, runID uuid.UUID, events ...*domain.RunEvent) error {
	return s.append(ctx, runID, true, events...)
}

type streamQuery interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// streamToken fingerprints the complete canonical stream, not only its last
// sequence. A bound prevents a replacement from consuming unbounded memory or
// work when the old recall layer is unexpectedly large.
func streamToken(ctx context.Context, db streamQuery, runID uuid.UUID) (string, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, sequence, event_type, CAST(timestamp AS TEXT), schema_version, data FROM run_events WHERE run_id = ? ORDER BY sequence`, runID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	h := sha256.New()
	var total int
	for rows.Next() {
		var id, kind, timestamp string
		var sequence, version int64
		var data []byte
		if err := rows.Scan(&id, &sequence, &kind, &timestamp, &version, &data); err != nil {
			return "", err
		}
		total += len(id) + len(kind) + len(timestamp) + len(data) + 32
		if total > 32<<20 {
			return "", errors.New("event stream exceeds replacement token limit")
		}
		fmt.Fprintf(h, "%d:%s:%d:%d:%s:%d:%s:%d:%d:", len(id), id, sequence, len(kind), kind, len(timestamp), timestamp, version, len(data))
		_, _ = h.Write(data)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if err := rows.Close(); err != nil {
		return "", err
	}
	var high int64
	if err := db.QueryRowContext(ctx, `SELECT COALESCE((SELECT high_sequence FROM run_event_highwater WHERE run_id = ?), -1)`, runID).Scan(&high); err != nil {
		return "", err
	}
	fmt.Fprintf(h, "high:%d", high)
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (s *SQLiteStore) StreamToken(ctx context.Context, runID uuid.UUID) (string, error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	return streamToken(ctx, tx, runID)
}

type sqliteReplacementGuard struct {
	store    *SQLiteStore
	finished bool
}

func (s *SQLiteStore) LockReplacement() StreamReplacementGuard {
	s.mu.Lock()
	return &sqliteReplacementGuard{store: s}
}

func (g *sqliteReplacementGuard) ReplaceInTx(ctx context.Context, tx *sql.Tx, runID uuid.UUID, expected string, events []*domain.RunEvent) error {
	if g.finished || tx == nil {
		return errors.New("replacement guard is not active")
	}
	actual, err := streamToken(ctx, tx, runID)
	if err != nil {
		return err
	}
	if actual != expected {
		return errors.New("event stream changed during replacement staging")
	}
	var previousHigh int64
	if err := tx.QueryRowContext(ctx, `SELECT MAX(COALESCE((SELECT MAX(sequence) FROM run_events WHERE run_id = ?), -1), COALESCE((SELECT high_sequence FROM run_event_highwater WHERE run_id = ?), -1))`, runID, runID).Scan(&previousHigh); err != nil {
		return err
	}
	deleted, err := tx.ExecContext(ctx, `DELETE FROM run_events WHERE run_id = ?`, runID)
	if err != nil {
		return err
	}
	count, err := deleted.RowsAffected()
	if err != nil {
		return err
	}
	if count > 0 {
		if _, err := tx.ExecContext(ctx, `UPDATE event_retention_state SET generation=generation+1, floor_rowid=COALESCE((SELECT MIN(rowid) FROM run_events),0), updated_at=? WHERE singleton=1`, sqliteTime(time.Now().UTC())); err != nil {
			return err
		}
	}
	return g.insertInTx(ctx, tx, runID, previousHigh, events)
}

func (g *sqliteReplacementGuard) insertInTx(ctx context.Context, tx *sql.Tx, runID uuid.UUID, previousHigh int64, events []*domain.RunEvent) error {
	for i, evt := range events {
		if evt == nil || evt.ID == uuid.Nil || evt.Timestamp.IsZero() {
			return errors.New("replacement event lacks identity or timestamp")
		}
		data, err := g.store.marshalEventData(evt)
		if err != nil {
			return err
		}
		version := evt.SchemaVersion
		if version == 0 {
			version = 1
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO run_events (id, run_id, sequence, event_type, timestamp, schema_version, data) VALUES (?, ?, ?, ?, ?, ?, ?)`, evt.ID, runID, previousHigh+1+int64(i), string(evt.EventType), sqliteTime(evt.Timestamp), version, data); err != nil {
			return err
		}
		evt.RunID, evt.Sequence, evt.SchemaVersion = runID, previousHigh+1+int64(i), version
	}
	return nil
}

func (g *sqliteReplacementGuard) AppendEmptyInTx(ctx context.Context, tx *sql.Tx, runID uuid.UUID, expected string, events []*domain.RunEvent) error {
	if g.finished || tx == nil {
		return errors.New("replacement guard is not active")
	}
	actual, err := streamToken(ctx, tx, runID)
	if err != nil {
		return err
	}
	if actual != expected {
		return errors.New("event stream changed during recall staging")
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM run_events WHERE run_id = ?`, runID).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return errors.New("recall stream is no longer empty")
	}
	var high int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE((SELECT high_sequence FROM run_event_highwater WHERE run_id = ?), -1)`, runID).Scan(&high); err != nil {
		return err
	}
	return g.insertInTx(ctx, tx, runID, high, events)
}

func (g *sqliteReplacementGuard) FinishAppend(runID uuid.UUID, committed bool, events []*domain.RunEvent) {
	if g.finished {
		return
	}
	if committed {
		for _, item := range events {
			for _, sub := range g.store.subscribers[runID] {
				deliverSubscriber(sub, item)
			}
		}
	}
	g.finished = true
	g.store.mu.Unlock()
}

func (g *sqliteReplacementGuard) Finish(runID uuid.UUID, committed bool) {
	if g.finished {
		return
	}
	if committed {
		for _, sub := range g.store.subscribers[runID] {
			sub.stop()
		}
		delete(g.store.subscribers, runID)
	}
	g.finished = true
	g.store.mu.Unlock()
}

func (s *SQLiteStore) append(ctx context.Context, runID uuid.UUID, requireEmpty bool, events ...*domain.RunEvent) error {
	if len(events) == 0 && !requireEmpty {
		return nil
	}

	conn, err := s.db.Conn(ctx)
	if err != nil {
		return dbError("get_connection", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return dbError("begin_immediate_transaction", err)
	}

	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()

	// Get the next sequence number
	var maxSeq int64
	query := `SELECT MAX(COALESCE((SELECT MAX(sequence) FROM run_events WHERE run_id = ?), -1), COALESCE((SELECT high_sequence FROM run_event_highwater WHERE run_id = ?), -1))`
	if err := conn.QueryRowContext(ctx, query, runID, runID).Scan(&maxSeq); err != nil {
		return dbError("get_max_sequence", err)
	}
	if requireEmpty {
		var present int
		if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM run_events WHERE run_id = ?`, runID).Scan(&present); err != nil {
			return dbError("count_existing_events", err)
		}
		if present != 0 {
			return fmt.Errorf("cannot rebuild recall: event stream is no longer empty")
		}
	}

	storedEvents := make([]*domain.RunEvent, 0, len(events))

	for _, evt := range events {
		evt.RunID = runID
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

		schemaVersion := evt.SchemaVersion
		if schemaVersion == 0 {
			schemaVersion = 1
		}
		evt.SchemaVersion = schemaVersion
		evt.PersistenceReplay = false

		var storedRun, storedType string
		var storedSeq int64
		var storedSchema int
		var storedData []byte
		var storedTime sqliteTime
		existingErr := conn.QueryRowContext(ctx, `SELECT run_id,sequence,event_type,schema_version,data,timestamp FROM run_events WHERE id=?`, evt.ID).Scan(&storedRun, &storedSeq, &storedType, &storedSchema, &storedData, &storedTime)
		if existingErr == nil {
			if storedRun != runID.String() || (!evt.KeepStoredOnReplay && (storedType != string(evt.EventType) || storedSchema != schemaVersion || !bytes.Equal(storedData, data))) {
				return &domain.DatabaseError{Operation: "replay_event_conflict", EntityType: "RunEvent", EntityID: evt.ID.String(), Cause: fmt.Errorf("event identity reused with different immutable evidence")}
			}
			evt.PersistenceReplay = true
			evt.Sequence = storedSeq
			evt.Timestamp = storedTime.Time()
			continue // committed replay does not notify subscribers again
		}
		if !errors.Is(existingErr, sql.ErrNoRows) {
			return dbError("read_event_replay", existingErr)
		}
		if evt.ID.Version() == 5 {
			// A deterministic transcript event re-read over a row stored before
			// ids were deterministic is the same source line: the run already
			// holds it. The unary + keeps the planner on the timestamp index (a few rows) instead of scanning the run.
			var twinSeq int64
			twinErr := conn.QueryRowContext(ctx, `SELECT sequence FROM run_events WHERE timestamp = ? AND +run_id = ? AND +event_type = ? AND data = ? AND substr(id, 15, 1) <> '5' LIMIT 1`,
				sqliteTime(evt.Timestamp), runID, string(evt.EventType), data).Scan(&twinSeq)
			if twinErr == nil {
				evt.PersistenceReplay = true
				evt.Sequence = twinSeq
				continue
			}
			if !errors.Is(twinErr, sql.ErrNoRows) {
				return dbError("read_legacy_twin", twinErr)
			}
		}
		maxSeq++
		evt.Sequence = maxSeq
		insertQuery := `INSERT INTO run_events (id, run_id, sequence, event_type, timestamp, schema_version, data)
			VALUES (?, ?, ?, ?, ?, ?, ?)`

		if _, err := conn.ExecContext(ctx, insertQuery,
			evt.ID, evt.RunID, evt.Sequence, string(evt.EventType), sqliteTime(evt.Timestamp), schemaVersion, data); err != nil {
			return dbError("insert_event", err)
		}

		// Keep a copy for notification
		copy := *evt
		storedEvents = append(storedEvents, &copy)
	}

	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return dbError("commit_transaction", err)
	}
	committed = true

	// Notify subscribers after successful commit
	s.notifySubscribers(runID, storedEvents)

	return nil
}

// DeleteBefore removes at most limit rows older than cutoff. Select candidates
// before opening a write transaction: LIMIT bounds rows returned, not the cost
// of scanning history, and SQLite reserves its writer before a DELETE's scan.
// Recheck eligibility by stable event identity in the short write transaction.
//
// An event whose run row no longer exists is deleted once it is past the
// cutoff even without a completed read-model projection: nothing can project a
// deleted run, so waiting for one kept such events forever (213 of 11,747
// orphans measured 2026-09-14). Imported runs are never age-deleted.
func (s *SQLiteStore) DeleteBefore(ctx context.Context, cutoff time.Time, limit int) (int, error) {
	if limit <= 0 {
		return 0, fmt.Errorf("event retention batch limit must be positive")
	}
	var ids []string
	if err := s.db.SelectContext(ctx, &ids, `SELECT events.id FROM run_events events
		LEFT JOIN invocation_read_model_watermarks watermark ON watermark.run_id = events.run_id
		LEFT JOIN runs run ON run.id = events.run_id
		WHERE events.timestamp < ?
		  AND (run.id IS NULL OR (watermark.projection_complete = 1 AND COALESCE(run.execution_mode, '') <> 'imported'))
		ORDER BY events.timestamp ASC LIMIT ?`, sqliteTime(cutoff), limit); err != nil {
		return 0, dbError("select_expired_events", err)
	}
	if len(ids) == 0 {
		return 0, nil
	}
	args := make([]any, 0, len(ids)+1)
	for _, id := range ids {
		args = append(args, id)
	}
	args = append(args, sqliteTime(cutoff))
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return 0, dbError("event_retention_connection", err)
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, dbError("begin_event_retention", err)
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `DELETE FROM run_events WHERE id IN (`+placeholders+`)
		AND timestamp < ?
		AND (NOT EXISTS (SELECT 1 FROM runs run WHERE run.id = run_events.run_id)
		  OR (EXISTS (SELECT 1 FROM invocation_read_model_watermarks watermark
		        WHERE watermark.run_id = run_events.run_id AND watermark.projection_complete = 1)
		      AND NOT EXISTS (SELECT 1 FROM runs run
		        WHERE run.id = run_events.run_id AND run.execution_mode = 'imported')))`, args...)
	if err != nil {
		return 0, dbError("delete_expired_events", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, dbError("count_deleted_expired_events", err)
	}
	if count > 0 {
		if _, err := tx.ExecContext(ctx, `UPDATE event_retention_state SET generation=generation+1, floor_rowid=COALESCE((SELECT MIN(rowid) FROM run_events),0), updated_at=? WHERE singleton=1`, sqliteTime(time.Now().UTC())); err != nil {
			return 0, dbError("advance_event_retention_generation", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, dbError("commit_event_retention", err)
	}
	return int(count), nil
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
	defer s.mu.RUnlock()
	subs := s.subscribers[runID]

	for _, evt := range events {
		for _, sub := range subs {
			deliverSubscriber(sub, evt)
		}
	}
}

// A full live buffer invalidates the stream: close it so the subscriber can
// resume from its last cursor rather than silently missing an event.
func deliverSubscriber(sub *eventSubscriber, event *domain.RunEvent) {
	select {
	case <-sub.done:
		return
	default:
	}
	select {
	case sub.ch <- event:
	default:
		sub.stop()
	}
}

// Get retrieves events for a run with optional filtering.
func (s *SQLiteStore) Get(ctx context.Context, runID uuid.UUID, opts GetOptions) ([]*domain.RunEvent, error) {
	where, args := eventSelection(runID, opts)

	query := fmt.Sprintf("SELECT %s FROM run_events WHERE %s ORDER BY sequence ASC", eventColumns, where)

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
		evt, err := row.toDomain()
		if err != nil {
			return nil, err
		}
		result[i] = evt
	}
	return result, nil
}

type eventSummaryRow struct {
	EventType     string `db:"event_type"`
	Count         int64  `db:"event_count"`
	Latest        int64  `db:"latest_sequence"`
	SelectedCount int64  `db:"selected_count"`
}

// Summarize groups the same ordered/filterable selection as Get, but reads only
// sequence and event_type. The limit+1 row is used for HasMore and excluded
// from the returned counts and latest sequence.
func (s *SQLiteStore) Summarize(ctx context.Context, runID uuid.UUID, opts GetOptions) (*Summary, error) {
	where, args := eventSelection(runID, opts)
	selectedQuery := fmt.Sprintf("SELECT sequence,event_type FROM run_events WHERE %s ORDER BY sequence ASC", where)
	selectedArgs := append([]interface{}(nil), args...)
	if opts.Limit > 0 {
		selectedQuery += " LIMIT ?"
		selectedArgs = append(selectedArgs, opts.Limit+1)
	} else if opts.Offset > 0 {
		selectedQuery += " LIMIT -1"
	}
	if opts.Offset > 0 {
		selectedQuery += " OFFSET ?"
		selectedArgs = append(selectedArgs, opts.Offset)
	}

	pageQuery := "SELECT sequence,event_type FROM selected ORDER BY sequence ASC"
	if opts.Limit > 0 {
		pageQuery += " LIMIT ?"
		selectedArgs = append(selectedArgs, opts.Limit)
	}
	query := fmt.Sprintf(`WITH selected AS (%s), page AS (%s), grouped AS (
		SELECT event_type,COUNT(*) AS event_count,MAX(sequence) AS latest_sequence FROM page GROUP BY event_type
	) SELECT event_type,event_count,latest_sequence,(SELECT COUNT(*) FROM selected) AS selected_count FROM grouped`, selectedQuery, pageQuery)
	var rows []eventSummaryRow
	if err := s.db.SelectContext(ctx, &rows, query, selectedArgs...); err != nil {
		return nil, dbError("summarize_events", err)
	}
	summary := &Summary{Counts: make(map[string]int64)}
	for _, row := range rows {
		summary.Counts[row.EventType] = row.Count
		latest := row.Latest
		if summary.LatestSequence == nil || latest > *summary.LatestSequence {
			summary.LatestSequence = &latest
		}
		if opts.Limit > 0 && row.SelectedCount > int64(opts.Limit) {
			summary.HasMore = true
		}
	}
	return summary, nil
}

func eventSelection(runID uuid.UUID, opts GetOptions) (string, []interface{}) {
	conditions := []string{"run_id = ?", "sequence > ?"}
	args := []interface{}{runID, opts.AfterSequence}
	if opts.Since != nil {
		conditions = append(conditions, "timestamp > ?")
		args = append(args, sqliteTime(*opts.Since))
	}
	if len(opts.EventTypes) > 0 {
		placeholders := make([]string, len(opts.EventTypes))
		for i, eventType := range opts.EventTypes {
			placeholders[i] = "?"
			args = append(args, string(eventType))
		}
		conditions = append(conditions, fmt.Sprintf("event_type IN (%s)", strings.Join(placeholders, ",")))
	}
	return strings.Join(conditions, " AND "), args
}

// Stream returns a channel that receives events in real-time.
// The channel is closed when the context is cancelled.
func (s *SQLiteStore) Stream(ctx context.Context, runID uuid.UUID, opts StreamOptions) (<-chan *domain.RunEvent, error) {
	bufSize := opts.BufferSize
	if bufSize <= 0 {
		bufSize = 100
	}

	sub := &eventSubscriber{ch: make(chan *domain.RunEvent, bufSize), done: make(chan struct{}), historyDone: make(chan struct{})}

	// Register subscriber
	s.mu.Lock()
	s.subscribers[runID] = append(s.subscribers[runID], sub)
	s.mu.Unlock()

	// Send existing events from the database in a goroutine
	go func() {
		defer close(sub.historyDone)
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
			case sub.ch <- evt:
			case <-sub.done:
				return
			}
		}
	}()

	// Clean up on context cancellation
	go func() {
		select {
		case <-ctx.Done():
		case <-sub.done:
		}
		s.mu.Lock()

		// Remove from subscribers
		subs := s.subscribers[runID]
		for i, registered := range subs {
			if registered == sub {
				s.subscribers[runID] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
		sub.stop()
		s.mu.Unlock()
		<-sub.historyDone
		close(sub.ch)
	}()

	return sub.ch, nil
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
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return dbError("delete_events_connection", err)
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return dbError("begin_delete_events", err)
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `DELETE FROM run_events WHERE run_id = ?`, runID)
	if err != nil {
		return dbError("delete_events", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return dbError("count_deleted_events", err)
	}
	if count > 0 {
		if _, err := tx.ExecContext(ctx, `UPDATE event_retention_state SET generation=generation+1, floor_rowid=COALESCE((SELECT MIN(rowid) FROM run_events),0), updated_at=? WHERE singleton=1`, sqliteTime(time.Now().UTC())); err != nil {
			return dbError("advance_event_retention_generation", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return dbError("commit_delete_events", err)
	}

	// Clean up subscribers
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sub := range s.subscribers[runID] {
		sub.stop()
	}
	delete(s.subscribers, runID)

	return nil
}

// LiveBytes reports the database's live (non-freelist) byte size using cheap
// pragmas — the same measure storage-health budgets against. Retention uses it
// to decide when imported-corpus eviction has brought the store back under
// budget, without scanning dbstat across a multi-GB file.
func (s *SQLiteStore) LiveBytes(ctx context.Context) (int64, error) {
	var pageCount, freelistCount, pageSize int64
	if err := s.db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pageCount); err != nil {
		return 0, dbError("pragma_page_count", err)
	}
	if err := s.db.QueryRowContext(ctx, "PRAGMA freelist_count").Scan(&freelistCount); err != nil {
		return 0, dbError("pragma_freelist_count", err)
	}
	if err := s.db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize); err != nil {
		return 0, dbError("pragma_page_size", err)
	}
	return (pageCount - freelistCount) * pageSize, nil
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
