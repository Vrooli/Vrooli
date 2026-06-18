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

// TestDogfood_ScenarioIsArchitecturallyClean is the closure for the
// implementation plan: cartographer must pass its own architecture
// audit. It runs the production detector chain slice that can operate on
// the committed self-graph fixture (cycle + layering + naming +
// mislocated_file) against:
//
//   - testdata/cartographer-self-graph.json — the committed snapshot
//     of cartographer's own internal/ package import graph (hand-
//     curated until go-code-graph ships).
//   - the domain map DERIVED from cartographer's own on-disk sources
//     (docs/concepts/DOMAINS.md via the extraction ladder) — there is
//     no per-scenario architecture manifest.
//
// The closure requires that conflicts.Service.ValidateConflicts (or,
// equivalently, the conflict list immediately after DetectConflicts)
// reports zero error-severity conflicts. Mislocated_file is included
// in the detector chain but receives nil VerdictProvider, so it
// becomes a no-op — exactly mirroring v0.1 production: no signals
// orchestration without a verdict source means no mislocation
// emission.
//
// Failure mode: a real cycle, blocker layering violation, or banned generic
// package/domain name introduced between the listed domains will fail this
// test. That's the intent.
func TestDogfood_ScenarioIsArchitecturallyClean(t *testing.T) {
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

	if len(got) != 0 {
		t.Fatalf("dogfood gate broken: %d conflicts detected; first=%+v", len(got), got[0])
	}
}
