package trials

import (
	"sort"
	"time"
)

// aggregate groups runs into per-day trend points (success rate, median tokens,
// median duration, run count). Input may be any order; output is oldest-day
// first. Success = VerdictPass over all runs that day (an errored or failed run
// counts against the rate — the trend must reflect real reliability).
func aggregate(runs []TrialRun) []HistoryPoint {
	byDay := map[string][]TrialRun{}
	for _, r := range runs {
		key := r.At.UTC().Format("2006-01-02")
		byDay[key] = append(byDay[key], r)
	}
	keys := make([]string, 0, len(byDay))
	for k := range byDay {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]HistoryPoint, 0, len(keys))
	for _, k := range keys {
		group := byDay[k]
		day, _ := time.Parse("2006-01-02", k)
		passes := 0
		tokens := make([]int64, 0, len(group))
		durations := make([]int64, 0, len(group))
		for _, r := range group {
			if r.Verdict == VerdictPass {
				passes++
			}
			tokens = append(tokens, r.Tokens)
			durations = append(durations, r.DurationMs)
		}
		pt := HistoryPoint{
			At:               day.UTC(),
			RunCount:         len(group),
			MedianTokens:     medianInt64(tokens),
			MedianDurationMs: medianInt64(durations),
		}
		if len(group) > 0 {
			pt.SuccessRate = float64(passes) / float64(len(group))
		}
		out = append(out, pt)
	}
	return out
}

// medianInt64 returns the median of vs (0 for empty). For an even count it
// returns the lower-middle element (deterministic, no float rounding).
func medianInt64(vs []int64) int64 {
	if len(vs) == 0 {
		return 0
	}
	sorted := append([]int64(nil), vs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	return sorted[(len(sorted)-1)/2]
}
