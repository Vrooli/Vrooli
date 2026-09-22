package review

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"
)

type (
	SQLExecutor interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
		ExecContext(context.Context, string, ...any) (sql.Result, error)
	}
	DailySummary struct {
		LocalDate                                                                                    string
		PlannedMinutes, RecordedActiveMinutes, FocusSessionCount, ActiveGoalCount, UnrecordedMinutes int64
		CoverageNote                                                                                 string
	}
	WeeklySummary struct {
		WeekStartLocalDate                                                        string
		Days                                                                      []DailySummary
		PlannedMinutes, RecordedActiveMinutes, FocusSessionCount, ActiveGoalCount int64
		CoverageNote                                                              string
	}
	ReviewReflection struct{ LocalDate, Text, UpdatedAt string }
	ReviewWin        struct{ LocalDate, Text, UpdatedAt string }
	Calibration      struct {
		Period          string             `json:"period"`
		AvgErrorPercent float64            `json:"avg_error_percent"`
		OverRatio       float64            `json:"over_ratio"`
		UnderRatio      float64            `json:"under_ratio"`
		ByCategory      map[string]float64 `json:"by_category"`
		AccuracyTrend   []float64          `json:"accuracy_trend"`
		SampleSize      int64              `json:"sample_size"`
		UpdatedAt       string             `json:"updated_at"`
	}
	AgentReadModel struct {
		EstimationBias Calibration `json:"estimation_bias"`
		GeneratedAt    string      `json:"generated_at"`
	}
	GoalVariance struct {
		GoalID        string `json:"goal_id"`
		TargetDate    string `json:"target_date"`
		CompletedDate string `json:"completed_date"`
		DeltaDays     int64  `json:"delta_days"`
		Label         string `json:"label"`
	}
	TodaySignals struct {
		OverdueCount   int64  `json:"overdue_count"`
		OverdueMinutes int64  `json:"overdue_minutes"`
		MomentumDays   int64  `json:"momentum_days"`
		Label          string `json:"label"`
	}
	GoalDrift struct {
		GoalID        string `json:"goal_id"`
		TargetDate    string `json:"target_date"`
		ExpectedBasis int64  `json:"expected_basis_points"`
		ActualBasis   int64  `json:"actual_basis_points"`
		DriftBasis    int64  `json:"drift_basis_points"`
		Label         string `json:"label"`
	}
	Reminder struct {
		ID        string `json:"id"`
		Kind      string `json:"kind"`
		Title     string `json:"title"`
		Body      string `json:"body"`
		StartDate string `json:"start_date"`
		StartMin  int64  `json:"start_minutes"`
	}
	ReminderPreferences struct {
		Enabled           bool   `json:"enabled"`
		QuietStartMinutes int64  `json:"quiet_start_minutes"`
		QuietEndMinutes   int64  `json:"quiet_end_minutes"`
		LeadMinutes       int64  `json:"lead_minutes"`
		UpdatedAt         string `json:"updated_at"`
	}
	Service interface {
		Daily(context.Context, string) (DailySummary, error)
		Weekly(context.Context, string) (WeeklySummary, error)
		Reflection(context.Context, string) (ReviewReflection, error)
		SaveReflection(context.Context, string, string) (ReviewReflection, error)
		Win(context.Context, string) (ReviewWin, error)
		SaveWin(context.Context, string, string) (ReviewWin, error)
		Calibration(context.Context, string) (Calibration, error)
		AgentReadModel(context.Context, string) (AgentReadModel, error)
		GoalVariances(context.Context) ([]GoalVariance, error)
		TodaySignals(context.Context, string) (TodaySignals, error)
		GoalDrifts(context.Context, string) ([]GoalDrift, error)
		Reminders(context.Context, string) ([]Reminder, error)
		ReminderPreferences(context.Context) (ReminderPreferences, error)
		SaveReminderPreferences(context.Context, ReminderPreferences) (ReminderPreferences, error)
	}
	service struct {
		db    SQLExecutor
		clock schedule.Clock
	}
)

func NewService(db SQLExecutor, clock schedule.Clock) Service { return &service{db: db, clock: clock} }

