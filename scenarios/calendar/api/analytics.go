package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	_ "github.com/lib/pq"
)

type AnalyticsManager struct {
	db *sql.DB
}

func NewAnalyticsManager(db *sql.DB) *AnalyticsManager {
	return &AnalyticsManager{db: db}
}

type ScheduleAnalytics struct {
	Overview            OverviewStats            `json:"overview"`
	TimePatterns        TimePatternAnalysis      `json:"time_patterns"`
	ProductivityMetrics ProductivityMetrics      `json:"productivity_metrics"`
	EventTypeBreakdown  EventTypeBreakdown       `json:"event_type_breakdown"`
	ConflictAnalysis    ConflictAnalytics        `json:"conflict_analysis"`
	Recommendations     []ScheduleRecommendation `json:"recommendations"`
}

type OverviewStats struct {
	TotalEvents        int     `json:"total_events"`
	UpcomingEvents     int     `json:"upcoming_events"`
	CompletedEvents    int     `json:"completed_events"`
	AverageEventLength float64 `json:"average_event_length_minutes"`
	TotalMeetingHours  float64 `json:"total_meeting_hours"`
	BusiestDay         string  `json:"busiest_day"`
	MostCommonType     string  `json:"most_common_event_type"`
}

type TimePatternAnalysis struct {
	PeakHours             []HourActivity     `json:"peak_hours"`
	DayOfWeekDistribution map[string]int     `json:"day_of_week_distribution"`
	MorningVsAfternoon    map[string]int     `json:"morning_vs_afternoon"`
	RecurringPatterns     []RecurringPattern `json:"recurring_patterns"`
}

type HourActivity struct {
	Hour       int     `json:"hour"`
	EventCount int     `json:"event_count"`
	Percentage float64 `json:"percentage"`
}

type RecurringPattern struct {
	Title          string `json:"title"`
	Frequency      string `json:"frequency"`
	Count          int    `json:"count"`
	NextOccurrence string `json:"next_occurrence,omitempty"`
}

type ProductivityMetrics struct {
	FocusTimeRatio     float64 `json:"focus_time_ratio"`
	MeetingDensity     float64 `json:"meeting_density"`
	AverageGapBetween  float64 `json:"average_gap_between_meetings_minutes"`
	LongestMeetingFree string  `json:"longest_meeting_free_period"`
	BackToBackCount    int     `json:"back_to_back_meetings_count"`
	OvertimeHours      float64 `json:"overtime_hours"`
}

type EventTypeBreakdown struct {
	Distribution map[string]EventTypeStats `json:"distribution"`
	Trends       []EventTypeTrend          `json:"trends"`
}

type EventTypeStats struct {
	Count       int     `json:"count"`
	Percentage  float64 `json:"percentage"`
	AvgDuration float64 `json:"avg_duration_minutes"`
	TotalHours  float64 `json:"total_hours"`
}

type EventTypeTrend struct {
	Type      string  `json:"type"`
	Direction string  `json:"direction"`
	Change    float64 `json:"change_percentage"`
}

type ConflictAnalytics struct {
	TotalConflicts      int      `json:"total_conflicts"`
	ConflictRate        float64  `json:"conflict_rate"`
	MostConflictedSlots []string `json:"most_conflicted_time_slots"`
	ResolutionRate      float64  `json:"resolution_rate"`
}

type ScheduleRecommendation struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Impact      string  `json:"impact"`
	Priority    string  `json:"priority"`
	Action      string  `json:"suggested_action"`
	Confidence  float64 `json:"confidence"`
}

func (am *AnalyticsManager) GetScheduleAnalytics(userID string, startDate, endDate time.Time) (*ScheduleAnalytics, error) {
	analytics := &ScheduleAnalytics{}

	// Get overview stats
	overview, err := am.getOverviewStats(userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get overview stats: %v", err)
	}
	analytics.Overview = *overview

	// Get time patterns
	timePatterns, err := am.getTimePatterns(userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get time patterns: %v", err)
	}
	analytics.TimePatterns = *timePatterns

	// Get productivity metrics
	productivity, err := am.getProductivityMetrics(userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get productivity metrics: %v", err)
	}
	analytics.ProductivityMetrics = *productivity

	// Get event type breakdown
	breakdown, err := am.getEventTypeBreakdown(userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get event type breakdown: %v", err)
	}
	analytics.EventTypeBreakdown = *breakdown

	// Get conflict analysis
	conflicts, err := am.getConflictAnalysis(userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get conflict analysis: %v", err)
	}
	analytics.ConflictAnalysis = *conflicts

	// Generate recommendations
	recommendations := am.generateRecommendations(analytics)
	analytics.Recommendations = recommendations

	return analytics, nil
}

