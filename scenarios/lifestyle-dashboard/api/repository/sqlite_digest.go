// DOC: docs/concepts/ARCHITECTURE.md#Abstraction-Layer
// DOC: PRD.md#OT-P1-002
//
// Package repository provides the SQLite implementation of DigestRepository.
// This generates weekly digests by comparing current week activity to a rolling baseline.
//
// [REQ:LD-DIGEST-WEEKLY] Weekly digest generation with baseline comparison.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"lifestyle-dashboard/domain"
)

// SQLiteDigestRepository implements DigestRepository using SQLite.
type SQLiteDigestRepository struct {
	db *sql.DB
}

// NewSQLiteDigestRepository creates a new SQLite-backed digest repository.
func NewSQLiteDigestRepository(db *sql.DB) *SQLiteDigestRepository {
	return &SQLiteDigestRepository{db: db}
}

// GenerateWeeklyDigest creates a weekly digest comparing current week to 4-week baseline.
// weekStart should be the Monday of the week being summarized (format: YYYY-MM-DD).
// [REQ:LD-DIGEST-WEEKLY]
func (r *SQLiteDigestRepository) GenerateWeeklyDigest(ctx context.Context, weekStart string) (*domain.WeeklyDigest, error) {
	now := time.Now().UTC()

	// Parse week start date, defaulting to last Monday if invalid
	startDate, err := time.Parse("2006-01-02", weekStart)
	if err != nil {
		startDate = getLastMonday(now)
		weekStart = startDate.Format("2006-01-02")
	}

	weekEnd := startDate.AddDate(0, 0, 6) // Sunday
	baselineStart := startDate.AddDate(0, 0, -28) // 4 weeks before

	digest := &domain.WeeklyDigest{
		GeneratedAt:   now.Format(time.RFC3339),
		WeekStartDate: weekStart,
		WeekEndDate:   weekEnd.Format("2006-01-02"),
		Correlations:  []domain.DigestCorrelation{}, // Placeholder for correlation engine
	}

	// Get domain changes
	domainChanges, err := r.getDomainChanges(ctx, weekStart, weekEnd.Format("2006-01-02"), baselineStart.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("failed to get domain changes: %w", err)
	}
	digest.DomainChanges = domainChanges

	// Get score trend
	scoreTrend, err := r.getScoreTrend(ctx, weekStart, weekEnd.Format("2006-01-02"), baselineStart.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("failed to get score trend: %w", err)
	}
	digest.ScoreTrend = scoreTrend

	// Generate highlights
	digest.Highlights = r.generateHighlights(domainChanges, scoreTrend)

	// Generate next week focus
	digest.NextWeekFocus = r.generateNextWeekFocus(domainChanges, scoreTrend)

	// Generate summary
	digest.Summary = r.generateSummary(domainChanges, scoreTrend)

	return digest, nil
}

// GetLatestDigest returns the most recent weekly digest.
// Since digests are generated on-demand, this returns nil.
// [REQ:LD-DIGEST-WEEKLY]
func (r *SQLiteDigestRepository) GetLatestDigest(ctx context.Context) (*domain.WeeklyDigest, error) {
	// Generate digest for the most recent complete week
	now := time.Now().UTC()
	lastMonday := getLastMonday(now)

	// If we're before Sunday 6pm, use the previous week
	if now.Weekday() == time.Sunday && now.Hour() < 18 {
		lastMonday = lastMonday.AddDate(0, 0, -7)
	} else if now.Weekday() != time.Sunday {
		lastMonday = lastMonday.AddDate(0, 0, -7)
	}

	return r.GenerateWeeklyDigest(ctx, lastMonday.Format("2006-01-02"))
}

