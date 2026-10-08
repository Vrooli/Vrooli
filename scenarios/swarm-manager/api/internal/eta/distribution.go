package eta

import (
	"fmt"
	"math"
	"sort"
)

// Confidence levels for an estimate, weakest to strongest.
const (
	ConfidenceLow    = "low"
	ConfidenceMedium = "medium"
	ConfidenceHigh   = "high"
)

// Basis identifies which rung of the D5 cold-start ladder a distribution rests
// on. The ladder, strongest first: live samples, then backfill samples, then
// operator/default priors.
const (
	BasisLive     = "live"
	BasisBackfill = "backfill"
	BasisPriors   = "priors"
	BasisDefault  = "default"
	// BasisMixed marks an aggregate estimate whose contributing classes rest
	// on different rungs of the ladder.
	BasisMixed = "mixed"
)

// SampleOrigin values, mirroring the eventlog DurationSamplePayload.Origin
// vocabulary. Kept as literals so this package stays pure (no eventlog import).
const (
	OriginLive     = "live"
	OriginBackfill = "backfill"
)

const (
	// minConfidentSamples is the live-sample count at/above which a class is
	// treated as sample-backed with high confidence. Mirrors
	// stats.MinSampleMeaningful — the smallest sample carrying summary signal.
	minConfidentSamples = 5
	// minSigma floors the multiplicative spread so a tight cluster of samples
	// never claims near-zero variance.
	minSigma = 0.25
	// coldStartK scales the small-sample spread inflation: sigma is multiplied
	// by (1 + coldStartK/sqrt(nEff)), so fewer effective samples widen the band.
	coldStartK = 1.5
	// backfillWeight discounts a backfill sample against a live one when
	// counting effective samples for the cold-start inflation.
	backfillWeight = 0.5
	// z80 is the standard-normal quantile at the 80th percentile, used to
	// derive a lognormal distribution's p80 from its (median, sigma).
	z80 = 0.8416212335729143
)

// Sample is one per-effort-class lead-time observation the estimator folds
// into a distribution. Origin is OriginLive or OriginBackfill; backfill samples
// are weighted lower in the cold-start inflation and never lift confidence to
// high on their own.
type Sample struct {
	EffortClass   string
	DurationHours float64
	Origin        string
}

// Prior is a lognormal belief about a class's lead time, used when no samples
// exist. MedianHours is the central hours; Sigma is the multiplicative spread
// (the standard deviation of ln(hours)). Operator priors and the built-in
// defaults share this shape.
type Prior struct {
	MedianHours float64
	Sigma       float64
}

// DefaultPriors returns the built-in per-effort-class priors at the bottom of
// the cold-start ladder. Values are wall-clock lead-time hours (created →
// completed), the same unit as the samples. Spreads are deliberately wide so a
// priors-only band reads as uncertain.
func DefaultPriors() map[string]Prior {
	return map[string]Prior{
		"XS": {MedianHours: 6, Sigma: 0.85},
		"S":  {MedianHours: 18, Sigma: 0.85},
		"M":  {MedianHours: 48, Sigma: 0.90},
		"L":  {MedianHours: 120, Sigma: 0.95},
		"XL": {MedianHours: 300, Sigma: 1.00},
	}
}

// DefaultGlobalPrior is the fallback for unsized items and the global pool.
func DefaultGlobalPrior() Prior { return Prior{MedianHours: 48, Sigma: 1.0} }

// DefaultGateWaitPrior models the extra latency a blocked item incurs waiting
// on a human gate/decision before it becomes actionable. Calibrated forward as
// gate-wait samples accrue (today the gate-wait history is too thin to fit).
func DefaultGateWaitPrior() Prior { return Prior{MedianHours: 24, Sigma: 1.0} }

// Distribution is a lognormal duration model for one effort class (or the
// global/gate-wait pool), tagged with the ladder rung it rests on and how many
// samples back it. Draws come from a lognormal(median, sigma); median and sigma
// are derived from samples when present and from the prior otherwise.
type Distribution struct {
	EffortClass string
	Basis       string
	Confidence  string
	// SampleCount is the number of samples (live + backfill) backing the
	// distribution; zero for priors/default.
	SampleCount int
	// Label is the basis label surfaced to operators: "N samples" when
	// sample-backed, "priors only" otherwise.
	Label string

	median float64
	sigma  float64
}

