package metrics

import (
	"strings"
	"testing"

	metricspb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics"
)

func TestProcessTimelineRowsShowSwapValueAndCoverage(t *testing.T) {
	rows := processTimelineRows([]*metricspb.ProcessTimelineEntry{{
		Owner: "unknown", Comm: "worker", SwapKb: 2048,
		SwapMeasuredSamples: 4, SwapUnknownSamples: 1, SampleCount: 5,
	}})
	if len(rows) != 1 {
		t.Fatalf("row count = %d, want 1", len(rows))
	}
	for _, want := range []string{"mean_swap=2.0MB", "4 measured/1 unknown", "samples=5"} {
		if !strings.Contains(rows[0], want) {
			t.Errorf("row %q does not contain %q", rows[0], want)
		}
	}
}