func (s *service) Daily(ctx context.Context, localDate string) (DailySummary, error) {
	zone := s.clock.Now().Location()
	day := s.clock.Now().In(zone)
	if localDate == "" {
		localDate = day.Format("2006-01-02")
	}
	parsed, err := time.ParseInLocation("2006-01-02", localDate, zone)
	if err != nil {
		return DailySummary{}, fmt.Errorf("local_date must be YYYY-MM-DD: %w", err)
	}
	start, end := parsed.Unix(), parsed.AddDate(0, 0, 1).Unix()
	var activeSeconds, sessions int64
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE((SELECT SUM(active_seconds) FROM focus_sessions WHERE started_at >= ? AND started_at < ?),0) + COALESCE((SELECT SUM(reported_minutes * 60) FROM manual_actuals WHERE local_date = ?),0), (SELECT COUNT(*) FROM focus_sessions WHERE started_at >= ? AND started_at < ?)`, start, end, localDate, start, end).Scan(&activeSeconds, &sessions); err != nil {
		return DailySummary{}, fmt.Errorf("read focus history: %w", err)
	}
	var activeGoals int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM goals WHERE status = 'active'`).Scan(&activeGoals); err != nil {
		return DailySummary{}, fmt.Errorf("read goals: %w", err)
	}
	var plannedMinutes int64
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(duration_minutes),0) FROM calendar_allocations WHERE local_date = ? AND state = 'accepted'`, localDate).Scan(&plannedMinutes); err != nil {
		return DailySummary{}, fmt.Errorf("read calendar allocations: %w", err)
	}
	return DailySummary{LocalDate: localDate, PlannedMinutes: plannedMinutes, RecordedActiveMinutes: activeSeconds / 60, FocusSessionCount: sessions, ActiveGoalCount: activeGoals, CoverageNote: "Accepted planned time and recorded activity (focus sessions plus manual actuals) are measured separately. Unrecorded time remains unknown; completing focus does not silently complete scheduled work."}, nil
}

func (s *service) Win(ctx context.Context, localDate string) (ReviewWin, error) {
	localDate, err := s.validDate(localDate)
	if err != nil {
		return ReviewWin{}, err
	}
	var win ReviewWin
	err = s.db.QueryRowContext(ctx, `SELECT local_date,text,updated_at FROM review_wins WHERE local_date=?`, localDate).Scan(&win.LocalDate, &win.Text, &win.UpdatedAt)
	if err == sql.ErrNoRows {
		return ReviewWin{LocalDate: localDate}, nil
	}
	if err != nil {
		return ReviewWin{}, fmt.Errorf("read win: %w", err)
	}
	return win, nil
}

func (s *service) SaveWin(ctx context.Context, localDate, text string) (ReviewWin, error) {
	localDate, err := s.validDate(localDate)
	if err != nil {
		return ReviewWin{}, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return ReviewWin{}, fmt.Errorf("win text must not be empty")
	}
	if len([]rune(text)) > 500 {
		return ReviewWin{}, fmt.Errorf("win text must be 500 characters or fewer")
	}
	updatedAt := s.clock.Now().UTC().Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `INSERT INTO review_wins(local_date,text,updated_at) VALUES (?,?,?) ON CONFLICT(local_date) DO UPDATE SET text=excluded.text,updated_at=excluded.updated_at`, localDate, text, updatedAt); err != nil {
		return ReviewWin{}, fmt.Errorf("save win: %w", err)
	}
	return ReviewWin{LocalDate: localDate, Text: text, UpdatedAt: updatedAt}, nil
}

func (s *service) AgentReadModel(ctx context.Context, period string) (AgentReadModel, error) {
	calibration, err := s.Calibration(ctx, period)
	if err != nil {
		return AgentReadModel{}, err
	}
	return AgentReadModel{EstimationBias: calibration, GeneratedAt: s.clock.Now().UTC().Format(time.RFC3339Nano)}, nil
}

func (s *service) ReminderPreferences(ctx context.Context) (ReminderPreferences, error) {
	var p ReminderPreferences
	var enabled int64
	err := s.db.QueryRowContext(ctx, `SELECT enabled,quiet_start_minutes,quiet_end_minutes,lead_minutes,updated_at FROM reminder_preferences WHERE id='workspace'`).Scan(&enabled, &p.QuietStartMinutes, &p.QuietEndMinutes, &p.LeadMinutes, &p.UpdatedAt)
	if err != nil {
		return ReminderPreferences{}, fmt.Errorf("read reminder preferences: %w", err)
	}
	p.Enabled = enabled != 0
	return p, nil
}

func (s *service) SaveReminderPreferences(ctx context.Context, p ReminderPreferences) (ReminderPreferences, error) {
	if p.QuietStartMinutes < 0 || p.QuietStartMinutes >= 1440 || p.QuietEndMinutes < 0 || p.QuietEndMinutes >= 1440 {
		return ReminderPreferences{}, fmt.Errorf("quiet hours must be within a day")
	}
	if p.LeadMinutes < 0 || p.LeadMinutes > 240 {
		return ReminderPreferences{}, fmt.Errorf("lead minutes must be between 0 and 240")
	}
	p.UpdatedAt = s.clock.Now().UTC().Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `INSERT INTO reminder_preferences(id,enabled,quiet_start_minutes,quiet_end_minutes,lead_minutes,updated_at) VALUES ('workspace',?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET enabled=excluded.enabled,quiet_start_minutes=excluded.quiet_start_minutes,quiet_end_minutes=excluded.quiet_end_minutes,lead_minutes=excluded.lead_minutes,updated_at=excluded.updated_at`, boolInt(p.Enabled), p.QuietStartMinutes, p.QuietEndMinutes, p.LeadMinutes, p.UpdatedAt); err != nil {
		return ReminderPreferences{}, fmt.Errorf("save reminder preferences: %w", err)
	}
	return p, nil
}

func (s *service) Weekly(ctx context.Context, weekStartLocalDate string) (WeeklySummary, error) {
	zone := s.clock.Now().Location()
	if weekStartLocalDate == "" {
		today := s.clock.Now().In(zone)
		weekStartLocalDate = formatWeekStart(today)
	}
	start, err := time.ParseInLocation("2006-01-02", weekStartLocalDate, zone)
	if err != nil {
		return WeeklySummary{}, fmt.Errorf("week_start_local_date must be YYYY-MM-DD: %w", err)
	}
	result := WeeklySummary{WeekStartLocalDate: weekStartLocalDate, CoverageNote: "Each day separates accepted planned time from recorded focus. Unrecorded time remains unknown; no weekly completion claim is inferred."}
	for day := 0; day < 7; day++ {
		summary, err := s.Daily(ctx, start.AddDate(0, 0, day).Format("2006-01-02"))
		if err != nil {
			return WeeklySummary{}, err
		}
		result.Days = append(result.Days, summary)
		result.PlannedMinutes += summary.PlannedMinutes
		result.RecordedActiveMinutes += summary.RecordedActiveMinutes
		result.FocusSessionCount += summary.FocusSessionCount
		if day == 0 {
			result.ActiveGoalCount = summary.ActiveGoalCount
		}
	}
	return result, nil
}

func (s *service) Reflection(ctx context.Context, localDate string) (ReviewReflection, error) {
	localDate, err := s.validDate(localDate)
	if err != nil {
		return ReviewReflection{}, err
	}
	var reflection ReviewReflection
	err = s.db.QueryRowContext(ctx, `SELECT local_date, text, updated_at FROM review_reflections WHERE local_date = ?`, localDate).Scan(&reflection.LocalDate, &reflection.Text, &reflection.UpdatedAt)
	if err == sql.ErrNoRows {
		return ReviewReflection{LocalDate: localDate}, nil
	}
	if err != nil {
		return ReviewReflection{}, fmt.Errorf("read reflection: %w", err)
	}
	return reflection, nil
}

func (s *service) SaveReflection(ctx context.Context, localDate, text string) (ReviewReflection, error) {
	localDate, err := s.validDate(localDate)
	if err != nil {
		return ReviewReflection{}, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return ReviewReflection{}, fmt.Errorf("reflection text must not be empty")
	}
	if len([]rune(text)) > 4000 {
		return ReviewReflection{}, fmt.Errorf("reflection text must be 4000 characters or fewer")
	}
	updatedAt := s.clock.Now().UTC().Format(time.RFC3339)
	if _, err := s.db.ExecContext(ctx, `INSERT INTO review_reflections(local_date, text, updated_at) VALUES (?, ?, ?) ON CONFLICT(local_date) DO UPDATE SET text = excluded.text, updated_at = excluded.updated_at`, localDate, text, updatedAt); err != nil {
		return ReviewReflection{}, fmt.Errorf("save reflection: %w", err)
	}
	return ReviewReflection{LocalDate: localDate, Text: text, UpdatedAt: updatedAt}, nil
}

// Calibration recomputes and persists the read model from actuals explicitly
// linked to planned allocations. The stored projection is the contract for
// the UI and ecosystem agents; callers never need to reconstruct it.
func (s *service) Calibration(ctx context.Context, period string) (Calibration, error) {
	if period == "" {
		period = "all_time"
	}
	if period != "all_time" && period != "last_30d" && period != "last_90d" {
		return Calibration{}, fmt.Errorf("period must be all_time, last_30d, or last_90d")
	}
	cutoff := ""
	if period != "all_time" {
		days := 30
		if period == "last_90d" {
			days = 90
		}
		cutoff = s.clock.Now().In(s.clock.Now().Location()).AddDate(0, 0, -days).Format("2006-01-02")
	}
	query := `SELECT m.reported_minutes, a.duration_minutes FROM manual_actuals m JOIN calendar_allocations a ON a.id=m.allocation_id WHERE m.allocation_id<>'' AND a.duration_minutes>0`
	args := []any{}
	if cutoff != "" {
		query += ` AND m.local_date>=?`
		args = append(args, cutoff)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return Calibration{}, fmt.Errorf("read calibration evidence: %w", err)
	}
	defer rows.Close()
	var totalPercent float64
	var over, under, samples int64
	trend := []float64{}
	for rows.Next() {
		var actual, planned int64
		if err := rows.Scan(&actual, &planned); err != nil {
			return Calibration{}, err
		}
		if planned <= 0 {
			continue
		}
		percent := float64(actual-planned) / float64(planned) * 100
		trend = append(trend, percent)
		totalPercent += percent
		samples++
		if actual > planned {
			over++
		} else if actual < planned {
			under++
		}
	}
	if err := rows.Err(); err != nil {
		return Calibration{}, err
	}
	avg, overRatio, underRatio := 0.0, 0.0, 0.0
	if samples > 0 {
		avg = totalPercent / float64(samples)
		overRatio = float64(over) / float64(samples)
		underRatio = float64(under) / float64(samples)
	}
	trendJSON, err := json.Marshal(trend)
	if err != nil {
		return Calibration{}, err
	}
	updated := s.clock.Now().UTC().Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `INSERT INTO estimation_bias(period,avg_error_percent,over_ratio,under_ratio,by_category_json,accuracy_trend_json,sample_size,updated_at) VALUES (?,?,?,?,?,?,?,?) ON CONFLICT(period) DO UPDATE SET avg_error_percent=excluded.avg_error_percent,over_ratio=excluded.over_ratio,under_ratio=excluded.under_ratio,by_category_json=excluded.by_category_json,accuracy_trend_json=excluded.accuracy_trend_json,sample_size=excluded.sample_size,updated_at=excluded.updated_at`, period, avg, overRatio, underRatio, `{}`, string(trendJSON), samples, updated); err != nil {
		return Calibration{}, fmt.Errorf("persist calibration: %w", err)
	}
	return Calibration{Period: period, AvgErrorPercent: avg, OverRatio: overRatio, UnderRatio: underRatio, ByCategory: map[string]float64{}, AccuracyTrend: trend, SampleSize: samples, UpdatedAt: updated}, nil
}

// GoalVariances is the durable goal-level learning read model. Dates are read
// from goals first, with milestone due dates as a fallback for older goals.
func (s *service) GoalVariances(ctx context.Context) ([]GoalVariance, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT g.id,
		       COALESCE(NULLIF(g.target_date,''), NULLIF(MIN(NULLIF(m.original_due_date,'')),''), NULLIF(MIN(NULLIF(m.due_date,'')),''), ''),
		       COALESCE(NULLIF(g.completed_date,''), NULLIF(MAX(NULLIF(m.completed_date,'')),''), '')
		FROM goals g LEFT JOIN milestones m ON m.goal_id=g.id
		GROUP BY g.id, g.target_date, g.completed_date
		HAVING COALESCE(NULLIF(g.target_date,''), NULLIF(MIN(NULLIF(m.original_due_date,'')),''), NULLIF(MIN(NULLIF(m.due_date,'')),''), '') <> ''
		   AND COALESCE(NULLIF(g.completed_date,''), NULLIF(MAX(NULLIF(m.completed_date,'')),''), '') <> ''`)
	if err != nil {
		return nil, fmt.Errorf("read goal variances: %w", err)
	}
	defer rows.Close()
	out := []GoalVariance{}
	for rows.Next() {
		var item GoalVariance
		if err := rows.Scan(&item.GoalID, &item.TargetDate, &item.CompletedDate); err != nil {
			return nil, fmt.Errorf("scan goal variance: %w", err)
		}
		target, err := time.Parse("2006-01-02", item.TargetDate)
		if err != nil {
			return nil, fmt.Errorf("parse goal target date: %w", err)
		}
		completed, err := time.Parse("2006-01-02", item.CompletedDate)
		if err != nil {
			return nil, fmt.Errorf("parse goal completed date: %w", err)
		}
		item.DeltaDays = int64(completed.Sub(target).Hours() / 24)
		switch {
		case item.DeltaDays < 0:
			item.Label = fmt.Sprintf("%d days early", -item.DeltaDays)
		case item.DeltaDays > 0:
			item.Label = fmt.Sprintf("%d days over", item.DeltaDays)
		default:
			item.Label = "on time"
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *service) TodaySignals(ctx context.Context, localDate string) (TodaySignals, error) {
	localDate, err := s.validDate(localDate)
	if err != nil {
		return TodaySignals{}, err
	}
	var signals TodaySignals
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*),COALESCE(SUM(a.duration_minutes),0) FROM calendar_allocations a JOIN work_items w ON w.id=a.work_item_id WHERE a.local_date<? AND a.state='accepted' AND w.status<>'complete'`, localDate).Scan(&signals.OverdueCount, &signals.OverdueMinutes); err != nil {
		return TodaySignals{}, fmt.Errorf("read overdue work: %w", err)
	}
	zone := s.clock.Now().Location()
	day, _ := time.ParseInLocation("2006-01-02", localDate, zone)
	for offset := 0; offset < 30; offset++ {
		candidate := day.AddDate(0, 0, -offset)
		start, end := candidate.Unix(), candidate.AddDate(0, 0, 1).Unix()
		var activity int64
		if err := s.db.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM focus_sessions WHERE started_at>=? AND started_at<?)+(SELECT COUNT(*) FROM manual_actuals WHERE local_date=?)`, start, end, candidate.Format("2006-01-02")).Scan(&activity); err != nil {
			return TodaySignals{}, fmt.Errorf("read momentum: %w", err)
		}
		if activity == 0 {
			break
		}
		signals.MomentumDays++
	}
	switch {
	case signals.OverdueCount > 0:
		signals.Label = fmt.Sprintf("%d overdue", signals.OverdueCount)
	case signals.MomentumDays > 0:
		signals.Label = fmt.Sprintf("%d-day momentum", signals.MomentumDays)
	default:
		signals.Label = "A clear start"
	}
	return signals, nil
}

