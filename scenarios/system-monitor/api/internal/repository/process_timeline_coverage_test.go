package repository

import (
	"testing"
	"time"
)

func TestProcessTimelineAccumulatorTracksSwapCoverageSeparatelyFromZero(t *testing.T) {
	start := time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)
	acc := NewProcessTimelineAccumulator()
	acc.AddRaw("desktop-tool:monitor", "gnome-system-mo", 10, 1, 10, 1024, 321, true, 0, start)
	acc.AddRaw("desktop-tool:monitor", "gnome-system-mo", 10, 1, 10, 1024, 99999, false, 0, start.Add(time.Minute))

	entries := acc.Entries(10, "rss")
	if len(entries) != 1 {
		t.Fatalf("entry count = %d, want 1", len(entries))
	}
	got := entries[0]
	if got.SwapKB != 321 || got.SwapMeasuredSamples != 1 || got.SwapUnknownSamples != 1 {
		t.Fatalf("swap value and coverage = %+v", got)
	}
	if !got.SwapCoverageStart.Equal(start) || !got.SwapCoverageEnd.Equal(start) {
		t.Fatalf("swap coverage window = [%s, %s], want [%s, %s]", got.SwapCoverageStart, got.SwapCoverageEnd, start, start)
	}
}

func TestProcessTimelineSwapRankSurfacesLargestMeasuredSwapConsumers(t *testing.T) {
	start := time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)
	acc := NewProcessTimelineAccumulator()
	acc.AddRaw("scenario-small", "small", 1, 5, 1, 4096, 32, true, 0, start)
	acc.AddRaw("unknown", "large", 2, 0, 0, 4096, 2048, true, 0, start)

	got := acc.Entries(1, "swap")
	if len(got) != 1 || got[0].Comm != "large" || got[0].SwapKB != 2048 {
		t.Fatalf("swap-ranked entries = %+v, want largest swap consumer", got)
	}
}

func TestProcessTimelineAccumulatorCarriesSwapCountsThroughRollups(t *testing.T) {
	minute := time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC)
	acc := NewProcessTimelineAccumulator()
	acc.AddRollup("scenario-a", "worker", 1, 2, 30, 1024, 512, 3, 2, 1, 0, minute)

	got := acc.Entries(10, "cpu_seconds")
	if len(got) != 1 || got[0].SwapMeasuredSamples != 2 || got[0].SwapUnknownSamples != 1 {
		t.Fatalf("rollup swap coverage = %+v", got)
	}
	if !got[0].SwapCoverageStart.Equal(minute) || !got[0].SwapCoverageEnd.Equal(minute.Add(time.Minute)) {
		t.Fatalf("rollup swap window = [%s, %s]", got[0].SwapCoverageStart, got[0].SwapCoverageEnd)
	}
}
