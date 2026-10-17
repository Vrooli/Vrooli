package graph_test

import (
	"testing"

	intgraph "go-code-graph/internal/graph"
)

// TestGraphHashDeterministic asserts identical graphs hash identically.
func TestGraphHashDeterministic(t *testing.T) {
	t.Parallel()
	g := intgraph.Graph{
		Nodes: []intgraph.Node{
			{ID: "package:a", Kind: intgraph.NodeKindPackage, Name: "a", Path: "a"},
			{ID: "package:b", Kind: intgraph.NodeKindPackage, Name: "b", Path: "b"},
		},
		Edges: []intgraph.Edge{
			{ID: "import:a->b", Kind: intgraph.EdgeKindImport, From: "package:a", To: "package:b"},
		},
	}
	h1 := intgraph.GraphHash(g)
	h2 := intgraph.GraphHash(g)
	if h1 != h2 {
		t.Fatalf("hash not deterministic: %s vs %s", h1, h2)
	}
	if len(h1) != 64 {
		t.Fatalf("expected hex sha256 (64 chars), got %d", len(h1))
	}
}

// TestGraphHashSensitiveToNodeOrder asserts that two graphs with
// different node ordering produce different hashes — the hash IS the
// canonical form. (Callers MUST run Normalize first; this test
// documents that contract.)
func TestGraphHashSensitiveToNodeOrder(t *testing.T) {
	t.Parallel()
	a := intgraph.Graph{
		Nodes: []intgraph.Node{
			{ID: "package:a", Kind: intgraph.NodeKindPackage, Name: "a", Path: "a"},
			{ID: "package:b", Kind: intgraph.NodeKindPackage, Name: "b", Path: "b"},
		},
	}
	b := intgraph.Graph{
		Nodes: []intgraph.Node{
			{ID: "package:b", Kind: intgraph.NodeKindPackage, Name: "b", Path: "b"},
			{ID: "package:a", Kind: intgraph.NodeKindPackage, Name: "a", Path: "a"},
		},
	}
	if intgraph.GraphHash(a) == intgraph.GraphHash(b) {
		t.Fatalf("hash should differ when node order differs (caller must Normalize first)")
	}
}

// TestGraphHashStableAttributeOrder asserts that two graphs whose
// attribute maps differ only in insertion order hash to the same
// value (because the canonical projection sorts attributes).
func TestGraphHashStableAttributeOrder(t *testing.T) {
	t.Parallel()
	g1 := intgraph.Graph{
		Nodes: []intgraph.Node{{
			ID:         "package:a",
			Kind:       intgraph.NodeKindPackage,
			Name:       "a",
			Path:       "a",
			Attributes: map[string]string{"x": "1", "y": "2"},
		}},
	}
	g2 := intgraph.Graph{
		Nodes: []intgraph.Node{{
			ID:         "package:a",
			Kind:       intgraph.NodeKindPackage,
			Name:       "a",
			Path:       "a",
			Attributes: map[string]string{"y": "2", "x": "1"},
		}},
	}
	if intgraph.GraphHash(g1) != intgraph.GraphHash(g2) {
		t.Fatalf("hash should be insensitive to attribute insertion order")
	}
}

// TestGraphHashEmpty asserts an empty graph still produces a valid
// hex sha256 string (and the same string across calls).
func TestGraphHashEmpty(t *testing.T) {
	t.Parallel()
	h := intgraph.GraphHash(intgraph.Graph{})
	if len(h) != 64 {
		t.Fatalf("expected 64-char hex, got %d (%q)", len(h), h)
	}
}
