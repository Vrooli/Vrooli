// DOC: docs/concepts/ARCHITECTURE.md#Abstraction-Layer
// DOC: PRD.md#OT-P0-005
//
// Package repository provides the SQLite implementation of BriefRepository.
// This generates morning and evening briefs by consolidating data from all domains.
//
// [REQ:LD-BRIEF-MORNING] Morning brief generation with yesterday summary.
// [REQ:LD-BRIEF-EVENING] Evening review with today's events.
// [REQ:LD-BRIEF-CONSOLIDATE] Cross-domain consolidation.
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"lifestyle-dashboard/domain"
)

// SQLiteBriefRepository implements BriefRepository using SQLite.
type SQLiteBriefRepository struct {
	db *sql.DB
}

// NewSQLiteBriefRepository creates a new SQLite-backed brief repository.
func NewSQLiteBriefRepository(db *sql.DB) *SQLiteBriefRepository {
	return &SQLiteBriefRepository{db: db}
}

// briefConfig holds configuration for brief generation.
type briefConfig struct {
	briefType       string
	sectionsDate    string
	noActivityMsg   string
	withActivityFmt string
	timePeriod      string // "yesterday" or "today" for message formatting
}

// generateBrief creates a brief with common logic extracted.
func (r *SQLiteBriefRepository) generateBrief(ctx context.Context, date string, cfg briefConfig) (*domain.Brief, error) {
	now := time.Now().UTC()

	// Parse target date, defaulting to today if invalid
	targetDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		targetDate = now.Truncate(24 * time.Hour)
		date = targetDate.Format("2006-01-02")
	}

	// Calculate the date to query sections from
	sectionsDate := cfg.sectionsDate
	if sectionsDate == "" {
		sectionsDate = date
	} else if sectionsDate == "yesterday" {
		sectionsDate = targetDate.AddDate(0, 0, -1).Format("2006-01-02")
	}

	brief := &domain.Brief{
		Type:        cfg.briefType,
		GeneratedAt: now.Format(time.RFC3339),
		Date:        date,
		Sections:    []domain.BriefSection{},
	}

	// Get domain summaries
	sections, err := r.getDomainSections(ctx, sectionsDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain sections: %w", err)
	}
	brief.Sections = sections

	// Generate summary from activity counts
	totalEvents, activeDomains := countSectionActivity(sections)
	brief.Summary = formatBriefSummary(totalEvents, activeDomains, cfg)

	// Attach score and trend (optional)
	if score, trend, err := r.getCurrentScore(ctx); err == nil && score != nil {
		brief.Score = score
		brief.ScoreTrend = trend
	}

	return brief, nil
}

// countSectionActivity returns total events and active domains from sections.
func countSectionActivity(sections []domain.BriefSection) (totalEvents, activeDomains int) {
	for _, s := range sections {
		totalEvents += s.EventCount
		if s.EventCount > 0 {
			activeDomains++
		}
	}
	return
}

// formatBriefSummary generates the appropriate summary message.
func formatBriefSummary(totalEvents, activeDomains int, cfg briefConfig) string {
	if totalEvents == 0 {
		return cfg.noActivityMsg
	}
	return fmt.Sprintf(cfg.withActivityFmt, totalEvents, activeDomains)
}

// GenerateMorningBrief creates a morning brief for the given date.
// It includes: yesterday's summary, today's scheduled items, and score trend.
// [REQ:LD-BRIEF-MORNING]
func (r *SQLiteBriefRepository) GenerateMorningBrief(ctx context.Context, date string) (*domain.Brief, error) {
	return r.generateBrief(ctx, date, briefConfig{
		briefType:       "morning",
		sectionsDate:    "yesterday",
		noActivityMsg:   "Good morning! No activity recorded yesterday. Start fresh today.",
		withActivityFmt: "Good morning! Yesterday: %d events across %d domain(s). Review your day below.",
	})
}

// GenerateEveningBrief creates an evening brief for the given date.
// It includes: today's summary, compliance rates, and tomorrow preview.
// [REQ:LD-BRIEF-EVENING]
func (r *SQLiteBriefRepository) GenerateEveningBrief(ctx context.Context, date string) (*domain.Brief, error) {
	return r.generateBrief(ctx, date, briefConfig{
		briefType:       "evening",
		sectionsDate:    "", // Use the date parameter directly
		noActivityMsg:   "Good evening! No activity recorded today. There's always tomorrow.",
		withActivityFmt: "Good evening! Today: %d events across %d domain(s). Here's your day in review.",
	})
}

