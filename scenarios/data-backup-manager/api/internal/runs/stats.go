package runs

import "sort"

// RunStats is aggregate run performance over the recent run-history window.
// Durations are wall-clock per terminal run; TotalBytes is the logical
// (pre-dedup) total of succeeded outcomes. TotalPhysicalBytes is the on-disk
// (deduped+compressed) repo growth those runs caused — a per-run repo-size
// delta summed over the window — and DedupRatio is logical ÷ physical.
//
// Physical attribution is approximate: it is a repo-size delta measured around
// each run, so concurrent runs writing to the same repo can mis-attribute one
// run's growth to another, and repo maintenance/compaction between the two
// measurements can shrink the repo (negative deltas are clamped to 0). It is an
// observability signal, never presented as an exact accounting of stored bytes.
//
// DedupRatio is computed only over the runs that actually measured physical
// growth (PhysicalBytes > 0) — their logical ÷ their physical — NOT TotalBytes
// ÷ TotalPhysicalBytes. Runs that predate the physical metric (or whose repo
// stats were unavailable) report 0 and would otherwise inflate the ratio by
// putting their logical bytes over a physical figure they never contributed to.
type RunStats struct {
	TotalRuns                int64
	Completed                int64
	PartialFailed            int64
	Failed                   int64
	SuccessRate              float64 // completed ÷ terminal runs, 0..1
	P50DurationMs            int64
	P95DurationMs            int64
	TotalBytes               int64
	AvgBytesPerRun           int64
	AvgThroughputBytesPerSec float64
	TotalPhysicalBytes       int64   // on-disk (deduped) repo growth over the window
	DedupRatio               float64 // logical ÷ physical (0 when physical unknown)
	Window                   int64   // runs the stats were computed over
}

// computeRunStats aggregates a window of runs (any order) into RunStats. It is
// pure (no clock, no I/O) so percentile/throughput math is unit-testable. Only
// terminal runs count toward rates and percentiles; non-terminal runs (an
// in-flight execution) are ignored rather than skewing duration.
func computeRunStats(runs []Run) RunStats {
	st := RunStats{Window: int64(len(runs))}
	durations := make([]int64, 0, len(runs))
	var (
		throughputSum   float64
		throughputCount int64
		terminal        int64
		// Logical bytes of only the runs that measured physical growth — the
		// numerator for an honest dedup ratio (paired with TotalPhysicalBytes).
		measuredLogical int64
	)
	for _, r := range runs {
		switch r.Status {
		case RunCompleted:
			st.Completed++
		case RunPartialFailed:
			st.PartialFailed++
		case RunFailed:
			st.Failed++
		default:
			continue // non-terminal: not counted
		}
		terminal++
		st.TotalRuns++

		var bytes int64
		for _, o := range r.Outcomes {
			if o.Status == OutcomeSucceeded {
				bytes += o.Bytes
			}
		}
		st.TotalBytes += bytes
		if r.PhysicalBytes > 0 {
			st.TotalPhysicalBytes += r.PhysicalBytes
			measuredLogical += bytes
		}

		if !r.StartedAt.IsZero() && r.FinishedAt.After(r.StartedAt) {
			ms := r.FinishedAt.Sub(r.StartedAt).Milliseconds()
			durations = append(durations, ms)
			if ms > 0 && bytes > 0 {
				throughputSum += float64(bytes) / (float64(ms) / 1000.0)
				throughputCount++
			}
		}
	}

	if terminal > 0 {
		st.SuccessRate = float64(st.Completed) / float64(terminal)
		st.AvgBytesPerRun = st.TotalBytes / terminal
	}
	if throughputCount > 0 {
		st.AvgThroughputBytesPerSec = throughputSum / float64(throughputCount)
	}
	if st.TotalPhysicalBytes > 0 {
		st.DedupRatio = float64(measuredLogical) / float64(st.TotalPhysicalBytes)
	}
	st.P50DurationMs = percentile(durations, 50)
	st.P95DurationMs = percentile(durations, 95)
	return st
}

// percentile returns the nearest-rank pth percentile of values (0 if empty).
func percentile(values []int64, p int) int64 {
	n := len(values)
	if n == 0 {
		return 0
	}
	sorted := append([]int64(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	// Nearest-rank: rank = ceil(p/100 * n), 1-based.
	rank := (p*n + 99) / 100
	if rank < 1 {
		rank = 1
	}
	if rank > n {
		rank = n
	}
	return sorted[rank-1]
}
