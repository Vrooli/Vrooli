package quality

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	apidb "github.com/vrooli/api-core/database"

	"knowledge-observatory/internal/sqlitetime"
)

// SQLite implements Repository against SQLite.
//
// The handle is a *apidb.RoutedDB rather than a *sql.DB so reads and writes
// follow per-request test routing; the method surface is identical.
type SQLite struct {
	DB *apidb.RoutedDB
}

// Compile-time proof the implementation satisfies the domain's interface.
var _ Repository = (*SQLite)(nil)

// NewSQLite returns a Repository backed by db.
func NewSQLite(db *apidb.RoutedDB) *SQLite { return &SQLite{DB: db} }

func (s *SQLite) InsertMetric(ctx context.Context, m Metric) (string, error) {
	if s == nil || s.DB == nil {
		return "", fmt.Errorf("quality repository not configured")
	}
	m.CollectionName = strings.TrimSpace(m.CollectionName)
	if m.CollectionName == "" {
		return "", fmt.Errorf("collection_name is required")
	}
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	if m.MeasuredAt.IsZero() {
		m.MeasuredAt = time.Now().UTC()
	}

	_, err := s.DB.ExecContext(ctx, `
INSERT INTO quality_metrics
  (id, collection_name, coherence_score, freshness_score, redundancy_score, coverage_score,
   total_entries, measured_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT(id) DO NOTHING
`, m.ID, m.CollectionName, m.Coherence, m.Freshness, m.Redundancy, m.Coverage,
		m.TotalEntries, sqlitetime.Format(m.MeasuredAt))
	if err != nil {
		return "", fmt.Errorf("insert quality metric: %w", err)
	}
	return m.ID, nil
}

func (s *SQLite) LatestMetric(ctx context.Context, collection string) (Metric, bool, error) {
	if s == nil || s.DB == nil {
		return Metric{}, false, fmt.Errorf("quality repository not configured")
	}
	collection = strings.TrimSpace(collection)
	if collection == "" {
		return Metric{}, false, fmt.Errorf("collection is required")
	}

	var m Metric
	err := s.DB.QueryRowContext(ctx, `
SELECT id, collection_name, coherence_score, freshness_score, redundancy_score, coverage_score,
       total_entries, avg_quality, measured_at, created_at, updated_at
FROM quality_metrics
WHERE collection_name = ?
ORDER BY measured_at DESC
LIMIT 1
`, collection).Scan(&m.ID, &m.CollectionName, &m.Coherence, &m.Freshness, &m.Redundancy, &m.Coverage,
		&m.TotalEntries, &m.AvgQuality, &m.MeasuredAt, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return Metric{}, false, nil
	}
	if err != nil {
		return Metric{}, false, fmt.Errorf("latest quality metric: %w", err)
	}
	return m, true, nil
}