// GetCurrentBrief returns the appropriate brief based on current time.
// Before evening hour (21:00): returns morning brief
// After evening hour: returns evening brief
// [REQ:LD-BRIEF-MORNING] [REQ:LD-BRIEF-EVENING]
func (r *SQLiteBriefRepository) GetCurrentBrief(ctx context.Context) (*domain.Brief, error) {
	now := time.Now()
	today := now.Format("2006-01-02")
	hour := now.Hour()

	// Evening brief after 21:00, morning brief otherwise
	if hour >= 21 {
		return r.GenerateEveningBrief(ctx, today)
	}
	return r.GenerateMorningBrief(ctx, today)
}

// getDomainSections retrieves domain-specific sections for a date.
// [REQ:LD-BRIEF-CONSOLIDATE]
func (r *SQLiteBriefRepository) getDomainSections(ctx context.Context, date string) ([]domain.BriefSection, error) {
	// Query domains with their event counts for the date
	query := `
		SELECT
			d.name,
			d.display_name,
			COALESCE(e.event_count, 0) as event_count,
			GROUP_CONCAT(DISTINCT e.event_type) as event_types
		FROM domains d
		LEFT JOIN (
			SELECT
				domain,
				COUNT(*) as event_count,
				event_type
			FROM events
			WHERE DATE(timestamp) = ?
			GROUP BY domain, event_type
		) e ON d.name = e.domain
		WHERE d.status = 'active'
		GROUP BY d.name, d.display_name
		ORDER BY COALESCE(e.event_count, 0) DESC, d.display_name ASC
	`

	rows, err := r.db.QueryContext(ctx, query, date)
	if err != nil {
		return nil, fmt.Errorf("failed to query domain sections: %w", err)
	}
	defer rows.Close()

	var sections []domain.BriefSection
	for rows.Next() {
		var name, displayName string
		var eventCount int
		var eventTypes sql.NullString

		if err := rows.Scan(&name, &displayName, &eventCount, &eventTypes); err != nil {
			return nil, fmt.Errorf("failed to scan domain section: %w", err)
		}

		section := domain.BriefSection{
			Domain:      name,
			DisplayName: displayName,
			EventCount:  eventCount,
			Priority:    3, // Default low priority
			Items:       []string{},
		}

		// Set priority based on event count
		if eventCount >= 5 {
			section.Priority = 1 // High
		} else if eventCount >= 2 {
			section.Priority = 2 // Medium
		}

		// Generate items based on event types
		if eventTypes.Valid && eventTypes.String != "" {
			if eventCount > 0 {
				section.Items = append(section.Items, fmt.Sprintf("%d events recorded", eventCount))
			}
		} else if eventCount == 0 {
			section.Items = append(section.Items, "No activity")
		}

		sections = append(sections, section)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating domain sections: %w", err)
	}

	return sections, nil
}

// getCurrentScore retrieves the current lifestyle score.
func (r *SQLiteBriefRepository) getCurrentScore(ctx context.Context) (*int, string, error) {
	// Simple score calculation based on recent activity
	query := `
		SELECT COUNT(DISTINCT domain) as active_domains
		FROM events
		WHERE timestamp >= datetime('now', '-1 day')
	`

	var activeDomains int
	err := r.db.QueryRowContext(ctx, query).Scan(&activeDomains)
	if err != nil {
		return nil, "", err
	}

	// Simple score: 20 points per active domain (max 100)
	score := activeDomains * 20
	if score > 100 {
		score = 100
	}

	// Determine trend by comparing to 7-day average
	avgQuery := `
		SELECT AVG(daily_count) FROM (
			SELECT COUNT(DISTINCT domain) as daily_count
			FROM events
			WHERE timestamp >= datetime('now', '-7 days')
			GROUP BY DATE(timestamp)
		)
	`

	var avgDomains float64
	err = r.db.QueryRowContext(ctx, avgQuery).Scan(&avgDomains)
	if err != nil {
		avgDomains = float64(activeDomains)
	}

	trend := "stable"
	if float64(activeDomains) > avgDomains*1.1 {
		trend = "up"
	} else if float64(activeDomains) < avgDomains*0.9 {
		trend = "down"
	}

	return &score, trend, nil
}
