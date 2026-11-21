package conflicts_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/detectors/cycle"
	"architecture-cartographer/internal/conflicts/detectors/layering"
	"architecture-cartographer/internal/conflicts/detectors/mislocatedfile"
	"architecture-cartographer/internal/conflicts/detectors/naming"
	conflictmocks "architecture-cartographer/internal/conflicts/mocks"
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/graph"

	"github.com/stretchr/testify/require"
)

// TestDogfood_ScenarioHasNoDeterministicGateFindings is the closure for the
// placement-engine and two-tier-findings redesign: cartographer must not
// fail its own deterministic architecture gate. It runs the production detector
// chain slice that can operate on the committed self-graph fixture
// (cycle + layering + naming + mislocated_file) against:
//
//   - testdata/cartographer-self-graph.json — the committed snapshot
//     of cartographer's own internal/ package import graph (hand-
//     curated until go-code-graph ships).
//   - the domain map DERIVED from cartographer's own on-disk sources
//     (docs/concepts/DOMAINS.md via the extraction ladder) — there is
//     no per-scenario architecture manifest.
//
// The closure requires that the conflict list immediately after
// DetectConflicts reports zero deterministic error/blocker conflicts. Heuristic
// findings may still be reported as advisory warnings, but they must not
// hard-fail the dogfood gate. mislocated_file is included in the detector chain;
// with a nil VerdictProvider it skips, matching production behavior when the
// caller did not wire the signals service.
//
// Failure mode: a real cycle or blocker layering violation introduced between
// the listed domains will fail this test. A cartographer-owned file-size
// detector would also fail, because that responsibility belongs to
// tidiness-manager.
func TestDogfood_ScenarioHasNoDeterministicGateFindings(t *testing.T) {
	graphPath := filepath.Join("testdata", "cartographer-self-graph.json")
	scenarioRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)

	raw := rawGraphFromFixture(t, graphPath)

	// Derive the domain map from cartographer's own sources — the
	// DOMAINS.md rung is authoritative. This is the dogfood: the same
	// derivation production runs on any target scenario.
	extractions, err := domains.RunLadder(context.Background(), scenarioRoot, domains.DefaultExtractors())
	require.NoError(t, err, "run domain-extraction ladder over own scenario")
	dmap, err := domains.Resolve("architecture-cartographer", extractions, time.Time{})
	require.NoError(t, err, "resolve own derived domain map")
	require.Equal(t, domains.SourceDomainsDoc, dmap.Authority)

	snap := graph.Normalize("architecture-cartographer", raw)

	registry := conflicts.NewRegistry(cycle.New(), layering.New(), mislocatedfile.New(), naming.New())
	svc := conflicts.NewService(&conflictmocks.FakeRepository{}, registry, conflicts.NewResolverRegistry())

	got, err := svc.DetectConflicts(context.Background(), conflicts.DetectOrchestrationInput{
		Scenario:  "architecture-cartographer",
		Snapshot:  snap,
		DomainMap: dmap,
		// VerdictProvider intentionally nil — see test doc above.
	})
	require.NoError(t, err)

	for _, finding := range got {
		if finding.Type == "file_cohesion" {
			t.Fatalf("cartographer must not emit tidiness-manager file_cohesion findings: %+v", finding)
		}
		if conflicts.IsDeterministicGateFinding(finding) {
			t.Fatalf("dogfood deterministic gate broken: %+v", finding)
		}
	}
}
