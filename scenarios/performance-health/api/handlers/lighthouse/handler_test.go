package lighthouse

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	internallh "performance-health/internal/lighthouse"

	lighthousev1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/lighthouse"
)

type fakeRunner struct {
	res    internallh.Result
	err    error
	called bool
}

func (f *fakeRunner) Run(_ context.Context, scenario, _ string) (internallh.Result, error) {
	f.called = true
	f.res.Scenario = scenario
	return f.res, f.err
}

// TestRunLighthouseScoredMapsToProto builds the REAL lighthouse service over a
// fake runner and asserts a Scored outcome with per-page scores maps correctly.
func TestRunLighthouseScoredMapsToProto(t *testing.T) {
	runner := &fakeRunner{res: internallh.Result{
		Outcome: internallh.OutcomeScored,
		Pages: []internallh.PageScore{{
			URL:           "http://localhost:3000/",
			Performance:   0.91,
			Accessibility: 0.88,
			BestPractices: 1.0,
			SEO:           0.95,
			Violations:    []string{"performance 0.91 < warn 0.95"},
		}},
	}}
	h := NewHandler(internallh.NewService(runner), nil)

	resp, err := h.RunLighthouse(context.Background(), connect.NewRequest(&lighthousev1.RunLighthouseRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("RunLighthouse: %v", err)
	}
	if !runner.called {
		t.Fatal("runner seam was not exercised")
	}
	msg := resp.Msg
	if msg.GetScenario() != "demo" {
		t.Errorf("scenario = %q", msg.GetScenario())
	}
	if msg.GetOutcome() != lighthousev1.LighthouseOutcome_LIGHTHOUSE_OUTCOME_SCORED {
		t.Errorf("outcome = %v, want SCORED", msg.GetOutcome())
	}
	if len(msg.GetPages()) != 1 {
		t.Fatalf("pages len = %d, want 1", len(msg.GetPages()))
	}
	p := msg.GetPages()[0]
	if p.GetUrl() != "http://localhost:3000/" || p.GetPerformance() != 0.91 || p.GetAccessibility() != 0.88 || p.GetBestPractices() != 1.0 || p.GetSeo() != 0.95 {
		t.Errorf("page score mapped wrong: %+v", p)
	}
	if len(p.GetViolations()) != 1 || p.GetViolations()[0] != "performance 0.91 < warn 0.95" {
		t.Errorf("violations mapped wrong: %v", p.GetViolations())
	}
}

// TestRunLighthouseSkippedMapsToProto proves a clean skip (no UI / no CLI) maps
// to the SKIPPED enum with the reason surfaced and no pages.
func TestRunLighthouseSkippedMapsToProto(t *testing.T) {
	runner := &fakeRunner{res: internallh.Result{Outcome: internallh.OutcomeSkipped, Reason: "no resolvable UI URL"}}
	h := NewHandler(internallh.NewService(runner), nil)

	resp, err := h.RunLighthouse(context.Background(), connect.NewRequest(&lighthousev1.RunLighthouseRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("RunLighthouse: %v", err)
	}
	if resp.Msg.GetOutcome() != lighthousev1.LighthouseOutcome_LIGHTHOUSE_OUTCOME_SKIPPED {
		t.Errorf("outcome = %v, want SKIPPED", resp.Msg.GetOutcome())
	}
	if resp.Msg.GetReason() != "no resolvable UI URL" {
		t.Errorf("reason = %q", resp.Msg.GetReason())
	}
	if len(resp.Msg.GetPages()) != 0 {
		t.Errorf("skipped run must have no pages, got %d", len(resp.Msg.GetPages()))
	}
}

// TestRunLighthouseRequiresScenario asserts the empty-scenario guard maps to
// InvalidArgument.
func TestRunLighthouseRequiresScenario(t *testing.T) {
	h := NewHandler(internallh.NewService(&fakeRunner{}), nil)
	_, err := h.RunLighthouse(context.Background(), connect.NewRequest(&lighthousev1.RunLighthouseRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want InvalidArgument, got %v (err=%v)", connect.CodeOf(err), err)
	}
}