// getDomainChanges calculates activity changes for each domain.
func (r *SQLiteDigestRepository) getDomainChanges(ctx context.Context, weekStart, weekEnd, baselineStart string) ([]domain.DigestDomainChange, error) {
	query := `
		WITH current_week AS (
			SELECT
				d.name,
				d.display_name,
				COALESCE(COUNT(e.id), 0) as event_count
			FROM domains d
			LEFT JOIN events e ON d.name = e.domain
				AND DATE(e.timestamp) >= ?
				AND DATE(e.timestamp) <= ?
			WHERE d.status = 'active'
			GROUP BY d.name, d.display_name
		),
		baseline AS (
			SELECT
				d.name,
				COALESCE(COUNT(e.id) / 4.0, 0) as avg_events
			FROM domains d
			LEFT JOIN events e ON d.name = e.domain
				AND DATE(e.timestamp) >= ?
				AND DATE(e.timestamp) < ?
			WHERE d.status = 'active'
			GROUP BY d.name
		)
		SELECT
			cw.name,
			cw.display_name,
			cw.event_count,
			COALESCE(b.avg_events, 0) as baseline_avg
		FROM current_week cw
		LEFT JOIN baseline b ON cw.name = b.name
		ORDER BY cw.event_count DESC
	`

	rows, err := r.db.QueryContext(ctx, query, weekStart, weekEnd, baselineStart, weekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to query domain changes: %w", err)
	}
	defer rows.Close()

	var changes []domain.DigestDomainChange
	for rows.Next() {
		var name, displayName string
		var eventCount int
		var baselineAvg float64

		if err := rows.Scan(&name, &displayName, &eventCount, &baselineAvg); err != nil {
			return nil, fmt.Errorf("failed to scan domain change: %w", err)
		}

		change := domain.DigestDomainChange{
			Domain:            name,
			DisplayName:       displayName,
			CurrentWeekEvents: eventCount,
			BaselineAvgEvents: baselineAvg,
		}

		// Calculate percent change using centralized decision helper
		change.PercentChange = domain.CalculatePercentChangeInt(eventCount, baselineAvg)

		// Determine direction using centralized decision helper (domain threshold)
		change.Direction = string(domain.DetermineDomainDirection(change.PercentChange))

		// Mark notable changes using centralized decision helper
		change.Notable = domain.IsNotableChange(change.PercentChange)

		// Generate message
		change.Message = r.generateDomainChangeMessage(change)

		changes = append(changes, change)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating domain changes: %w", err)
	}

	return changes, nil
}

// getScoreTrend calculates lifestyle score trend compared to baseline.
func (r *SQLiteDigestRepository) getScoreTrend(ctx context.Context, weekStart, weekEnd, baselineStart string) (domain.DigestScoreTrend, error) {
	// Calculate current week average score (based on domain activity)
	currentQuery := `
		SELECT COALESCE(AVG(daily_score), 0) FROM (
			SELECT
				DATE(timestamp) as day,
				MIN(COUNT(DISTINCT domain) * 20, 100) as daily_score
			FROM events
			WHERE DATE(timestamp) >= ? AND DATE(timestamp) <= ?
			GROUP BY DATE(timestamp)
		)
	`

	var currentAvg float64
	err := r.db.QueryRowContext(ctx, currentQuery, weekStart, weekEnd).Scan(&currentAvg)
	if err != nil && err != sql.ErrNoRows {
		return domain.DigestScoreTrend{}, fmt.Errorf("failed to get current week score: %w", err)
	}

	// Calculate baseline average score
	baselineQuery := `
		SELECT COALESCE(AVG(daily_score), 0) FROM (
			SELECT
				DATE(timestamp) as day,
				MIN(COUNT(DISTINCT domain) * 20, 100) as daily_score
			FROM events
			WHERE DATE(timestamp) >= ? AND DATE(timestamp) < ?
			GROUP BY DATE(timestamp)
		)
	`

	var baselineAvg float64
	err = r.db.QueryRowContext(ctx, baselineQuery, baselineStart, weekStart).Scan(&baselineAvg)
	if err != nil && err != sql.ErrNoRows {
		return domain.DigestScoreTrend{}, fmt.Errorf("failed to get baseline score: %w", err)
	}

	trend := domain.DigestScoreTrend{
		CurrentWeekAvg: currentAvg,
		BaselineAvg:    baselineAvg,
	}

	// Calculate percent change using centralized decision helper
	trend.PercentChange = domain.CalculatePercentChange(currentAvg, baselineAvg)

	// Determine direction using centralized decision helper (score threshold)
	trend.Direction = string(domain.DetermineDirection(trend.PercentChange))

	// Generate message
	trend.Message = r.generateScoreTrendMessage(trend)

	return trend, nil
}

// generateDomainChangeMessage creates a human-readable message for domain change.
func (r *SQLiteDigestRepository) generateDomainChangeMessage(change domain.DigestDomainChange) string {
	if change.CurrentWeekEvents == 0 && change.BaselineAvgEvents == 0 {
		return fmt.Sprintf("No activity in %s", change.DisplayName)
	}

	if change.BaselineAvgEvents == 0 {
		return fmt.Sprintf("Started tracking %s with %d events", change.DisplayName, change.CurrentWeekEvents)
	}

	switch change.Direction {
	case "up":
		return fmt.Sprintf("%s activity up %.0f%% (%d events vs %.1f avg)",
			change.DisplayName, change.PercentChange, change.CurrentWeekEvents, change.BaselineAvgEvents)
	case "down":
		return fmt.Sprintf("%s activity down %.0f%% (%d events vs %.1f avg)",
			change.DisplayName, math.Abs(change.PercentChange), change.CurrentWeekEvents, change.BaselineAvgEvents)
	default:
		return fmt.Sprintf("%s activity steady (%d events)", change.DisplayName, change.CurrentWeekEvents)
	}
}

