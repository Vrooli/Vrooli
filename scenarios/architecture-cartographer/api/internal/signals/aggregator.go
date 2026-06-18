package signals

import (
	"context"
	"sort"

	"architecture-cartographer/internal/graph"
)

// Aggregator combines per-signal ScoreResults into a Verdict, honoring
// configured weights and the tier thresholds (defaults below,
// overridable via the global control surface).
//
// Weight bookkeeping separates two concerns:
//   - DirectionValue is normalized by scoring signals only, so an abstaining
//     signal never lowers a domain's directional score.
//   - Confidence is the scoring-weight / available-weight participation
//     fraction, so sparse evidence cannot masquerade as high certainty.
//   - Unavailable signals (IsAvailable=false) are silently skipped and do NOT
//     contribute weight; they aren't running.
type Aggregator struct {
	registry         *Registry
	weights          map[string]float64
	autoPlaceMinimum float64
	suggestMinimum   float64
	tieDelta         float64
	quorumHigh       float64
	quorumLow        float64
}

// Default tier thresholds. Per SIGNAL_LADDER.md.
const (
	DefaultAutoPlaceMinimum = 0.85
	DefaultSuggestMinimum   = 0.55
	DefaultTieDelta         = 0.10
	DefaultQuorumHigh       = 0.45
	DefaultQuorumLow        = 0.30
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
		quorumHigh:       DefaultQuorumHigh,
		quorumLow:        DefaultQuorumLow,
	}
}

// WithThresholds returns a copy of the aggregator with custom
// thresholds (from the global control surface).
func (a *Aggregator) WithThresholds(autoPlace, suggest, tieDelta, quorumHigh, quorumLow float64) *Aggregator {
	cp := *a
	cp.autoPlaceMinimum = autoPlace
	cp.suggestMinimum = suggest
	cp.tieDelta = tieDelta
	cp.quorumHigh = quorumHigh
	cp.quorumLow = quorumLow
	return &cp
}

// WithWeightOverrides returns a copy of the aggregator with selected
// signal weights replaced. Used by content-only verdicts to remove
// path-token authority without rebuilding the registry.
func (a *Aggregator) WithWeightOverrides(overrides map[string]float64) *Aggregator {
	cp := *a
	cp.weights = make(map[string]float64, len(a.weights))
	for name, weight := range a.weights {
		cp.weights[name] = weight
	}
	for name, weight := range overrides {
		cp.weights[name] = weight
	}
	return &cp
}

// Aggregate runs every registered + available signal against the chunk
// and returns the verdict.
func (a *Aggregator) Aggregate(ctx context.Context, gctx GraphContext, chunk graph.Chunk) Verdict {
	scores, abstentions, availableWeight, scoringWeight := a.collect(ctx, gctx, chunk)
	domainTotals := sumByDomain(scores, a.weights)
	domainValues := normalizeDomainValues(domainTotals, scoringWeight)
	confidence := participationConfidence(scoringWeight, availableWeight)

	v := Verdict{
		ChunkID:        chunk.ID,
		ChunkPath:      chunk.Path,
		Confidence:     confidence,
		QuorumMet:      confidence >= a.quorumLow,
		Scores:         scores,
		Abstentions:    abstentions,
		DomainValues:   domainValues,
		DirectionValue: 0,
	}

	if len(domainValues) == 0 {
		v.Tier = TierConflict
		return v
	}
	v.TopDomain = domainValues[0].Domain
	v.TopValue = domainValues[0].DirectionValue
	v.DirectionValue = v.TopValue
	if len(domainValues) > 1 {
		v.RunnerUpDomain = domainValues[1].Domain
		v.RunnerUpValue = domainValues[1].DirectionValue
	}

	if v.RunnerUpValue > 0 && v.TopValue-v.RunnerUpValue < a.tieDelta {
		v.Tied = true
		v.Tier = TierConflict
		return v
	}

	switch {
	case v.TopValue >= a.autoPlaceMinimum && v.Confidence >= a.quorumHigh:
		v.Tier = TierAutoPlace
		v.QuorumMet = true
	case v.TopValue >= a.suggestMinimum && v.Confidence >= a.quorumLow:
		v.Tier = TierSuggest
		v.QuorumMet = true
	default:
		v.Tier = TierConflict
		v.QuorumMet = false
	}
	return v
}

