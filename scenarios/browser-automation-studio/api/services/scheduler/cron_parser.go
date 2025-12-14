package scheduler

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// Common cron expression presets for user-friendly scheduling.
var CronPresets = map[string]string{
	"@hourly":   "0 0 * * * *", // Every hour at minute 0
	"@daily":    "0 0 0 * * *", // Every day at midnight
	"@weekly":   "0 0 0 * * 0", // Every Sunday at midnight
	"@monthly":  "0 0 0 1 * *", // First day of month at midnight
	"@yearly":   "0 0 0 1 1 *", // January 1st at midnight
	"@annually": "0 0 0 1 1 *", // Same as yearly
	"@midnight": "0 0 0 * * *", // Same as daily
}

// ValidateCronExpression validates a cron expression and returns a detailed error if invalid.
func ValidateCronExpression(expr string) error {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return errors.New("cron expression cannot be empty")
	}

	// Check for presets
	if preset, ok := CronPresets[strings.ToLower(expr)]; ok {
		expr = preset
	}

	// Parse to validate
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

	// Try parsing with 6 fields first (second minute hour day month weekday)
	_, err := parser.Parse(expr)
	if err != nil {
		// Try prepending seconds if 5-field expression
		fields := splitCronFields(expr)
		if len(fields) == 5 {
			expr = "0 " + expr
			_, err = parser.Parse(expr)
		}
	}

	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	return nil
}

// ParseCronExpression parses a cron expression and returns the cron schedule.
// Supports both 5-field (standard) and 6-field (with seconds) formats.
func ParseCronExpression(expr string) (cron.Schedule, error) {
	expr = strings.TrimSpace(expr)

	// Check for presets
	if preset, ok := CronPresets[strings.ToLower(expr)]; ok {
		expr = preset
	}

	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

	// If 5-field expression, prepend "0" for seconds
	fields := splitCronFields(expr)
	if len(fields) == 5 {
		expr = "0 " + expr
	}

	return parser.Parse(expr)
}

// CalculateNextRun calculates the next run time for a cron expression in the given timezone.
func CalculateNextRun(cronExpr string, timezone string) (time.Time, error) {
	schedule, err := ParseCronExpression(cronExpr)
	if err != nil {
		return time.Time{}, err
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	// Calculate next run in the specified timezone
	now := time.Now().In(loc)
	next := schedule.Next(now)

	return next, nil
}

// CalculateNextNRuns calculates the next N run times for a cron expression.
// Useful for showing users when their schedule will run.
func CalculateNextNRuns(cronExpr string, timezone string, n int) ([]time.Time, error) {
	schedule, err := ParseCronExpression(cronExpr)
	if err != nil {
		return nil, err
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	runs := make([]time.Time, 0, n)
	current := time.Now().In(loc)

	for i := 0; i < n; i++ {
		next := schedule.Next(current)
		runs = append(runs, next)
		current = next
	}

	return runs, nil
}

// DescribeCronExpression returns a human-readable description of a cron expression.
func DescribeCronExpression(expr string) string {
	expr = strings.TrimSpace(expr)

	// Check for presets
	switch strings.ToLower(expr) {
	case "@hourly":
		return "Every hour"
	case "@daily", "@midnight":
		return "Every day at midnight"
	case "@weekly":
		return "Every Sunday at midnight"
	case "@monthly":
		return "First day of every month at midnight"
	case "@yearly", "@annually":
		return "Every year on January 1st"
	}

	fields := splitCronFields(expr)
	if len(fields) < 5 {
		return "Invalid expression"
	}

	// For now, just return basic descriptions
	// A full implementation would parse all field variations
	minute := fields[0]
	hour := fields[1]
	dayOfMonth := fields[2]
	month := fields[3]
	dayOfWeek := fields[4]

	if len(fields) == 6 {
		minute = fields[1]
		hour = fields[2]
		dayOfMonth = fields[3]
		month = fields[4]
		dayOfWeek = fields[5]
	}

	// Simple cases
	if minute == "0" && hour == "*" && dayOfMonth == "*" && month == "*" && dayOfWeek == "*" {
		return "Every hour"
	}
	if dayOfMonth == "*" && month == "*" && dayOfWeek == "*" {
		if minute == "0" && hour == "0" {
			return "Every day at midnight"
		}
		if minute == "0" {
			return fmt.Sprintf("Every day at %s:00", hour)
		}
	}
	if minute == "0" && hour == "9" && dayOfMonth == "*" && month == "*" && dayOfWeek == "1-5" {
		return "Weekdays at 9:00 AM"
	}

	// Fallback to technical description
	return fmt.Sprintf("Cron: %s", expr)
}

// splitCronFields splits a cron expression into its individual fields.
func splitCronFields(expr string) []string {
	expr = strings.TrimSpace(expr)
	fields := strings.Fields(expr)
	return fields
}

// CommonSchedules provides common schedule patterns for UI dropdowns.
var CommonSchedules = []struct {
	Label          string
	CronExpression string
	Description    string
}{
	{"Every minute", "* * * * *", "Runs at the start of every minute"},
	{"Every 5 minutes", "*/5 * * * *", "Runs every 5 minutes"},
	{"Every 15 minutes", "*/15 * * * *", "Runs every 15 minutes"},
	{"Every 30 minutes", "*/30 * * * *", "Runs every 30 minutes"},
	{"Every hour", "0 * * * *", "Runs at the start of every hour"},
	{"Every 2 hours", "0 */2 * * *", "Runs every 2 hours"},
	{"Every 6 hours", "0 */6 * * *", "Runs every 6 hours"},
	{"Every 12 hours", "0 */12 * * *", "Runs at noon and midnight"},
	{"Daily at midnight", "0 0 * * *", "Runs every day at 00:00"},
	{"Daily at 6 AM", "0 6 * * *", "Runs every day at 06:00"},
	{"Daily at 9 AM", "0 9 * * *", "Runs every day at 09:00"},
	{"Daily at noon", "0 12 * * *", "Runs every day at 12:00"},
	{"Daily at 6 PM", "0 18 * * *", "Runs every day at 18:00"},
	{"Weekdays at 9 AM", "0 9 * * 1-5", "Runs Monday-Friday at 09:00"},
	{"Weekends at 10 AM", "0 10 * * 0,6", "Runs Saturday and Sunday at 10:00"},
	{"Weekly on Monday", "0 0 * * 1", "Runs every Monday at midnight"},
	{"Weekly on Sunday", "0 0 * * 0", "Runs every Sunday at midnight"},
	{"Monthly on the 1st", "0 0 1 * *", "Runs on the 1st of each month"},
	{"Monthly on the 15th", "0 0 15 * *", "Runs on the 15th of each month"},
	{"Quarterly", "0 0 1 1,4,7,10 *", "Runs on the 1st of Jan, Apr, Jul, Oct"},
	{"Yearly on Jan 1st", "0 0 1 1 *", "Runs every year on January 1st"},
}

// GetTimezones returns a list of common timezones for UI dropdowns.
func GetTimezones() []string {
	return []string{
		"UTC",
		"America/New_York",
		"America/Chicago",
		"America/Denver",
		"America/Los_Angeles",
		"America/Toronto",
		"America/Vancouver",
		"America/Sao_Paulo",
		"Europe/London",
		"Europe/Paris",
		"Europe/Berlin",
		"Europe/Moscow",
		"Asia/Dubai",
		"Asia/Kolkata",
		"Asia/Singapore",
		"Asia/Tokyo",
		"Asia/Shanghai",
		"Asia/Hong_Kong",
		"Australia/Sydney",
		"Australia/Melbourne",
		"Pacific/Auckland",
	}
}
