package agenttranscript

import (
	"bytes"
	"encoding/json"
)

// JSONLRecord is the common outer shape of a native JSON-lines transcript
// record. Payload and Timestamp remain raw so the harness-specific codec can
// decode only the fields it owns.
type JSONLRecord struct {
	Type      string
	Payload   json.RawMessage
	Timestamp json.RawMessage
}

// DecodeJSONLRecord reads the shared envelope fields from one JSONL record.
// It returns false for blank, malformed, or non-object records.
func DecodeJSONLRecord(line []byte) (JSONLRecord, bool) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 || line[0] != '{' {
		return JSONLRecord{}, false
	}
	var wire struct {
		Type      string          `json:"type"`
		Payload   json.RawMessage `json:"payload"`
		Timestamp json.RawMessage `json:"timestamp"`
	}
	if err := json.Unmarshal(line, &wire); err != nil {
		return JSONLRecord{}, false
	}
	return JSONLRecord{Type: wire.Type, Payload: wire.Payload, Timestamp: wire.Timestamp}, true
}
