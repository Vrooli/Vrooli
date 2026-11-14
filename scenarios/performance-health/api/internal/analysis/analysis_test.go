package analysis

import (
	"context"
	"testing"
)

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
