package conflicts_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/conflicts/detectors/cycle"
	"architecture-cartographer/internal/conflicts/detectors/mislocatedfile"
	conflictmocks "architecture-cartographer/internal/conflicts/mocks"
	"architecture-cartographer/internal/graph"
	"architecture-cartographer/internal/manifest"

	"github.com/stretchr/testify/require"
)

// TestDogfood_ScenarioIsArchitecturallyClean is the closure for the
// implementation plan: cartographer must pass its own architecture
// audit. It runs the production detector chain (cycle +
// mislocated_file) against:
//
//   - testdata/cartographer-self-graph.json — the committed snapshot
//     of cartographer's own internal/ package import graph (hand-
//     curated until go-code-graph ships).
//   - .cartographer/manifest.yaml — the manifest authored alongside
//     this test.
//
// The closure requires that conflicts.Service.ValidateConflicts (or,
// equivalently, the conflict list immediately after DetectConflicts)
// reports zero error-severity conflicts. Mislocated_file is included
// in the detector chain but receives nil VerdictProvider, so it
// becomes a no-op — exactly mirroring v0.1 production: no signals
// orchestration without a verdict source means no mislocation
// emission.
//
// Failure mode: a real cycle introduced between the listed domains
// will fail this test. That's the intent.
func TestDogfood_ScenarioIsArchitecturallyClean(t *testing.T) {
	graphPath := filepath.Join("testdata", "cartographer-self-graph.json")
	manifestPath := filepath.Join("..", "..", "..", ".cartographer", "manifest.yaml")

	raw := rawGraphFromFixture(t, graphPath)
	manifestBytes, err := os.ReadFile(manifestPath)
	require.NoError(t, err, "read manifest %s", manifestPath)
	m, _, _, err := manifest.Parse(manifestBytes, manifest.ContentTypeYAML)
	require.NoError(t, err, "parse manifest %s", manifestPath)
	require.Equal(t, "architecture-cartographer", m.Scenario)

	snap := graph.Normalize(m.Scenario, raw)

	registry := conflicts.NewRegistry(cycle.New(), mislocatedfile.New())
	svc := conflicts.NewService(&conflictmocks.FakeRepository{}, registry, conflicts.NewResolverRegistry())

	got, err := svc.DetectConflicts(context.Background(), conflicts.DetectOrchestrationInput{
		Scenario: m.Scenario,
		Snapshot: snap,
		Manifest: m,
		// VerdictProvider intentionally nil — see test doc above.
	})
	require.NoError(t, err)

	if len(got) != 0 {
		t.Fatalf("dogfood gate broken: %d conflicts detected; first=%+v", len(got), got[0])
	}
}
