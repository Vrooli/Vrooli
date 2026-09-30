// Package sqlitetime normalises how this scenario stores timestamps in SQLite.
//
// SQLite has no date type; a TIMESTAMP column holds text, and the text format
// depends on who wrote the row. Left alone, one column ends up with two shapes:
//
//	CURRENT_TIMESTAMP default   -> "2026-08-01 22:43:52"
//	driver-bound time.Time      -> "2026-03-04 05:06:07 +0000 UTC"
//
// The driver hides that when scanning a plain column, because it knows the
// declared type and parses either form. It does not hide it once a value passes
// through an expression: MAX(created_at) erases the type affinity, the driver
// returns a bare string, and a scan into time.Time fails at runtime.
//
// Every write therefore goes through Format, so stored text always matches the
// SQLite canonical form that CURRENT_TIMESTAMP produces. Parse handles reads
// that come back as text.
package sqlitetime

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Layout is the canonical SQLite timestamp format, and the one CURRENT_TIMESTAMP
// emits. It is fixed-width and UTC, so lexicographic ordering matches
// chronological ordering — which is what makes plain `<` comparisons on these
// columns correct.
const Layout = "2006-01-02 15:04:05"

// readLayouts are accepted when parsing. The first is canonical; the rest cover
// text written before normalisation or by another writer.
var readLayouts = []string{
	Layout,
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05 -0700 MST", // Go's time.Time.String()
	time.RFC3339Nano,
	time.RFC3339,
}

// Format renders t for storage. The zero time becomes the zero value of the
// layout rather than a Go-specific string, so it still sorts before real data.
func Format(t time.Time) string {
	return t.UTC().Format(Layout)
}

// FormatPtr renders an optional timestamp for binding: nil stays SQL NULL.
func FormatPtr(t *time.Time) any {
	if t == nil {
		return nil
	}
	return Format(*t)
}

// Parse reads a stored timestamp. It accepts every layout this database could
// hold so a value written before normalisation still reads back.
func Parse(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	for _, layout := range readLayouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised timestamp %q", value)
}

// ParseNull converts a nullable text column into an optional timestamp. An
// unparseable value is reported as absent rather than as an error, because
// these columns feed reporting rather than control flow.
func ParseNull(value sql.NullString) *time.Time {
	if !value.Valid {
		return nil
	}
	parsed, err := Parse(value.String)
	if err != nil {
		return nil
	}
	return &parsed
}
