package evals

import (
	"math"
	"strings"
	"testing"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

func TestFormatAggregateShowsFederatedRatesAndUnsetDirectRates(t *testing.T) {
	routing, recall := 0.75, 0.5
	federated := &evalv1.EvalRun{Aggregate: &evalv1.EvalAggregate{
		Cases: 4, Met: 2, PassRate: 0.5, RoutingPrecision: &routing, RetrievalRecall: &recall,
	}}
	formatted := formatAggregate(federated)
	for _, want := range []string{"pass_rate=0.500", "routing_precision=0.750", "retrieval_recall=0.500"} {
		if !strings.Contains(formatted, want) {
			t.Fatalf("formatted aggregate %q does not contain %q", formatted, want)
		}
	}

	direct := formatAggregate(&evalv1.EvalRun{Aggregate: &evalv1.EvalAggregate{PassRate: 1}})
	if !strings.Contains(direct, "routing_precision=unset") || !strings.Contains(direct, "retrieval_recall=unset") {
		t.Fatalf("direct aggregate did not preserve unset rates: %q", direct)
	}
}

func TestRateDeltaRequiresPresenceOnBothRuns(t *testing.T) {
	if delta, ok := rateDelta(0.4, 0.7, true, true); !ok || math.Abs(delta-0.3) > 1e-9 {
		t.Fatalf("rate delta = %v, %v; want 0.3, true", delta, ok)
	}
	if _, ok := rateDelta(0.4, 0.7, true, false); ok {
		t.Fatal("unset rate pair unexpectedly produced a delta")
	}
}
