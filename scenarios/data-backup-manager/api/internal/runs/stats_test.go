package runs

import (
	"testing"
	"time"
)

func run(status RunStatus, startSec, durSec int, succeededBytes ...int64) Run {
	start := time.Date(2026, 6, 3, 0, 0, startSec, 0, time.UTC)
	r := Run{Status: status, StartedAt: start, FinishedAt: start.Add(time.Duration(durSec) * time.Second)}
	for _, b := range succeededBytes {
		r.Outcomes = append(r.Outcomes, TargetOutcome{Status: OutcomeSucceeded, Bytes: b})
	}
	return r
}

func TestComputeRunStats(t *testing.T) {
	runs := []Run{
		run(RunCompleted, 0, 2, 100, 100),             // dur 2s, 200 bytes
		run(RunCompleted, 10, 4, 400),                 // dur 4s, 400 bytes
		run(RunPartialFailed, 20, 6, 600),             // dur 6s, 600 bytes (counts toward total/terminal)
		run(RunFailed, 30, 8),                         // dur 8s, 0 bytes
		{Status: RunCapturing, StartedAt: time.Now()}, // non-terminal: ignored
	}

	st := computeRunStats(runs)

	if st.Window != 5 {
		t.Errorf("window = %d, want 5 (all rows)", st.Window)
	}
	if st.TotalRuns != 4 {
		t.Errorf("total terminal runs = %d, want 4", st.TotalRuns)
	}
	if st.Completed != 2 || st.PartialFailed != 1 || st.Failed != 1 {
		t.Errorf("counts = c%d p%d f%d, want 2/1/1", st.Completed, st.PartialFailed, st.Failed)
	}
	if st.SuccessRate != 0.5 {
		t.Errorf("success rate = %v, want 0.5", st.SuccessRate)
	}
	if st.TotalBytes != 1200 {
		t.Errorf("total bytes = %d, want 1200", st.TotalBytes)
	}
	if st.AvgBytesPerRun != 300 {
		t.Errorf("avg bytes/run = %d, want 300 (1200/4)", st.AvgBytesPerRun)
	}
	// Durations sorted: [2000, 4000, 6000, 8000]ms. Nearest-rank:
	// p50 → rank ceil(.5*4)=2 → 4000; p95 → rank ceil(.95*4)=4 → 8000.
	if st.P50DurationMs != 4000 {
		t.Errorf("p50 = %d, want 4000", st.P50DurationMs)
	}
	if st.P95DurationMs != 8000 {
		t.Errorf("p95 = %d, want 8000", st.P95DurationMs)
	}
	// Throughput averaged over runs with dur>0 AND bytes>0: 100, 100, 100 → 100.
	// (200B/2s, 400B/4s, 600B/6s = 100 B/s each; the 0-byte failed run excluded.)
	if st.AvgThroughputBytesPerSec != 100 {
		t.Errorf("avg throughput = %v, want 100 B/s", st.AvgThroughputBytesPerSec)
	}
}

func TestComputeRunStats_Empty(t *testing.T) {
	st := computeRunStats(nil)
	if st.TotalRuns != 0 || st.SuccessRate != 0 || st.P50DurationMs != 0 {
		t.Errorf("empty stats not zero-valued: %+v", st)
	}
}
