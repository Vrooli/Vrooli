package graph_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"typescript-code-graph/internal/graph"
	"typescript-code-graph/internal/sidecar"
)

func TestNormalize_SortsNodesByID(t *testing.T) {
	raw := sidecar.RawGraph{
		Nodes: []sidecar.RawNode{
			{ID: "c", Kind: 1},
			{ID: "a", Kind: 1},
			{ID: "b", Kind: 1},
		},
	}
	g := graph.Normalize(raw)
	require.Equal(t, []string{"a", "b", "c"},
		[]string{g.Nodes[0].ID, g.Nodes[1].ID, g.Nodes[2].ID})
}

func TestNormalize_SortsEdgesByFromTo(t *testing.T) {
	raw := sidecar.RawGraph{
		Edges: []sidecar.RawEdge{
			{ID: "e1", FromNodeID: "b", ToNodeID: "a", Kind: 1},
			{ID: "e2", FromNodeID: "a", ToNodeID: "z", Kind: 1},
			{ID: "e3", FromNodeID: "a", ToNodeID: "b", Kind: 1},
		},
	}
	g := graph.Normalize(raw)
	require.Equal(t, "a", g.Edges[0].From)
	require.Equal(t, "b", g.Edges[0].To)
	require.Equal(t, "a", g.Edges[1].From)
	require.Equal(t, "z", g.Edges[1].To)
	require.Equal(t, "b", g.Edges[2].From)
}

func TestNormalize_DecodesTsNodeKinds(t *testing.T) {
	cases := []struct {
		in   int32
		want graph.NodeKind
	}{
		{201, graph.NodeKindComponent},
		{202, graph.NodeKindHook},
		{203, graph.NodeKindClass},
		{204, graph.NodeKindInterface},
		{205, graph.NodeKindType},
		{206, graph.NodeKindFunction},
		{207, graph.NodeKindVar},
		{208, graph.NodeKindConst},
		{209, graph.NodeKindReExport},
		{210, graph.NodeKindImportBinding},
		{211, graph.NodeKindReference},
		{212, graph.NodeKindCall},
		{213, graph.NodeKindJsxUsage},
		{214, graph.NodeKindExport},
		{215, graph.NodeKindRoute},
		{200, graph.NodeKindModule},
		{1, graph.NodeKindFile},
		{2, graph.NodeKindModule},
	}
	for _, tc := range cases {
		raw := sidecar.RawGraph{Nodes: []sidecar.RawNode{{ID: "n", Kind: tc.in}}}
		g := graph.Normalize(raw)
		require.Equal(t, tc.want, g.Nodes[0].Kind, "input %d", tc.in)
	}
}

// The sidecar populates Attributes["kind"] with the TS-specific enum
// name (TS_NODE_KIND_*) directly; normalize passes it through verbatim.
func TestNormalize_PreservesTsKindInAttributes(t *testing.T) {
	raw := sidecar.RawGraph{
		Nodes: []sidecar.RawNode{{
			ID:         "ts_component:m:Btn",
			Kind:       201,
			Name:       "Btn",
			Attributes: map[string]string{"kind": "TS_NODE_KIND_COMPONENT"},
		}},
	}
	g := graph.Normalize(raw)
	require.Equal(t, "TS_NODE_KIND_COMPONENT", g.Nodes[0].Attributes["kind"])
}

func TestNormalize_KeepsAttributesIntact(t *testing.T) {
	raw := sidecar.RawGraph{
		Nodes: []sidecar.RawNode{{
			ID:         "n",
			Kind:       1,
			Attributes: map[string]string{"language": "typescript", "exported": "true"},
		}},
	}
	g := graph.Normalize(raw)
	require.Equal(t, "typescript", g.Nodes[0].Attributes["language"])
	require.Equal(t, "true", g.Nodes[0].Attributes["exported"])
}

// An unknown numeric kind falls back to NodeKindFile; the sidecar may
// also include attributes["kind"] for future enum values — those pass
// through unchanged.
func TestNormalize_UnknownKindFallsBackToFile(t *testing.T) {
	raw := sidecar.RawGraph{
		Nodes: []sidecar.RawNode{{
			ID:         "n",
			Kind:       299,
			Attributes: map[string]string{"kind": "TS_NODE_KIND_FUTURE"},
		}},
	}
	g := graph.Normalize(raw)
	require.Equal(t, graph.NodeKindFile, g.Nodes[0].Kind)
	require.Equal(t, "TS_NODE_KIND_FUTURE", g.Nodes[0].Attributes["kind"])
}
