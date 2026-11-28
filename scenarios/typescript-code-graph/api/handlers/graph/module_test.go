package graph

import (
	"io"
	"log"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"

	intgraph "typescript-code-graph/internal/graph"
	intrewrite "typescript-code-graph/internal/rewrite"
	sidecarmocks "typescript-code-graph/internal/sidecar/mocks"
)

func TestModule_RegistersAndIsMountable(t *testing.T) {
	fake := &sidecarmocks.FakeSidecarClient{}
	pathMu := intgraph.NewPathMutex()
	gsvc := intgraph.NewService(fake, pathMu)
	rsvc := intrewrite.NewService(intrewrite.NewMemoryPlanStore(), fake, pathMu)
	m := Module(gsvc, rsvc, log.New(io.Discard, "", 0))
	require.Equal(t, "graph", m.Name)
	require.NotNil(t, m.Mount)
	require.NotEmpty(t, m.Endpoints)

	r := mux.NewRouter()
	m.Mount(r)
}

func TestEndpoints_HaveStablePaths(t *testing.T) {
	ids := map[string]bool{}
	for _, ep := range Endpoints {
		require.NotEmpty(t, ep.Path, "endpoint %s has empty path", ep.ID)
		require.False(t, ids[ep.ID], "duplicate endpoint id %s", ep.ID)
		ids[ep.ID] = true
	}
	require.True(t, ids["graph_extract"])
	// rewrite_plan / rewrite_apply now live in handlers/rewrite/endpoints.go
	// (Phase 5 split). They MUST NOT appear here anymore.
	require.False(t, ids["rewrite_plan"], "rewrite_plan moved to handlers/rewrite/endpoints.go")
	require.False(t, ids["rewrite_apply"], "rewrite_apply moved to handlers/rewrite/endpoints.go")
}

func TestSchemaIsEmpty(t *testing.T) {
	require.Equal(t, "", Schema())
}
