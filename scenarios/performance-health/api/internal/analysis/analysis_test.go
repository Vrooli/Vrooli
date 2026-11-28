package analysis

import (
	"context"
	"testing"

	"performance-health/internal/perfsample"
)

func TestPopulateSampleCopiesNavigationAndAggregatesInteractionMetrics(t *testing.T) {
	result := Result{
		ResponseEndMs: 101, DOMInteractiveMs: 203, DOMContentLoadedMs: 307, LoadEventEndMs: 409,
		NavigationType: "navigate",
		LongTaskMs:     37, LongTaskMaxMs: 22.5,
		FrameSummary: FrameSummary{ApproxDrawnFPS: 55.5, DroppedFrameRate: 0.125},
		InputEvents:  []EventSummary{{Name: "pointermove", Count: 3}, {Name: "keydown", Count: 2}},
		BrowserWork: []EventSummary{
			{Name: "RasterTask", TotalMs: 4.5}, {Name: "RasterTask", TotalMs: 1.5},
			{Name: "Layout", TotalMs: 7}, {Name: "UpdateLayoutTree", TotalMs: 2.25},
			{Name: "Paint", TotalMs: 3.75},
		},
		Components: []ComponentTiming{{Component: "slower", AvgMs: 8, MaxMs: 20}, {Component: "fast", AvgMs: 3, MaxMs: 4}},
	}
	sample := perfsample.Sample{Scenario: "keep-scenario", Flow: "keep-flow", Note: "analysis"}

	PopulateSample(&sample, result)

	want := perfsample.Sample{
		Scenario: "keep-scenario", Flow: "keep-flow", Note: "analysis",
		ResponseEndMs: 101, DOMInteractiveMs: 203, DOMContentLoadedMs: 307, LoadEventEndMs: 409,
		NavigationType: "navigate", DrawnFPS: 55.5, DroppedFrameRate: 0.125,
		LongTaskTotalMs: 37, LongTaskMaxMs: 22.5, InputEventCount: 5,
		RasterTotalMs: 6, LayoutTotalMs: 9.25, PaintTotalMs: 3.75,
		SlowestComponent: "slower", SlowestComponentAvgMs: 8, SlowestComponentMaxMs: 20,
	}
	if sample != want {
		t.Fatalf("PopulateSample() = %#v, want %#v", sample, want)
	}
}

type fakeLoader struct {
	byArtifact map[string]Result
	err        error
}

func (f fakeLoader) Load(_ context.Context, _ string, artifact string) (Result, error) {
	if f.err != nil {
		return Result{}, f.err
	}
	return f.byArtifact[artifact], nil
}

// [REQ:PH-ANALYSIS-001] Analyze parses a trace into a per-component table sorted
// by average commit time descending.
func TestAnalyzeSortsComponents(t *testing.T) {
	loader := fakeLoader{byArtifact: map[string]Result{
		"trace": {Components: []ComponentTiming{
			{Component: "Small", AvgMs: 2},
			{Component: "Big", AvgMs: 30},
		}, LCPMs: 1200},
	}}
	svc := NewService(loader)
	res, err := svc.Analyze(context.Background(), "demo", "trace")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(res.Components) != 2 || res.Components[0].Component != "Big" {
		t.Fatalf("expected Big first, got %#v", res.Components)
	}
	if res.LCPMs != 1200 {
		t.Fatalf("LCP not carried through: %d", res.LCPMs)
	}
}

func TestAnalyzeRequiresArtifact(t *testing.T) {
	svc := NewService(fakeLoader{})
	if _, err := svc.Analyze(context.Background(), "demo", ""); err == nil {
		t.Fatal("expected error for empty artifact")
	}
}
