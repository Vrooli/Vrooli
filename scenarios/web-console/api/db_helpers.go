package main

import (
	"time"

	"web-console/internal/dbx"
)

// reads is the handle for request-path reads: standalone queries run on the
// query-only read pool beside the single writer, so a slow scan never queues
// other requests behind it; statements and transactions still go to the
// writer. Test-mode requests reach the leased test pool on both sides.
func (s *Server) reads() dbx.Handle {
	return dbx.ReadWrite{Writes: s.db, Reads: s.db.Reader()}
}

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