func (s *SQLite) CountMetrics(ctx context.Context) (int64, error) {
	if s == nil || s.DB == nil {
		return 0, fmt.Errorf("quality repository not configured")
	}
	var n int64
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM quality_metrics`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count quality metrics: %w", err)
	}
	return n, nil
}

func (s *SQLite) UpsertCollectionStat(ctx context.Context, stat CollectionStat) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("quality repository not configured")
	}
	stat.CollectionName = strings.TrimSpace(stat.CollectionName)
	if stat.CollectionName == "" {
		return fmt.Errorf("collection_name is required")
	}
	if stat.ID == "" {
		stat.ID = uuid.NewString()
	}

	_, err := s.DB.ExecContext(ctx, `
INSERT INTO collection_stats
  (id, collection_name, total_entries, total_searches, avg_search_score,
   most_searched_terms, growth_rate, last_updated, created_at)
VALUES (?, ?, ?, ?, ?, NULLIF(?, ''), ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT(collection_name) DO UPDATE SET
  total_entries = excluded.total_entries,
  total_searches = excluded.total_searches,
  avg_search_score = COALESCE(excluded.avg_search_score, collection_stats.avg_search_score),
  most_searched_terms = COALESCE(excluded.most_searched_terms, collection_stats.most_searched_terms),
  growth_rate = COALESCE(excluded.growth_rate, collection_stats.growth_rate),
  last_updated = CURRENT_TIMESTAMP
`, stat.ID, stat.CollectionName, stat.TotalEntries, stat.TotalSearches, stat.AvgSearchScore,
		stat.MostSearchedTerms, stat.GrowthRate)
	if err != nil {
		return fmt.Errorf("upsert collection stat: %w", err)
	}
	return nil
}

func (s *SQLite) GetCollectionStat(ctx context.Context, collection string) (CollectionStat, bool, error) {
	if s == nil || s.DB == nil {
		return CollectionStat{}, false, fmt.Errorf("quality repository not configured")
	}
	collection = strings.TrimSpace(collection)
	if collection == "" {
		return CollectionStat{}, false, fmt.Errorf("collection is required")
	}

	var (
		stat  CollectionStat
		terms sql.NullString
	)
	err := s.DB.QueryRowContext(ctx, `
SELECT id, collection_name, total_entries, total_searches, avg_search_score,
       most_searched_terms, growth_rate, last_updated, created_at
FROM collection_stats
WHERE collection_name = ?
`, collection).Scan(&stat.ID, &stat.CollectionName, &stat.TotalEntries, &stat.TotalSearches,
		&stat.AvgSearchScore, &terms, &stat.GrowthRate, &stat.LastUpdated, &stat.CreatedAt)
	if err == sql.ErrNoRows {
		return CollectionStat{}, false, nil
	}
	if err != nil {
		return CollectionStat{}, false, fmt.Errorf("get collection stat: %w", err)
	}
	stat.MostSearchedTerms = terms.String
	return stat, true, nil
}

func (s *SQLite) Dashboard(ctx context.Context) ([]DashboardRow, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("quality repository not configured")
	}
	rows, err := s.DB.QueryContext(ctx, `
SELECT collection_name, total_entries, coherence_score, freshness_score, redundancy_score,
       coverage_score, avg_quality, total_searches, avg_search_score, measured_at
FROM dashboard_metrics
ORDER BY collection_name
`)
	if err != nil {
		return nil, fmt.Errorf("query dashboard metrics: %w", err)
	}
	defer rows.Close()

	var out []DashboardRow
	for rows.Next() {
		var (
			r        DashboardRow
			measured sql.NullTime
		)
		if err := rows.Scan(&r.CollectionName, &r.TotalEntries, &r.Coherence, &r.Freshness,
			&r.Redundancy, &r.Coverage, &r.AvgQuality, &r.TotalSearches, &r.AvgSearchScore,
			&measured); err != nil {
			return nil, fmt.Errorf("scan dashboard metric: %w", err)
		}
		if measured.Valid {
			t := measured.Time.UTC()
			r.MeasuredAt = &t
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *SQLite) PruneMetricsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	if s == nil || s.DB == nil {
		return 0, fmt.Errorf("quality repository not configured")
	}
	res, err := s.DB.ExecContext(ctx,
		`DELETE FROM quality_metrics WHERE measured_at < ?`, sqlitetime.Format(cutoff))
	if err != nil {
		return 0, fmt.Errorf("prune quality metrics: %w", err)
	}
	return res.RowsAffected()
}

// DownsampleMetricsOlderThan keeps, for each (collection, calendar day) before
// cutoff, only the sample with the newest measured_at and deletes the rest.
//
// Re-running it is a no-op once a range is already collapsed, because a day
// that holds a single row has nothing left to delete.
func (s *SQLite) DownsampleMetricsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	if s == nil || s.DB == nil {
		return 0, fmt.Errorf("quality repository not configured")
	}
	res, err := s.DB.ExecContext(ctx, `
DELETE FROM quality_metrics
WHERE measured_at < ?
  AND id NOT IN (
      SELECT id FROM (
          SELECT id,
                 ROW_NUMBER() OVER (
                     PARTITION BY collection_name, date(measured_at)
                     ORDER BY measured_at DESC, id DESC
                 ) AS rn
          FROM quality_metrics
          WHERE measured_at < ?
      )
      WHERE rn = 1
  )
`, sqlitetime.Format(cutoff), sqlitetime.Format(cutoff))
	if err != nil {
		return 0, fmt.Errorf("downsample quality metrics: %w", err)
	}
	return res.RowsAffected()
}
