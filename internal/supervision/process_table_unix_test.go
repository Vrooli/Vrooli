//go:build linux || darwin

package supervision

import (
	"testing"
	"time"
)

func TestParseProcessStartTableGivenPSOutputThenReturnsPortableStartTimes(t *testing.T) {
	location := time.FixedZone("test", -4*60*60)
	processes, err := parseProcessStartTable("  4242 Fri Aug 29 00:12:34 2026\n", location)
	if err != nil {
		t.Fatalf("parseProcessStartTable: %v", err)
	}
	want := time.Date(2026, 8, 29, 0, 12, 34, 0, location)
	if got := processes[4242].StartedAt; !got.Equal(want) {
		t.Fatalf("StartedAt = %v, want %v", got, want)
	}
}
