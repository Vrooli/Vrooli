package tunnel

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface sqliteRepository depends on.
// Declared at the consumer per seam-discovery: both *sql.DB (repository unit
// tests via testutil/db.NewSQLite) and *database.RoutedDB (production main.go)
// satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

// NewSQLiteRepository constructs the production MetricsRepository.
func NewSQLiteRepository(db SQLExecutor, clk schedule.Clock) MetricsRepository {
	return &sqliteRepository{db: db, clock: clk}
}

// Compile-time guarantee.
var _ MetricsRepository = (*sqliteRepository)(nil)

const (
	// metricsTimeFormat matches the wire format and the round-trip in
	// scanSample.
	metricsTimeFormat = time.RFC3339Nano

	metricsColumns = `id, ha_connections, request_errors, active_streams, smoothed_rtt_ms, scraped_at`

	insertMetricsSQL = `
INSERT INTO metrics (id, ha_connections, request_errors, active_streams, smoothed_rtt_ms, scraped_at)
VALUES (?, ?, ?, ?, ?, ?)
`
)

func (s *sqliteRepository) Store(ctx context.Context, m MetricsSample) (MetricsSample, error) {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	if m.ScrapedAt.IsZero() {
		m.ScrapedAt = s.clock.Now().UTC()
	}

	_, err := s.db.ExecContext(ctx, insertMetricsSQL,
		m.ID, m.HAConnections, m.RequestErrors, m.ActiveStreams, m.SmoothedRTTMS,
		m.ScrapedAt.Format(metricsTimeFormat),
	)
	if err != nil {
		return MetricsSample{}, fmt.Errorf("insert metrics %q: %w", m.ID, err)
	}
	if err := s.pruneBefore(ctx, m.ScrapedAt.Add(-MetricsRetentionWindow)); err != nil {
		return MetricsSample{}, err
	}
	return m, nil
}

func (s *sqliteRepository) pruneBefore(ctx context.Context, cutoff time.Time) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM metrics WHERE scraped_at < ?", cutoff.UTC().Format(metricsTimeFormat))
	if err != nil {
		return fmt.Errorf("prune metrics before %s: %w", cutoff.UTC().Format(metricsTimeFormat), err)
	}
	return nil
}

func (s *sqliteRepository) Query(ctx context.Context, from, to time.Time) ([]MetricsSample, error) {
	query := "SELECT " + metricsColumns + " FROM metrics"
	var (
		clauses []string
		args    []any
	)
	if !from.IsZero() {
		clauses = append(clauses, "scraped_at >= ?")
		args = append(args, from.UTC().Format(metricsTimeFormat))
	}
	if !to.IsZero() {
		clauses = append(clauses, "scraped_at <= ?")
		args = append(args, to.UTC().Format(metricsTimeFormat))
	}
	for i, c := range clauses {
		if i == 0 {
			query += " WHERE " + c
		} else {
			query += " AND " + c
		}
	}
	query += " ORDER BY scraped_at ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query metrics: %w", err)
	}
	defer rows.Close()

	var samples []MetricsSample
	for rows.Next() {
		m, err := scanSample(rows)
		if err != nil {
			return nil, fmt.Errorf("scan metrics: %w", err)
		}
		samples = append(samples, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate metrics: %w", err)
	}
	return samples, nil
}

func (s *sqliteRepository) Latest(ctx context.Context) (MetricsSample, error) {
	row := s.db.QueryRowContext(ctx, "SELECT "+metricsColumns+" FROM metrics ORDER BY scraped_at DESC LIMIT 1")
	m, err := scanSample(row)
	if errors.Is(err, sql.ErrNoRows) {
		return MetricsSample{}, ErrNoMetrics{}
	}
	if err != nil {
		return MetricsSample{}, fmt.Errorf("latest metrics: %w", err)
	}
	return m, nil
}

// rowScanner unifies *sql.Row and *sql.Rows under their common Scan surface so
// scanSample works for both single-row and multi-row reads.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanSample(sc rowScanner) (MetricsSample, error) {
	var (
		m         MetricsSample
		scrapeRaw string
	)
	if err := sc.Scan(&m.ID, &m.HAConnections, &m.RequestErrors, &m.ActiveStreams, &m.SmoothedRTTMS, &scrapeRaw); err != nil {
		return MetricsSample{}, err
	}
	scraped, err := time.Parse(metricsTimeFormat, scrapeRaw)
	if err != nil {
		return MetricsSample{}, fmt.Errorf("parse scraped_at %q: %w", scrapeRaw, err)
	}
	m.ScrapedAt = scraped
	return m, nil
}
