package procsampler

import "sort"

// RankPagingContributors returns up to limit processes ordered by paging
// activity: measured fault rate first, then swapped memory. Processes with
// neither a positive measured fault rate nor measured swapped memory are not
// contributors and are excluded, so an idle host yields an empty slice
// rather than a list of arbitrary processes.
func RankPagingContributors(samples []ProcessSample, limit int) []ProcessSample {
	faultRate := func(s ProcessSample) float64 {
		if s.FaultStatus != FaultMeasured {
			return 0
		}
		return s.MajorFaultsPerSecond
	}
	swapKB := func(s ProcessSample) int64 {
		if s.SwapStatus != "measured" {
			return 0
		}
		return s.SwapKB
	}
	candidates := make([]ProcessSample, 0, len(samples))
	for _, sample := range samples {
		if faultRate(sample) > 0 || swapKB(sample) > 0 {
			candidates = append(candidates, sample)
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if a, b := faultRate(candidates[i]), faultRate(candidates[j]); a != b {
			return a > b
		}
		if a, b := swapKB(candidates[i]), swapKB(candidates[j]); a != b {
			return a > b
		}
		return candidates[i].PID < candidates[j].PID
	})
	if limit > 0 && len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}
