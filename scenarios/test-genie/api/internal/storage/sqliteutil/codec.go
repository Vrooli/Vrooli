package sqliteutil

import (
	"encoding/json"
	"fmt"
	"time"
)

// FormatTimestamp normalizes timestamps before persistence so SQLite stores a
// stable RFC3339 text value.
func FormatTimestamp(ts time.Time) string {
	return ts.UTC().Format(time.RFC3339Nano)
}

// ParseTimestamp decodes a timestamp returned from SQLite text storage.
func ParseTimestamp(src any) (time.Time, error) {
	text, err := Text(src)
	if err != nil {
		return time.Time{}, err
	}
	if text == "" {
		return time.Time{}, fmt.Errorf("timestamp value is empty")
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if ts, parseErr := time.Parse(layout, text); parseErr == nil {
			return ts.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported timestamp format %q", text)
}

// MarshalStringSlice encodes a string list as JSON text for SQLite storage.
func MarshalStringSlice(values []string) (string, error) {
	if values == nil {
		values = []string{}
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// UnmarshalStringSlice decodes a JSON string list from SQLite text storage.
func UnmarshalStringSlice(src any) ([]string, error) {
	data, err := Bytes(src)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	var values []string
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, err
	}
	return values, nil
}

// Bytes converts SQLite text/blob values into a byte slice.
func Bytes(src any) ([]byte, error) {
	switch value := src.(type) {
	case nil:
		return nil, nil
	case []byte:
		return append([]byte(nil), value...), nil
	case string:
		return []byte(value), nil
	default:
		return nil, fmt.Errorf("unsupported sqlite value type %T", src)
	}
}

// Text converts SQLite text/blob values into a string.
func Text(src any) (string, error) {
	data, err := Bytes(src)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// UnmarshalJSON decodes arbitrary JSON stored as SQLite text.
func UnmarshalJSON[T any](src any, dest *T) error {
	data, err := Bytes(src)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, dest)
}
