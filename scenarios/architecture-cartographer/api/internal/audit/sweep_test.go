package audit_test

import (
	"context"
	"testing"
	"time"

	"architecture-cartographer/internal/audit"
	conflictsmocks "architecture-cartographer/internal/conflicts/mocks"
	"architecture-cartographer/internal/domains"
	domainsmocks "architecture-cartographer/internal/domains/mocks"
	graphmocks "architecture-cartographer/internal/graph/mocks"
	"architecture-cartographer/internal/testutil/mocks"
)

// fakeLister returns the configured list of scenario names verbatim.
type fakeLister struct{ names []string }

func (f *fakeLister) List(_ context.Context) ([]string, error) { return f.names, nil }

// TestRunAll_AllowLowAuthorityScenarios_SilencesOnlyListed asserts the
// per-scenario opt-out: when --allow-low-authority-scenarios=A is set
// (and the global --allow-low-authority is false), scenario A returns
// CLEAN while B still returns FINDINGS for missing authority. Closes
// Plan Problem 6 (run-all all-or-nothing).
func TestRunAll_AllowLowAuthorityScenarios_SilencesOnlyListed(t *testing.T) {
	// Both scenarios have NO authority (LPBS-shaped).
	g := &graphmocks.FakeService{Snapshots: nil, FromCache: false}
	d := &domainsmocks.FakeService{Err: domains.ErrNoAuthority{Scenario: ""}}
	c := &conflictsmocks.FakeService{}
	lister := &fakeLister{names: []string{"a", "b"}}
	svc := audit.NewService(g, d, c, nil, nil, lister, mocks.NewFakeClock(time.Time{}))

	sweep, err := svc.RunAll(context.Background(), audit.RunAllInput{
		AllowLowAuthorityScenarios: []string{"a"},
	})
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	if len(sweep.Reports) != 2 {
		t.Fatalf("want 2 reports, got %d", len(sweep.Reports))
	}
	byName := map[string]audit.Report{}
	for _, r := range sweep.Reports {
		byName[r.Scenario] = r
	}
	if byName["a"].Outcome != audit.OutcomeClean {
		t.Errorf("scenario a: outcome=%s want %s (in allow list)", byName["a"].Outcome, audit.OutcomeClean)
	}
	if byName["b"].Outcome != audit.OutcomeFindings {
		t.Errorf("scenario b: outcome=%s want %s (NOT in allow list)", byName["b"].Outcome, audit.OutcomeFindings)
	}
}

// TestRunAll_GlobalAllowLowAuthorityOverridesAll confirms the existing
// global bool still works (any list value is OR'd in on top).
func TestRunAll_GlobalAllowLowAuthorityOverridesAll(t *testing.T) {
	g := &graphmocks.FakeService{Snapshots: nil, FromCache: false}
	d := &domainsmocks.FakeService{Err: domains.ErrNoAuthority{Scenario: ""}}
	c := &conflictsmocks.FakeService{}
	lister := &fakeLister{names: []string{"a", "b"}}
	svc := audit.NewService(g, d, c, nil, nil, lister, mocks.NewFakeClock(time.Time{}))

	sweep, err := svc.RunAll(context.Background(), audit.RunAllInput{
		AllowLowAuthority: true,
	})
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	for _, r := range sweep.Reports {
		if r.Outcome != audit.OutcomeClean {
			t.Errorf("scenario %s: outcome=%s want %s (global allow-low-authority)", r.Scenario, r.Outcome, audit.OutcomeClean)
		}
	}
}
