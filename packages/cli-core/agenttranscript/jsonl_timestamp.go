package agenttranscript

import (
	"bufio"
	"bytes"
	"io"
	"time"
)

// FirstRFC3339JSONLTimestamp scans a bounded JSONL header and returns the first
// valid string timestamp. Numeric timestamp values are ignored because callers
// use this for native formats whose timestamp contract is RFC3339 text.
func FirstRFC3339JSONLTimestamp(reader io.Reader, maxLines, maxLineBytes int) (time.Time, bool) {
	if reader == nil || maxLines <= 0 || maxLineBytes <= 0 {
		return time.Time{}, false
	}
	initialBuffer := min(maxLineBytes, 64*1024)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, initialBuffer), maxLineBytes)
	for line := 0; line < maxLines && scanner.Scan(); line++ {
		record, ok := DecodeJSONLRecord(scanner.Bytes())
		if !ok {
			continue
		}
		raw := bytes.TrimSpace(record.Timestamp)
		if len(raw) == 0 || raw[0] != '"' {
			continue
		}
		if timestamp := ParseTimestampValue(raw); !timestamp.IsZero() {
			return timestamp, true
		}
	}
	return time.Time{}, false
}
