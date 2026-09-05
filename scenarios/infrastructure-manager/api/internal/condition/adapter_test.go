package condition

import (
	"testing"
	"time"
)

func TestHeadroomPeerAvailabilityNamesStorageManager(t *testing.T) {
	checkedAt := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	availability := PeerSourceAvailability("headroom", checkedAt)
	for _, source := range availability {
		if source.Source != "storage-manager" {
			continue
		}
		if !source.Available {
			t.Fatalf("storage-manager availability = %#v, want configured", source)
		}
		if source.CheckedAt != checkedAt {
			t.Fatalf("checked_at = %v, want %v", source.CheckedAt, checkedAt)
		}
		return
	}
	t.Fatal("storage-manager source availability was not returned")
}

func TestHeadroomSourceIDIsStorageManager(t *testing.T) {
	if got := sourceID("headroom"); got != "storage-manager" {
		t.Fatalf("sourceID(headroom) = %q, want storage-manager", got)
	}
}
