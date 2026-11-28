package prototime

import (
	"testing"
	"time"
)

func TestOptionalTimestampPreservesOptionalUTCContract(t *testing.T) {
	if got := OptionalTimestamp(time.Time{}); got != nil {
		t.Fatalf("OptionalTimestamp(zero) = %v, want nil", got)
	}

	zone := time.FixedZone("offset", -5*60*60)
	input := time.Date(2026, time.June, 2, 9, 30, 0, 0, zone)
	got := OptionalTimestamp(input)
	if got == nil {
		t.Fatal("OptionalTimestamp(nonzero) = nil, want timestamp")
	}
	if want := input.UTC(); !got.AsTime().Equal(want) || got.AsTime().Location() != time.UTC {
		t.Fatalf("OptionalTimestamp(nonzero) = %v, want %v in UTC", got.AsTime(), want)
	}
}
