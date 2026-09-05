package audit_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"architecture-cartographer/internal/audit"
	conflictsmocks "architecture-cartographer/internal/conflicts/mocks"
	"architecture-cartographer/internal/domains"
	domainsmocks "architecture-cartographer/internal/domains/mocks"
	"architecture-cartographer/internal/graph"
	graphmocks "architecture-cartographer/internal/graph/mocks"

	"github.com/vrooli/api-core/scheduletest"
)

// TestRun_SkippedAdaptersFlipCleanToPartial proves that when graph
// extract returned a snapshot with SkippedAdapters populated (the
// graph layer's signal that --skip-ts or workspace_unsupported
// silently degraded extraction), an otherwise-clean audit returns
// outcome=partial. Closes Plan Problem 7 (TS workspace_unsupported
// makes whole audit fail).
func TestRun_SkippedAdaptersFlipCleanToPartial(t *testing.T) {
	g := &graphmocks.FakeService{
		Snapshots: []graph.GraphSnapshot{{
			ID:              "snap:demo:hash",
			Scenario:        "demo",
			ContentHash:     "hash",
			SkippedAdapters: []string{"typescript"},
		}},
		FromCache: true,
	}
	d := &domainsmocks.FakeService{Map: domains.DerivedDomainMap{
		Authority:           domains.SourceDomainsDoc,
		AuthorityConfidence: domains.ConfidenceHigh,
	}}
	c := &conflictsmocks.FakeService{}
	svc := audit.NewService(g, d, c, nil, nil, nil, scheduletest.New(time.Time{}))

	rep, err := svc.Run(context.Background(), audit.RunInput{Scenario: "demo"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Outcome != audit.OutcomePartial {
		t.Fatalf("outcome=%s want %s", rep.Outcome, audit.OutcomePartial)
	}
	if !strings.Contains(rep.OutcomeReason, "typescript") {
		t.Fatalf("reason must name the skipped adapter, got %q", rep.OutcomeReason)
	}
}

// TestRun_SkippedAdaptersStillFindingsOnReal asserts findings still
// flip the outcome — partial is only the "would be clean" fallback.
func TestRun_SkippedAdaptersStillFindingsOnReal(t *testing.T) {
	g := &graphmocks.FakeService{
		Snapshots: []graph.GraphSnapshot{{
			ID:              "snap:demo:hash",
			Scenario:        "demo",
			SkippedAdapters: []string{"typescript"},
		}},
	}
	d := &domainsmocks.FakeService{Map: domains.DerivedDomainMap{
		Authority:           domains.SourceDomainsDoc,
		AuthorityConfidence: domains.ConfidenceHigh,
	}}
	cmock := &conflictsmocks.FakeService{}
	svc := audit.NewService(g, d, cmock, nil, nil, nil, scheduletest.New(time.Time{}))

	rep, err := svc.Run(context.Background(), audit.RunInput{Scenario: "demo"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	// No conflicts seeded → outcome partial (not clean).
	if rep.Outcome != audit.OutcomePartial {
		t.Fatalf("outcome=%s want %s", rep.Outcome, audit.OutcomePartial)
	}
}