func (am *AnalyticsManager) getOverviewStats(userID string, startDate, endDate time.Time) (*OverviewStats, error) {
	overview := &OverviewStats{}

	// Get total and upcoming events
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN start_time > NOW() THEN 1 END) as upcoming,
			COUNT(CASE WHEN end_time < NOW() THEN 1 END) as completed,
			AVG(EXTRACT(EPOCH FROM (end_time - start_time))/60) as avg_length,
			SUM(EXTRACT(EPOCH FROM (end_time - start_time))/3600) as total_hours
		FROM events 
		WHERE user_id = $1 AND start_time BETWEEN $2 AND $3`

	err := am.db.QueryRow(query, userID, startDate, endDate).Scan(
		&overview.TotalEvents,
		&overview.UpcomingEvents,
		&overview.CompletedEvents,
		&overview.AverageEventLength,
		&overview.TotalMeetingHours,
	)
	if err != nil {
		return nil, err
	}

	// Get busiest day
	dayQuery := `
		SELECT TO_CHAR(start_time, 'Day'), COUNT(*)
		FROM events
		WHERE user_id = $1 AND start_time BETWEEN $2 AND $3
		GROUP BY TO_CHAR(start_time, 'Day')
		ORDER BY COUNT(*) DESC
		LIMIT 1`

	var busiestDay sql.NullString
	var dayCount int
	err = am.db.QueryRow(dayQuery, userID, startDate, endDate).Scan(&busiestDay, &dayCount)
	if err == nil && busiestDay.Valid {
		overview.BusiestDay = busiestDay.String
	}

	// Get most common event type
	typeQuery := `
		SELECT event_type, COUNT(*)
		FROM events
		WHERE user_id = $1 AND start_time BETWEEN $2 AND $3 AND event_type != ''
		GROUP BY event_type
		ORDER BY COUNT(*) DESC
		LIMIT 1`

	var eventType sql.NullString
	var typeCount int
	err = am.db.QueryRow(typeQuery, userID, startDate, endDate).Scan(&eventType, &typeCount)
	if err == nil && eventType.Valid {
		overview.MostCommonType = eventType.String
	}

	return overview, nil
}

func (am *AnalyticsManager) getTimePatterns(userID string, startDate, endDate time.Time) (*TimePatternAnalysis, error) {
	patterns := &TimePatternAnalysis{
		DayOfWeekDistribution: make(map[string]int),
		MorningVsAfternoon:    make(map[string]int),
	}

	// Get peak hours
	hourQuery := `
		SELECT EXTRACT(HOUR FROM start_time) as hour, COUNT(*) as count
		FROM events
		WHERE user_id = $1 AND start_time BETWEEN $2 AND $3
		GROUP BY hour
		ORDER BY count DESC`

	rows, err := am.db.Query(hourQuery, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	totalEvents := 0
	var hourActivities []HourActivity
	for rows.Next() {
		var hour, count int
		if err := rows.Scan(&hour, &count); err != nil {
			continue
		}
		hourActivities = append(hourActivities, HourActivity{
			Hour:       hour,
			EventCount: count,
		})
		totalEvents += count
	}

	// Calculate percentages
	for i := range hourActivities {
		if totalEvents > 0 {
			hourActivities[i].Percentage = float64(hourActivities[i].EventCount) / float64(totalEvents) * 100
		}
	}
	patterns.PeakHours = hourActivities

	// Get day of week distribution
	dayQuery := `
		SELECT TO_CHAR(start_time, 'Day'), COUNT(*)
		FROM events
		WHERE user_id = $1 AND start_time BETWEEN $2 AND $3
		GROUP BY TO_CHAR(start_time, 'Day')`

	rows, err = am.db.Query(dayQuery, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var day string
		var count int
		if err := rows.Scan(&day, &count); err != nil {
			continue
		}
		patterns.DayOfWeekDistribution[day] = count
	}

	// Morning vs Afternoon analysis
	morningQuery := `
		SELECT 
			CASE 
				WHEN EXTRACT(HOUR FROM start_time) < 12 THEN 'Morning'
				WHEN EXTRACT(HOUR FROM start_time) < 17 THEN 'Afternoon'
				ELSE 'Evening'
			END as period,
			COUNT(*)
		FROM events
		WHERE user_id = $1 AND start_time BETWEEN $2 AND $3
		GROUP BY period`

	rows, err = am.db.Query(morningQuery, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var period string
		var count int
		if err := rows.Scan(&period, &count); err != nil {
			continue
		}
		patterns.MorningVsAfternoon[period] = count
	}

	// Get recurring patterns
	recurQuery := `
		SELECT DISTINCT title, COUNT(*) as count
		FROM events
		WHERE user_id = $1 AND start_time BETWEEN $2 AND $3
		GROUP BY title
		HAVING COUNT(*) > 2
		ORDER BY count DESC
		LIMIT 5`

	rows, err = am.db.Query(recurQuery, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var title string
		var count int
		if err := rows.Scan(&title, &count); err != nil {
			continue
		}
		patterns.RecurringPatterns = append(patterns.RecurringPatterns, RecurringPattern{
			Title:     title,
			Count:     count,
			Frequency: am.detectFrequency(userID, title),
		})
	}

	return patterns, nil
}

func (am *AnalyticsManager) getProductivityMetrics(userID string, startDate, endDate time.Time) (*ProductivityMetrics, error) {
	metrics := &ProductivityMetrics{}

	// Calculate focus time ratio (time without meetings vs total work time)
	workHoursPerDay := 8.0
	totalWorkDays := int(endDate.Sub(startDate).Hours() / 24)
	totalWorkHours := float64(totalWorkDays) * workHoursPerDay

	var totalMeetingHours float64
	query := `SELECT COALESCE(SUM(EXTRACT(EPOCH FROM (end_time - start_time))/3600), 0) FROM events WHERE user_id = $1 AND start_time BETWEEN $2 AND $3`
	am.db.QueryRow(query, userID, startDate, endDate).Scan(&totalMeetingHours)

	if totalWorkHours > 0 {
		metrics.FocusTimeRatio = (totalWorkHours - totalMeetingHours) / totalWorkHours
		metrics.MeetingDensity = totalMeetingHours / totalWorkHours
	}

	// Calculate average gap between meetings
	gapQuery := `
		WITH ordered_events AS (
			SELECT start_time, end_time,
				   LEAD(start_time) OVER (ORDER BY start_time) as next_start
			FROM events
			WHERE user_id = $1 AND start_time BETWEEN $2 AND $3
		)
		SELECT AVG(EXTRACT(EPOCH FROM (next_start - end_time))/60)
		FROM ordered_events
		WHERE next_start IS NOT NULL`

	am.db.QueryRow(gapQuery, userID, startDate, endDate).Scan(&metrics.AverageGapBetween)

	// Count back-to-back meetings (less than 15 minutes gap)
	backToBackQuery := `
		WITH ordered_events AS (
			SELECT start_time, end_time,
				   LEAD(start_time) OVER (ORDER BY start_time) as next_start
			FROM events
			WHERE user_id = $1 AND start_time BETWEEN $2 AND $3
		)
		SELECT COUNT(*)
		FROM ordered_events
		WHERE next_start IS NOT NULL 
		AND EXTRACT(EPOCH FROM (next_start - end_time))/60 < 15`

	am.db.QueryRow(backToBackQuery, userID, startDate, endDate).Scan(&metrics.BackToBackCount)

	// Calculate overtime (meetings outside 9-5)
	overtimeQuery := `
		SELECT COALESCE(SUM(
			CASE 
				WHEN EXTRACT(HOUR FROM start_time) < 9 OR EXTRACT(HOUR FROM end_time) > 17 
				THEN EXTRACT(EPOCH FROM (end_time - start_time))/3600
				ELSE 0
			END
		), 0)
		FROM events
		WHERE user_id = $1 AND start_time BETWEEN $2 AND $3`

	am.db.QueryRow(overtimeQuery, userID, startDate, endDate).Scan(&metrics.OvertimeHours)

	return metrics, nil
}

func (am *AnalyticsManager) getEventTypeBreakdown(userID string, startDate, endDate time.Time) (*EventTypeBreakdown, error) {
	breakdown := &EventTypeBreakdown{
		Distribution: make(map[string]EventTypeStats),
	}

	// Get distribution by type
	query := `
		SELECT 
			COALESCE(event_type, 'unspecified') as type,
			COUNT(*) as count,
			AVG(EXTRACT(EPOCH FROM (end_time - start_time))/60) as avg_duration,
			SUM(EXTRACT(EPOCH FROM (end_time - start_time))/3600) as total_hours
		FROM events
		WHERE user_id = $1 AND start_time BETWEEN $2 AND $3
		GROUP BY event_type`

	rows, err := am.db.Query(query, userID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	totalEvents := 0
	for rows.Next() {
		var eventType string
		var stats EventTypeStats
		if err := rows.Scan(&eventType, &stats.Count, &stats.AvgDuration, &stats.TotalHours); err != nil {
			continue
		}
		breakdown.Distribution[eventType] = stats
		totalEvents += stats.Count
	}

	// Calculate percentages
	for k, v := range breakdown.Distribution {
		if totalEvents > 0 {
			v.Percentage = float64(v.Count) / float64(totalEvents) * 100
			breakdown.Distribution[k] = v
		}
	}

	// Calculate trends (comparing last 30 days to previous 30 days)
	midPoint := startDate.Add(endDate.Sub(startDate) / 2)
	for eventType := range breakdown.Distribution {
		var countFirst, countSecond int

		am.db.QueryRow(`
			SELECT COUNT(*) FROM events 
			WHERE user_id = $1 AND event_type = $2 AND start_time BETWEEN $3 AND $4`,
			userID, eventType, startDate, midPoint).Scan(&countFirst)

		am.db.QueryRow(`
			SELECT COUNT(*) FROM events 
			WHERE user_id = $1 AND event_type = $2 AND start_time BETWEEN $3 AND $4`,
			userID, eventType, midPoint, endDate).Scan(&countSecond)

		if countFirst > 0 {
			change := float64(countSecond-countFirst) / float64(countFirst) * 100
			direction := "stable"
			if change > 10 {
				direction = "increasing"
			} else if change < -10 {
				direction = "decreasing"
			}

			breakdown.Trends = append(breakdown.Trends, EventTypeTrend{
				Type:      eventType,
				Direction: direction,
				Change:    change,
			})
		}
	}

	return breakdown, nil
}

func (am *AnalyticsManager) getConflictAnalysis(userID string, startDate, endDate time.Time) (*ConflictAnalytics, error) {
	conflicts := &ConflictAnalytics{}

	// Count overlapping events
	conflictQuery := `
		SELECT COUNT(*)
		FROM events e1
		JOIN events e2 ON e1.user_id = e2.user_id
		WHERE e1.user_id = $1 
		AND e1.id != e2.id
		AND e1.start_time BETWEEN $2 AND $3
		AND e1.start_time < e2.end_time 
		AND e1.end_time > e2.start_time`

	am.db.QueryRow(conflictQuery, userID, startDate, endDate).Scan(&conflicts.TotalConflicts)

	// Calculate conflict rate
	var totalEvents int
	am.db.QueryRow(`SELECT COUNT(*) FROM events WHERE user_id = $1 AND start_time BETWEEN $2 AND $3`,
		userID, startDate, endDate).Scan(&totalEvents)

	if totalEvents > 0 {
		conflicts.ConflictRate = float64(conflicts.TotalConflicts) / float64(totalEvents) * 100
	}

	// Find most conflicted time slots
	slotQuery := `
		SELECT TO_CHAR(start_time, 'Day HH24:00'), COUNT(*)
		FROM events
		WHERE user_id = $1 AND start_time BETWEEN $2 AND $3
		GROUP BY TO_CHAR(start_time, 'Day HH24:00')
		HAVING COUNT(*) > 1
		ORDER BY COUNT(*) DESC
		LIMIT 3`

	rows, err := am.db.Query(slotQuery, userID, startDate, endDate)
	if err != nil {
		return conflicts, nil
	}
	defer rows.Close()

	for rows.Next() {
		var slot string
		var count int
		if err := rows.Scan(&slot, &count); err != nil {
			continue
		}
		conflicts.MostConflictedSlots = append(conflicts.MostConflictedSlots, fmt.Sprintf("%s (%d conflicts)", slot, count))
	}

	return conflicts, nil
}

func (am *AnalyticsManager) detectFrequency(userID, title string) string {
	// Simple frequency detection based on event intervals
	var avgDays float64
	query := `
		WITH event_dates AS (
			SELECT start_time,
				   LAG(start_time) OVER (ORDER BY start_time) as prev_start
			FROM events
			WHERE user_id = $1 AND title = $2
			ORDER BY start_time
		)
		SELECT AVG(EXTRACT(EPOCH FROM (start_time - prev_start))/86400)
		FROM event_dates
		WHERE prev_start IS NOT NULL`

	am.db.QueryRow(query, userID, title).Scan(&avgDays)

	switch {
	case avgDays < 2:
		return "Daily"
	case avgDays < 9:
		return "Weekly"
	case avgDays < 16:
		return "Bi-weekly"
	case avgDays < 35:
		return "Monthly"
	default:
		return "Irregular"
	}
}

func (am *AnalyticsManager) generateRecommendations(analytics *ScheduleAnalytics) []ScheduleRecommendation {
	var recommendations []ScheduleRecommendation

	// Check meeting density
	if analytics.ProductivityMetrics.MeetingDensity > 0.7 {
		recommendations = append(recommendations, ScheduleRecommendation{
			Type:        "meeting_overload",
			Description: "Your calendar is 70%+ meetings. Consider blocking focus time.",
			Impact:      "Improve deep work and reduce meeting fatigue",
			Priority:    "high",
			Action:      "Block 2-hour focus sessions each day",
			Confidence:  0.9,
		})
	}

	// Check back-to-back meetings
	if analytics.ProductivityMetrics.BackToBackCount > 5 {
		recommendations = append(recommendations, ScheduleRecommendation{
			Type:        "back_to_back_meetings",
			Description: fmt.Sprintf("You have %d back-to-back meetings. Add buffer time.", analytics.ProductivityMetrics.BackToBackCount),
			Impact:      "Reduce stress and improve meeting effectiveness",
			Priority:    "medium",
			Action:      "Add 15-minute buffers between meetings",
			Confidence:  0.85,
		})
	}

	// Check overtime
	if analytics.ProductivityMetrics.OvertimeHours > 10 {
		recommendations = append(recommendations, ScheduleRecommendation{
			Type:        "work_life_balance",
			Description: fmt.Sprintf("%.1f hours of meetings outside work hours", analytics.ProductivityMetrics.OvertimeHours),
			Impact:      "Improve work-life balance and prevent burnout",
			Priority:    "high",
			Action:      "Decline or reschedule after-hours meetings",
			Confidence:  0.8,
		})
	}

	// Check conflict rate
	if analytics.ConflictAnalysis.ConflictRate > 5 {
		recommendations = append(recommendations, ScheduleRecommendation{
			Type:        "scheduling_conflicts",
			Description: fmt.Sprintf("%.1f%% of your events have conflicts", analytics.ConflictAnalysis.ConflictRate),
			Impact:      "Reduce double-booking and improve reliability",
			Priority:    "high",
			Action:      "Review and resolve scheduling conflicts",
			Confidence:  0.95,
		})
	}

	// Check for optimal meeting times based on patterns
	if len(analytics.TimePatterns.PeakHours) > 0 {
		peakHour := analytics.TimePatterns.PeakHours[0].Hour
		recommendations = append(recommendations, ScheduleRecommendation{
			Type:        "optimal_scheduling",
			Description: fmt.Sprintf("Your peak meeting time is %d:00. Schedule important meetings then.", peakHour),
			Impact:      "Maximize energy and engagement levels",
			Priority:    "low",
			Action:      fmt.Sprintf("Schedule high-priority meetings at %d:00", peakHour),
			Confidence:  0.7,
		})
	}

	// Check meeting length optimization
	if analytics.Overview.AverageEventLength > 60 {
		recommendations = append(recommendations, ScheduleRecommendation{
			Type:        "meeting_efficiency",
			Description: "Average meeting length exceeds 1 hour. Consider shorter, focused meetings.",
			Impact:      "Increase productivity and reduce meeting fatigue",
			Priority:    "medium",
			Action:      "Default to 30-minute meetings with clear agendas",
			Confidence:  0.75,
		})
	}

	return recommendations
}

// HTTP Handler for analytics endpoint
func (am *AnalyticsManager) HandleAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user ID from context
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "test-user" // Default for testing
	}

	// Parse date range from query params
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")

	var startDate, endDate time.Time
	var err error

	if startDateStr != "" {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			http.Error(w, "Invalid start_date format", http.StatusBadRequest)
			return
		}
	} else {
		// Default to last 30 days
		startDate = time.Now().AddDate(0, 0, -30)
	}

	if endDateStr != "" {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			http.Error(w, "Invalid end_date format", http.StatusBadRequest)
			return
		}
	} else {
		endDate = time.Now()
	}

	// Get analytics
	analytics, err := am.GetScheduleAnalytics(userID, startDate, endDate)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate analytics: %v", err), http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}
