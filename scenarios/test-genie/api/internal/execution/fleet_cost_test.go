package execution

import (
	"database/sql"
	"testing"
	"time"
)

// Queue latency is the half of "why was my suite slow" that phase timings
// cannot see. It was unmeasurable because the run manager re-stamped its start
// time when a slot opened, erasing the wait.

func TestQueueLatencyIsMeasuredWhenBothTimestampsAreKnown(t *testing.T) {
	requested := sql.NullString{String: "2026-08-20T12:00:00Z", Valid: true}
	started := sql.NullString{String: "2026-08-20T12:00:45Z", Valid: true}
	if got := queueLatencyMs(requested, started); got != 45_000 {
		t.Fatalf("queue latency = %d, want 45000", got)
	}
}

// TestQueueLatencyIsUnknownNotZero is the honesty property. Rows recorded
// before requested_at existed genuinely cannot say how long they waited, and
// reporting 0 would claim measured immediacy for every one of them — dragging
// the fleet median toward zero with fabricated data.
func TestQueueLatencyIsUnknownNotZero(t *testing.T) {
	known := sql.NullString{String: "2026-08-20T12:00:00Z", Valid: true}
	cases := map[string][2]sql.NullString{
		"no requested_at": {{}, known},
		"no started_at":   {known, {}},
		"both absent":     {{}, {}},
		"unparseable":     {{String: "not-a-time", Valid: true}, known},
		// A start before the request is impossible; report unknown rather than
		// a negative latency.
		"negative": {{String: "2026-08-20T12:00:45Z", Valid: true}, {String: "2026-08-20T12:00:00Z", Valid: true}},
	}
	for name, pair := range cases {
		t.Run(name, func(t *testing.T) {
			if got := queueLatencyMs(pair[0], pair[1]); got != -1 {
				t.Fatalf("expected -1 (unknown), got %d", got)
			}
		})
	}
}

func TestQueueLatencyAcceptsHistoricalTimestampFormats(t *testing.T) {
	started := sql.NullString{String: "2026-08-20T12:00:10Z", Valid: true}
	for _, layout := range []string{
		"2026-08-20T12:00:00Z",
		"2026-08-20T12:00:00.000000000Z",
	} {
		if got := queueLatencyMs(sql.NullString{String: layout, Valid: true}, started); got != 10_000 {
			t.Fatalf("layout %q gave %d, want 10000", layout, got)
		}
	}
}

// --- fleet folding ------------------------------------------------------

