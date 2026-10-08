package scoring

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	snapshotColumns      = "id, scenario, category, digest, composite, classification, working_rung, breakdown_json, importance, source, created_at, last_run_at, last_status"
	latestSnapshotIDsCTE = `WITH latest AS (
	SELECT id FROM (
		SELECT id, ROW_NUMBER() OVER (PARTITION BY scenario ORDER BY created_at DESC, id DESC) AS rn
		FROM score_snapshots
	) WHERE rn = 1
)`
)

type snapshotDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// SQLiteSnapshotRepository stores score history in SQLite.
type SQLiteSnapshotRepository struct {
	db snapshotDB
}

var _ SnapshotRepository = (*SQLiteSnapshotRepository)(nil)

// NewSQLiteSnapshotRepository builds the production snapshot repository.
func NewSQLiteSnapshotRepository(db snapshotDB) *SQLiteSnapshotRepository {
	return &SQLiteSnapshotRepository{db: db}
}

func (r *SQLiteSnapshotRepository) LatestFor(ctx context.Context, scenario string) (Snapshot, bool, error) {
	row := r.db.QueryRowContext(ctx, "SELECT "+snapshotColumns+" FROM score_snapshots WHERE scenario = ? ORDER BY created_at DESC, id DESC LIMIT 1", scenario)
	return scanSnapshotRow(row)
}

func (r *SQLiteSnapshotRepository) LatestDifferingDigest(ctx context.Context, scenario, digest string) (Snapshot, bool, error) {
	row := r.db.QueryRowContext(ctx, "SELECT "+snapshotColumns+" FROM score_snapshots WHERE scenario = ? AND digest <> ? ORDER BY created_at DESC, id DESC LIMIT 1", scenario, digest)
	return scanSnapshotRow(row)
}

