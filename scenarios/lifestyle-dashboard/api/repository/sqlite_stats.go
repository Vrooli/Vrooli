package repository

import (
	"context"
	"database/sql"
	"strconv"

	"lifestyle-dashboard/domain"
)

// SQLiteStatsRepository implements StatsRepository for SQLite.
type SQLiteStatsRepository struct {
	db *sql.DB
}

// NewSQLiteStatsRepository creates a new SQLite stats repository.
func NewSQLiteStatsRepository(db *sql.DB) *SQLiteStatsRepository {
	return &SQLiteStatsRepository{db: db}
}

// GetTimeline returns event counts grouped by day and domain.
// [REQ:LD-QUERY-AGGREGATE] Aggregates events for timeline visualization.
func (r *SQLiteStatsRepository) GetTimeline(ctx context.Context, days int) ([]domain.TimelineEntry, error) {
	if days <= 0 {
		days = 7
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			date(timestamp) as day,
			domain,
			count(*) as event_count
		FROM events
		WHERE timestamp >= datetime('now', '-' || ? || ' days')
		GROUP BY date(timestamp), domain
		ORDER BY day DESC, domain
	`, strconv.Itoa(days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	timeline := []domain.TimelineEntry{}
	for rows.Next() {
		var entry domain.TimelineEntry
		if err := rows.Scan(&entry.Day, &entry.Domain, &entry.Count); err != nil {
			continue
		}
		timeline = append(timeline, entry)
	}

	return timeline, rows.Err()
}

// GetSummary returns aggregated statistics across all domains.
// [REQ:LD-QUERY-AGGREGATE] Provides dashboard-level statistics.
func (r *SQLiteStatsRepository) GetSummary(ctx context.Context) (*domain.SummaryResponse, error) {
	// Get counts by domain
	domainRows, err := r.db.QueryContext(ctx, `
		SELECT domain, count(*) as event_count
		FROM events
		GROUP BY domain
		ORDER BY event_count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer domainRows.Close()

	eventsByDomain := []domain.DomainCount{}
	for domainRows.Next() {
		var dc domain.DomainCount
		if err := domainRows.Scan(&dc.Domain, &dc.Count); err != nil {
			continue
		}
		eventsByDomain = append(eventsByDomain, dc)
	}

	// Get total counts
	var totalEvents int
	r.db.QueryRowContext(ctx, "SELECT count(*) FROM events").Scan(&totalEvents)

	var activeDomains int
	r.db.QueryRowContext(ctx, "SELECT count(*) FROM domains WHERE status = 'active'").Scan(&activeDomains)

	// Get recent activity
	var lastEventTime sql.NullString
	r.db.QueryRowContext(ctx, "SELECT max(timestamp) FROM events").Scan(&lastEventTime)

	return &domain.SummaryResponse{
		TotalEvents:    totalEvents,
		ActiveDomains:  activeDomains,
		EventsByDomain: eventsByDomain,
		LastEventAt:    lastEventTime.String,
	}, nil
}
