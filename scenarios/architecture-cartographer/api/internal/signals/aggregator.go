package signals

import (
	"context"
	"sort"

	"architecture-cartographer/internal/graph"
)

// Aggregator combines per-signal Scores into a Verdict, honoring
// manifest-overlaid weights and the tier thresholds. Phase 3 fills in
// the production day-one signals; Phase 2 ships the aggregation core
// + tier dispatch so it can be tested with mocks.FakeSignal.
type Aggregator struct {
	registry         *Registry
	weights          map[string]float64
	autoPlaceMinimum float64
	suggestMinimum   float64
	tieDelta         float64
}

// Default tier thresholds. Per SIGNAL_LADDER.md.
const (
	DefaultAutoPlaceMinimum = 0.85
	DefaultSuggestMinimum   = 0.55
	DefaultTieDelta         = 0.10
)

// NewAggregator constructs the aggregator. weightOverrides apply on
// top of each signal's DefaultWeight().
func NewAggregator(reg *Registry, weightOverrides map[string]float64) *Aggregator {
	w := make(map[string]float64, len(reg.All()))
	for _, s := range reg.All() {
		w[s.Name()] = s.DefaultWeight()
	}
	for name, override := range weightOverrides {
		w[name] = override
	}
	return &Aggregator{
		registry:         reg,
		weights:          w,
		autoPlaceMinimum: DefaultAutoPlaceMinimum,
		suggestMinimum:   DefaultSuggestMinimum,
		tieDelta:         DefaultTieDelta,
	}
}

// WithThresholds returns a copy of the aggregator with custom
// thresholds (per-scenario manifest overlay).
func (a *Aggregator) WithThresholds(autoPlace, suggest, tieDelta float64) *Aggregator {
	cp := *a
	cp.autoPlaceMinimum = autoPlace
	cp.suggestMinimum = suggest
	cp.tieDelta = tieDelta
	return &cp
}

// Aggregate runs every registered + available signal against the chunk
// and returns the verdict.
func (a *Aggregator) Aggregate(ctx context.Context, gctx GraphContext, chunk graph.Chunk) Verdict {
	scores := a.collect(ctx, gctx, chunk)
	domainTotals, weightSum := a.sumByDomain(scores)
	domainValues := normalizeDomainValues(domainTotals, weightSum)

	v := Verdict{
		ChunkID:      chunk.ID,
		ChunkPath:    chunk.Path,
		Scores:       scores,
		DomainValues: domainValues,
	}

	if len(domainValues) == 0 {
		v.Tier = TierConflict
		return v
	}
	v.TopDomain = domainValues[0].Domain
	v.TopValue = domainValues[0].Value
	if len(domainValues) > 1 {
		v.RunnerUpDomain = domainValues[1].Domain
		v.RunnerUpValue = domainValues[1].Value
	}

	if v.RunnerUpValue > 0 && v.TopValue-v.RunnerUpValue < a.tieDelta {
		v.Tied = true
		v.Tier = TierConflict
		return v
	}

	switch {
	case v.TopValue >= a.autoPlaceMinimum:
		v.Tier = TierAutoPlace
	case v.TopValue >= a.suggestMinimum:
		v.Tier = TierSuggest
	default:
		v.Tier = TierConflict
	}
	return v
}

func (a *Aggregator) collect(ctx context.Context, gctx GraphContext, chunk graph.Chunk) []Score {
	var out []Score
	for _, s := range a.registry.All() {
		if ok, _ := s.IsAvailable(ctx); !ok {
			continue
		}
		scores := s.Score(ctx, gctx, chunk)
		for _, sc := range scores {
			if len(sc.Evidence) == 0 {
				// Broken signal: drop the score.
				continue
			}
			out = append(out, sc)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Signal != out[j].Signal {
			return out[i].Signal < out[j].Signal
		}
		return out[i].Domain < out[j].Domain
	})
	return out
}

func (a *Aggregator) sumByDomain(scores []Score) (map[string]float64, float64) {
	totals := make(map[string]float64)
	weightSum := 0.0
	weightsUsed := make(map[string]struct{})
	for _, s := range scores {
		w := a.weights[s.Signal]
		if w == 0 {
			continue
		}
		totals[s.Domain] += w * s.Value
		// Each signal contributes its weight once to the denominator,
		// regardless of how many candidate domains it scored.
		if _, seen := weightsUsed[s.Signal]; !seen {
			weightSum += w
			weightsUsed[s.Signal] = struct{}{}
		}
	}
	return totals, weightSum
}

func normalizeDomainValues(totals map[string]float64, weightSum float64) []DomainValue {
	out := make([]DomainValue, 0, len(totals))
	for d, v := range totals {
		val := 0.0
		if weightSum > 0 {
			val = v / weightSum
			if val > 1 {
				val = 1
			}
			if val < 0 {
				val = 0
			}
		}
		out = append(out, DomainValue{Domain: d, Value: val})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Value != out[j].Value {
			return out[i].Value > out[j].Value
		}
		return out[i].Domain < out[j].Domain
	})
	return out
}
