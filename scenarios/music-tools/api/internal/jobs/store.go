package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// jobTimeFormat keeps UTC timestamps fixed-width so SQLite text comparisons are
// chronological at nanosecond precision.
const jobTimeFormat = "2006-01-02T15:04:05.000000000Z"

// SQLExecutor is the narrow database surface the store depends on. Both *sql.DB
// (unit tests) and *database.RoutedDB (production) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// store persists Job records.
type store struct {
	db SQLExecutor
}

func newStore(db SQLExecutor) *store { return &store{db: db} }

// migrate verifies the owner-installed schema and upgrades legacy timestamp
// strings once before any query depends on their lexical ordering.
func (s *store) migrate(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, "SELECT meta FROM jobs LIMIT 0")
	if err != nil {
		return fmt.Errorf("jobs: apply owner schema compatibility before start: %w", err)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	var applied int
	err = s.db.QueryRowContext(ctx, "SELECT version FROM jobs_schema_migrations WHERE version=?", 1).Scan(&applied)
	if err == nil {
		return nil
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("jobs: read timestamp migration marker: %w", err)
	}

	// Each batch is read and closed before its updates. If startup is interrupted,
	// retrying is safe because canonical rows stay canonical and the marker is
	// written only after every row is successfully upgraded.
	const batchSize = 500
	lastID := ""
	firstBatch := true
	for {
		batch, err := s.readLegacyTimestampBatch(ctx, lastID, firstBatch, batchSize)
		if err != nil {
			return err
		}
		if len(batch) == 0 {
			break
		}
		for _, row := range batch {
			created, err := canonicalStoredTime(row.id, "created_at", row.createdAt, false)
			if err != nil {
				return err
			}
			started, err := canonicalStoredTime(row.id, "started_at", row.startedAt, true)
			if err != nil {
				return err
			}
			finished, err := canonicalStoredTime(row.id, "finished_at", row.finishedAt, true)
			if err != nil {
				return err
			}
			if created == row.createdAt && started == row.startedAt && finished == row.finishedAt {
				continue
			}
			if _, err := s.db.ExecContext(ctx, "UPDATE jobs SET created_at=?, started_at=?, finished_at=? WHERE id=?", created, started, finished, row.id); err != nil {
				return fmt.Errorf("jobs: normalize timestamps for job %q: %w", row.id, err)
			}
		}
		lastID = batch[len(batch)-1].id
		firstBatch = false
	}
	if _, err := s.db.ExecContext(ctx, "INSERT OR IGNORE INTO jobs_schema_migrations(version) VALUES (?)", 1); err != nil {
		return fmt.Errorf("jobs: record timestamp migration: %w", err)
	}
	return nil
}

type legacyTimestampRow struct {
	id                               string
	createdAt, startedAt, finishedAt string
}

