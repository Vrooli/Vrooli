package agenttranscript

import (
	"bytes"
	"encoding/json"
	"strconv"
	"time"
)

// ParseTimestamp reads the vendor timestamp from one native transcript record.
// RFC3339 strings and integer Unix timestamps are supported; large integer
// values are interpreted as milliseconds, matching OpenCode's native format.
// Missing, malformed, and unsupported values return the zero time.
func ParseTimestamp(line []byte) time.Time {
	record, ok := DecodeJSONLRecord(line)
	if !ok {
		return time.Time{}
	}
	return ParseTimestampValue(record.Timestamp)
}

// ParseTimestampValue parses a native timestamp JSON value already extracted
// by a caller that decoded its containing record. This avoids decoding a
// transcript line a second time just to read its timestamp.
func ParseTimestampValue(raw json.RawMessage) time.Time {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		if parsed, err := time.Parse(time.RFC3339Nano, text); err == nil {
			return parsed
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var number json.Number
	if decoder.Decode(&number) == nil {
		value, err := strconv.ParseInt(number.String(), 10, 64)
		if err == nil {
			if value > 100000000000 {
				return time.UnixMilli(value).UTC()
			}
			return time.Unix(value, 0).UTC()
		}
	}
	return time.Time{}
}
