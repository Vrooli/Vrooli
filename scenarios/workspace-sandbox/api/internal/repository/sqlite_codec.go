package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// SQLite codec helpers.
//
// SQLite stores arrays and JSON-shaped values as TEXT and timestamps as
// RFC3339Nano UTC TEXT. These helpers centralize the marshal/unmarshal logic
// so query call sites read like ordinary Go.
// ---------------------------------------------------------------------------

// jsonStrings encodes a slice of strings as a JSON array string suitable for
// storage in a TEXT column. nil and empty inputs both serialize to "[]".
func jsonStrings(xs []string) string {
	if len(xs) == 0 {
		return "[]"
	}
	b, err := json.Marshal(xs)
	if err != nil {
		// json.Marshal of a []string never fails; if it ever did, fall back
		// to the empty array rather than corrupting the column.
		return "[]"
	}
	return string(b)
}

// jsonInts encodes a slice of ints as a JSON array string.
func jsonInts(xs []int) string {
	if len(xs) == 0 {
		return "[]"
	}
	b, err := json.Marshal(xs)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// parseStrings decodes a JSON array TEXT column into []string. Empty/null
// values yield a nil slice (not an error).
func parseStrings(s string) ([]string, error) {
	if s == "" || s == "null" {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, fmt.Errorf("decode string array: %w", err)
	}
	return out, nil
}

// parseInts decodes a JSON array TEXT column into []int.
func parseInts(s string) ([]int, error) {
	if s == "" || s == "null" {
		return nil, nil
	}
	var out []int
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, fmt.Errorf("decode int array: %w", err)
	}
	return out, nil
}

// jsonObject encodes any value (typically a map or struct) as a JSON object
// string. nil inputs serialize to "{}".
func jsonObject(v any) (string, error) {
	if v == nil {
		return "{}", nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	if len(b) == 0 || string(b) == "null" {
		return "{}", nil
	}
	return string(b), nil
}

// parseJSONObject decodes a TEXT column into the supplied destination. Empty
// strings are treated as "no data" and leave dst unchanged.
func parseJSONObject(s string, dst any) error {
	if s == "" || s == "null" {
		return nil
	}
	return json.Unmarshal([]byte(s), dst)
}

// formatTime renders a time as the canonical RFC3339Nano UTC TEXT used by all
// timestamp columns.
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

// formatTimePtr renders a *time.Time, returning sql.NullString so nil maps to
// NULL and non-nil maps to the formatted UTC string.
func formatTimePtr(t *time.Time) sql.NullString {
	if t == nil || t.IsZero() {
		return sql.NullString{}
	}
	return sql.NullString{String: t.UTC().Format(time.RFC3339Nano), Valid: true}
}

// parseTime decodes a TEXT timestamp into time.Time. Accepts RFC3339Nano and
// RFC3339 as fallback for older rows.
func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC(), nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time %q: %w", s, err)
	}
	return t.UTC(), nil
}

// parseTimePtr decodes a sql.NullString into *time.Time. NULL/empty rows
// produce a nil pointer.
func parseTimePtr(ns sql.NullString) (*time.Time, error) {
	if !ns.Valid || ns.String == "" {
		return nil, nil
	}
	t, err := parseTime(ns.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// uuidText returns the canonical 36-char form of a uuid.UUID.
func uuidText(id uuid.UUID) string {
	return id.String()
}

// parseUUID parses a TEXT column into a uuid.UUID.
func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// boolInt converts a Go bool to the 0/1 integer SQLite stores.
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// nullableString returns a sql.NullString that is NULL when s is empty.
func nullableString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
