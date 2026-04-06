// Package sqlutil provides shared SQLite helpers used across store, policy,
// and subscription packages, eliminating duplicated format strings and
// conversion functions.
package sqlutil

import "time"

// TimestampFormat is the SQLite datetime format used by strftime('%Y-%m-%dT%H:%M:%f','now').
const TimestampFormat = "2006-01-02T15:04:05.000"

// ParseTime parses a SQLite timestamp string. Returns the zero time if parsing fails.
func ParseTime(s string) time.Time {
	t, _ := time.Parse(TimestampFormat, s)
	return t
}

// BoolToInt converts a bool to a SQLite-compatible integer (0 or 1).
func BoolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
