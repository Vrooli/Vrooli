package telemetry

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"program-runtime/internal/sessions"

	"github.com/google/uuid"
	telemetryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/telemetry"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type outboxRow struct {
	eventID     string
	event       *telemetryv1.ProgramEvent
	attempts    int
	lastError   string
	nextAttempt time.Time
	state       string
}

type Repository interface {
	Append(context.Context, *telemetryv1.ProgramEvent, time.Time) error
	List(context.Context, string, telemetryv1.EventKind) ([]*telemetryv1.ProgramEvent, error)
	Pending(context.Context, time.Time, int) ([]outboxRow, error)
	MarkDelivered(context.Context, string) error
	MarkFailed(context.Context, string, int, string, time.Time) error
	MarkDead(context.Context, string, int, string) error
	DeleteDelivered(context.Context) (int64, error)
}

type outboxRepository struct{ db sessions.SQLExecutor }

func NewRepository(db sessions.SQLExecutor) Repository { return &outboxRepository{db: db} }

func (r *outboxRepository) Append(ctx context.Context, event *telemetryv1.ProgramEvent, now time.Time) error {
	payload, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal telemetry event: %w", err)
	}
	if event.GetEventId() == "" {
		return errors.New("telemetry event id is required")
	}
	if _, err := r.db.ExecContext(ctx, `INSERT INTO event_outbox (event_id, session_id, program_id, kind, occurred_at, payload, attempt_count, last_error, next_attempt_at, state) VALUES (?, ?, ?, ?, ?, ?, 0, '', ?, 'pending')`, event.GetEventId(), event.GetSessionId(), event.GetProgramId(), event.GetKind(), event.GetOccurredAt(), string(payload), formatTime(now), formatTime(now)); err != nil {
		return fmt.Errorf("append telemetry event: %w", err)
	}
	return nil
}

func (r *outboxRepository) List(ctx context.Context, sessionID string, kind telemetryv1.EventKind) ([]*telemetryv1.ProgramEvent, error) {
	query := `SELECT payload FROM event_outbox WHERE 1=1`
	args := make([]any, 0, 2)
	if sessionID != "" {
		query += ` AND session_id = ?`
		args = append(args, sessionID)
	}
	if kind != telemetryv1.EventKind_EVENT_KIND_UNSPECIFIED {
		query += ` AND kind = ?`
		args = append(args, kind)
	}
	query += ` ORDER BY occurred_at, event_id`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list telemetry events: %w", err)
	}
	defer rows.Close()
	var out []*telemetryv1.ProgramEvent
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, fmt.Errorf("scan telemetry payload: %w", err)
		}
		event := new(telemetryv1.ProgramEvent)
		if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal([]byte(payload), event); err != nil {
			return nil, fmt.Errorf("decode telemetry payload: %w", err)
		}
		out = append(out, event)
	}
	return out, rows.Err()
}

func (r *outboxRepository) Pending(ctx context.Context, now time.Time, limit int) ([]outboxRow, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `SELECT event_id, payload, attempt_count, last_error, next_attempt_at, state FROM event_outbox WHERE state = 'pending' AND next_attempt_at <= ? ORDER BY occurred_at, event_id LIMIT ?`, formatTime(now), limit)
	if err != nil {
		return nil, fmt.Errorf("list pending telemetry: %w", err)
	}
	defer rows.Close()
	var out []outboxRow
	for rows.Next() {
		var eventID, payload, lastError, nextAttempt, state string
		var attempts int
		if err := rows.Scan(&eventID, &payload, &attempts, &lastError, &nextAttempt, &state); err != nil {
			return nil, fmt.Errorf("scan pending telemetry: %w", err)
		}
		event := new(telemetryv1.ProgramEvent)
		if err := protojson.Unmarshal([]byte(payload), event); err != nil {
			return nil, fmt.Errorf("decode pending telemetry %q: %w", eventID, err)
		}
		parsed, err := time.Parse(time.RFC3339Nano, nextAttempt)
		if err != nil {
			return nil, fmt.Errorf("parse telemetry retry time: %w", err)
		}
		out = append(out, outboxRow{eventID: eventID, event: event, attempts: attempts, lastError: lastError, nextAttempt: parsed, state: state})
	}
	return out, rows.Err()
}

func (r *outboxRepository) MarkDelivered(ctx context.Context, eventID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE event_outbox SET state = 'delivered', last_error = '' WHERE event_id = ?`, eventID)
	return err
}

func (r *outboxRepository) MarkFailed(ctx context.Context, eventID string, attempts int, lastError string, next time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE event_outbox SET attempt_count = ?, last_error = ?, next_attempt_at = ? WHERE event_id = ?`, attempts, lastError, formatTime(next), eventID)
	return err
}

