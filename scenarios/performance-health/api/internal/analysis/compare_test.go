package analysis

import (
	"context"
	"testing"
)

// [REQ:PH-ANALYSIS-002] Compare diffs two traces of the same interaction by
// component, surfacing the largest regression first.
func TestCompareDiffsComponents(t *testing.T) {
	loader := fakeLoader{byArtifact: map[string]Result{
		"before": {Components: []ComponentTiming{{Component: "List", AvgMs: 10}}, LongTaskMs: 100},
		"after":  {Components: []ComponentTiming{{Component: "List", AvgMs: 25}}, LongTaskMs: 180},
	}}
	svc := NewService(loader)
	cmp, err := svc.Compare(context.Background(), "demo", "before", "after")
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if len(cmp.Components) != 1 || cmp.Components[0].DeltaMs != 15 {
		t.Fatalf("expected +15ms List delta, got %#v", cmp.Components)
	}
	if cmp.LongTaskDeltaMs != 80 {
		t.Fatalf("expected +80ms long-task delta, got %d", cmp.LongTaskDeltaMs)
	}
}

func TestCompareRequiresBothArtifacts(t *testing.T) {
	svc := NewService(fakeLoader{})
	if _, err := svc.Compare(context.Background(), "demo", "a", ""); err == nil {
		t.Fatal("expected error for missing candidate artifact")
	}
}

// [REQ:PH-ANALYSIS-003] Compare diffs two real fixture traces of the same
// interaction into a per-component delta (count + avg + max) plus long-task and
// LCP deltas, surfacing the "commit count rose but per-commit cost dropped"
// case (ProjectList: 2→3 commits, 20.0ms→2.7ms avg).
func TestCompareRealFixtures(t *testing.T) {
	svc := NewService(FileTraceLoader{})
	cmp, err := svc.Compare(context.Background(), "demo", "testdata/performance.json", "testdata/after/performance.json")
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}

	byName := map[string]ComponentDelta{}
	for _, d := range cmp.Components {
		byName[d.Component] = d
	}

	pl, ok := byName["ProjectList"]
	if !ok {
		t.Fatalf("ProjectList missing from delta: %#v", cmp.Components)
	}
	if pl.BaselineCount != 2 || pl.CandidateCount != 3 || pl.CountDelta != 1 {
		t.Errorf("ProjectList count delta: got %d→%d (Δ%d), want 2→3 (Δ1)", pl.BaselineCount, pl.CandidateCount, pl.CountDelta)
	}
	if pl.BaselineAvgMs != 20.0 || pl.CandidateAvgMs != 2.7 || pl.DeltaMs != -17.3 {
		t.Errorf("ProjectList avg delta: got %.1f→%.1f (Δ%.1f), want 20.0→2.7 (Δ-17.3)", pl.BaselineAvgMs, pl.CandidateAvgMs, pl.DeltaMs)
	}

	// Sidebar present only in baseline → one-sided delta.
	sb, ok := byName["Sidebar"]
	if !ok || sb.CandidateCount != 0 || sb.CountDelta != -1 {
		t.Errorf("Sidebar one-sided delta wrong: %#v", sb)
	}

	if cmp.LongTaskDeltaMs != -130 {
		t.Errorf("long-task delta: got %d, want -130", cmp.LongTaskDeltaMs)
	}
	if cmp.LCPDeltaMs != -140 {
		t.Errorf("LCP delta: got %d, want -140", cmp.LCPDeltaMs)
	}
	if cmp.FrameDelta.DrawnFrameCountDelta != 1 || cmp.FrameDelta.DroppedFrameCountDelta != -1 {
		t.Errorf("frame count delta: got %#v, want drawn +1 dropped -1", cmp.FrameDelta)
	}
	if cmp.FrameDelta.ApproxDrawnFPSDelta != 5.1 {
		t.Errorf("drawn FPS delta: got %.1f, want +5.1", cmp.FrameDelta.ApproxDrawnFPSDelta)
	}
	if got := eventDeltaByName(cmp.BrowserWork, "RasterTask"); got == nil || got.TotalDeltaMs != -490 {
		t.Errorf("RasterTask delta: got %#v, want total -490ms", got)
	}
	if got := eventDeltaByName(cmp.InputEvents, "pointermove"); got == nil || got.CountDelta != 5 {
		t.Errorf("pointermove delta: got %#v, want count +5", got)
	}
}

func eventDeltaByName(events []EventDelta, name string) *EventDelta {
	for i := range events {
		if events[i].Name == name {
			return &events[i]
		}
	}
	return nil
}
