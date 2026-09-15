package graph

import (
	"testing"

	"github.com/stretchr/testify/require"

	intgraph "typescript-code-graph/internal/graph"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestDomainToProtoGraph_FoldsKinds(t *testing.T) {
	g := intgraph.Graph{
		Nodes: []intgraph.Node{
			{ID: "f", Kind: intgraph.NodeKindFile, Path: "src/a.ts"},
			{ID: "m", Kind: intgraph.NodeKindModule, Path: "."},
			{
				ID: "c", Kind: intgraph.NodeKindComponent, Name: "Btn", Path: "src/Btn.tsx",
				LeadingComments: []string{"/** doc */"},
			},
		},
	}
	p := domainToProtoGraph(g)
	require.Len(t, p.Nodes, 3)

	byID := map[string]*commonv1.CodeGraphNode{}
	for _, n := range p.Nodes {
		byID[n.GetId()] = n
	}
	require.Equal(t, commonv1.NodeKind_NODE_KIND_FILE, byID["f"].GetKind())
	require.Equal(t, commonv1.NodeKind_NODE_KIND_MODULE, byID["m"].GetKind())
	require.Equal(t, commonv1.NodeKind_NODE_KIND_PACKAGE, byID["c"].GetKind())
	require.Equal(t, "TS_NODE_KIND_COMPONENT", byID["c"].GetAttributes()["kind"])
	require.Equal(t, []string{"/** doc */"}, byID["c"].GetLeadingComments())
}

func TestDomainToProtoGraph_EdgeKinds(t *testing.T) {
	g := intgraph.Graph{
		Edges: []intgraph.Edge{
			{ID: "e1", Kind: intgraph.EdgeKindImport, From: "a", To: "b"},
			{ID: "e2", Kind: intgraph.EdgeKindReExport, From: "a", To: "c"},
		},
	}
	p := domainToProtoGraph(g)
	require.Equal(t, commonv1.EdgeKind_EDGE_KIND_IMPORT, p.Edges[0].GetKind())
	require.Equal(t, commonv1.EdgeKind_EDGE_KIND_RE_EXPORT, p.Edges[1].GetKind())
}

func TestWarningsToProto_Mapping(t *testing.T) {
	in := []intgraph.Warning{
		{Kind: intgraph.WarningKindParseError, File: "a.ts", Message: "x"},
		{Kind: intgraph.WarningKindUnresolvedImport, Message: "y"},
		{Kind: intgraph.WarningKindTypeCheckFailure, Message: "z"},
	}
	out := warningsToProto(in)
	require.Equal(t, commonv1.CodeGraphWarningKind_CODE_GRAPH_WARNING_KIND_PARSE_ERROR, out[0].GetKind())
	require.Equal(t, "a.ts", out[0].GetFile())
	require.Equal(t, commonv1.CodeGraphWarningKind_CODE_GRAPH_WARNING_KIND_UNRESOLVED_IMPORT, out[1].GetKind())
	require.Equal(t, commonv1.CodeGraphWarningKind_CODE_GRAPH_WARNING_KIND_TYPE_CHECK_FAILURE, out[2].GetKind())
}

func TestWarningsToProto_EmptyReturnsNil(t *testing.T) {
	require.Nil(t, warningsToProto(nil))
	require.Nil(t, warningsToProto([]intgraph.Warning{}))
}
