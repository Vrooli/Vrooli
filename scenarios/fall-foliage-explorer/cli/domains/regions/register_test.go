package regions

import (
	"strings"
	"testing"

	"fall-foliage-explorer/cli/internal/support"
)

// [REQ:REQ-P0-007] Regions CLI formats API region records with location and peak metadata.
func TestRegionRows(t *testing.T) {
	elevation := 1200
	week := 42
	rows := regionRows([]support.Region{{
		ID:              5,
		Name:            "Blue Ridge Parkway",
		State:           "Virginia",
		Country:         "USA",
		Latitude:        37.5615,
		Longitude:       -79.3553,
		ElevationMeters: &elevation,
		TypicalPeakWeek: &week,
	}})

	if len(rows) != 1 {
		t.Fatalf("regionRows length = %d, want 1", len(rows))
	}
	for _, want := range []string{"5. Blue Ridge Parkway, Virginia", "elev=1200m", "typical peak=w42"} {
		if !strings.Contains(rows[0], want) {
			t.Fatalf("regionRows output %q missing %q", rows[0], want)
		}
	}
}

func TestRegionRowsEmpty(t *testing.T) {
	rows := regionRows(nil)
	if len(rows) != 1 || rows[0] != "No regions available" {
		t.Fatalf("regionRows(nil) = %#v", rows)
	}
}
