package persistence

import (
	"database/sql"
	"fmt"
	"time"
)

const (
	maxHealthResultDetailsBytes = 64 * 1024
	maxActionLogTextBytes       = 8 * 1024
	defaultRetentionBatchSize   = 200
)

func parseDBTime(raw any) (time.Time, error) {
	switch v := raw.(type) {
	case time.Time:
		return v, nil
	case string:
		return parseTimeString(v)
	case []byte:
		return parseTimeString(string(v))
	default:
		return time.Time{}, fmt.Errorf("unsupported timestamp type %T", raw)
	}
}

func parseTimeString(s string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, s); err == nil {
			return ts, nil
		}
	}
	return time.Time{}, fmt.Errorf("failed to parse time %q", s)
}

func nullableTimeToDBText(ts time.Time) interface{} {
	if ts.IsZero() {
		return nil
	}
	return ts.UTC().Format(time.RFC3339Nano)
}

func parseNullableDBTime(raw any) (time.Time, bool) {
	if raw == nil {
		return time.Time{}, false
	}
	switch v := raw.(type) {
	case time.Time:
		if v.IsZero() {
			return time.Time{}, false
		}
		return v, true
	case string:
		if v == "" {
			return time.Time{}, false
		}
		ts, err := parseTimeString(v)
		if err != nil {
			return time.Time{}, false
		}
		return ts, true
	case []byte:
		if len(v) == 0 {
			return time.Time{}, false
		}
		ts, err := parseTimeString(string(v))
		if err != nil {
			return time.Time{}, false
		}
		return ts, true
	case sql.NullTime:
		if !v.Valid || v.Time.IsZero() {
			return time.Time{}, false
		}
		return v.Time, true
	default:
		return time.Time{}, false
	}
}
