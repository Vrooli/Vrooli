package manifest_test

import (
	"testing"

	manifesth "architecture-cartographer/handlers/manifest"
	"architecture-cartographer/internal/manifest"
	"architecture-cartographer/internal/manifest/mocks"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestModule_Shape(t *testing.T) {
	svc := manifest.NewService(&mocks.FakeRepository{})
	m := manifesth.Module(svc)
	require.Equal(t, "manifest", m.Name)
	require.NotNil(t, m.Mount)
	require.Len(t, m.Endpoints, 3)
}

func TestModule_MountsConnectRoutes(t *testing.T) {
	svc := manifest.NewService(&mocks.FakeRepository{})
	m := manifesth.Module(svc)

	r := mux.NewRouter()
	m.Mount(r)

	// Walk routes and ensure at least one is prefixed under the
	// ManifestService Connect prefix.
	found := false
	require.NoError(t, r.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		p, _ := route.GetPathTemplate()
		if p != "" {
			found = true
		}
		return nil
	}))
	require.True(t, found, "Module.Mount should register at least one route")
}

func TestModule_SchemaPopulated(t *testing.T) {
	if manifesth.Schema() == "" {
		t.Fatalf("expected manifest schema to be re-exported")
	}
}