func (r *outboxRepository) MarkDead(ctx context.Context, eventID string, attempts int, lastError string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE event_outbox SET state = 'dead', attempt_count = ?, last_error = ? WHERE event_id = ?`, attempts, lastError, eventID)
	return err
}

func (r *outboxRepository) DeleteDelivered(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM event_outbox WHERE state = 'delivered'`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

type memoryRepository struct {
	mu   sync.RWMutex
	rows map[string]outboxRow
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{rows: make(map[string]outboxRow)}
}

func (r *memoryRepository) Append(_ context.Context, event *telemetryv1.ProgramEvent, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if event.GetEventId() == "" {
		event.EventId = uuid.NewString()
	}
	r.rows[event.GetEventId()] = outboxRow{eventID: event.GetEventId(), event: protoClone(event), nextAttempt: now, state: "pending"}
	return nil
}

func (r *memoryRepository) List(_ context.Context, sessionID string, kind telemetryv1.EventKind) ([]*telemetryv1.ProgramEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*telemetryv1.ProgramEvent
	for _, row := range r.rows {
		if sessionID != "" && row.event.GetSessionId() != sessionID || kind != telemetryv1.EventKind_EVENT_KIND_UNSPECIFIED && row.event.GetKind() != kind {
			continue
		}
		out = append(out, protoClone(row.event))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].GetOccurredAt() < out[j].GetOccurredAt() })
	return out, nil
}

func (r *memoryRepository) Pending(_ context.Context, now time.Time, limit int) ([]outboxRow, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []outboxRow
	for _, row := range r.rows {
		if row.state == "pending" && !row.nextAttempt.After(now) {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].event.GetOccurredAt() < out[j].event.GetOccurredAt() })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *memoryRepository) MarkDelivered(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[id]
	if ok {
		row.state = "delivered"
		r.rows[id] = row
	}
	return nil
}

func (r *memoryRepository) MarkFailed(_ context.Context, id string, attempts int, lastError string, next time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[id]
	if ok {
		row.attempts, row.lastError, row.nextAttempt = attempts, lastError, next
		r.rows[id] = row
	}
	return nil
}

func (r *memoryRepository) MarkDead(_ context.Context, id string, attempts int, lastError string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[id]
	if ok {
		row.attempts, row.lastError, row.state = attempts, lastError, "dead"
		r.rows[id] = row
	}
	return nil
}

func (r *memoryRepository) DeleteDelivered(_ context.Context) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var count int64
	for id, row := range r.rows {
		if row.state == "delivered" {
			delete(r.rows, id)
			count++
		}
	}
	return count, nil
}

type Store struct {
	repo      Repository
	clock     func() time.Time
	publisher Publisher
	drainer   *Drainer
}

func NewStore() *Store { return NewStoreWithOptions(Options{}) }
func NewStoreWithPublisher(publisher Publisher) *Store {
	return NewStoreWithOptions(Options{Publisher: publisher})
}

func NewStoreWithDB(db sessions.SQLExecutor, publisher Publisher) *Store {
	return NewStoreWithOptions(Options{DB: db, Publisher: publisher})
}

type Options struct {
	DB        sessions.SQLExecutor
	Publisher Publisher
	Clock     func() time.Time
}

func NewStoreWithOptions(options Options) *Store {
	clock := options.Clock
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	repo := Repository(newMemoryRepository())
	if options.DB != nil {
		repo = NewRepository(options.DB)
	}
	return &Store{repo: repo, clock: clock, publisher: options.Publisher}
}

func (s *Store) Append(event *telemetryv1.ProgramEvent) {
	_ = s.AppendContext(context.Background(), event)
}

func (s *Store) AppendContext(ctx context.Context, event *telemetryv1.ProgramEvent) error {
	if event == nil {
		return nil
	}
	copy := protoClone(event)
	if copy.GetEventId() == "" {
		copy.EventId = uuid.NewString()
	}
	if copy.GetOccurredAt() == "" {
		copy.OccurredAt = formatTime(s.clock())
	}
	return s.repo.Append(ctx, copy, s.clock().UTC())
}

func (s *Store) List(sessionID string, kind telemetryv1.EventKind) []*telemetryv1.ProgramEvent {
	return s.ListContext(context.Background(), sessionID, kind)
}

func (s *Store) ListContext(ctx context.Context, sessionID string, kind telemetryv1.EventKind) []*telemetryv1.ProgramEvent {
	out, _ := s.repo.List(ctx, sessionID, kind)
	return out
}

func (s *Store) Start(ctx context.Context) {
	if s.publisher != nil {
		s.drainer = NewDrainer(s.repo, s.publisher, s.clock)
		s.drainer.Start(ctx)
	}
}

func (s *Store) Stop() {
	if s.drainer != nil {
		s.drainer.Stop()
	}
}

func (s *Store) DrainOnce(ctx context.Context) error {
	if s.publisher == nil {
		return nil
	}
	return NewDrainer(s.repo, s.publisher, s.clock).DrainOnce(ctx)
}

func protoClone(event *telemetryv1.ProgramEvent) *telemetryv1.ProgramEvent {
	return proto.Clone(event).(*telemetryv1.ProgramEvent)
}
func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

var _ sessions.SQLExecutor = (*sql.DB)(nil)
