package graph_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"typescript-code-graph/internal/graph"
)

func TestGraphHash_EmptyIsStable(t *testing.T) {
	h1 := graph.GraphHash(graph.Graph{})
	h2 := graph.GraphHash(graph.Graph{})
	require.Equal(t, h1, h2)
	require.Len(t, h1, 64, "hex sha256 must be 64 chars")
}

func TestGraphHash_DeterministicWithAttributes(t *testing.T) {
	g := graph.Graph{
		Nodes: []graph.Node{{
			ID:   "file:src/a.ts",
			Kind: graph.NodeKindFile,
			Name: "a.ts",
			Path: "src/a.ts",
			Attributes: map[string]string{
				"language":  "typescript",
				"module_id": "ts_module:root",
			},
		}},
	}
	h1 := graph.GraphHash(g)
	for i := 0; i < 8; i++ {
		require.Equal(t, h1, graph.GraphHash(g))
	}
}

func TestGraphHash_LeadingCommentsAffectHash(t *testing.T) {
	g1 := graph.Graph{Nodes: []graph.Node{{
		ID: "ts_component:m:Button", Kind: graph.NodeKindComponent,
		LeadingComments: []string{"/** @vrooliWidget */"},
	}}}
	g2 := graph.Graph{Nodes: []graph.Node{{
		ID: "ts_component:m:Button", Kind: graph.NodeKindComponent,
		LeadingComments: []string{"/** @vrooliWidget changed */"},
	}}}
	require.NotEqual(t, graph.GraphHash(g1), graph.GraphHash(g2),
		"comment edits must alter the graph hash so consumers can detect drift")
}

func TestGraphHash_DiffersByID(t *testing.T) {
	a := graph.Graph{Nodes: []graph.Node{{ID: "a", Kind: graph.NodeKindFile}}}
	b := graph.Graph{Nodes: []graph.Node{{ID: "b", Kind: graph.NodeKindFile}}}
	require.NotEqual(t, graph.GraphHash(a), graph.GraphHash(b))
}
