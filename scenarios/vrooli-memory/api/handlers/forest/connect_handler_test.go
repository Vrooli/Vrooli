package forest

import (
	"testing"

	"github.com/stretchr/testify/require"

	internalforest "vrooli-memory/internal/forest"
)

func TestNodesProtoPreservesForestNodeIdentity(t *testing.T) { // [REQ:VMEM-P1-003] [REQ:VMEM-P1-006]
	nodes := nodesProto([]internalforest.Node{
		{ID: "summary-1", FacetID: "episode", Depth: 2},
		{ID: "entry-1", EntryID: "entry-1", FacetID: "standing-rule", Depth: 0},
	})

	require.Len(t, nodes, 2)
	require.Equal(t, "summary-1", nodes[0].GetId())
	require.Empty(t, nodes[0].GetEntryId())
	require.Equal(t, int32(2), nodes[0].GetDepth())
	require.Equal(t, "entry-1", nodes[1].GetId())
	require.Equal(t, "entry-1", nodes[1].GetEntryId())
	require.Equal(t, "standing-rule", nodes[1].GetFacetId())
}
