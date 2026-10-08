package lighthouse

import (
	"context"
	"testing"
)

type fakeRunner struct {
	res Result
	err error
}

func (f fakeRunner) Run(context.Context, string, string) (Result, error) { return f.res, f.err }

// [REQ:PH-LH-001] Score wraps the runner and surfaces per-page category scores.
func TestScoreReturnsPages(t *testing.T) {
	svc := NewService(fakeRunner{res: Result{Outcome: OutcomeScored, Pages: []PageScore{
		{URL: "http://localhost:3000", Performance: 0.92},
	}}})
	res, err := svc.Score(context.Background(), "demo", "")
	if err != nil {
		t.Fatalf("Score: %v", err)
	}
	if res.Outcome != OutcomeScored || len(res.Pages) != 1 || res.Pages[0].Performance != 0.92 {
		t.Fatalf("unexpected result: %#v", res)
	}
}

func TestScoreRequiresScenario(t *testing.T) {
	svc := NewService(fakeRunner{})
	if _, err := svc.Score(context.Background(), "", ""); err == nil {
		t.Fatal("expected error for empty scenario")
	}
}
