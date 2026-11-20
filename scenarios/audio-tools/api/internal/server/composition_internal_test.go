package server

import (
	"strings"
	"testing"

	"audio-tools/internal/modules"
	"audio-tools/internal/testutil/mocks"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

// TestCompose_MountsEveryDeclaredEndpoint is the composition drift gate.
// modules.AllEndpoints() is the static endpoint inventory used by docs and
// gateway registration; Compose is the only runtime mount point. A module that
// stops mounting a procedure, or an endpoint descriptor that is added without
// a module, fails here instead of surfacing as a 404 in production.
//
// The composition intentionally tolerates absent optional services (nil DB,
// nil chains, nil stores): handler constructors validate only their required
// logger/clock and defer dependency use to request time, so the route table can
// be built without a live environment.
func TestCompose_MountsEveryDeclaredEndpoint(t *testing.T) {
	srv := Compose(CompositionDeps{
		Logger: &mocks.FakeLogger{},
		Env:    Env{TestIsolationActive: func() bool { return false }},
	})
	require.NotNil(t, srv)
	require.NotNil(t, srv.router)

	mounted := make([]string, 0, 32)
	err := srv.router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		template, err := route.GetPathTemplate()
		if err != nil {
			return nil // regex-only routes have no template
		}
		mounted = append(mounted, template)
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, mounted, "compose registered no routes")

	for _, endpoint := range modules.AllEndpoints() {
		require.Truef(t, routeServes(mounted, endpoint.Path),
			"endpoint %s %s is declared in modules.AllEndpoints but no module mounts it (mounted: %v)",
			endpoint.Method, endpoint.Path, mounted)
	}
}

// routeServes reports whether a mounted route template serves path. Connect
// mounts use PathPrefix (one procedure per method under the service prefix),
// while simple probes are exact routes.
func routeServes(templates []string, path string) bool {
	for _, template := range templates {
		if template == path {
			return true
		}
		if strings.HasSuffix(template, "/") && strings.HasPrefix(path, template) {
			return true
		}
	}
	return false
}
