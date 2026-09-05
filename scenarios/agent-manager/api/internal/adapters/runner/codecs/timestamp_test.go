package codecs

import (
	"testing"
	"time"
)

func TestTranscriptLineTimestamp(t *testing.T) {
	want := time.Date(2026, 7, 22, 2, 36, 55, 108000000, time.UTC)
	if got := transcriptLineTimestamp(`{"timestamp":"2026-07-22T02:36:55.108Z"}`); !got.Equal(want) {
		t.Fatalf("RFC3339 timestamp=%s want %s", got, want)
	}
	if got := transcriptLineTimestamp(`{"timestamp":1782095815108}`); got.IsZero() {
		t.Fatal("Unix-millisecond timestamp was not parsed")
	}
}