// generateScoreTrendMessage creates a human-readable message for score trend.
func (r *SQLiteDigestRepository) generateScoreTrendMessage(trend domain.DigestScoreTrend) string {
	if trend.CurrentWeekAvg == 0 && trend.BaselineAvg == 0 {
		return "No activity data to calculate lifestyle score"
	}

	if trend.BaselineAvg == 0 {
		return fmt.Sprintf("First week with lifestyle data! Average score: %.0f", trend.CurrentWeekAvg)
	}

	switch trend.Direction {
	case "up":
		return fmt.Sprintf("Lifestyle score improved %.0f%% (%.0f avg vs %.0f baseline)",
			trend.PercentChange, trend.CurrentWeekAvg, trend.BaselineAvg)
	case "down":
		return fmt.Sprintf("Lifestyle score decreased %.0f%% (%.0f avg vs %.0f baseline)",
			math.Abs(trend.PercentChange), trend.CurrentWeekAvg, trend.BaselineAvg)
	default:
		return fmt.Sprintf("Lifestyle score stable at %.0f avg", trend.CurrentWeekAvg)
	}
}

// generateHighlights creates notable achievements and concerns.
// Uses centralized decision helpers for consistent highlight criteria.
func (r *SQLiteDigestRepository) generateHighlights(changes []domain.DigestDomainChange, score domain.DigestScoreTrend) []string {
	var highlights []string

	// Add notable domain changes using centralized decision helper
	for _, change := range changes {
		shouldHighlight, highlightType := domain.ShouldHighlightDomainChange(change.PercentChange, domain.Direction(change.Direction))
		if shouldHighlight {
			if highlightType == domain.HighlightTypePositive {
				highlights = append(highlights, fmt.Sprintf("🎉 %s activity increased significantly (+%.0f%%)", change.DisplayName, change.PercentChange))
			} else if highlightType == domain.HighlightTypeWarning {
				highlights = append(highlights, fmt.Sprintf("⚠️ %s activity dropped (%.0f%%)", change.DisplayName, change.PercentChange))
			}
		}
	}

	// Add score highlights using centralized decision helper
	if domain.ShouldHighlightScoreImprovement(score.PercentChange, domain.Direction(score.Direction)) {
		highlights = append(highlights, fmt.Sprintf("🏆 Great week! Lifestyle score up %.0f%%", score.PercentChange))
	}

	// Add new tracking highlight
	for _, change := range changes {
		if change.BaselineAvgEvents == 0 && change.CurrentWeekEvents > 0 {
			highlights = append(highlights, fmt.Sprintf("🆕 Started tracking %s this week", change.DisplayName))
		}
	}

	if len(highlights) == 0 {
		highlights = append(highlights, "📊 Steady week with consistent activity")
	}

	return highlights
}

// generateNextWeekFocus creates recommendations for next week.
// Uses centralized decision helpers for consistent recommendation criteria.
func (r *SQLiteDigestRepository) generateNextWeekFocus(changes []domain.DigestDomainChange, score domain.DigestScoreTrend) []string {
	var focus []string

	// Generate focus recommendations using centralized decision helper
	for _, change := range changes {
		if recommendation, shouldAdd := domain.GenerateFocusRecommendation(
			change.DisplayName,
			change.PercentChange,
			domain.Direction(change.Direction),
		); shouldAdd {
			focus = append(focus, recommendation)
		}
	}

	// General recommendations based on score direction
	if domain.Direction(score.Direction) == domain.DirectionDown {
		focus = append(focus, "Review which domains need more attention")
	}

	if len(focus) == 0 {
		focus = append(focus, "Maintain current habits and track consistently")
	}

	return focus
}

// generateSummary creates the overall digest summary.
func (r *SQLiteDigestRepository) generateSummary(changes []domain.DigestDomainChange, score domain.DigestScoreTrend) string {
	activeCount := 0
	upCount := 0
	downCount := 0

	for _, change := range changes {
		if change.CurrentWeekEvents > 0 {
			activeCount++
		}
		if change.Notable && change.Direction == "up" {
			upCount++
		}
		if change.Notable && change.Direction == "down" {
			downCount++
		}
	}

	if activeCount == 0 {
		return "No activity recorded this week. Start tracking to see your weekly summary."
	}

	summary := fmt.Sprintf("This week: %d active domain(s)", activeCount)

	if upCount > 0 || downCount > 0 {
		if upCount > 0 && downCount > 0 {
			summary += fmt.Sprintf(", %d improving, %d declining", upCount, downCount)
		} else if upCount > 0 {
			summary += fmt.Sprintf(", %d improving", upCount)
		} else {
			summary += fmt.Sprintf(", %d declining", downCount)
		}
	}

	summary += ". " + score.Message

	return summary
}

// getLastMonday returns the most recent Monday (including today if Monday).
func getLastMonday(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday is 7, not 0
	}
	daysToSubtract := weekday - 1 // Days since Monday
	return t.AddDate(0, 0, -daysToSubtract).Truncate(24 * time.Hour)
}
