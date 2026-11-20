// Package activity owns the append-only mutation log. Actor attribution is
// durable here; the workbench must not reconstruct "who changed what" from
// unrelated catalog timestamps.
//
// The package is deliberately small: Record is the only write seam, and List
// plus Summary are read projections. A mutating service writes an event in the
// same transaction as its change where practical, or through an outbox that
// calls Record. Nothing in this package mutates or deletes a recorded event.
package activity

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// timestampLayout is fixed-width UTC so lexical ordering matches time order.
// RFC3339Nano trims trailing zeros, which makes "12:00:00Z" sort after
// "12:00:00.1Z"; the fixed nanosecond form keeps SQLite's text comparison
// honest.
const timestampLayout = "2006-01-02T15:04:05.000000000Z"

const (
	defaultListLimit    = 100
	maxListLimit        = 500
	defaultSummaryDays  = 30
	maxSummaryDays      = 365
	idRandomBytes       = 8
	activityIDTimestamp = 16
)

// ErrInvalid marks a caller-supplied value this domain cannot accept. Handlers
// map it to Connect's CodeInvalidArgument rather than a storage failure.
var ErrInvalid = errors.New("invalid activity event")

// Event is one durable mutation record. BeforeJSON and AfterJSON are opaque
// snapshots owned by the mutating service.
type Event struct {
	ID          string
	Actor       string
	Action      string
	SubjectKind string
	SubjectID   string
	Summary     string
	BeforeJSON  string
	AfterJSON   string
	At          string
	TraceID     string
}

// Filter narrows List. Since is an inclusive RFC3339 lower bound.
type Filter struct {
	SubjectKind string
	SubjectID   string
	Actor       string
	Since       string
}

// DayCount is one day's measured event total.
type DayCount struct {
	Date  string
	Count int
}

// ActorCount is one actor's measured event total.
type ActorCount struct {
	Actor string
	Count int
}

// ActionCount is one action's measured event total.
type ActionCount struct {
	Action string
	Count  int
}

// Summary is the bounded activity view used by Stats. Empty means the queried
// window contains zero events; that is a measured zero, not an unavailable
// source.
type Summary struct {
	Total    int
	ByDay    []DayCount
	ByActor  []ActorCount
	ByAction []ActionCount
	Empty    bool
}

// Recorder is the append seam other services depend on. Keeping it an
// interface lets an outbox or test double stand in without a second write path
// to activity_events.
type Recorder interface {
	Record(ctx context.Context, event Event) error
}

// Store is the SQLite-backed activity owner.
type Store struct {
	db     *sql.DB
	clock  func() time.Time
	random io.Reader
}

var _ Recorder = (*Store)(nil)

// NewSQLiteStore returns the durable store over the scenario's primary pool.
func NewSQLiteStore(db *sql.DB) *Store {
	return &Store{db: db, clock: time.Now, random: rand.Reader}
}

