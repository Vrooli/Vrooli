package audit_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"architecture-cartographer/internal/audit"
	"architecture-cartographer/internal/conflicts"
	conflictsmocks "architecture-cartographer/internal/conflicts/mocks"
	"architecture-cartographer/internal/domains"
	domainsmocks "architecture-cartographer/internal/domains/mocks"
	graphmocks "architecture-cartographer/internal/graph/mocks"
	"architecture-cartographer/internal/testutil/mocks"
)

// TestRun_NoLadderRungs_ReportsMissingAuthority models the LPBS-shaped
// regression: a scenario with no DOMAINS.md and no api/internal/<name>/
// subfolders. domains.Service returns ErrNoAuthority. Pre-fix, the
// audit silently fell through to OutcomeClean. Post-fix, missing
// authority must produce OutcomeFindings with a remediation-led reason.
func TestRun_NoLadderRungs_ReportsMissingAuthority(t *testing.T) {
	g := &graphmocks.FakeService{Snapshots: nil, FromCache: false}
	d := &domainsmocks.FakeService{Err: domains.ErrNoAuthority{Scenario: "demo"}}
	c := &conflictsmocks.FakeService{}
	svc := audit.NewService(g, d, c, nil, nil, nil, mocks.NewFakeClock(time.Time{}))

	rep, err := svc.Run(context.Background(), audit.RunInput{Scenario: "demo"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Outcome != audit.OutcomeFindings {
		t.Fatalf("outcome=%s want %s — missing authority must not silently pass",
			rep.Outcome, audit.OutcomeFindings)
	}
	if rep.Domains.Confidence != string(domains.ConfidenceMissing) {
		t.Fatalf("Domains.Confidence=%q want %q",
			rep.Domains.Confidence, domains.ConfidenceMissing)
	}
	if !strings.Contains(rep.OutcomeReason, "DOMAINS.md") {
		t.Fatalf("reason must name DOMAINS.md (fix), got %q", rep.OutcomeReason)
	}
}

// TestRun_NoLadderRungs_AllowLowBypasses confirms --allow-low-authority
// flips the missing-authority case back to clean (advisory mode).
func TestRun_NoLadderRungs_AllowLowBypasses(t *testing.T) {
	g := &graphmocks.FakeService{Snapshots: nil, FromCache: false}
	d := &domainsmocks.FakeService{Err: domains.ErrNoAuthority{Scenario: "demo"}}
	c := &conflictsmocks.FakeService{}
	svc := audit.NewService(g, d, c, nil, nil, nil, mocks.NewFakeClock(time.Time{}))

	rep, err := svc.Run(context.Background(), audit.RunInput{
		Scenario: "demo", AllowLowAuthority: true,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Outcome != audit.OutcomeClean {
		t.Fatalf("outcome=%s want %s — --allow-low-authority must bypass",
			rep.Outcome, audit.OutcomeClean)
	}
}

// TestRun_NoLadderRungs_FindingsStillFlipOutcome confirms an error-severity
// finding still flips outcome even when authority is missing (findings beat
// authority axis).
func TestRun_NoLadderRungs_FindingsStillFlipOutcome(t *testing.T) {
	g := &graphmocks.FakeService{Snapshots: nil, FromCache: false}
	d := &domainsmocks.FakeService{Err: domains.ErrNoAuthority{Scenario: "demo"}}
	c := &conflictsmocks.FakeService{Conflicts: []conflicts.Conflict{{
		Type: "cycle", Severity: conflicts.SeverityError, Locations: []string{"a"},
	}}}
	svc := audit.NewService(g, d, c, nil, nil, nil, mocks.NewFakeClock(time.Time{}))

	rep, err := svc.Run(context.Background(), audit.RunInput{Scenario: "demo"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Outcome != audit.OutcomeFindings {
		t.Fatalf("outcome=%s want %s", rep.Outcome, audit.OutcomeFindings)
	}
}