func (s *service) GoalDrifts(ctx context.Context, localDate string) ([]GoalDrift, error) {
	localDate, err := s.validDate(localDate)
	if err != nil {
		return nil, err
	}
	zone := s.clock.Now().Location()
	today, _ := time.ParseInLocation("2006-01-02", localDate, zone)
	rows, err := s.db.QueryContext(ctx, `SELECT id,progress_basis_points,target_date,created_at FROM goals WHERE status='active'`)
	if err != nil {
		return nil, fmt.Errorf("read goal drift: %w", err)
	}
	defer rows.Close()
	out := []GoalDrift{}
	for rows.Next() {
		var item GoalDrift
		var created int64
		if err := rows.Scan(&item.GoalID, &item.ActualBasis, &item.TargetDate, &created); err != nil {
			return nil, err
		}
		if item.TargetDate == "" {
			out = append(out, item)
			continue
		}
		targetDate, err := time.ParseInLocation("2006-01-02", item.TargetDate, zone)
		if err != nil {
			return nil, fmt.Errorf("parse goal drift target: %w", err)
		}
		createdDate := time.Unix(created, 0).In(zone)
		totalDays := targetDate.Sub(createdDate).Hours() / 24
		elapsedDays := today.Sub(createdDate).Hours() / 24
		if totalDays <= 0 {
			continue
		}
		progress := elapsedDays / totalDays
		if progress < 0 {
			progress = 0
		}
		if progress > 1 {
			progress = 1
		}
		item.ExpectedBasis = int64(progress * 10000)
		item.DriftBasis = item.ActualBasis - item.ExpectedBasis
		switch {
		case item.DriftBasis < -500:
			item.Label = "Behind pace"
		case item.DriftBasis > 500:
			item.Label = "Ahead of pace"
		default:
			item.Label = "On pace"
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *service) Reminders(ctx context.Context, localDate string) ([]Reminder, error) {
	localDate, err := s.validDate(localDate)
	if err != nil {
		return nil, err
	}
	zone := s.clock.Now().Location()
	now := s.clock.Now().In(zone)
	nowMinutes := int64(now.Hour()*60 + now.Minute())
	prefs, err := s.ReminderPreferences(ctx)
	if err != nil {
		return nil, err
	}
	if !prefs.Enabled || inQuietHours(nowMinutes, prefs.QuietStartMinutes, prefs.QuietEndMinutes) {
		return []Reminder{}, nil
	}
	out := []Reminder{}
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.id,a.start_minutes,a.duration_minutes,w.title
		FROM calendar_allocations a JOIN work_items w ON w.id=a.work_item_id
		WHERE a.local_date=? AND a.state='accepted' AND w.status NOT IN ('complete','cancelled')
		ORDER BY a.start_minutes`, localDate)
	if err != nil {
		return nil, fmt.Errorf("read work reminders: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, title string
		var start, duration int64
		if err := rows.Scan(&id, &start, &duration, &title); err != nil {
			return nil, err
		}
		if start >= nowMinutes && start-nowMinutes <= prefs.LeadMinutes {
			out = append(out, Reminder{ID: "upcoming-" + id, Kind: "upcoming", Title: title, Body: fmt.Sprintf("Starts in %d minutes", start-nowMinutes), StartDate: localDate, StartMin: start})
		}
		if start+duration < nowMinutes {
			out = append(out, Reminder{ID: "overdue-" + id, Kind: "overdue", Title: title, Body: "This accepted block is still open", StartDate: localDate, StartMin: start})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	goalRows, err := s.db.QueryContext(ctx, `SELECT id,title FROM goals WHERE status='active' AND target_date=?`, localDate)
	if err != nil {
		return nil, fmt.Errorf("read goal reminders: %w", err)
	}
	defer goalRows.Close()
	for goalRows.Next() {
		var id, title string
		if err := goalRows.Scan(&id, &title); err != nil {
			return nil, err
		}
		out = append(out, Reminder{ID: "goal-" + id, Kind: "goal", Title: title, Body: "Target date is today", StartDate: localDate})
	}
	return out, goalRows.Err()
}

func boolInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func inQuietHours(now, start, end int64) bool {
	if start == end {
		return false
	}
	if start < end {
		return now >= start && now < end
	}
	return now >= start || now < end
}

func (s *service) validDate(localDate string) (string, error) {
	zone := s.clock.Now().Location()
	if localDate == "" {
		localDate = s.clock.Now().In(zone).Format("2006-01-02")
	}
	if _, err := time.ParseInLocation("2006-01-02", localDate, zone); err != nil {
		return "", fmt.Errorf("local_date must be YYYY-MM-DD: %w", err)
	}
	return localDate, nil
}

func formatWeekStart(date time.Time) string {
	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return date.AddDate(0, 0, -(weekday - 1)).Format("2006-01-02")
}
