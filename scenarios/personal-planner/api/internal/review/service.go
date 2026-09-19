package review

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"
)

type (
	SQLExecutor interface {
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
	Service          interface {
		Daily(context.Context, string) (DailySummary, error)
		Weekly(context.Context, string) (WeeklySummary, error)
		Reflection(context.Context, string) (ReviewReflection, error)
		SaveReflection(context.Context, string, string) (ReviewReflection, error)
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
