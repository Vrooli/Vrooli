package runs

import "sort"

// RunStats is aggregate run performance over the recent run-history window.
// Durations are wall-clock per terminal run; bytes are the logical (pre-dedup)
// total of succeeded outcomes. Deduped/physical bytes and a dedup ratio are
// deliberately absent — kopia's snapshot-create JSON does not expose uploaded
// bytes in this build and repo stats is separately broken (see PROBLEMS.md).
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
	Window                   int64 // runs the stats were computed over
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
