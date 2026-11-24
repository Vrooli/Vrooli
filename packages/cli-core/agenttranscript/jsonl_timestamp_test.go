package agenttranscript

import (
	"strings"
	"testing"
	"time"
)

func TestFirstRFC3339JSONLTimestampSkipsPreambleAndNumericValues(t *testing.T) {
	input := strings.Join([]string{
		``,
		`not json`,
		`{"type":"mode"}`,
		`{"type":"checkpoint","timestamp":1795521600}`,
		`{"type":"user","timestamp":"2026-11-24T17:48:12.123456789Z"}`,
		`{"type":"assistant","timestamp":"2026-11-24T17:49:00Z"}`,
	}, "\n")
	got, ok := FirstRFC3339JSONLTimestamp(strings.NewReader(input), 6, 1024)
	if !ok {
		t.Fatal("timestamp not found")
	}
	want := time.Date(2026, 11, 24, 17, 48, 12, 123456789, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("timestamp = %s, want %s", got, want)
	}
}

func TestFirstRFC3339JSONLTimestampHonorsHeaderBounds(t *testing.T) {
	input := `{"type":"mode"}` + "\n" + `{"type":"user","timestamp":"2026-11-24T17:48:12Z"}`
	if _, ok := FirstRFC3339JSONLTimestamp(strings.NewReader(input), 1, 1024); ok {
		t.Fatal("timestamp beyond maxLines was read")
	}
	if _, ok := FirstRFC3339JSONLTimestamp(strings.NewReader(input), 2, 8); ok {
		t.Fatal("oversized line was accepted")
	}
}
