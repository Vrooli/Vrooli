package eta

import (
	"math"
	"strings"
	"testing"
)

func liveSamples(class string, hours ...float64) []Sample {
	out := make([]Sample, 0, len(hours))
	for _, h := range hours {
		out = append(out, Sample{EffortClass: class, DurationHours: h, Origin: OriginLive})
	}
	return out
}

func backfillSamples(class string, hours ...float64) []Sample {
	out := make([]Sample, 0, len(hours))
	for _, h := range hours {
		out = append(out, Sample{EffortClass: class, DurationHours: h, Origin: OriginBackfill})
	}
	return out
}

func TestBuildDistribution_PriorsOnlyColdStart(t *testing.T) {
	prior := DefaultPriors()["M"]
	d := BuildDistribution("M", nil, prior, false)

	if d.Basis != BasisDefault {
		t.Errorf("basis = %q, want %q", d.Basis, BasisDefault)
	}
	if d.Label != "priors only" {
		t.Errorf("label = %q, want %q", d.Label, "priors only")
	}
	if d.Confidence != ConfidenceLow {
		t.Errorf("confidence = %q, want low", d.Confidence)
	}
	if d.SampleCount != 0 {
		t.Errorf("sample count = %d, want 0", d.SampleCount)
	}
	if d.P50() != prior.MedianHours {
		t.Errorf("p50 = %v, want prior median %v", d.P50(), prior.MedianHours)
	}
	if d.P80() <= d.P50() {
		t.Errorf("p80 %v must exceed p50 %v", d.P80(), d.P50())
	}
}

func TestBuildDistribution_OperatorPriorBasis(t *testing.T) {
	d := BuildDistribution("M", nil, Prior{MedianHours: 40, Sigma: 0.7}, true)
	if d.Basis != BasisPriors {
		t.Errorf("basis = %q, want %q (operator prior)", d.Basis, BasisPriors)
	}
	if d.Label != "priors only" {
		t.Errorf("label = %q, want %q", d.Label, "priors only")
	}
}

func TestBuildDistribution_LadderDegradesLabelAndNarrows(t *testing.T) {
	prior := DefaultPriors()["M"]
	priorsOnly := BuildDistribution("M", nil, prior, false)

	// A confident cluster of live samples near 40h.
	live := liveSamples("M", 38, 40, 42, 39, 41, 40)
	sampled := BuildDistribution("M", live, prior, false)

	if sampled.Basis != BasisLive {
		t.Errorf("basis = %q, want %q with %d live samples", sampled.Basis, BasisLive, len(live))
	}
	if sampled.Confidence != ConfidenceHigh {
		t.Errorf("confidence = %q, want high with %d live samples", sampled.Confidence, len(live))
	}
	wantLabel := "6 samples"
	if sampled.Label != wantLabel {
		t.Errorf("label = %q, want %q", sampled.Label, wantLabel)
	}

	// The band must narrow as samples replace priors: compare the p80/p50 ratio.
	priorSpread := priorsOnly.P80() / priorsOnly.P50()
	sampledSpread := sampled.P80() / sampled.P50()
	if !(sampledSpread < priorSpread) {
		t.Errorf("sample-backed spread %.3f must be tighter than priors-only spread %.3f", sampledSpread, priorSpread)
	}
}

func TestBuildDistribution_FewSamplesLowConfidence(t *testing.T) {
	d := BuildDistribution("L", liveSamples("L", 100, 140), DefaultPriors()["L"], false)
	if d.Basis != BasisLive {
		t.Errorf("basis = %q, want live", d.Basis)
	}
	if d.Confidence != ConfidenceLow {
		t.Errorf("confidence = %q, want low for 2 samples", d.Confidence)
	}
	if d.SampleCount != 2 {
		t.Errorf("sample count = %d, want 2", d.SampleCount)
	}
}

func TestBuildDistribution_BackfillOnlyMediumWhenEnough(t *testing.T) {
	d := BuildDistribution("S", backfillSamples("S", 10, 12, 14, 16, 18, 20), DefaultPriors()["S"], false)
	if d.Basis != BasisBackfill {
		t.Errorf("basis = %q, want backfill", d.Basis)
	}
	if d.Confidence != ConfidenceMedium {
		t.Errorf("confidence = %q, want medium for 6 backfill samples", d.Confidence)
	}
	if !strings.HasSuffix(d.Label, "samples") {
		t.Errorf("label = %q, want a sample count", d.Label)
	}
}

func TestBuildDistribution_LiveOutranksBackfill(t *testing.T) {
	// 3 live + 4 backfill: enough combined for medium, and live presence keeps
	// the basis on the live rung.
	samples := append(liveSamples("M", 30, 40, 50), backfillSamples("M", 20, 60, 45, 55)...)
	d := BuildDistribution("M", samples, DefaultPriors()["M"], false)
	if d.Basis != BasisLive {
		t.Errorf("basis = %q, want live (live present)", d.Basis)
	}
	if d.Confidence != ConfidenceMedium {
		t.Errorf("confidence = %q, want medium", d.Confidence)
	}
	if d.SampleCount != 7 {
		t.Errorf("sample count = %d, want 7", d.SampleCount)
	}
}

func TestDistribution_P50P80Ordering(t *testing.T) {
	d := BuildDistribution("M", liveSamples("M", 30, 45, 60, 40, 50, 35), DefaultPriors()["M"], false)
	if d.P80() < d.P50() {
		t.Errorf("p80 %v must be >= p50 %v", d.P80(), d.P50())
	}
}

func TestLogStats_SingleSampleZeroSigma(t *testing.T) {
	medianLog, sigma := logStats([]float64{math.E})
	if math.Abs(medianLog-1) > 1e-9 {
		t.Errorf("medianLog = %v, want 1", medianLog)
	}
	if sigma != 0 {
		t.Errorf("sigma = %v, want 0 for a single sample", sigma)
	}
}
