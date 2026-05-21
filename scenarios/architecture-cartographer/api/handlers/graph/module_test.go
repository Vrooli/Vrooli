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
	require.Len(t, m.Endpoints, 5)
}

func TestModule_SchemaPopulated(t *testing.T) {
	if graphh.Schema() == "" {
		t.Fatalf("graphh.Schema() returned empty")
	}
}
