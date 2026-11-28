package agenttranscript

import (
	"testing"
	"time"
)

func TestParseTimestamp(t *testing.T) {
	want := time.Date(2026, 7, 22, 2, 36, 55, 108000000, time.UTC)
	if got := ParseTimestamp([]byte(`{"timestamp":"2026-07-22T02:36:55.108Z"}`)); !got.Equal(want) {
		t.Fatalf("RFC3339 timestamp=%s want %s", got, want)
	}
	if got := ParseTimestamp([]byte(`{"timestamp":1782095815108}`)); !got.Equal(time.UnixMilli(1782095815108).UTC()) {
		t.Fatalf("Unix-millisecond timestamp=%s", got)
	}
	if got := ParseTimestamp([]byte(`{"timestamp":1782095815}`)); !got.Equal(time.Unix(1782095815, 0).UTC()) {
		t.Fatalf("Unix-second timestamp=%s", got)
	}
	for _, raw := range []string{`{}`, `not-json`, `{"timestamp":"not-a-time"}`, `{"timestamp":1.2}`} {
		if got := ParseTimestamp([]byte(raw)); !got.IsZero() {
			t.Errorf("ParseTimestamp(%q)=%s want zero", raw, got)
		}
	}
}
