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
