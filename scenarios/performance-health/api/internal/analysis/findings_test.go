package analysis

import "testing"

// [REQ:PH-ANALYSIS-003] DeriveFindings flags only components over a positive
// budget, attaching quantified evidence (deterministic; no AI).
func TestDeriveFindingsOverBudget(t *testing.T) {
	components := []ComponentTiming{
		{Component: "Fast", CommitCount: 5, AvgMs: 2, MaxMs: 4},
		{Component: "Slow", CommitCount: 120, AvgMs: 18, MaxMs: 40, Definition: "ui/src/Slow.tsx:12"},
	}
	findings := DeriveFindings(components, 8)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	f := findings[0]
	if f.Component != "Slow" || f.Definition != "ui/src/Slow.tsx:12" {
		t.Fatalf("unexpected finding target: %#v", f)
	}
	if f.Evidence == "" {
		t.Fatal("finding must carry quantified evidence")
	}
}

func TestDeriveFindingsNoBudget(t *testing.T) {
	findings := DeriveFindings([]ComponentTiming{{Component: "X", AvgMs: 999}}, 0)
	if len(findings) != 0 {
		t.Fatalf("expected no findings with no budget, got %d", len(findings))
	}
}

func TestDeriveInteractionFindingsFlagsPoorTrace(t *testing.T) {
	findings := DeriveInteractionFindings(
		FrameSummary{TraceDurationMs: 1000, DrawnFrameCount: 20, DroppedFrameCount: 30, ApproxDrawnFPS: 20, DroppedFrameRate: 0.6},
		[]EventSummary{{Name: "RasterTask", Count: 3, TotalMs: 700, AvgMs: 233.3, MaxMs: 300}, {Name: "Layout", Count: 2, TotalMs: 140, AvgMs: 70, MaxMs: 80}},
		[]EventSummary{{Name: "wheel", Count: 2, TotalMs: 4, AvgMs: 2, MaxMs: 2}},
	)
	for _, code := range []string{
		"PERF_INTERACTION_INPUT_TOO_SMALL",
		"PERF_LOW_DRAWN_FPS",
		"PERF_HIGH_DROPPED_FRAME_RATE",
		"PERF_HIGH_RASTER_COST",
		"PERF_HIGH_LAYOUT_COST",
	} {
		if !hasFinding(findings, code) {
			t.Fatalf("expected finding %s in %#v", code, findings)
		}
	}
}

func TestDeriveInteractionFindingsFlagsMissingEvidence(t *testing.T) {
	findings := DeriveInteractionFindings(FrameSummary{}, nil, nil)
	for _, code := range []string{"PERF_INTERACTION_INPUT_MISSING", "PERF_FRAME_HEALTH_MISSING"} {
		if !hasFinding(findings, code) {
			t.Fatalf("expected finding %s in %#v", code, findings)
		}
	}
}
