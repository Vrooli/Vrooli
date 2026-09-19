package storagetime

import (
	"testing"
	"time"
)

func TestFormatUTC(t *testing.T) {
	zone := time.FixedZone("east", -5*60*60)
	got := FormatUTC(time.Date(2026, 8, 27, 12, 34, 56, 123456789, zone))
	if got != "2026-08-27T17:34:56.123456789Z" {
		t.Fatalf("FormatUTC() = %q", got)
	}
}

func TestFormatOptionalUTC(t *testing.T) {
	if got := FormatOptionalUTC(nil); got != nil {
		t.Fatalf("nil timestamp = %#v", got)
	}
	zero := time.Time{}
	if got := FormatOptionalUTC(&zero); got != nil {
		t.Fatalf("zero timestamp = %#v", got)
	}
	value := time.Date(2026, 8, 27, 0, 0, 0, 1, time.UTC)
	if got := FormatOptionalUTC(&value); got != "2026-08-27T00:00:00.000000001Z" {
		t.Fatalf("timestamp = %#v", got)
	}
}

func TestFormatUTCIsFixedWidthAcrossNanosecondBoundaries(t *testing.T) {
	first := FormatUTC(time.Date(2026, 1, 1, 0, 0, 0, 9, time.UTC))
	second := FormatUTC(time.Date(2026, 1, 1, 0, 0, 0, 10, time.UTC))
	if first >= second {
		t.Fatalf("timestamps are not lexically sortable: %q >= %q", first, second)
	}
}