func (r *SQLiteSnapshotRepository) SeriesFor(ctx context.Context, q TrendQuery) ([]Snapshot, error) {
	limit := q.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	args := []any{q.Scenario}
	where := "scenario = ?"
	if !q.Since.IsZero() {
		where += " AND created_at >= ?"
		args = append(args, formatSnapshotTime(q.Since))
	}
	args = append(args, limit)
	rows, err := r.db.QueryContext(ctx, "SELECT "+snapshotColumns+" FROM score_snapshots WHERE "+where+" ORDER BY created_at DESC, id DESC LIMIT ?", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSnapshots(rows)
}

func (r *SQLiteSnapshotRepository) UpsertSnapshot(ctx context.Context, snap Snapshot) (bool, error) {
	if snap.CreatedAt.IsZero() {
		return false, errors.New("created_at is required")
	}
	lastRunAt := ""
	if !snap.LastRunAt.IsZero() {
		lastRunAt = formatSnapshotTime(snap.LastRunAt)
	}
	res, err := r.db.ExecContext(ctx, `INSERT OR IGNORE INTO score_snapshots
		(scenario, category, digest, composite, classification, working_rung, breakdown_json, importance, source, created_at, last_run_at, last_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		snap.Scenario,
		defaultString(snap.Category, "utility"),
		snap.Digest,
		snap.Composite,
		snap.Classification,
		snap.WorkingRung,
		defaultString(snap.BreakdownJSON, "{}"),
		nullableFloat(snap.Importance),
		defaultString(snap.Source, "sweeper"),
		formatSnapshotTime(snap.CreatedAt),
		lastRunAt,
		snap.LastStatus,
	)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	// Test recency is orthogonal to the digest-deduplicated score: a new test
	// run can complete without the tree (and thus the digest) changing, so the
	// INSERT OR IGNORE above is a no-op for that row. Always advance recency on
	// the (scenario, digest) row when the incoming run is newer, so re-scores
	// and `--recompute` keep last_run_at/last_status current without minting a
	// duplicate score row. Monotonic guard: never regress to an older run.
	if lastRunAt != "" {
		if _, err := r.db.ExecContext(ctx, `UPDATE score_snapshots
			SET last_run_at = ?, last_status = ?
			WHERE scenario = ? AND digest = ? AND (last_run_at = '' OR last_run_at < ?)`,
			lastRunAt, snap.LastStatus, snap.Scenario, snap.Digest, lastRunAt,
		); err != nil {
			return false, err
		}
	}
	// Importance enrichment is also orthogonal to the digest: it is computed on a
	// separate, slower cadence than the fast score sweep (which writes
	// importance=NULL). Without this update the INSERT OR IGNORE above would
	// silently drop a freshly-computed importance onto an existing (scenario,
	// digest) row — leaving importance dark fleet-wide. Upsert it whenever the
	// incoming snapshot carries one, so the importance-refresh path lands.
	if snap.Importance != nil {
		if _, err := r.db.ExecContext(ctx, `UPDATE score_snapshots
			SET importance = ?
			WHERE scenario = ? AND digest = ?`,
			*snap.Importance, snap.Scenario, snap.Digest,
		); err != nil {
			return false, err
		}
	}
	return n > 0, nil
}

func (r *SQLiteSnapshotRepository) ListPage(ctx context.Context, q ListQuery) (ListResult, error) {
	limit := q.Limit
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	offset := q.Offset
	if offset < 0 {
		offset = 0
	}
	where, args := listWhere(q)
	orderBy := listOrder(q.SortBy, q.Order)
	args = append(args, limit+1, offset)
	query := latestSnapshotIDsCTE + " SELECT " + snapshotColumns +
		" FROM score_snapshots JOIN latest USING (id)" + where +
		" ORDER BY " + orderBy + " LIMIT ? OFFSET ?"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return ListResult{}, err
	}
	defer rows.Close()
	snaps, err := scanSnapshots(rows)
	if err != nil {
		return ListResult{}, err
	}
	hasNext := len(snaps) > limit
	if hasNext {
		snaps = snaps[:limit]
	}
	return ListResult{
		Snapshots:  snaps,
		NextOffset: offset + len(snaps),
		HasNext:    hasNext,
	}, nil
}

func (r *SQLiteSnapshotRepository) CountLatestBelowRung(ctx context.Context, thresholdRank int, window MeasureWindow) (int64, error) {
	if thresholdRank < 0 {
		thresholdRank = 0
	}
	if thresholdRank > 5 {
		thresholdRank = 5
	}
	row := r.db.QueryRowContext(ctx, `WITH latest AS (
		SELECT id FROM (
			SELECT id, ROW_NUMBER() OVER (PARTITION BY scenario ORDER BY created_at DESC, id DESC) AS rn
			FROM score_snapshots
			WHERE created_at >= ? AND created_at < ?
		) WHERE rn = 1
	)
	SELECT COUNT(*)
	FROM score_snapshots JOIN latest USING (id)
	WHERE CASE
		WHEN working_rung LIKE 'R0%' THEN 0
		WHEN working_rung LIKE 'R1%' THEN 1
		WHEN working_rung LIKE 'R2%' THEN 2
		WHEN working_rung LIKE 'R3%' THEN 3
		WHEN working_rung LIKE 'R4%' THEN 4
		ELSE 5
	END < ?`, formatSnapshotTime(window.From), formatSnapshotTime(window.To), thresholdRank)
	var n int64
	if err := row.Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func (r *SQLiteSnapshotRepository) AverageLatestComposite(ctx context.Context, window MeasureWindow) (float64, bool, error) {
	row := r.db.QueryRowContext(ctx, `WITH latest AS (
		SELECT id FROM (
			SELECT id, ROW_NUMBER() OVER (PARTITION BY scenario ORDER BY created_at DESC, id DESC) AS rn
			FROM score_snapshots
			WHERE created_at >= ? AND created_at < ?
		) WHERE rn = 1
	)
	SELECT AVG(composite)
	FROM score_snapshots JOIN latest USING (id)`, formatSnapshotTime(window.From), formatSnapshotTime(window.To))
	var avg sql.NullFloat64
	if err := row.Scan(&avg); err != nil {
		return 0, false, err
	}
	if !avg.Valid {
		return 0, false, nil
	}
	return avg.Float64, true, nil
}

func (r *SQLiteSnapshotRepository) FleetScoreSeries(ctx context.Context, window MeasureWindow) ([]ScoreSeriesPoint, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT substr(created_at, 1, 10) AS bucket, AVG(composite), COUNT(*)
		FROM score_snapshots
		WHERE created_at >= ? AND created_at < ?
		GROUP BY bucket
		ORDER BY bucket ASC`, formatSnapshotTime(window.From), formatSnapshotTime(window.To))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ScoreSeriesPoint
	for rows.Next() {
		var (
			bucket string
			avg    float64
			count  int
		)
		if err := rows.Scan(&bucket, &avg, &count); err != nil {
			return nil, err
		}
		t, err := time.ParseInLocation("2006-01-02", bucket, time.UTC)
		if err != nil {
			return nil, fmt.Errorf("parse score series bucket %q: %w", bucket, err)
		}
		out = append(out, ScoreSeriesPoint{Bucket: t, Score: avg, Count: count})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func listWhere(q ListQuery) (string, []any) {
	clauses := make([]string, 0, 4)
	args := make([]any, 0, 4)
	if q.MinScore != nil {
		clauses = append(clauses, "composite >= ?")
		args = append(args, *q.MinScore)
	}
	if q.MaxScore != nil {
		clauses = append(clauses, "composite <= ?")
		args = append(args, *q.MaxScore)
	}
	if strings.TrimSpace(q.Rung) != "" {
		clauses = append(clauses, "working_rung = ?")
		args = append(args, strings.TrimSpace(q.Rung))
	}
	if strings.TrimSpace(q.Category) != "" {
		clauses = append(clauses, "category = ?")
		args = append(args, strings.TrimSpace(q.Category))
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func listOrder(sortBy SortBy, order SortOrder) string {
	dir := "DESC"
	if order == SortAsc {
		dir = "ASC"
	}
	switch sortBy {
	case SortByScenario:
		return "scenario " + dir + ", id DESC"
	case SortByRung:
		return "working_rung " + dir + ", scenario ASC"
	case SortByLastScored:
		return "created_at " + dir + ", scenario ASC"
	case SortByPriority:
		return "(COALESCE(importance, 0) * ((100 - composite) / 100.0)) " + dir + ", scenario ASC"
	default:
		return "composite " + dir + ", scenario ASC"
	}
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSnapshotRow(row rowScanner) (Snapshot, bool, error) {
	var snap Snapshot
	if err := scanSnapshot(row, &snap); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Snapshot{}, false, nil
		}
		return Snapshot{}, false, err
	}
	return snap, true, nil
}

func scanSnapshots(rows *sql.Rows) ([]Snapshot, error) {
	var out []Snapshot
	for rows.Next() {
		var snap Snapshot
		if err := scanSnapshot(rows, &snap); err != nil {
			return nil, err
		}
		out = append(out, snap)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanSnapshot(row rowScanner, snap *Snapshot) error {
	var importance sql.NullFloat64
	var createdAt string
	var lastRunAt string
	if err := row.Scan(
		&snap.ID,
		&snap.Scenario,
		&snap.Category,
		&snap.Digest,
		&snap.Composite,
		&snap.Classification,
		&snap.WorkingRung,
		&snap.BreakdownJSON,
		&importance,
		&snap.Source,
		&createdAt,
		&lastRunAt,
		&snap.LastStatus,
	); err != nil {
		return err
	}
	if importance.Valid {
		value := importance.Float64
		snap.Importance = &value
	}
	parsed, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return fmt.Errorf("parse created_at %q: %w", createdAt, err)
	}
	snap.CreatedAt = parsed
	if strings.TrimSpace(lastRunAt) != "" {
		runAt, err := time.Parse(time.RFC3339Nano, lastRunAt)
		if err != nil {
			return fmt.Errorf("parse last_run_at %q: %w", lastRunAt, err)
		}
		snap.LastRunAt = runAt
	}
	return nil
}

func formatSnapshotTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func nullableFloat(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

func defaultString(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

// EncodePageOffset converts an offset cursor into the opaque page token used
// by the API contract. The current token is intentionally simple; callers
// should still treat it as opaque.
func EncodePageOffset(offset int) string {
	if offset <= 0 {
		return ""
	}
	return strconv.Itoa(offset)
}

// DecodePageOffset parses a ListScores page token.
func DecodePageOffset(token string) (int, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return 0, nil
	}
	offset, err := strconv.Atoi(token)
	if err != nil || offset < 0 {
		return 0, fmt.Errorf("invalid page token %q", token)
	}
	return offset, nil
}