// collect runs every available signal, validates the self-explaining
// invariant, and returns scores, abstentions, available weight, and scoring
// weight. Any invoked-and-available signal contributes to available weight;
// only signals with valid scores contribute to scoring weight.
func (a *Aggregator) collect(ctx context.Context, gctx GraphContext, chunk graph.Chunk) ([]Score, []Abstention, float64, float64) {
	var scores []Score
	var abstentions []Abstention
	availableWeight := 0.0
	scoringWeight := 0.0
	for _, s := range a.registry.All() {
		if ok, _ := s.IsAvailable(ctx); !ok {
			continue
		}
		name := s.Name()
		weight := a.weights[name]
		availableWeight += weight
		result := s.Score(ctx, gctx, chunk)
		// Self-explaining invariant: a signal must emit either non-empty
		// Scores (each with ≥1 Evidence) or a non-nil Abstention with
		// ≥1 Evidence. Anything else is a contract violation; we
		// synthesize a diagnostic abstention so the breakage surfaces in
		// the verdict instead of being silently dropped.
		validScores := validScores(result.Scores, name)
		validAbstention := validAbstention(result.Abstention, name)

		switch {
		case len(validScores) > 0:
			scores = append(scores, validScores...)
			scoringWeight += weight
		case validAbstention != nil:
			abstentions = append(abstentions, *validAbstention)
		default:
			abstentions = append(abstentions, Abstention{
				Signal: name,
				Reason: "signal returned empty ScoreResult (broken self-explaining contract)",
				Evidence: []Evidence{{
					Kind:    "broken_contract",
					Summary: "signal " + name + " returned no Scores and no Abstention",
					Locator: chunk.Path,
				}},
			})
		}
	}
	sort.SliceStable(scores, func(i, j int) bool {
		if scores[i].Signal != scores[j].Signal {
			return scores[i].Signal < scores[j].Signal
		}
		return scores[i].Domain < scores[j].Domain
	})
	sort.SliceStable(abstentions, func(i, j int) bool {
		return abstentions[i].Signal < abstentions[j].Signal
	})
	return scores, abstentions, availableWeight, scoringWeight
}

// validScores filters out Scores that lack Evidence (contract
// violation) so they don't contribute to direction. The signal's weight still
// counts toward available participation through the synthetic abstention path.
func validScores(in []Score, signal string) []Score {
	if len(in) == 0 {
		return nil
	}
	out := make([]Score, 0, len(in))
	for _, s := range in {
		if len(s.Evidence) == 0 {
			continue
		}
		if s.Signal == "" {
			s.Signal = signal
		}
		out = append(out, s)
	}
	return out
}

// validAbstention enforces the Reason + ≥1 Evidence contract on an
// emitted Abstention. Returns nil if the abstention is missing either.
func validAbstention(in *Abstention, signal string) *Abstention {
	if in == nil {
		return nil
	}
	if in.Reason == "" || len(in.Evidence) == 0 {
		return nil
	}
	cp := *in
	if cp.Signal == "" {
		cp.Signal = signal
	}
	return &cp
}

func sumByDomain(scores []Score, weights map[string]float64) map[string]float64 {
	totals := make(map[string]float64)
	for _, s := range scores {
		w := weights[s.Signal]
		if w == 0 {
			continue
		}
		totals[s.Domain] += w * s.Value
	}
	return totals
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
		out = append(out, DomainValue{Domain: d, DirectionValue: val})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].DirectionValue != out[j].DirectionValue {
			return out[i].DirectionValue > out[j].DirectionValue
		}
		return out[i].Domain < out[j].Domain
	})
	return out
}

func participationConfidence(scoringWeight, availableWeight float64) float64 {
	if availableWeight <= 0 {
		return 0
	}
	v := scoringWeight / availableWeight
	if v > 1 {
		return 1
	}
	if v < 0 {
		return 0
	}
	return v
}