func TestFoldFleetMergesByPhase(t *testing.T) {
	rows := []CostSummary{
		{
			Scenario: "a", Phase: "unit", ProviderScenario: "unit-health", SampleCount: 10, ReliableSampleCount: 10,
			TotalWallClockMs: 1000, MedianWallClockMs: 100, P90WallClockMs: 200, MaxPeakRSSBytes: 500, CacheHitCount: 2,
			RepeatFailureWallClockMs: 300, RepeatFailureSampleCount: 3, QueueLatencyMedianMs: -1, QueueLatencyP90Ms: -1,
		},
		{
			Scenario: "b", Phase: "unit", ProviderScenario: "unit-health", SampleCount: 10, ReliableSampleCount: 10,
			TotalWallClockMs: 3000, MedianWallClockMs: 300, P90WallClockMs: 400, MaxPeakRSSBytes: 900, CacheHitCount: 8,
			RepeatFailureWallClockMs: 100, RepeatFailureSampleCount: 1, QueueLatencyMedianMs: -1, QueueLatencyP90Ms: -1,
		},
		{
			Scenario: "a", Phase: "security", ProviderScenario: "security-health", SampleCount: 4, ReliableSampleCount: 4,
			TotalWallClockMs: 8000, MedianWallClockMs: 2000, P90WallClockMs: 2500, QueueLatencyMedianMs: -1, QueueLatencyP90Ms: -1,
		},
	}

	folded := FoldFleet(rows)
	if len(folded) != 2 {
		t.Fatalf("expected one row per phase, got %d", len(folded))
	}
	// Ordered by total wall clock, so the most expensive phase leads.
	if folded[0].Phase != "security" {
		t.Fatalf("expected the costliest phase first, got %q", folded[0].Phase)
	}

	var unit CostSummary
	for _, row := range folded {
		if row.Phase == "unit" {
			unit = row
		}
	}
	if unit.TotalWallClockMs != 4000 {
		t.Fatalf("totals must sum: got %d, want 4000", unit.TotalWallClockMs)
	}
	if unit.SampleCount != 20 || unit.CacheHitCount != 10 {
		t.Fatalf("counts must sum: samples=%d hits=%d", unit.SampleCount, unit.CacheHitCount)
	}
	// Peak RSS is the largest single observation, not a sum: the fleet never
	// held 1400 bytes at once.
	if unit.MaxPeakRSSBytes != 900 {
		t.Fatalf("peak RSS = %d, want the maximum 900", unit.MaxPeakRSSBytes)
	}
	// Percentiles are sample-weighted, not summed and not averaged blindly.
	if unit.MedianWallClockMs != 200 {
		t.Fatalf("weighted median = %d, want 200", unit.MedianWallClockMs)
	}
	if unit.CacheHitRatePercent != 50 {
		t.Fatalf("cache hit rate = %.1f, want 50", unit.CacheHitRatePercent)
	}
	if unit.RepeatFailureWallClockMs != 400 || unit.RepeatFailureSampleCount != 4 {
		t.Fatalf("repeat-failure cost must sum: %dms over %d", unit.RepeatFailureWallClockMs, unit.RepeatFailureSampleCount)
	}
	if unit.ProviderScenario != "unit-health" {
		t.Fatalf("provider attribution lost: %q", unit.ProviderScenario)
	}
	if unit.Scenario != "*" {
		t.Fatalf("a folded row must not claim to be one scenario, got %q", unit.Scenario)
	}
}

// TestFoldFleetKeepsQueueLatencyUnknownWhenNoSampleKnowsIt stops the fold from
// inventing a measured zero out of unknowns.
func TestFoldFleetKeepsQueueLatencyUnknownWhenNoSampleKnowsIt(t *testing.T) {
	rows := []CostSummary{
		{Scenario: "a", Phase: "unit", SampleCount: 5, QueueLatencyMedianMs: -1, QueueLatencyP90Ms: -1},
		{Scenario: "b", Phase: "unit", SampleCount: 5, QueueLatencyMedianMs: -1, QueueLatencyP90Ms: -1},
	}
	folded := FoldFleet(rows)
	if folded[0].QueueLatencyMedianMs != -1 {
		t.Fatalf("queue latency = %d, want -1 (unknown)", folded[0].QueueLatencyMedianMs)
	}
}

func TestFoldFleetWeightsKnownQueueLatency(t *testing.T) {
	rows := []CostSummary{
		{Scenario: "a", Phase: "unit", SampleCount: 1, QueueLatencyMedianMs: 1000, QueueLatencyP90Ms: 1000},
		{Scenario: "b", Phase: "unit", SampleCount: 3, QueueLatencyMedianMs: 5000, QueueLatencyP90Ms: 5000},
		// A scenario that cannot report latency must not drag the mean toward 0.
		{Scenario: "c", Phase: "unit", SampleCount: 100, QueueLatencyMedianMs: -1, QueueLatencyP90Ms: -1},
	}
	folded := FoldFleet(rows)
	if got := folded[0].QueueLatencyMedianMs; got != 4000 {
		t.Fatalf("weighted queue latency = %d, want 4000", got)
	}
}

func TestFoldFleetOnEmptyInput(t *testing.T) {
	if got := FoldFleet(nil); len(got) != 0 {
		t.Fatalf("expected no rows, got %d", len(got))
	}
}

func TestCostTimestampParsingIsUTC(t *testing.T) {
	parsed, ok := parseCostTimestamp("2026-08-20T08:00:00-04:00")
	if !ok {
		t.Fatal("expected an offset timestamp to parse")
	}
	if parsed.Location() != time.UTC {
		t.Fatalf("timestamps must normalize to UTC, got %v", parsed.Location())
	}
	if parsed.Hour() != 12 {
		t.Fatalf("offset was not applied: hour = %d", parsed.Hour())
	}
}
