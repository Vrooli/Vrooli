package measures

import (
	"context"
	"testing"
	"time"

	measurelib "github.com/vrooli/measures-go"
)

type fakeMetrics struct{ aggregate Aggregate }

func (f fakeMetrics) Aggregate(context.Context, time.Time, time.Time) (Aggregate, error) {
	return f.aggregate, nil
}

func TestRegistryComputesDeclaredMeasuresFromOneAggregate(t *testing.T) {
	registry, err := declarationRegistry(fakeMetrics{aggregate: Aggregate{
		TerminalExecutions: 10, SuccessfulExecutions: 9, DurationP95Ms: 1234.5,
		StepCount: 20, FailedSteps: 2, SelectorTraces: 8, FailedSelectors: 1,
	}}, func() time.Time { return time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		ExecutionSuccessRate: "0.9",
		ExecutionDurationP95: "1234.5",
		StepFailureRate:      "0.1",
		SelectorFailureRate:  "0.125",
	}
	for name, want := range cases {
		got, err := registry.Execute(context.Background(), measurelib.MeasureRequest{Measure: name, Params: map[string]string{"window": "last_7d"}})
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if got.Value != want {
			t.Errorf("%s: got %q want %q", name, got.Value, want)
		}
		if got.Provenance.ExecutedQuery == "" || got.Provenance.ComputedAt.IsZero() {
			t.Errorf("%s: missing provenance: %+v", name, got.Provenance)
		}
	}
}
