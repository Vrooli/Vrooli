package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

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

// GetLifestyleScore calculates the composite lifestyle score.
// [REQ:LD-UI-SCORE] Provides score data for dashboard display.
//
// Scoring algorithm (P0 simplified version):
// - Each active domain contributes equally (weight = 1/N where N = active domains)
// - Domain score = min(100, events_today * 20) -- each event adds 20 points, capped at 100
// - Composite = weighted average of domain scores
// - Data quality: "good" if >=3 domains with events, "limited" if 1-2, "insufficient" if 0
func (r *SQLiteStatsRepository) GetLifestyleScore(ctx context.Context, historyDays int) (*domain.ScoreResponse, error) {
	if historyDays <= 0 {
		historyDays = 7
	}

	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	// Get today's events by domain with domain display names
	domainScores, err := r.getTodayDomainScores(ctx, today)
	if err != nil {
		return nil, fmt.Errorf("get domain scores: %w", err)
	}

	// Calculate composite score
	var totalScore float64
	var totalWeight float64
	domainsWithData := 0
	for _, ds := range domainScores {
		if ds.EventCount > 0 {
			domainsWithData++
		}
		totalScore += float64(ds.Score) * ds.Weight
		totalWeight += ds.Weight
	}

	compositeScore := 0
	if totalWeight > 0 {
		compositeScore = int(totalScore / totalWeight)
	}

	// Determine data quality
	dataQuality := "insufficient"
	if domainsWithData >= 3 {
		dataQuality = "good"
	} else if domainsWithData >= 1 {
		dataQuality = "limited"
	}

	// Get yesterday's score for trend calculation
	yesterdayScore, err := r.calculateDayScore(ctx, yesterday)
	if err != nil {
		yesterdayScore = 0
	}

	// Determine trend
	changeFromYesterday := compositeScore - yesterdayScore
	trend := "stable"
	if changeFromYesterday > 5 {
		trend = "up"
	} else if changeFromYesterday < -5 {
		trend = "down"
	}

	// Generate message
	message := r.generateScoreMessage(compositeScore, trend, dataQuality, domainsWithData)

	// Get history
	history, err := r.getScoreHistory(ctx, historyDays)
	if err != nil {
		history = []domain.ScoreHistoryEntry{}
	}

	return &domain.ScoreResponse{
		Current: domain.LifestyleScore{
			Score:               compositeScore,
			Date:                today,
			DomainScores:        domainScores,
			Trend:               trend,
			ChangeFromYesterday: changeFromYesterday,
			DataQuality:         dataQuality,
			Message:             message,
		},
		History: history,
	}, nil
}

// scoreMultiplier is the points per event (20 points per event).
const scoreMultiplier = 20

// maxDomainScore is the cap per domain (100 max).
const maxDomainScore = 100

// calculateDomainScore converts event count to a capped score.
// Each event adds 20 points, capped at 100.
func calculateDomainScore(eventCount int) int {
	score := eventCount * scoreMultiplier
	if score > maxDomainScore {
		return maxDomainScore
	}
	return score
}

// getTodayDomainScores calculates per-domain scores for the given date.
func (r *SQLiteStatsRepository) getTodayDomainScores(ctx context.Context, date string) ([]domain.DomainScore, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			d.name,
			d.display_name,
			COALESCE(e.event_count, 0) as event_count
		FROM domains d
		LEFT JOIN (
			SELECT domain, count(*) as event_count
			FROM events
			WHERE date(timestamp) = ?
			GROUP BY domain
		) e ON d.name = e.domain
		WHERE d.status = 'active'
		ORDER BY d.name
	`, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scores []domain.DomainScore
	for rows.Next() {
		var ds domain.DomainScore
		if err := rows.Scan(&ds.Domain, &ds.DisplayName, &ds.EventCount); err != nil {
			continue
		}
		ds.Score = calculateDomainScore(ds.EventCount)
		scores = append(scores, ds)
	}

	// Set equal weights across all domains
	if len(scores) > 0 {
		weight := 1.0 / float64(len(scores))
		for i := range scores {
			scores[i].Weight = weight
		}
	}

	return scores, rows.Err()
}

// calculateDayScore computes the composite score for a specific date.
func (r *SQLiteStatsRepository) calculateDayScore(ctx context.Context, date string) (int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT domain, count(*) as event_count
		FROM events
		WHERE date(timestamp) = ?
		GROUP BY domain
	`, date)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var domainCount, totalScore int
	for rows.Next() {
		var domainName string
		var eventCount int
		if err := rows.Scan(&domainName, &eventCount); err != nil {
			continue
		}
		totalScore += calculateDomainScore(eventCount)
		domainCount++
	}

	if domainCount == 0 {
		return 0, nil
	}
	return totalScore / domainCount, rows.Err()
}

// getScoreHistory returns historical scores for the past N days.
func (r *SQLiteStatsRepository) getScoreHistory(ctx context.Context, days int) ([]domain.ScoreHistoryEntry, error) {
	history := make([]domain.ScoreHistoryEntry, 0, days)

	for i := days - 1; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		score, err := r.calculateDayScore(ctx, date)
		if err != nil {
			score = 0
		}
		history = append(history, domain.ScoreHistoryEntry{
			Date:  date,
			Score: score,
		})
	}

	return history, nil
}

// generateScoreMessage creates a human-readable summary of the score.
func (r *SQLiteStatsRepository) generateScoreMessage(score int, trend, dataQuality string, domainsWithData int) string {
	if dataQuality == "insufficient" {
		return "No activity recorded today. Start tracking to see your lifestyle score."
	}

	var qualityNote string
	if dataQuality == "limited" {
		qualityNote = fmt.Sprintf(" Based on %d domain(s).", domainsWithData)
	}

	var trendNote string
	switch trend {
	case "up":
		trendNote = " Trending up from yesterday!"
	case "down":
		trendNote = " Down from yesterday."
	default:
		trendNote = " Steady from yesterday."
	}

	var scoreNote string
	if score >= 80 {
		scoreNote = "Excellent day!"
	} else if score >= 60 {
		scoreNote = "Good progress today."
	} else if score >= 40 {
		scoreNote = "Moderate activity."
	} else {
		scoreNote = "Light activity today."
	}

	return scoreNote + trendNote + qualityNote
}
