package trips

import (
	"strings"
	"testing"

	"fall-foliage-explorer/cli/internal/support"
)

// [REQ:REQ-P1-003] Trips CLI renders saved plans with dates, regions, and notes.
func TestTripRows(t *testing.T) {
	rows := tripRows([]support.TripPlan{{
		ID:        4,
		Name:      "Northern loop",
		StartDate: "2025-10-01",
		EndDate:   "2025-10-05",
		Regions:   []int{1, 3, 5},
		Notes:     "Book lodging early",
	}})

	if len(rows) != 1 {
		t.Fatalf("tripRows length = %d, want 1", len(rows))
	}
	for _, want := range []string{"#4", "Northern loop", "2025-10-01", "2025-10-05", "regions=1,3,5", "Book lodging early"} {
		if !strings.Contains(rows[0], want) {
			t.Fatalf("tripRows output %q missing %q", rows[0], want)
		}
	}
}

func TestIntSliceToCSV(t *testing.T) {
	if got := intSliceToCSV([]int{2, 4, 6}); got != "2,4,6" {
		t.Fatalf("intSliceToCSV populated = %q", got)
	}
	if got := intSliceToCSV(nil); got != "(none)" {
		t.Fatalf("intSliceToCSV empty = %q", got)
	}
}
