package usageattribution

// Responsibility: parse the one time-bound vocabulary every usage window
// accepts (--since/--until on the CLI, the same fields over RPC and HTTP).

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseWindowBound reads a window bound written as RFC 3339 (fractional
// seconds allowed), a calendar date YYYY-MM-DD (UTC midnight), or a relative
// age such as 7d, 36h or 90m meaning that long before now. An empty value is
// the zero time, which callers treat as "use the default". The error names
// field and every accepted form, so a caller can return it as-is.
func ParseWindowBound(field, value string, now time.Time) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	if at, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return at.UTC(), nil
	}
	if at, err := time.ParseInLocation("2006-01-02", value, time.UTC); err == nil {
		return at, nil
	}
	if age, ok := parseRelativeAge(value); ok {
		return now.UTC().Add(-age), nil
	}
	return time.Time{}, fmt.Errorf("%s %q must be RFC 3339, YYYY-MM-DD or a relative age such as 7d or 24h", field, value)
}

// parseRelativeAge accepts a positive whole number of days (7d) or any
// positive Go duration (24h, 90m, 1h30m).
func parseRelativeAge(value string) (time.Duration, bool) {
	if days, found := strings.CutSuffix(value, "d"); found {
		n, err := strconv.Atoi(days)
		if err != nil || n <= 0 {
			return 0, false
		}
		return time.Duration(n) * 24 * time.Hour, true
	}
	age, err := time.ParseDuration(value)
	if err != nil || age <= 0 {
		return 0, false
	}
	return age, true
}
