package analytics

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface sqliteRepository depends
// on; both *sql.DB and *database.RoutedDB satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor, clk schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

const eventTimeFormat = time.RFC3339Nano

const (
	insertEventSQL = `
INSERT INTO analytics_events
  (id, kind, scenario, domain, conflict_id, chunk_id, plan_id, run_id,
   corrects_event_id, payload, actor, recorded_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	selectEventsSQL = `
SELECT id, kind, scenario, domain, conflict_id, chunk_id, plan_id, run_id,
       corrects_event_id, payload, actor, recorded_at
FROM analytics_events
WHERE scenario = ?
ORDER BY recorded_at DESC, id DESC
LIMIT ?`

	insertPlacementSQL = `
INSERT INTO analytics_placements
  (id, scenario, chunk_id, chunk_path, verdict_json, outcome, auto_acted, recorded_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	selectPlacementsSQL = `
SELECT id, scenario, chunk_id, chunk_path, verdict_json, outcome, auto_acted, recorded_at
FROM analytics_placements
WHERE scenario = ?
ORDER BY recorded_at DESC, id DESC
LIMIT ?`

	insertOverrideSQL = `
INSERT INTO analytics_overrides
  (id, scenario, chunk_id, verdict_domain, chosen_domain, note, verdict_event_id, recorded_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
)

func (s *sqliteRepository) AppendEvent(ctx context.Context, e Event) (Event, error) {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.RecordedAt.IsZero() {
		e.RecordedAt = s.clock.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, insertEventSQL,
		e.ID, string(e.Kind), e.Scenario, e.Domain,
		e.ConflictID, e.ChunkID, e.PlanID, e.RunID,
		e.CorrectsEventID, e.Payload, e.Actor,
		e.RecordedAt.Format(eventTimeFormat),
	)
	if err != nil {
		return Event{}, fmt.Errorf("insert analytics_event %q: %w", e.ID, err)
	}
	return e, nil
}

func (s *sqliteRepository) ListEvents(ctx context.Context, f EventFilter) (EventPage, error) {
	limit := f.PageSize
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, selectEventsSQL, f.Scenario, limit)
	if err != nil {
		return EventPage{}, fmt.Errorf("list analytics_events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return EventPage{}, err
		}
		if !matchesKinds(e.Kind, f.Kinds) {
			continue
		}
		if !f.Since.IsZero() && e.RecordedAt.Before(f.Since) {
			continue
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return EventPage{}, fmt.Errorf("iterate analytics_events: %w", err)
	}
	return EventPage{Events: events}, nil
}

func (s *sqliteRepository) AppendPlacement(ctx context.Context, p Placement) (Placement, error) {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	if p.RecordedAt.IsZero() {
		p.RecordedAt = s.clock.Now().UTC()
	}
	autoActed := 0
	if p.AutoActed {
		autoActed = 1
	}
	_, err := s.db.ExecContext(ctx, insertPlacementSQL,
		p.ID, p.Scenario, p.ChunkID, p.ChunkPath, p.VerdictJSON,
		p.Outcome, autoActed, p.RecordedAt.Format(eventTimeFormat),
	)
	if err != nil {
		return Placement{}, fmt.Errorf("insert analytics_placement %q: %w", p.ID, err)
	}
	return p, nil
}

func (s *sqliteRepository) ListPlacements(ctx context.Context, f PlacementFilter) (PlacementPage, error) {
	limit := f.PageSize
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, selectPlacementsSQL, f.Scenario, limit)
	if err != nil {
		return PlacementPage{}, fmt.Errorf("list analytics_placements: %w", err)
	}
	defer rows.Close()

	var placements []Placement
	for rows.Next() {
		p, err := scanPlacement(rows)
		if err != nil {
			return PlacementPage{}, err
		}
		if !matchesOutcomes(p.Outcome, f.Outcomes) {
			continue
		}
		placements = append(placements, p)
	}
	if err := rows.Err(); err != nil {
		return PlacementPage{}, fmt.Errorf("iterate analytics_placements: %w", err)
	}
	return PlacementPage{Placements: placements}, nil
}

func (s *sqliteRepository) AppendOverride(ctx context.Context, o Override) (Override, error) {
	if o.ID == "" {
		o.ID = uuid.NewString()
	}
	if o.RecordedAt.IsZero() {
		o.RecordedAt = s.clock.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, insertOverrideSQL,
		o.ID, o.Scenario, o.ChunkID, o.VerdictDomain, o.ChosenDomain,
		o.Note, o.VerdictEventID, o.RecordedAt.Format(eventTimeFormat),
	)
	if err != nil {
		return Override{}, fmt.Errorf("insert analytics_override %q: %w", o.ID, err)
	}
	return o, nil
}

const statsSQL = `
SELECT
  (SELECT COUNT(*) FROM analytics_events WHERE scenario = ? AND kind = ?),
  (SELECT COUNT(*) FROM analytics_events WHERE scenario = ? AND kind = ?),
  (SELECT COUNT(*) FROM analytics_events WHERE scenario = ? AND kind = ?),
  (SELECT COUNT(*) FROM analytics_placements WHERE scenario = ? AND outcome = ?),
  (SELECT COUNT(*) FROM analytics_placements WHERE scenario = ? AND outcome = ?),
  (SELECT COUNT(*) FROM analytics_overrides WHERE scenario = ?),
  (SELECT COUNT(*) FROM analytics_events WHERE scenario = ? AND kind = ?)
`

func (s *sqliteRepository) Stats(ctx context.Context, scenario string) (StatsSummary, error) {
	row := s.db.QueryRowContext(ctx, statsSQL,
		scenario, string(EventKindConflictDetected),
		scenario, string(EventKindConflictResolved),
		scenario, string(EventKindConflictForceResolved),
		scenario, "auto_placed",
		scenario, "suggested",
		scenario,
		scenario, string(EventKindVerdictProduced),
	)
	var ss StatsSummary
	ss.Scenario = scenario
	if err := row.Scan(
		&ss.ConflictsDetected,
		&ss.ConflictsResolved,
		&ss.ConflictsForceResolved,
		&ss.PlacementsAuto,
		&ss.PlacementsSuggest,
		&ss.Overrides,
		&ss.VerdictObservationCount,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ss, nil
		}
		return StatsSummary{}, fmt.Errorf("stats: %w", err)
	}
	// Verdict success rate = 1 - (overrides / verdict_observations).
	// Suppressed when N < threshold.
	if ss.VerdictObservationCount < MinVerdictObservations {
		ss.VerdictSuccessRateSuppressed = true
	} else {
		ss.VerdictSuccessRate = 1.0 - float64(ss.Overrides)/float64(ss.VerdictObservationCount)
		if ss.VerdictSuccessRate < 0 {
			ss.VerdictSuccessRate = 0
		}
	}
	return ss, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEvent(s rowScanner) (Event, error) {
	var (
		e          Event
		kind       string
		recordedAt string
	)
	if err := s.Scan(
		&e.ID, &kind, &e.Scenario, &e.Domain,
		&e.ConflictID, &e.ChunkID, &e.PlanID, &e.RunID,
		&e.CorrectsEventID, &e.Payload, &e.Actor, &recordedAt,
	); err != nil {
		return Event{}, err
	}
	e.Kind = EventKind(kind)
	t, err := time.Parse(eventTimeFormat, recordedAt)
	if err != nil {
		return Event{}, fmt.Errorf("parse recorded_at %q: %w", recordedAt, err)
	}
	e.RecordedAt = t
	return e, nil
}

func scanPlacement(s rowScanner) (Placement, error) {
	var (
		p          Placement
		autoActed  int
		recordedAt string
	)
	if err := s.Scan(
		&p.ID, &p.Scenario, &p.ChunkID, &p.ChunkPath,
		&p.VerdictJSON, &p.Outcome, &autoActed, &recordedAt,
	); err != nil {
		return Placement{}, err
	}
	p.AutoActed = autoActed != 0
	t, err := time.Parse(eventTimeFormat, recordedAt)
	if err != nil {
		return Placement{}, fmt.Errorf("parse recorded_at %q: %w", recordedAt, err)
	}
	p.RecordedAt = t
	return p, nil
}

func matchesKinds(k EventKind, allowed []EventKind) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, a := range allowed {
		if a == k {
			return true
		}
	}
	return false
}

func matchesOutcomes(o string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, a := range allowed {
		if a == o {
			return true
		}
	}
	return false
}
