package graph_test

import (
	"testing"

	graphh "architecture-cartographer/handlers/graph"
	"architecture-cartographer/internal/graph/mocks"

	"github.com/stretchr/testify/require"
)

func TestModule_Shape(t *testing.T) {
	m := graphh.Module(&mocks.FakeService{})
	require.Equal(t, "graph", m.Name)
	require.NotNil(t, m.Mount)
	require.Len(t, m.Endpoints, 8)
	require.Equal(t, "zones.show", m.Endpoints[5].ID)
	require.Equal(t, "zones", m.Endpoints[5].Category)
	require.Equal(t, "slice.show", m.Endpoints[6].ID)
	require.Equal(t, "slice", m.Endpoints[6].Category)
	require.Equal(t, "archetype.infer", m.Endpoints[7].ID)
	require.Equal(t, "archetype", m.Endpoints[7].Category)
}

func TestModule_SchemaPopulated(t *testing.T) {
	if graphh.Schema() == "" {
		t.Fatalf("graphh.Schema() returned empty")
	}
}
