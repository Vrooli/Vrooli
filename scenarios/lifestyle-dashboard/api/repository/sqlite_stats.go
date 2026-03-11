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
// [REQ:LD-UI-SCORE] [REQ:LD-SCORE-CALC] Provides score data for dashboard display.
//
// Scoring algorithm (P1 version with configurable weights):
// - Domain weights are configurable: high (3x), medium (2x), low (1x), none (0x)
// - Domain score = min(100, events_today * 20) -- each event adds 20 points, capped at 100
// - Composite = weighted average of domain scores using configured multipliers
// - Data quality: determined by domain.DetermineDataQuality()
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

	// Determine data quality using domain decision helper
	dataQuality := domain.DetermineDataQuality(domainsWithData)

	// Get yesterday's score for trend calculation
	yesterdayScore, err := r.calculateDayScore(ctx, yesterday)
	if err != nil {
		yesterdayScore = 0
	}

	// Determine trend using domain decision helper
	changeFromYesterday := compositeScore - yesterdayScore
	trend := domain.DetermineDirection(float64(changeFromYesterday))

	// Generate message using domain decision helpers
	message := r.generateScoreMessage(compositeScore, string(trend), string(dataQuality), domainsWithData)

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
			Trend:               string(trend),
			ChangeFromYesterday: changeFromYesterday,
			DataQuality:         string(dataQuality),
			Message:             message,
		},
		History: history,
	}, nil
}

// calculateDomainScore converts event count to a capped score.
// Delegates to domain.CalculateDomainScore for centralized decision logic.
func calculateDomainScore(eventCount int) int {
	return domain.CalculateDomainScore(eventCount)
}

// getTodayDomainScores calculates per-domain scores for the given date.
// [REQ:LD-SCORE-CALC] Uses configurable domain weights from domain_weights table.
func (r *SQLiteStatsRepository) getTodayDomainScores(ctx context.Context, date string) ([]domain.DomainScore, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			d.name,
			d.display_name,
			COALESCE(e.event_count, 0) as event_count,
			COALESCE(w.weight, 'medium') as weight_label
		FROM domains d
		LEFT JOIN (
			SELECT domain, count(*) as event_count
			FROM events
			WHERE date(timestamp) = ?
			GROUP BY domain
		) e ON d.name = e.domain
		LEFT JOIN domain_weights w ON d.name = w.domain
		WHERE d.status = 'active'
		ORDER BY d.name
	`, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scores []domain.DomainScore
	var totalMultiplier float64
	for rows.Next() {
		var ds domain.DomainScore
		var weightLabel string
		if err := rows.Scan(&ds.Domain, &ds.DisplayName, &ds.EventCount, &weightLabel); err != nil {
			continue
		}
		ds.Score = calculateDomainScore(ds.EventCount)

		// Apply preset if no explicit weight is set
		if weightLabel == "medium" {
			if preset, ok := domain.WeightPresets[ds.Domain]; ok {
				weightLabel = preset
			}
		}

		// Get multiplier for this weight level
		multiplier := domain.WeightMultipliers[weightLabel]
		if multiplier == 0 {
			multiplier = 2.0 // Default to medium if unknown
		}
		ds.Weight = multiplier
		totalMultiplier += multiplier
		scores = append(scores, ds)
	}

	// Normalize weights to sum to 1.0 for the weighted average calculation
	if totalMultiplier > 0 {
		for i := range scores {
			scores[i].Weight = scores[i].Weight / totalMultiplier
		}
	}

	return scores, rows.Err()
}

// calculateDayScore computes the composite score for a specific date.
// [REQ:LD-SCORE-CALC] Uses configurable domain weights for historical consistency.
func (r *SQLiteStatsRepository) calculateDayScore(ctx context.Context, date string) (int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			e.domain,
			count(*) as event_count,
			COALESCE(w.weight, 'medium') as weight_label
		FROM events e
		LEFT JOIN domain_weights w ON e.domain = w.domain
		WHERE date(e.timestamp) = ?
		GROUP BY e.domain
	`, date)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var totalWeightedScore, totalMultiplier float64
	for rows.Next() {
		var domainName, weightLabel string
		var eventCount int
		if err := rows.Scan(&domainName, &eventCount, &weightLabel); err != nil {
			continue
		}

		// Apply preset if no explicit weight is set
		if weightLabel == "medium" {
			if preset, ok := domain.WeightPresets[domainName]; ok {
				weightLabel = preset
			}
		}

		// Get multiplier for this weight level
		multiplier := domain.WeightMultipliers[weightLabel]
		if multiplier == 0 {
			multiplier = 2.0 // Default to medium if unknown
		}

		score := calculateDomainScore(eventCount)
		totalWeightedScore += float64(score) * multiplier
		totalMultiplier += multiplier
	}

	if totalMultiplier == 0 {
		return 0, nil
	}
	return int(totalWeightedScore / totalMultiplier), rows.Err()
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
// Uses domain decision helpers for consistent messaging across the codebase.
func (r *SQLiteStatsRepository) generateScoreMessage(score int, trend, dataQuality string, domainsWithData int) string {
	if dataQuality == string(domain.DataQualityInsufficient) {
		return "No activity recorded today. Start tracking to see your lifestyle score."
	}

	var qualityNote string
	if dataQuality == string(domain.DataQualityLimited) {
		qualityNote = fmt.Sprintf(" Based on %d domain(s).", domainsWithData)
	}

	// Use domain helpers for consistent messaging
	trendNote := domain.TrendMessage(domain.Direction(trend))
	scoreLevel := domain.DetermineScoreLevel(score)
	scoreNote := domain.ScoreLevelMessage(scoreLevel)

	return scoreNote + trendNote + qualityNote
}