// BuildDistribution folds a class's samples and prior into a Distribution
// following the D5 ladder. priorIsOperator distinguishes an operator-supplied
// prior (basis "priors") from the built-in default (basis "default"); both
// render the same "priors only" label.
func BuildDistribution(class string, samples []Sample, prior Prior, priorIsOperator bool) Distribution {
	var live, backfill []float64
	for _, s := range samples {
		if s.DurationHours <= 0 {
			continue
		}
		if s.Origin == OriginBackfill {
			backfill = append(backfill, s.DurationHours)
		} else {
			live = append(live, s.DurationHours)
		}
	}
	total := len(live) + len(backfill)

	d := Distribution{EffortClass: class}
	switch {
	case len(live) >= minConfidentSamples:
		d.Basis, d.Confidence = BasisLive, ConfidenceHigh
	case total >= minConfidentSamples:
		// Enough combined signal, but not enough live on its own.
		d.Confidence = ConfidenceMedium
		if len(live) > 0 {
			d.Basis = BasisLive
		} else {
			d.Basis = BasisBackfill
		}
	case total > 0:
		// A handful of samples: usable but low confidence, wide band.
		d.Confidence = ConfidenceLow
		if len(live) > 0 {
			d.Basis = BasisLive
		} else {
			d.Basis = BasisBackfill
		}
	default:
		// No samples: rest on the prior.
		d.Confidence = ConfidenceLow
		if priorIsOperator {
			d.Basis = BasisPriors
		} else {
			d.Basis = BasisDefault
		}
	}

	if total == 0 {
		d.median = prior.MedianHours
		d.sigma = prior.Sigma
		if d.sigma < minSigma {
			d.sigma = minSigma
		}
		d.Label = "priors only"
		return d
	}

	// Sample-backed: prefer live-only once it is confident, otherwise pool live
	// and backfill so backfill fills the cold start.
	hours := live
	if len(live) < minConfidentSamples {
		hours = append(append([]float64(nil), live...), backfill...)
	}
	medianLog, sigmaLog := logStats(hours)
	nEff := float64(len(live)) + backfillWeight*float64(len(backfill))
	inflation := 1.0
	if nEff > 0 {
		inflation = 1.0 + coldStartK/math.Sqrt(nEff)
	}
	sigma := math.Max(sigmaLog, minSigma) * inflation

	d.SampleCount = total
	d.median = math.Exp(medianLog)
	d.sigma = sigma
	d.Label = fmt.Sprintf("%d samples", total)
	return d
}

// P50 returns the distribution's median hours.
func (d Distribution) P50() float64 { return d.median }

// P80 returns the distribution's 80th-percentile hours.
func (d Distribution) P80() float64 { return d.median * math.Exp(z80*d.sigma) }

// draw samples one lognormal duration in hours using the supplied normal
// deviate z (z ~ N(0,1)). Isolating the draw from the RNG keeps the math
// testable.
func (d Distribution) draw(z float64) float64 {
	if d.median <= 0 {
		return 0
	}
	return d.median * math.Exp(d.sigma*z)
}

// logStats returns the median and sample standard deviation of ln(hours) over
// the strictly-positive inputs. It assumes len(hours) > 0.
func logStats(hours []float64) (medianLog, sigma float64) {
	logs := make([]float64, 0, len(hours))
	for _, h := range hours {
		if h > 0 {
			logs = append(logs, math.Log(h))
		}
	}
	if len(logs) == 0 {
		return 0, 0
	}
	sort.Float64s(logs)
	medianLog = medianSorted(logs)
	if len(logs) < 2 {
		return medianLog, 0
	}
	mean := 0.0
	for _, l := range logs {
		mean += l
	}
	mean /= float64(len(logs))
	var ss float64
	for _, l := range logs {
		ss += (l - mean) * (l - mean)
	}
	sigma = math.Sqrt(ss / float64(len(logs)-1))
	return medianLog, sigma
}

// medianSorted returns the median of an already-sorted slice.
func medianSorted(sorted []float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}
