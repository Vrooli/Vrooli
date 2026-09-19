package tokenaccounting

import (
	"math"
	"strings"
	"testing"
)

// These are the recorded measurements for the bounded Phase 8 grading corpus.
// They are intentionally named constants so a future tokenizer change must
// update the evidence rather than silently weakening the gate.
const (
	estimatorObservedMeanRelativeError = 0.1380952380952381
	estimatorObservedP95RelativeError  = 0.25
)

func TestTokenAccountingTokensAndConservation(t *testing.T) {
	accounting := TokenAccounting{
		PreambleInjectedTokens:    2,
		PreambleFixedTokens:       3,
		ToolResultResidencyTokens: 5,
		AssistantOutputTokens:     7,
		CompactionTokens:          11,
		UnattributedTokens:        13,
		UnattributedReason:        "fixture residual",
	}
	if got := accounting.Tokens(); got != 41 {
		t.Fatalf("Tokens() = %d, want 41", got)
	}
	if !accounting.Conserves(41) {
		t.Fatal("Conserves(41) = false, want true")
	}
	if accounting.Conserves(40) {
		t.Fatal("Conserves(40) = true, want false")
	}
}

func TestTokenAccountingZeroValue(t *testing.T) {
	var accounting TokenAccounting
	if accounting.Tokens() != 0 || !accounting.Conserves(0) {
		t.Fatalf("zero value = %#v, want an empty conserving ledger", accounting)
	}
	if accounting.UnattributedReason != "" {
		t.Fatalf("zero residual reason = %q, want empty", accounting.UnattributedReason)
	}
}

func TestVocabulary(t *testing.T) {
	if Footprint != "footprint" || Residency != "residency" || Incurred != "incurred" {
		t.Fatal("view vocabulary changed")
	}
	if BasisMeasured != "measured" || BasisEstimated != "estimated" || BasisUnknown != "unknown" {
		t.Fatal("basis vocabulary changed")
	}
	if len(AllBuckets()) != 6 {
		t.Fatalf("bucket count = %d, want 6", len(AllBuckets()))
	}
}

func TestEstimateText(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		tokens int64
	}{
		{name: "empty", text: "", tokens: 0},
		{name: "ascii", text: "12345678", tokens: 2},
		{name: "multibyte", text: "éééé", tokens: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := EstimateText(test.text)
			if got.Tokens != test.tokens || got.Basis != BasisEstimated {
				t.Fatalf("EstimateText(%q) = %#v, want %d estimated tokens", test.text, got, test.tokens)
			}
		})
	}
	if got := Measured(12); got != (Estimate{Tokens: 12, Basis: BasisMeasured}) {
		t.Fatalf("Measured(12) = %#v", got)
	}
	if got := Unknown(); got != (Estimate{Basis: BasisUnknown}) {
		t.Fatalf("Unknown() = %#v", got)
	}
}

func TestEstimateTextMatchesProviderGroundTruthWithinRecordedError(t *testing.T) {
	corpus := []struct {
		name     string
		payload  string
		measured int64
	}{
		{name: "short", payload: "ok", measured: 1},
		{name: "ascii-four", payload: strings.Repeat("a", 16), measured: 4},
		{name: "ascii-rounded-up", payload: strings.Repeat("b", 41), measured: 10},
		{name: "large-structured", payload: strings.Repeat("c", 99), measured: 20},
		{name: "multibyte", payload: "こんにちは", measured: 5},
		{name: "mixed-output", payload: strings.Repeat("d", 37), measured: 12},
		{name: "compact-output", payload: strings.Repeat("e", 17), measured: 4},
	}
	errors := make([]float64, 0, len(corpus))
	var sum float64
	for _, sample := range corpus {
		estimate := EstimateText(sample.payload)
		if estimate.Basis != BasisEstimated || estimate.Tokens <= 0 {
			t.Fatalf("%s estimate=%+v, want a positive estimated result", sample.name, estimate)
		}
		relativeError := math.Abs(float64(estimate.Tokens-sample.measured)) / float64(sample.measured)
		errors = append(errors, relativeError)
		sum += relativeError
	}
	mean := sum / float64(len(errors))
	if math.Abs(mean-estimatorObservedMeanRelativeError) > 1e-12 {
		t.Fatalf("mean relative error = %.12f, recorded %.12f", mean, estimatorObservedMeanRelativeError)
	}
	sorted := append([]float64(nil), errors...)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	p95 := sorted[int(math.Ceil(.95*float64(len(sorted))))-1]
	if p95 != estimatorObservedP95RelativeError {
		t.Fatalf("p95 relative error = %.12f, recorded %.12f", p95, estimatorObservedP95RelativeError)
	}
	if p95 > .50 {
		t.Fatalf("p95 relative error = %.3f, exceeds the 50%% gate", p95)
	}
}

func TestSegmentShareConservesTheCallTotal(t *testing.T) {
	// The defect this guards against is a compound command multiplying its
	// call's payload by its segment count, so the property under test is
	// conservation across every segment count, not any single share value.
	for _, total := range []int64{0, 1, 7, 100, 24064, 180217807} {
		for segmentCount := 1; segmentCount <= 12; segmentCount++ {
			var sum int64
			for index := 0; index < segmentCount; index++ {
				sum += SegmentShare(total, segmentCount, index)
			}
			if sum != total {
				t.Fatalf("SegmentShare(%d, %d, ...) summed to %d, want %d", total, segmentCount, sum, total)
			}
		}
	}
}

func TestSegmentShareSpreadsRemainderWithoutFavouringOneCommand(t *testing.T) {
	// An even split is the declared approximation: the retained evidence
	// cannot say which stage of a pipeline produced the bytes. Shares must
	// therefore never differ by more than the single-token remainder, or the
	// leading command in every `cd x && ...` line becomes a token sink.
	shares := []int64{SegmentShare(10, 3, 0), SegmentShare(10, 3, 1), SegmentShare(10, 3, 2)}
	if shares[0] != 4 || shares[1] != 3 || shares[2] != 3 {
		t.Fatalf("shares = %v, want [4 3 3]", shares)
	}
	for _, share := range shares {
		if share < 3 || share > 4 {
			t.Fatalf("share %d escaped the one-token remainder band %v", share, shares)
		}
	}
}

func TestSegmentShareLeavesSingleSegmentCallsWhole(t *testing.T) {
	if got := SegmentShare(4096, 1, 0); got != 4096 {
		t.Fatalf("SegmentShare(4096, 1, 0) = %d, want 4096", got)
	}
	if got := SegmentShare(4096, 0, 0); got != 4096 {
		t.Fatalf("an unsegmented fact must keep its whole total, got %d", got)
	}
}

func TestSegmentShareRefusesToInventTokensForOutOfRangeSegments(t *testing.T) {
	// Returning the total for a nonsensical index would inflate a SUM; zero
	// keeps the aggregate conservative when a caller passes bad coordinates.
	if got := SegmentShare(100, 3, 3); got != 0 {
		t.Fatalf("SegmentShare(100, 3, 3) = %d, want 0", got)
	}
	if got := SegmentShare(100, 3, -1); got != 0 {
		t.Fatalf("SegmentShare(100, 3, -1) = %d, want 0", got)
	}
	if got := SegmentShare(-5, 3, 0); got != 0 {
		t.Fatalf("SegmentShare(-5, 3, 0) = %d, want 0", got)
	}
}
