// Package storagetime owns the timestamp representation persisted by SQLite
// stores in the control plane.
package storagetime

import "time"

// FormatUTC is the only timestamp format safe in a SQL bind position. It is
// UTC-normalized and fixed-width, so values sort lexically in SQLite.
func FormatUTC(t time.Time) string {
	return formatTime(t)
}

func formatTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000000000Z07:00")
}

// FormatOptionalUTC formats a nullable persisted timestamp.
func FormatOptionalUTC(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return FormatUTC(*t)
}
