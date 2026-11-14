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

// formatTime renders a time as UTC RFC3339Nano. Used by storage layers
// that write CreatedAt / UpdatedAt columns.
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}
