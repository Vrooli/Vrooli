package runs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"vrooli-bridge/internal/clock"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface sqliteRepository depends on. Both
// *sql.DB (repository unit tests) and *database.RoutedDB (production) satisfy
// it, so production participates in per-request routing without the test
// fixture wrapping its handle.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock clock.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

// Compile-time guarantee.
var _ Repository = (*sqliteRepository)(nil)

// runTimeFormat sorts lexicographically in time order for a fixed zone, so a
// string range/order comparison is a correct filter — matching the wire format
// and the registry domain convention.
const runTimeFormat = time.RFC3339Nano

const (
	insertRunSQL = `
INSERT INTO runs (id, node_id, scenario, verb, args, status, exit_code, timeout_seconds, created_at, started_at, finished_at, artifact_refs)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

	selectRunColumns = `
SELECT id, node_id, scenario, verb, args, status, exit_code, timeout_seconds, created_at, started_at, finished_at, artifact_refs
FROM runs
`

	selectRunByIDSQL = selectRunColumns + `WHERE id = ?`

	updateRunSQL = `
UPDATE runs
SET status = ?, exit_code = ?, started_at = ?, finished_at = ?, artifact_refs = ?
WHERE id = ?
`

	insertRunEventSQL = `
INSERT OR IGNORE INTO run_events (run_id, sequence, kind, log_chunk, status, exit_code, artifact_ref, emitted_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`

	listRunEventsSQL = `
SELECT run_id, sequence, kind, log_chunk, status, exit_code, artifact_ref, emitted_at
FROM run_events
WHERE run_id = ?
ORDER BY sequence ASC
`
)

func (s *sqliteRepository) Create(ctx context.Context, r Run) (Run, error) {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = s.clock.Now().UTC()
	}
	if r.Status == StatusUnspecified {
		r.Status = StatusQueued
	}
	args, err := marshalStrings(r.Args)
	if err != nil {
		return Run{}, fmt.Errorf("encode args: %w", err)
	}
	refs, err := marshalStrings(r.ArtifactRefs)
	if err != nil {
		return Run{}, fmt.Errorf("encode artifact_refs: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, insertRunSQL,
		r.ID, r.NodeID, r.Scenario, r.Verb, args, int(r.Status), r.ExitCode, r.TimeoutSeconds,
		r.CreatedAt.Format(runTimeFormat), formatNullableTime(r.StartedAt), formatNullableTime(r.FinishedAt), refs,
	); err != nil {
		return Run{}, fmt.Errorf("insert run %q: %w", r.ID, err)
	}
	return r, nil
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Run, error) {
	row := s.db.QueryRowContext(ctx, selectRunByIDSQL, id)
	r, err := scanRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Run{}, ErrRunNotFound{ID: id}
	}
	if err != nil {
		return Run{}, fmt.Errorf("get run %q: %w", id, err)
	}
	return r, nil
}

func (s *sqliteRepository) List(ctx context.Context, filter ListFilter) ([]Run, error) {
	query := selectRunColumns
	args := make([]any, 0, 2)
	if filter.NodeID != "" {
		query += `WHERE node_id = ? `
		args = append(args, filter.NodeID)
	}
	query += `ORDER BY created_at DESC, id DESC`
	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list runs: %w", err)
	}
	defer rows.Close()

	var runs []Run
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, fmt.Errorf("scan run: %w", err)
		}
		runs = append(runs, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate runs: %w", err)
	}
	return runs, nil
}

func (s *sqliteRepository) Update(ctx context.Context, r Run) (Run, error) {
	existing, err := s.Get(ctx, r.ID)
	if err != nil {
		return Run{}, err
	}
	existing.Status = r.Status
	existing.ExitCode = r.ExitCode
	existing.StartedAt = r.StartedAt
	existing.FinishedAt = r.FinishedAt
	existing.ArtifactRefs = r.ArtifactRefs

	refs, err := marshalStrings(existing.ArtifactRefs)
	if err != nil {
		return Run{}, fmt.Errorf("encode artifact_refs: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, updateRunSQL,
		int(existing.Status), existing.ExitCode,
		formatNullableTime(existing.StartedAt), formatNullableTime(existing.FinishedAt), refs, existing.ID,
	); err != nil {
		return Run{}, fmt.Errorf("update run %q: %w", r.ID, err)
	}
	return existing, nil
}

func (s *sqliteRepository) AppendEvent(ctx context.Context, ev RunEvent) error {
	emitted := formatNullableTime(ev.EmittedAt)
	if _, err := s.db.ExecContext(ctx, insertRunEventSQL,
		ev.RunID, ev.Sequence, int(ev.Kind), ev.LogChunk, ev.Status, ev.ExitCode, ev.ArtifactRef, emitted,
	); err != nil {
		return fmt.Errorf("append event for run %q: %w", ev.RunID, err)
	}
	return nil
}

func (s *sqliteRepository) ListEvents(ctx context.Context, runID string) ([]RunEvent, error) {
	rows, err := s.db.QueryContext(ctx, listRunEventsSQL, runID)
	if err != nil {
		return nil, fmt.Errorf("list events for run %q: %w", runID, err)
	}
	defer rows.Close()

	var events []RunEvent
	for rows.Next() {
		var (
			ev         RunEvent
			kind       int
			emittedRaw string
		)
		if err := rows.Scan(&ev.RunID, &ev.Sequence, &kind, &ev.LogChunk, &ev.Status, &ev.ExitCode, &ev.ArtifactRef, &emittedRaw); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		ev.Kind = EventKind(kind)
		if ev.EmittedAt, err = parseNullableTime(emittedRaw); err != nil {
			return nil, fmt.Errorf("parse emitted_at %q: %w", emittedRaw, err)
		}
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}
	return events, nil
}

// rowScanner unifies *sql.Row and *sql.Rows under their common Scan surface.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanRun(sc rowScanner) (Run, error) {
	var (
		r           Run
		argsRaw     string
		status      int
		createdRaw  string
		startedRaw  string
		finishedRaw string
		refsRaw     string
	)
	if err := sc.Scan(&r.ID, &r.NodeID, &r.Scenario, &r.Verb, &argsRaw, &status, &r.ExitCode,
		&r.TimeoutSeconds, &createdRaw, &startedRaw, &finishedRaw, &refsRaw); err != nil {
		return Run{}, err
	}
	r.Status = RunStatus(status)

	args, err := unmarshalStrings(argsRaw)
	if err != nil {
		return Run{}, fmt.Errorf("decode args: %w", err)
	}
	refs, err := unmarshalStrings(refsRaw)
	if err != nil {
		return Run{}, fmt.Errorf("decode artifact_refs: %w", err)
	}
	r.Args = args
	r.ArtifactRefs = refs

	if r.CreatedAt, err = time.Parse(runTimeFormat, createdRaw); err != nil {
		return Run{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	if r.StartedAt, err = parseNullableTime(startedRaw); err != nil {
		return Run{}, fmt.Errorf("parse started_at %q: %w", startedRaw, err)
	}
	if r.FinishedAt, err = parseNullableTime(finishedRaw); err != nil {
		return Run{}, fmt.Errorf("parse finished_at %q: %w", finishedRaw, err)
	}
	return r, nil
}

// marshalStrings encodes a string slice as a JSON array, normalising nil to
// "[]" so the column never holds NULL/"".
func marshalStrings(v []string) (string, error) {
	if v == nil {
		v = []string{}
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func unmarshalStrings(raw string) ([]string, error) {
	if raw == "" {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

// formatNullableTime renders a zero time as "" (the column default) so absence
// is distinguishable from a real timestamp.
func formatNullableTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(runTimeFormat)
}

func parseNullableTime(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, nil
	}
	return time.Parse(runTimeFormat, raw)
}
