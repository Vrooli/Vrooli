package budgets

import (
	"fmt"
	"sort"
	"time"
)

// LatencyObservation is one completed execution measurement. Status and thermal
// state are mandatory strata; binding and model phases must be captured so a
// slow dependency can be distinguished from runtime overhead.
type LatencyObservation struct {
	Status       string
	ThermalState string
	Total        time.Duration
	BindingPhase time.Duration
	ModelPhase   time.Duration
}

// LatencyEvidencePolicy states the minimum evidence needed before a p95 may be
// used to propose a deadline. The default requires every status/thermal stratum
// to contain at least 20 observations. It does not itself change any timeout.
type LatencyEvidencePolicy struct {
	MinSamplesPerStratum int
}

// StratifiedLatencySummary is returned only when evidence passes the quality
// gate. P95 values use nearest-rank quantiles and remain split by status and
// thermal state; PooledP95 is descriptive and must not replace the strata.
type StratifiedLatencySummary struct {
	Counts    map[string]int
	P95       map[string]time.Duration
	PooledP95 time.Duration
}

var latencyStatuses = []string{"success", "failure"}
var latencyThermalStates = []string{"cold", "warm"}

func DefaultLatencyEvidencePolicy() LatencyEvidencePolicy {
	return LatencyEvidencePolicy{MinSamplesPerStratum: 20}
}

// QualifyLatencyEvidence fails closed when a stratum is absent/undersampled,
// a label is unknown, or phase measurements are missing. This prevents a
// mixed-status or cold/warm sample from silently authorizing a p95 deadline.
func QualifyLatencyEvidence(samples []LatencyObservation, policy LatencyEvidencePolicy) (StratifiedLatencySummary, error) {
	if policy.MinSamplesPerStratum < 1 {
		return StratifiedLatencySummary{}, fmt.Errorf("minimum samples per stratum must be positive")
	}
	values := make(map[string][]time.Duration, 4)
	all := make([]time.Duration, 0, len(samples))
	for i, sample := range samples {
		if sample.Status != "success" && sample.Status != "failure" {
			return StratifiedLatencySummary{}, fmt.Errorf("sample %d has unknown status %q", i, sample.Status)
		}
		if sample.ThermalState != "cold" && sample.ThermalState != "warm" {
			return StratifiedLatencySummary{}, fmt.Errorf("sample %d has unknown thermal state %q", i, sample.ThermalState)
		}
		if sample.Total <= 0 || sample.BindingPhase <= 0 || sample.ModelPhase <= 0 {
			return StratifiedLatencySummary{}, fmt.Errorf("sample %d lacks positive total, binding, or model phase timing", i)
		}
		key := sample.Status + "/" + sample.ThermalState
		values[key] = append(values[key], sample.Total)
		all = append(all, sample.Total)
	}
	summary := StratifiedLatencySummary{Counts: make(map[string]int, 4), P95: make(map[string]time.Duration, 4)}
	for _, status := range latencyStatuses {
		for _, thermal := range latencyThermalStates {
			key := status + "/" + thermal
			stratum := values[key]
			summary.Counts[key] = len(stratum)
			if len(stratum) < policy.MinSamplesPerStratum {
				return StratifiedLatencySummary{}, fmt.Errorf("latency evidence stratum %s has %d samples; need at least %d", key, len(stratum), policy.MinSamplesPerStratum)
			}
			summary.P95[key] = nearestRankP95(stratum)
		}
	}
	summary.PooledP95 = nearestRankP95(all)
	return summary, nil
}

func nearestRankP95(values []time.Duration) time.Duration {
	ordered := append([]time.Duration(nil), values...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	index := (95*len(ordered)+99)/100 - 1
	return ordered[index]
}
