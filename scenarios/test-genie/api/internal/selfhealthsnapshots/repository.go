// Package selfhealthsnapshots is the persisted self-health trend store: a
// per-domain SQLite table (storage-steer §8 — all SQL lives here, never in
// handlers or the sweeper composition) holding timestamped reliability +
// conformance rollups. The background sweeper is the sole writer; the read
// path composes Latest/LatestDifferingDigest/Series to surface trend deltas.
package selfhealthsnapshots

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"test-genie/internal/dbexec"
)

// timeLayout is the stable textual timestamp encoding (UTC, RFC3339Nano) so
// captured_at sorts lexicographically in the same order as chronologically.
const timeLayout = time.RFC3339Nano

// Snapshot is one persisted self-health rollup. PayloadJSON carries the full
// serialized rollup (per-phase reliability etc.); the promoted columns are the
// headline metrics the trend surface and meta-opt baseline read directly.
type Snapshot struct {
	CapturedAt     time.Time
	WindowDays     int
	RunCount       int
	Availability   float64
	HardViolations int
	MetricsAdopted int
	ProvidersTotal int
	PayloadJSON    string
	Digest         string
	Source         string
}

// SeriesQuery bounds a windowed trend series read.
type SeriesQuery struct {
	// Since, when non-zero, returns only snapshots captured at/after it.
	Since time.Time
	// Limit caps the number of newest-first points returned (0 = no cap).
	Limit int
}

// SnapshotRepository is the storage seam the sweeper writes and the read path
// composes. Mirrors the SCS SnapshotRepository shape.
type SnapshotRepository interface {
	// Insert persists a snapshot, deduplicating on digest via INSERT OR IGNORE.
	// It returns inserted=false (no error) when an identical-digest row exists.
	Insert(ctx context.Context, snap Snapshot) (bool, error)
	// Latest returns the most recently captured snapshot, if any.
	Latest(ctx context.Context) (Snapshot, bool, error)
	// LatestDifferingDigest returns the most recent snapshot whose digest differs
	// from the supplied one — the baseline a trend delta is computed against.
	LatestDifferingDigest(ctx context.Context, digest string) (Snapshot, bool, error)
	// Series returns snapshots newest-first, bounded by the query.
	Series(ctx context.Context, q SeriesQuery) ([]Snapshot, error)
}

// SqliteRepository is the SQLite-backed SnapshotRepository.
type SqliteRepository struct {
	db dbexec.Executor
}

var _ SnapshotRepository = (*SqliteRepository)(nil)

// NewSqliteRepository binds a repository to the narrow dbexec seam.
func NewSqliteRepository(db dbexec.Executor) *SqliteRepository {
	return &SqliteRepository{db: db}
}

func (r *SqliteRepository) Insert(ctx context.Context, snap Snapshot) (bool, error) {
	if r == nil || r.db == nil {
		return false, errors.New("selfhealthsnapshots: nil repository")
	}
	if snap.Digest == "" {
		return false, errors.New("selfhealthsnapshots: snapshot digest is required")
	}
	capturedAt := snap.CapturedAt
	if capturedAt.IsZero() {
		return false, errors.New("selfhealthsnapshots: captured_at is required")
	}
	source := snap.Source
	if source == "" {
		source = "sweeper"
	}
	payload := snap.PayloadJSON
	if payload == "" {
		payload = "{}"
	}
	res, err := r.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO selfhealth_snapshots
			(captured_at, window_days, run_count, availability, hard_violations, metrics_adopted, providers_total, payload_json, digest, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		capturedAt.UTC().Format(timeLayout), snap.WindowDays, snap.RunCount, snap.Availability,
		snap.HardViolations, snap.MetricsAdopted, snap.ProvidersTotal, payload, snap.Digest, source,
	)
	if err != nil {
		return false, fmt.Errorf("insert self-health snapshot: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("insert self-health snapshot rows affected: %w", err)
	}
	return affected > 0, nil
}

func (r *SqliteRepository) Latest(ctx context.Context) (Snapshot, bool, error) {
	return r.queryOne(ctx, `
		SELECT captured_at, window_days, run_count, availability, hard_violations, metrics_adopted, providers_total, payload_json, digest, source
		FROM selfhealth_snapshots
		ORDER BY captured_at DESC, id DESC
		LIMIT 1`)
}

func (r *SqliteRepository) LatestDifferingDigest(ctx context.Context, digest string) (Snapshot, bool, error) {
	return r.queryOne(ctx, `
		SELECT captured_at, window_days, run_count, availability, hard_violations, metrics_adopted, providers_total, payload_json, digest, source
		FROM selfhealth_snapshots
		WHERE digest != ?
		ORDER BY captured_at DESC, id DESC
		LIMIT 1`, digest)
}

func (r *SqliteRepository) Series(ctx context.Context, q SeriesQuery) ([]Snapshot, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("selfhealthsnapshots: nil repository")
	}
	query := `
		SELECT captured_at, window_days, run_count, availability, hard_violations, metrics_adopted, providers_total, payload_json, digest, source
		FROM selfhealth_snapshots`
	var args []any
	if !q.Since.IsZero() {
		query += " WHERE captured_at >= ?"
		args = append(args, q.Since.UTC().Format(timeLayout))
	}
	query += " ORDER BY captured_at DESC, id DESC"
	if q.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, q.Limit)
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query self-health series: %w", err)
	}
	defer rows.Close()
	var out []Snapshot
	for rows.Next() {
		snap, err := scanSnapshot(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, snap)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate self-health series: %w", err)
	}
	return out, nil
}

func (r *SqliteRepository) queryOne(ctx context.Context, query string, args ...any) (Snapshot, bool, error) {
	if r == nil || r.db == nil {
		return Snapshot{}, false, errors.New("selfhealthsnapshots: nil repository")
	}
	snap, err := scanSnapshot(r.db.QueryRowContext(ctx, query, args...).Scan)
	if errors.Is(err, sql.ErrNoRows) {
		return Snapshot{}, false, nil
	}
	if err != nil {
		return Snapshot{}, false, err
	}
	return snap, true, nil
}

// scanSnapshot decodes one row from either *sql.Row or *sql.Rows via their
// shared Scan signature.
func scanSnapshot(scan func(dest ...any) error) (Snapshot, error) {
	var (
		snap       Snapshot
		capturedAt string
	)
	if err := scan(&capturedAt, &snap.WindowDays, &snap.RunCount, &snap.Availability,
		&snap.HardViolations, &snap.MetricsAdopted, &snap.ProvidersTotal, &snap.PayloadJSON, &snap.Digest, &snap.Source); err != nil {
		return Snapshot{}, err
	}
	parsed, err := time.Parse(timeLayout, capturedAt)
	if err != nil {
		return Snapshot{}, fmt.Errorf("parse captured_at %q: %w", capturedAt, err)
	}
	snap.CapturedAt = parsed
	return snap, nil
}
