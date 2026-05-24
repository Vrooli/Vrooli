package graph_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"typescript-code-graph/internal/graph"
)

// TestExtractDeterminism re-runs Extract 10x against the ts-junk-drawer
// fixture and asserts both GraphHash and the canonical JSON bytes are
// identical every run. Catches any source of map-iteration order, time
// leakage, or ts-morph node-position drift sneaking into the graph.
func TestExtractDeterminism(t *testing.T) {
	sup := startRealSupervisor(t)
	svc := graph.NewService(sup, graph.NewPathMutex())
	fixDir := fixtureAbsPath(t, "ts-junk-drawer")

	const runs = 10
	var firstHash string
	var firstJSON []byte
	for i := 0; i < runs; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		out, err := svc.Extract(ctx, graph.ExtractInput{ScenarioPath: fixDir})
		cancel()
		require.NoError(t, err, "run %d", i)

		canonical := canonicalGraphJSON(t, out.Graph)
		if i == 0 {
			firstHash = out.GraphHash
			firstJSON = canonical
			require.NotEmpty(t, firstHash)
			continue
		}
		require.Equal(t, firstHash, out.GraphHash,
			"run %d: GraphHash drifted across identical extractions", i)
		require.Equal(t, string(firstJSON), string(canonical),
			"run %d: canonical JSON drifted across identical extractions", i)
	}
}