func (s *store) readLegacyTimestampBatch(ctx context.Context, afterID string, firstBatch bool, limit int) ([]legacyTimestampRow, error) {
	query := "SELECT id, created_at, started_at, finished_at FROM jobs WHERE id>? ORDER BY id LIMIT ?"
	args := []any{afterID, limit}
	if firstBatch {
		query = "SELECT id, created_at, started_at, finished_at FROM jobs ORDER BY id LIMIT ?"
		args = []any{limit}
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("jobs: read legacy timestamps: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var batch []legacyTimestampRow
	for rows.Next() {
		var row legacyTimestampRow
		if err := rows.Scan(&row.id, &row.createdAt, &row.startedAt, &row.finishedAt); err != nil {
			return nil, fmt.Errorf("jobs: scan legacy timestamps: %w", err)
		}
		batch = append(batch, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("jobs: read legacy timestamp rows: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("jobs: close legacy timestamp rows: %w", err)
	}
	return batch, nil
}

func canonicalStoredTime(jobID, column, value string, optional bool) (string, error) {
	if value == "" && optional {
		return "", nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return "", fmt.Errorf("jobs: normalize %s for job %q: parse %q: %w", column, jobID, value, err)
	}
	return parsed.UTC().Format(jobTimeFormat), nil
}

const insertJobSQL = `
INSERT INTO jobs (id, operation, lane, state, progress, message, error, result_ref, meta, payload, estimated_seconds, created_at, started_at, finished_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

func (s *store) insert(ctx context.Context, j Job) error {
	if !j.State.Valid() {
		return fmt.Errorf("jobs: insert: unknown state %q", j.State)
	}
	_, err := s.db.ExecContext(ctx, insertJobSQL,
		j.ID, j.Operation, string(j.Lane), string(j.State), j.Progress, j.Message, j.Error, j.ResultRef, encodeMeta(j.Meta),
		j.Payload, j.EstimatedSeconds, j.CreatedAt.Format(jobTimeFormat), formatTimePtr(j.StartedAt), formatTimePtr(j.FinishedAt),
	)
	if err != nil {
		return fmt.Errorf("jobs: insert: %w", err)
	}
	return nil
}

const updateJobSQL = `
UPDATE jobs SET state=?, progress=?, message=?, error=?, result_ref=?, meta=?, started_at=?, finished_at=?
WHERE id=? AND state IN ('queued','running')`

func (s *store) update(ctx context.Context, j Job) error {
	if !j.State.Valid() {
		return fmt.Errorf("jobs: update: unknown state %q", j.State)
	}
	result, err := s.db.ExecContext(ctx, updateJobSQL,
		string(j.State), j.Progress, j.Message, j.Error, j.ResultRef, encodeMeta(j.Meta),
		formatTimePtr(j.StartedAt), formatTimePtr(j.FinishedAt), j.ID,
	)
	if err != nil {
		return fmt.Errorf("jobs: update: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		authoritative, readErr := s.get(ctx, j.ID)
		if readErr != nil {
			return fmt.Errorf("jobs: terminal fence read: %w", readErr)
		}
		if authoritative.State.Terminal() {
			return &terminalWinner{job: authoritative}
		}
	}
	if n != 1 {
		return fmt.Errorf("jobs: expected one persisted active row, updated %d", n)
	}
	return nil
}

// terminalWinner is an authoritative persisted first terminal result. It is
// adoption, never permission to retry a stale terminal overwrite.
type terminalWinner struct{ job Job }

func (e *terminalWinner) Error() string { return "jobs: authoritative terminal already persisted" }

const selectJobColumns = `id, operation, lane, state, progress, message, error, result_ref, meta, payload, estimated_seconds, created_at, started_at, finished_at`

// encodeMeta stores an absent map as the empty string rather than "{}" so a row
// that carries no backend record is distinguishable from one that carries an
// empty record, which is the same distinction ai.MetaCostUSD is written to
// preserve: "cost nothing" and "nobody reported a cost" are different facts.
func encodeMeta(meta map[string]string) string {
	if len(meta) == 0 {
		return ""
	}
	encoded, err := json.Marshal(meta)
	if err != nil {
		// Unreachable: a map[string]string always marshals.
		return ""
	}
	return string(encoded)
}

func decodeMeta(raw string) map[string]string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var meta map[string]string
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return nil
	}
	return meta
}

func (s *store) get(ctx context.Context, id string) (Job, error) {
	row := s.db.QueryRowContext(ctx, "SELECT "+selectJobColumns+" FROM jobs WHERE id=?", id)
	return scanJob(row)
}

func (s *store) list(ctx context.Context, limit int) ([]Job, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, "SELECT "+selectJobColumns+" FROM jobs ORDER BY created_at DESC LIMIT ?", limit)
	if err != nil {
		return nil, fmt.Errorf("jobs: list: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("jobs: list rows: %w", err)
	}
	return out, nil
}

// listRetentionCandidates returns the oldest terminal jobs whose managed
// output is eligible for owner-side retention. The query deliberately requires
// an out/ blob reference: inputs, conditioning assets, model-install paths,
// and caller-owned absolute paths are not reclaimable through this owner.
func (s *store) listRetentionCandidates(ctx context.Context, before time.Time, limit int) ([]Job, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx, "SELECT "+selectJobColumns+" FROM jobs WHERE state=? AND result_ref LIKE 'out/%' AND finished_at<>'' AND finished_at<? ORDER BY finished_at ASC LIMIT ?", string(StateSucceeded), before.UTC().Format(jobTimeFormat), limit)
	if err != nil {
		return nil, fmt.Errorf("jobs: list retention candidates: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("jobs: list retention candidates rows: %w", err)
	}
	return out, nil
}

// listNonTerminal returns jobs still queued/running (used at boot for recovery).
func (s *store) listNonTerminal(ctx context.Context) ([]Job, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+selectJobColumns+" FROM jobs WHERE state IN (?, ?)", string(StateQueued), string(StateRunning))
	if err != nil {
		return nil, fmt.Errorf("jobs: list non-terminal: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanJob(sc scanner) (Job, error) {
	var (
		j                     Job
		lane, state           string
		createdAt             string
		startedAt, finishedAt string
		meta                  string
		payload               []byte
	)
	if err := sc.Scan(
		&j.ID, &j.Operation, &lane, &state, &j.Progress, &j.Message, &j.Error, &j.ResultRef, &meta,
		&payload, &j.EstimatedSeconds, &createdAt, &startedAt, &finishedAt,
	); err != nil {
		return Job{}, fmt.Errorf("jobs: scan: %w", err)
	}
	j.Lane = Lane(lane)
	j.State = State(state)
	if !j.State.Valid() {
		return Job{}, fmt.Errorf("jobs: scan: unknown state %q for job %q", j.State, j.ID)
	}
	j.Meta = decodeMeta(meta)
	j.Payload = payload
	t, err := time.Parse(jobTimeFormat, createdAt)
	if err != nil {
		return Job{}, fmt.Errorf("jobs: parse created_at: %w", err)
	}
	j.CreatedAt = t
	j.StartedAt = parseTimePtr(startedAt)
	j.FinishedAt = parseTimePtr(finishedAt)
	return j, nil
}

func formatTimePtr(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format(jobTimeFormat)
}

func parseTimePtr(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(jobTimeFormat, s)
	if err != nil {
		return nil
	}
	return &t
}
