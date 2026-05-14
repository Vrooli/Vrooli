package main

import "time"

// boolToInt converts a bool to 0/1 for SQLite columns that store booleans
// as integers.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// formatTimeOrEmpty renders a time as RFC3339 UTC, or empty string for the
// zero time. Used by callers that mirror Metadata's wire format.
func formatTimeOrEmpty(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
