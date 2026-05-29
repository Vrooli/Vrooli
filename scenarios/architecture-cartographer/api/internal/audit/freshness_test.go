package audit_test

import (
	"context"
	"testing"
	"time"

	"architecture-cartographer/internal/audit"
	conflictsmocks "architecture-cartographer/internal/conflicts/mocks"
	"architecture-cartographer/internal/domains"
	domainsmocks "architecture-cartographer/internal/domains/mocks"
	"architecture-cartographer/internal/graph"
	graphmocks "architecture-cartographer/internal/graph/mocks"
	"architecture-cartographer/internal/testutil/mocks"
)

// runFreshness exercises Service.Run with a graph fake configured to
// represent each of the three (cacheHit, priorExists) combinations.
//
// graphmocks.FakeService.ListSnapshots returns Snapshots verbatim; its
// ExtractGraph returns the first Snapshot (or an empty one) and the
// FromCache flag. That gives us enough surface to model:
//   - CACHED     → Snapshots non-empty AND FromCache=true
//   - REEXTRACT  → Snapshots non-empty AND FromCache=false
//   - FRESH      → Snapshots empty       AND FromCache=false
func runFreshness(t *testing.T, snaps []graph.GraphSnapshot, fromCache bool) audit.SnapshotFreshness {
	t.Helper()
	g := &graphmocks.FakeService{Snapshots: append([]graph.GraphSnapshot(nil), snaps...), FromCache: fromCache}
	d := &domainsmocks.FakeService{Map: domains.DerivedDomainMap{
		Authority:           domains.SourceAPIFolders,
		AuthorityConfidence: "low",
	}}
	c := &conflictsmocks.FakeService{}
	svc := audit.NewService(g, d, c, nil, nil, nil, mocks.NewFakeClock(time.Time{}))
	rep, err := svc.Run(context.Background(), audit.RunInput{Scenario: "demo"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return rep.SnapshotFreshness
}

func TestFreshSnapshot_CacheHit(t *testing.T) {
	got := runFreshness(t, []graph.GraphSnapshot{{ID: "old", ContentHash: "h1"}}, true)
	if got != audit.SnapshotFreshnessCached {
		t.Fatalf("got %q want %q", got, audit.SnapshotFreshnessCached)
	}
}

func TestFreshSnapshot_PriorExistsHashDiffers(t *testing.T) {
	got := runFreshness(t, []graph.GraphSnapshot{{ID: "old", ContentHash: "h1"}}, false)
	if got != audit.SnapshotFreshnessReExtracted {
		t.Fatalf("got %q want %q", got, audit.SnapshotFreshnessReExtracted)
	}
}

func TestFreshSnapshot_NoPrior(t *testing.T) {
	got := runFreshness(t, nil, false)
	if got != audit.SnapshotFreshnessFresh {
		t.Fatalf("got %q want %q", got, audit.SnapshotFreshnessFresh)
	}
}