// Record appends one event. The caller may supply an id for outbox
// idempotency; otherwise a deterministic time+random hex id is generated.
func (s *Store) Record(ctx context.Context, event Event) error {
	if s == nil || s.db == nil {
		return errors.New("activity store is not configured")
	}
	now := s.now()
	clean, at, err := normalizeEvent(event, now)
	if err != nil {
		return err
	}
	providedID := clean.ID != ""
	if !providedID {
		clean.ID = s.newID(at)
	}
	// A caller-supplied id is the outbox retry key: replaying the same event is
	// a no-op. Generated ids are left strict so a random collision is visible
	// rather than silently dropping an event.
	insert := "INSERT"
	if providedID {
		insert = "INSERT OR IGNORE"
	}
	_, err = s.db.ExecContext(ctx, insert+` INTO activity_events(id, actor, action, subject_kind, subject_id, summary, before_json, after_json, at, trace_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		clean.ID, clean.Actor, clean.Action, clean.SubjectKind, clean.SubjectID,
		clean.Summary, clean.BeforeJSON, clean.AfterJSON, clean.At, clean.TraceID)
	if err != nil {
		return fmt.Errorf("record activity event: %w", err)
	}
	return nil
}

// List returns events newest-first. Limit is bounded so a caller cannot turn
// the Stats surface into an unbounded table scan.
func (s *Store) List(ctx context.Context, filter Filter, limit int) ([]Event, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("activity store is not configured")
	}
	limit = clampLimit(limit)
	query := `SELECT id, actor, action, subject_kind, subject_id, summary, before_json, after_json, at, trace_id
FROM activity_events`
	conditions := make([]string, 0, 4)
	args := make([]any, 0, 4)
	if kind := strings.TrimSpace(filter.SubjectKind); kind != "" {
		conditions = append(conditions, "subject_kind = ?")
		args = append(args, kind)
	}
	if id := strings.TrimSpace(filter.SubjectID); id != "" {
		conditions = append(conditions, "subject_id = ?")
		args = append(args, id)
	}
	if actor := strings.TrimSpace(filter.Actor); actor != "" {
		conditions = append(conditions, "actor = ?")
		args = append(args, actor)
	}
	if since := strings.TrimSpace(filter.Since); since != "" {
		normalized, err := normalizeTimestamp(since)
		if err != nil {
			return nil, fmt.Errorf("%w: since must be RFC3339", ErrInvalid)
		}
		conditions = append(conditions, "at >= ?")
		args = append(args, normalized)
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY at DESC, id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list activity events: %w", err)
	}
	defer rows.Close()

	events := make([]Event, 0)
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.ID, &event.Actor, &event.Action, &event.SubjectKind,
			&event.SubjectID, &event.Summary, &event.BeforeJSON, &event.AfterJSON,
			&event.At, &event.TraceID); err != nil {
			return nil, fmt.Errorf("scan activity event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate activity events: %w", err)
	}
	return events, nil
}

// Summary returns measured totals for the last days (bounded 1..365, with 0
// meaning the server default). Group rows are ordered for stable rendering:
// days ascending, actors/actions by count descending then name ascending.
func (s *Store) Summary(ctx context.Context, days int) (Summary, error) {
	if s == nil || s.db == nil {
		return Summary{}, errors.New("activity store is not configured")
	}
	days = clampDays(days)
	cutoff := s.now().UTC().Add(-time.Duration(days) * 24 * time.Hour).Format(timestampLayout)

	summary := Summary{}
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM activity_events WHERE at >= ?`, cutoff).Scan(&summary.Total); err != nil {
		return Summary{}, fmt.Errorf("count activity events: %w", err)
	}
	summary.Empty = summary.Total == 0

	byDay, err := s.groupedCounts(ctx,
		`SELECT substr(at, 1, 10) AS day, COUNT(*) FROM activity_events WHERE at >= ?
		 GROUP BY day ORDER BY day ASC`, cutoff)
	if err != nil {
		return Summary{}, fmt.Errorf("group activity by day: %w", err)
	}
	summary.ByDay = make([]DayCount, 0, len(byDay))
	for _, row := range byDay {
		summary.ByDay = append(summary.ByDay, DayCount{Date: row.Key, Count: row.Count})
	}

	byActor, err := s.groupedCounts(ctx,
		`SELECT actor, COUNT(*) FROM activity_events WHERE at >= ?
		 GROUP BY actor ORDER BY COUNT(*) DESC, actor ASC`, cutoff)
	if err != nil {
		return Summary{}, fmt.Errorf("group activity by actor: %w", err)
	}
	summary.ByActor = make([]ActorCount, 0, len(byActor))
	for _, row := range byActor {
		summary.ByActor = append(summary.ByActor, ActorCount{Actor: row.Key, Count: row.Count})
	}

	byAction, err := s.groupedCounts(ctx,
		`SELECT action, COUNT(*) FROM activity_events WHERE at >= ?
		 GROUP BY action ORDER BY COUNT(*) DESC, action ASC`, cutoff)
	if err != nil {
		return Summary{}, fmt.Errorf("group activity by action: %w", err)
	}
	summary.ByAction = make([]ActionCount, 0, len(byAction))
	for _, row := range byAction {
		summary.ByAction = append(summary.ByAction, ActionCount{Action: row.Key, Count: row.Count})
	}
	return summary, nil
}

type groupedCount struct {
	Key   string
	Count int
}

func (s *Store) groupedCounts(ctx context.Context, query string, args ...any) ([]groupedCount, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]groupedCount, 0)
	for rows.Next() {
		var row groupedCount
		if err := rows.Scan(&row.Key, &row.Count); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func normalizeEvent(event Event, now time.Time) (Event, time.Time, error) {
	event.ID = strings.TrimSpace(event.ID)
	event.Actor = strings.TrimSpace(event.Actor)
	event.Action = strings.TrimSpace(event.Action)
	event.SubjectKind = strings.TrimSpace(event.SubjectKind)
	event.SubjectID = strings.TrimSpace(event.SubjectID)
	event.Summary = strings.TrimSpace(event.Summary)
	event.TraceID = strings.TrimSpace(event.TraceID)

	switch {
	case event.Actor == "":
		return Event{}, time.Time{}, fmt.Errorf("%w: actor is required", ErrInvalid)
	case event.Action == "":
		return Event{}, time.Time{}, fmt.Errorf("%w: action is required", ErrInvalid)
	case event.SubjectKind == "":
		return Event{}, time.Time{}, fmt.Errorf("%w: subject_kind is required", ErrInvalid)
	case event.SubjectID == "":
		return Event{}, time.Time{}, fmt.Errorf("%w: subject_id is required", ErrInvalid)
	}

	at := now.UTC()
	if value := strings.TrimSpace(event.At); value != "" {
		parsed, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			return Event{}, time.Time{}, fmt.Errorf("%w: at must be RFC3339", ErrInvalid)
		}
		at = parsed.UTC()
	}
	event.At = at.Format(timestampLayout)
	return event, at, nil
}

func normalizeTimestamp(value string) (string, error) {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return "", err
	}
	return parsed.UTC().Format(timestampLayout), nil
}

func (s *Store) now() time.Time {
	if s.clock != nil {
		return s.clock()
	}
	return time.Now()
}

// newID is deterministic given the clock and random source: the timestamp
// prefix preserves insertion order, and the random suffix prevents collisions
// between events recorded in the same nanosecond. Tests inject both seams.
func (s *Store) newID(at time.Time) string {
	suffix := make([]byte, idRandomBytes)
	if s.random != nil {
		_, _ = io.ReadFull(s.random, suffix)
	} else {
		_, _ = rand.Read(suffix)
	}
	return fmt.Sprintf("%0*x%s", activityIDTimestamp, uint64(at.UnixNano()), hex.EncodeToString(suffix))
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return defaultListLimit
	}
	if limit > maxListLimit {
		return maxListLimit
	}
	return limit
}

func clampDays(days int) int {
	if days <= 0 {
		return defaultSummaryDays
	}
	if days > maxSummaryDays {
		return maxSummaryDays
	}
	return days
}
