package graph_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"go-code-graph/internal/graph"
)

// canonicalGraphBytes serializes g using the same scheme as graph.GraphHash:
// sorted attribute pairs, no HTML escaping, trailing newline. Keeping this
// helper inside the test (and matching hash.go exactly) ensures the
// expected-graph.json bytes the test compares against are exactly what
// GraphHash would consume.
func canonicalGraphBytes(t *testing.T, g graph.Graph) []byte {
	t.Helper()
	// Round-trip through the public GraphHash path indirectly: build the
	// same canonical structure and Encode it identically.
	type canonNode struct {
		ID         string         `json:"id"`
		Kind       graph.NodeKind `json:"kind"`
		Name       string         `json:"name"`
		Path       string         `json:"path"`
		Attributes [][]string     `json:"attributes,omitempty"`
	}
	type canonEdge struct {
		ID         string         `json:"id"`
		Kind       graph.EdgeKind `json:"kind"`
		From       string         `json:"from"`
		To         string         `json:"to"`
		Attributes [][]string     `json:"attributes,omitempty"`
	}
	type canonGraph struct {
		Nodes []canonNode `json:"nodes"`
		Edges []canonEdge `json:"edges"`
	}
	out := canonGraph{Nodes: make([]canonNode, 0, len(g.Nodes)), Edges: make([]canonEdge, 0, len(g.Edges))}
	for _, n := range g.Nodes {
		out.Nodes = append(out.Nodes, canonNode{ID: n.ID, Kind: n.Kind, Name: n.Name, Path: n.Path, Attributes: sortedPairs(n.Attributes)})
	}
	for _, e := range g.Edges {
		out.Edges = append(out.Edges, canonEdge{ID: e.ID, Kind: e.Kind, From: e.From, To: e.To, Attributes: sortedPairs(e.Attributes)})
	}
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		t.Fatalf("encode canonical graph: %v", err)
	}
	return buf.Bytes()
}

func sortedPairs(m map[string]string) [][]string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Stable lexicographic order.
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	out := make([][]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, []string{k, m[k]})
	}
	return out
}

// resolveFixture returns the absolute path to a fixture directory given
// its path relative to api/.
func resolveFixture(t *testing.T, rel string) string {
	t.Helper()
	abs, err := filepath.Abs(rel)
	if err != nil {
		t.Fatalf("resolve fixture %s: %v", rel, err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("fixture %s missing: %v", abs, err)
	}
	return abs
}

func newRealService() *graph.Service {
	return graph.NewService(graph.NewPackagesLoader(), graph.NewPathMutex())
}

func TestExtractFixtures(t *testing.T) {
	cases := []struct {
		name       string
		fixtureRel string
	}{
		{"go-cycles", "../../../bas/fixtures/go-cycles"},
		{"go-mislocated", "../../../bas/fixtures/go-mislocated"},
	}

	svc := newRealService()
	update := os.Getenv("UPDATE_FIXTURES") == "1"

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			abs := resolveFixture(t, tc.fixtureRel)
			g, _, err := svc.Extract(context.Background(), graph.ExtractInput{ScenarioPath: abs})
			if err != nil {
				t.Fatalf("Extract: %v", err)
			}

			actual := canonicalGraphBytes(t, g)
			expectedPath := filepath.Join(abs, "expected-graph.json")

			if update {
				if err := os.WriteFile(expectedPath, actual, 0o644); err != nil {
					t.Fatalf("write expected-graph.json: %v", err)
				}
				t.Logf("wrote %s (%d bytes)", expectedPath, len(actual))
				return
			}

			expected, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatalf("read expected-graph.json: %v (run with UPDATE_FIXTURES=1 to bootstrap)", err)
			}
			if !bytes.Equal(actual, expected) {
				t.Fatalf("canonical graph mismatch for %s\n--- expected (%d bytes) ---\n%s\n--- actual (%d bytes) ---\n%s",
					tc.name, len(expected), string(expected), len(actual), string(actual))
			}
		})
	}
}

func TestConcurrentExtractSamePath(t *testing.T) {
	abs := resolveFixture(t, "../../../bas/fixtures/go-cycles")
	svc := newRealService()

	const N = 5
	hashes := make([]string, N)
	var wg sync.WaitGroup
	errs := make([]error, N)
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			g, _, err := svc.Extract(context.Background(), graph.ExtractInput{ScenarioPath: abs})
			if err != nil {
				errs[i] = err
				return
			}
			hashes[i] = graph.GraphHash(g)
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: %v", i, err)
		}
	}
	for i := 1; i < N; i++ {
		if hashes[i] != hashes[0] {
			t.Fatalf("hash mismatch: hashes[0]=%s hashes[%d]=%s", hashes[0], i, hashes[i])
		}
	}
}

func TestConcurrentExtractDifferentPaths(t *testing.T) {
	a := resolveFixture(t, "../../../bas/fixtures/go-cycles")
	b := resolveFixture(t, "../../../bas/fixtures/go-mislocated")
	svc := newRealService()

	var wg sync.WaitGroup
	var errA, errB error
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _, errA = svc.Extract(context.Background(), graph.ExtractInput{ScenarioPath: a})
	}()
	go func() {
		defer wg.Done()
		_, _, errB = svc.Extract(context.Background(), graph.ExtractInput{ScenarioPath: b})
	}()
	wg.Wait()
	if errA != nil {
		t.Fatalf("extract go-cycles: %v", errA)
	}
	if errB != nil {
		t.Fatalf("extract go-mislocated: %v", errB)
	}
}
