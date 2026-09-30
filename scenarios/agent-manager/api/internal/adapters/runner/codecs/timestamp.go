package codecs

import (
	"bytes"
	"encoding/json"
	"strconv"
	"time"
)

// transcriptLineTimestamp reads only the vendor timestamp. Codex, Claude,
// and Grok use RFC3339 strings; OpenCode uses Unix milliseconds (with seconds
// accepted for older exports).
func transcriptLineTimestamp(line string) time.Time {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(line), &envelope); err != nil {
		return time.Time{}
	}
	raw, ok := envelope["timestamp"]
	if !ok {
		return time.Time{}
	}
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
