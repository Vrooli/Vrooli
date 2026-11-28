package agenttranscript

import (
	"encoding/json"
	"testing"
)

func TestDecodeJSONLRecord(t *testing.T) {
	record, ok := DecodeJSONLRecord([]byte(`{"type":"session_meta","payload":{"id":"thread-1"},"timestamp":1782095815}`))
	if !ok || record.Type != "session_meta" {
		t.Fatalf("record=%+v ok=%v", record, ok)
	}
	var payload struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(record.Payload, &payload); err != nil || payload.ID != "thread-1" {
		t.Fatalf("payload=%+v err=%v", payload, err)
	}
	if got := ParseTimestampValue(record.Timestamp); got.IsZero() {
		t.Fatal("decoded timestamp was not parsed")
	}
	for _, line := range []string{"", "null", `[]`, `"line"`, `not-json`} {
		if _, ok := DecodeJSONLRecord([]byte(line)); ok {
			t.Errorf("DecodeJSONLRecord(%q) accepted invalid record", line)
		}
	}
}
