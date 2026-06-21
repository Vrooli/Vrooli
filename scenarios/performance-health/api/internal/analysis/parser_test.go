package analysis

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

const fixtureScenarioRoot = "testdata/fixture-scenario"

// [REQ:PH-ANALYSIS-001] The real FileTraceLoader parses a checked-in CDP trace
// into the per-component count/avg/max table (⚛ marks paired by id2.local) and
// extracts long-task / FCP / LCP from the sibling web-vitals file.
func TestFileTraceLoaderParsesFixture(t *testing.T) {
	loader := FileTraceLoader{}
	res, err := loader.Load(context.Background(), "", "testdata/performance.json")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := map[string]ComponentTiming{
		"App":         {Component: "App", CommitCount: 3, AvgMs: 2.0, MaxMs: 3.0},
		"ProjectList": {Component: "ProjectList", CommitCount: 2, AvgMs: 20.0, MaxMs: 25.0},
		"Sidebar":     {Component: "Sidebar", CommitCount: 1, AvgMs: 0.5, MaxMs: 0.5},
	}
	if len(res.Components) != len(want) {
		t.Fatalf("expected %d components, got %d (%#v)", len(want), len(res.Components), res.Components)
	}
	// Sorted by avg desc → ProjectList first.
	if res.Components[0].Component != "ProjectList" {
		t.Fatalf("expected ProjectList first (highest avg), got %s", res.Components[0].Component)
	}
	for _, got := range res.Components {
		w, ok := want[got.Component]
		if !ok {
			t.Fatalf("unexpected component %q", got.Component)
		}
		if got.CommitCount != w.CommitCount || got.AvgMs != w.AvgMs || got.MaxMs != w.MaxMs {
			t.Fatalf("%s: got count=%d avg=%.1f max=%.1f, want count=%d avg=%.1f max=%.1f",
				got.Component, got.CommitCount, got.AvgMs, got.MaxMs, w.CommitCount, w.AvgMs, w.MaxMs)
		}
	}

	if res.LongTaskMs != 160 {
		t.Errorf("long-task total: got %d, want 160", res.LongTaskMs)
	}
	if res.FCPMs != 148 {
		t.Errorf("FCP: got %d, want 148", res.FCPMs)
	}
	if res.LCPMs != 1240 {
		t.Errorf("LCP: got %d, want 1240", res.LCPMs)
	}
}

// [REQ:PH-ANALYSIS-001] A Tier-0 trace (no ⚛ marks) yields an empty component
// table and never errors; web-vitals are still parsed.
func TestFileTraceLoaderTier0NoComponentMarks(t *testing.T) {
	loader := FileTraceLoader{}
	res, err := loader.Load(context.Background(), "", "testdata/tier0/performance.json")
	if err != nil {
		t.Fatalf("Tier-0 Load must not error: %v", err)
	}
	if len(res.Components) != 0 {
		t.Fatalf("Tier-0 trace must have no components, got %#v", res.Components)
	}
	if res.LongTaskMs != 41 {
		t.Errorf("Tier-0 long-task: got %d, want 41", res.LongTaskMs)
	}
	if res.FCPMs != 210 || res.LCPMs != 980 {
		t.Errorf("Tier-0 web-vitals: got FCP=%d LCP=%d, want 210/980", res.FCPMs, res.LCPMs)
	}
	if len(res.Findings) != 0 {
		t.Errorf("Tier-0 trace must produce no findings, got %#v", res.Findings)
	}
}

// [REQ:PH-ANALYSIS-002] With a budget + symbol locator, the hot component is
// flagged with quantified evidence AND located to file:line in the fixture UI.
func TestFileTraceLoaderEmitsLocatedFindings(t *testing.T) {
	abs, err := filepath.Abs(fixtureScenarioRoot)
	if err != nil {
		t.Fatal(err)
	}
	loader := FileTraceLoader{
		Locator:  SourceLocator{ScenarioRoot: abs},
		BudgetMs: 8.0,
	}
	res, err := loader.Load(context.Background(), "fixture-scenario", "testdata/performance.json")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(res.Findings) != 1 {
		t.Fatalf("expected exactly 1 over-budget finding (ProjectList avg 20ms), got %d: %#v", len(res.Findings), res.Findings)
	}
	f := res.Findings[0]
	if f.Component != "ProjectList" {
		t.Fatalf("expected ProjectList finding, got %q", f.Component)
	}
	wantDef := "ui/src/features/list/ProjectList.tsx:9"
	if f.Definition != wantDef {
		t.Fatalf("definition: got %q, want %q", f.Definition, wantDef)
	}
	if f.Evidence == "" || f.Code != "PERF_COMPONENT_COMMIT_OVER_BUDGET" {
		t.Fatalf("finding must carry code + quantified evidence: %#v", f)
	}
}

// A finding whose component definition cannot be located is still emitted, with
// a "definition not located" note rather than being dropped.
func TestFindingsUnlocatedNote(t *testing.T) {
	got := DeriveFindings([]ComponentTiming{
		{Component: "Mystery", CommitCount: 9, AvgMs: 50, MaxMs: 90},
	}, 8.0)
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(got))
	}
	if got[0].Definition != "" {
		t.Fatalf("expected empty definition, got %q", got[0].Definition)
	}
	if got[0].Message == "" || !strings.Contains(got[0].Message, "definition not located") {
		t.Fatalf("expected unlocated note in message, got %q", got[0].Message)
	}
}
