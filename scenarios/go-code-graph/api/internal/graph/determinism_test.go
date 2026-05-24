package graph_test

import (
	"bytes"
	"context"
	"testing"

	"go-code-graph/internal/graph"
)

// TestExtractDeterministic runs Extract 10 times against go-cycles and
// asserts every canonical-JSON serialization and every GraphHash is
// byte-identical. Powers REQ-P0-001 (byte-stable JSON output) at the
// integration level.
func TestExtractDeterministic(t *testing.T) {
	abs := resolveFixture(t, "../../../bas/fixtures/go-cycles")
	svc := newRealService()

	const N = 10
	var firstBytes []byte
	var firstHash string
	for i := 0; i < N; i++ {
		g, _, err := svc.Extract(context.Background(), graph.ExtractInput{ScenarioPath: abs})
		if err != nil {
			t.Fatalf("iteration %d Extract: %v", i, err)
		}
		b := canonicalGraphBytes(t, g)
		h := graph.GraphHash(g)
		if i == 0 {
			firstBytes = b
			firstHash = h
			continue
		}
		if !bytes.Equal(b, firstBytes) {
			t.Fatalf("iteration %d canonical bytes differ from iteration 0", i)
		}
		if h != firstHash {
			t.Fatalf("iteration %d hash %s differs from iteration 0 hash %s", i, h, firstHash)
		}
	}
}
