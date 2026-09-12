package reports

import (
	"strings"
	"testing"

	"fall-foliage-explorer/cli/internal/support"
)

// [REQ:REQ-P1-001] Reports CLI keeps user report summaries compact and photo-aware.
func TestReportRows(t *testing.T) {
	rows := reportRows([]support.UserReport{{
		ID:            12,
		ReportDate:    "2025-10-18",
		FoliageStatus: "peak",
		Description:   "Crimson canopy",
		PhotoURL:      "photo://local",
	}})

	if len(rows) != 1 {
		t.Fatalf("reportRows length = %d, want 1", len(rows))
	}
	for _, want := range []string{"#12", "2025-10-18", "peak", "Crimson canopy", "photo=photo://local"} {
		if !strings.Contains(rows[0], want) {
			t.Fatalf("reportRows output %q missing %q", rows[0], want)
		}
	}
}

func TestReportRowsEmpty(t *testing.T) {
	rows := reportRows(nil)
	if len(rows) != 1 || rows[0] != "No reports for this region" {
		t.Fatalf("reportRows(nil) = %#v", rows)
	}
}
