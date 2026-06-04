package aisearch

import (
	"math"
	"sort"
	"time"

	"prompt-manager/store"
)

// DistributionStats summarizes a numeric sample (e.g. returned-count per call).
type DistributionStats struct {
	Count  int     `json:"count"`
	Min    float64 `json:"min"`
	P10    float64 `json:"p10"`
	Median float64 `json:"median"`
	P90    float64 `json:"p90"`
	Max    float64 `json:"max"`
	Mean   float64 `json:"mean"`
}

// ComplexityMetric breaks the headline numbers down by complexity tier so a
// tuner can see whether, say, "architectural" calls trim more than "minor".
type ComplexityMetric struct {
	CallCount      int     `json:"callCount"`
	OverBudgetRate float64 `json:"overBudgetRate"`
	MedianReturned float64 `json:"medianReturned"`
}

// BudgetHogSkill is a skill that is large enough to pressure the budget. It is
// the "what's crowding out smaller skills" signal for tuning.
type BudgetHogSkill struct {
	ID                  string `json:"id"`
	MaxChars            int    `json:"maxChars"`
	Occurrences         int    `json:"occurrences"`
	OverBudgetSightings int    `json:"overBudgetSightings"`
}

// DiscoveryMetricsReport is the aggregate view of discovery telemetry over a
// window. It is the read-side counterpart to the per-call DiscoveryCall records
// and the evidence base for threshold/budget tuning (Phase 4 of the plan).
type DiscoveryMetricsReport struct {
	Since             string            `json:"since"`
	CallCount         int               `json:"callCount"`
	ReturnedCount     DistributionStats `json:"returnedCount"`
	BudgetedCallCount int               `json:"budgetedCallCount"`
	OverBudgetRate    float64           `json:"overBudgetRate"`
	// NearThresholdRate is the fraction of calls (with at least one result)
	// whose lowest returned score sits within nearThresholdEpsilon ABOVE the
	// active threshold. A high rate means results are clustering on the floor,
	// so lowering the threshold would likely admit more — the inference signal
	// used when the clipping probe is off.
	NearThresholdRate float64                     `json:"nearThresholdRate"`
	ProbedCallCount   int                         `json:"probedCallCount"`
	ThresholdClipRate float64                     `json:"thresholdClipRate"`
	ClippedPerProbe   DistributionStats           `json:"clippedPerProbe"`
	PerComplexity     map[string]ComplexityMetric `json:"perComplexity"`
	BudgetHogs        []BudgetHogSkill            `json:"budgetHogs"`
}

// nearThresholdEpsilon is the band above the threshold within which a returned
// score counts as "on the floor" for NearThresholdRate.
const nearThresholdEpsilon = 0.05

// maxBudgetHogs bounds the budget-hog list so the report stays compact.
const maxBudgetHogs = 10

