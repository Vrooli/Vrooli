package eval

import (
	"testing"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

func TestFederatedRatesPreserveMeasuredZeroAndUnset(t *testing.T) {
	tests := []struct {
		name      string
		results   []*evalv1.CaseResult
		precision *float64
		recall    *float64
	}{
		{
			name: "all routed and recalled",
			results: []*evalv1.CaseResult{
				{CaseId: "a", Outcome: "met", ProviderRouted: true, ExpectedRank: 1},
				{CaseId: "b", Outcome: "thin_margin", ProviderRouted: true, ExpectedRank: 2},
			},
			precision: float64p(1), recall: float64p(1),
		},
		{
			name: "all routed and none recalled",
			results: []*evalv1.CaseResult{
				{CaseId: "a", Outcome: "below_expectation", ProviderRouted: true},
				{CaseId: "b", Outcome: "below_expectation", ProviderRouted: true, ExpectedRank: 4},
			},
			precision: float64p(1), recall: float64p(0),
		},
		{
			name: "none routed",
			results: []*evalv1.CaseResult{
				{CaseId: "a", Outcome: "misrouted"},
				{CaseId: "b", Outcome: "answered_by_sibling"},
			},
			precision: float64p(0),
		},
		{
			name: "mixed routed and retrieval",
			results: []*evalv1.CaseResult{
				{CaseId: "a", Outcome: "met", ProviderRouted: true, ExpectedRank: 1},
				{CaseId: "b", Outcome: "below_expectation", ProviderRouted: true, ExpectedRank: 3},
				{CaseId: "c", Outcome: "misrouted"},
			},
			precision: float64p(2.0 / 3.0), recall: float64p(0.5),
		},
		{
			name:    "zero gradeable cases",
			results: []*evalv1.CaseResult{{CaseId: "a", Outcome: "n/a"}},
		},
		{
			name: "every case degraded",
			results: []*evalv1.CaseResult{
				{CaseId: "a", Outcome: "degraded", ProviderRouted: true, ExpectedRank: 1},
				{CaseId: "b", Outcome: "unavailable"},
			},
		},
		{
			name: "negative cases excluded from retrieval recall",
			results: []*evalv1.CaseResult{
				{CaseId: "negative", Outcome: "met", ProviderRouted: true, ExpectedRank: 0},
				{CaseId: "positive", Outcome: "met", ProviderRouted: true, ExpectedRank: 1},
			},
			precision: float64p(1), recall: float64p(1),
		},
	}

	suite := &evalv1.EvalSuite{Cases: []*evalv1.EvalCase{
		{CaseId: "a", ExpectWithinTopK: 1},
		{CaseId: "b", ExpectWithinTopK: 2},
		{CaseId: "c", ExpectWithinTopK: 1},
		{CaseId: "negative", ExpectNoStrongHit: true, ExpectMaxScore: 0.3},
		{CaseId: "positive", ExpectWithinTopK: 1, ExpectIds: []string{"wanted"}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			precision, recall := federatedRates(suite, tt.results)
			assertOptionalRate(t, "routing_precision", tt.precision, precision)
			assertOptionalRate(t, "retrieval_recall", tt.recall, recall)
		})
	}
}

func assertOptionalRate(t *testing.T, name string, want, got *float64) {
	t.Helper()
	if (want == nil) != (got == nil) {
		t.Fatalf("%s presence = %v, want %v", name, got != nil, want != nil)
	}
	if want != nil && *want != *got {
		t.Fatalf("%s = %v, want %v", name, *got, *want)
	}
}

func float64p(value float64) *float64 { return &value }