// DiscoveryMetrics reads the call window, optionally filters by type, and
// aggregates the headline discovery-health metrics. Empty typeFilter = all.
func (s *Service) DiscoveryMetrics(window time.Duration, typeFilter string) (*DiscoveryMetricsReport, error) {
	report := &DiscoveryMetricsReport{
		Since:         window.String(),
		PerComplexity: map[string]ComplexityMetric{},
		BudgetHogs:    []BudgetHogSkill{},
	}
	if s.callStore == nil {
		return report, nil
	}
	calls, err := s.callStore.ReadSince(window)
	if err != nil {
		return nil, err
	}
	if typeFilter != "" {
		typeFilter = normalizeDiscoverType(typeFilter)
	}

	returned := []float64{}
	clippedSamples := []float64{}
	budgetedCount := 0
	overBudgetCount := 0
	nearThresholdCount := 0
	withResultsCount := 0

	type complexityAcc struct {
		count    int
		over     int
		budgeted int
		returned []float64
	}
	byComplexity := map[string]*complexityAcc{}

	hogs := map[string]*budgetHogAcc{}

	for _, call := range calls {
		if typeFilter != "" && typeFilter != "all" && normalizeDiscoverType(call.Type) != typeFilter {
			continue
		}
		report.CallCount++
		returned = append(returned, float64(call.ReturnedCount))

		isOver := call.BudgetStatus == "over"
		if call.BudgetStatus != "" {
			budgetedCount++
			if isOver {
				overBudgetCount++
			}
		}

		if call.ReturnedCount > 0 {
			withResultsCount++
			if minScore, ok := minReturnedScore(call.Results); ok {
				if minScore-call.Threshold <= nearThresholdEpsilon {
					nearThresholdCount++
				}
			}
		}

		if call.ClippedBelowThreshold != nil {
			report.ProbedCallCount++
			clippedSamples = append(clippedSamples, float64(*call.ClippedBelowThreshold))
		}

		if call.Complexity != "" {
			acc := byComplexity[call.Complexity]
			if acc == nil {
				acc = &complexityAcc{}
				byComplexity[call.Complexity] = acc
			}
			acc.count++
			acc.returned = append(acc.returned, float64(call.ReturnedCount))
			if call.BudgetStatus != "" {
				acc.budgeted++
				if isOver {
					acc.over++
				}
			}
		}

		for _, r := range call.Results {
			if r.Type == "action" {
				continue
			}
			hog := hogs[r.ID]
			if hog == nil {
				hog = &budgetHogAcc{}
				hogs[r.ID] = hog
			}
			hog.occurrences++
			if r.Chars > hog.maxChars {
				hog.maxChars = r.Chars
			}
			if isOver {
				hog.overBudget++
			}
		}
	}

	report.ReturnedCount = distribution(returned)
	report.BudgetedCallCount = budgetedCount
	if budgetedCount > 0 {
		report.OverBudgetRate = float64(overBudgetCount) / float64(budgetedCount)
	}
	if withResultsCount > 0 {
		report.NearThresholdRate = float64(nearThresholdCount) / float64(withResultsCount)
	}
	report.ClippedPerProbe = distribution(clippedSamples)
	if report.ProbedCallCount > 0 {
		clipped := 0
		for _, c := range clippedSamples {
			if c > 0 {
				clipped++
			}
		}
		report.ThresholdClipRate = float64(clipped) / float64(report.ProbedCallCount)
	}

	for tier, acc := range byComplexity {
		metric := ComplexityMetric{CallCount: acc.count}
		if acc.budgeted > 0 {
			metric.OverBudgetRate = float64(acc.over) / float64(acc.budgeted)
		}
		metric.MedianReturned = distribution(acc.returned).Median
		report.PerComplexity[tier] = metric
	}

	report.BudgetHogs = topBudgetHogs(hogs)
	return report, nil
}

// budgetHogAcc accumulates per-skill size/frequency while scanning calls.
type budgetHogAcc struct {
	maxChars    int
	occurrences int
	overBudget  int
}

func minReturnedScore(results []store.DiscoveryCallResult) (float64, bool) {
	found := false
	min := 0.0
	for _, r := range results {
		if !found || r.Score < min {
			min = r.Score
			found = true
		}
	}
	return min, found
}

func topBudgetHogs(hogs map[string]*budgetHogAcc) []BudgetHogSkill {
	out := make([]BudgetHogSkill, 0, len(hogs))
	for id, hog := range hogs {
		out = append(out, BudgetHogSkill{
			ID:                  id,
			MaxChars:            hog.maxChars,
			Occurrences:         hog.occurrences,
			OverBudgetSightings: hog.overBudget,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].MaxChars != out[j].MaxChars {
			return out[i].MaxChars > out[j].MaxChars
		}
		return out[i].ID < out[j].ID
	})
	if len(out) > maxBudgetHogs {
		out = out[:maxBudgetHogs]
	}
	return out
}

// distribution computes summary stats over a numeric sample. An empty sample
// yields a zero-value DistributionStats.
func distribution(values []float64) DistributionStats {
	stats := DistributionStats{Count: len(values)}
	if len(values) == 0 {
		return stats
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	sum := 0.0
	for _, v := range sorted {
		sum += v
	}
	stats.Min = sorted[0]
	stats.Max = sorted[len(sorted)-1]
	stats.Mean = sum / float64(len(sorted))
	stats.P10 = percentile(sorted, 0.10)
	stats.Median = percentile(sorted, 0.50)
	stats.P90 = percentile(sorted, 0.90)
	return stats
}

// percentile returns the linearly-interpolated p-quantile of a sorted sample.
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	rank := p * float64(len(sorted)-1)
	lo := int(math.Floor(rank))
	hi := int(math.Ceil(rank))
	if lo == hi {
		return sorted[lo]
	}
	frac := rank - float64(lo)
	return sorted[lo]*(1-frac) + sorted[hi]*frac
}
