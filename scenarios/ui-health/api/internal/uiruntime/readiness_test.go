package uiruntime

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadinessSurfaceFindingsEnforcesRequiredSurfaceLifecycle(t *testing.T) {
	wants := []requiredSurface{{id: "results", required: true, kind: "async", states: map[string]bool{"loading": true, "ready": true, "empty": true}}}
	missing := readinessSurfaceFindings(`{"experienceSurfaces":[]}`, wants, "http://scenario", "desktop")
	require.Equal(t, "runtime_required_surface_missing", missing[0].Code)

	hidden := readinessSurfaceFindings(`{"experienceSurfaces":[{"id":"results","state":"ready","visible":false}]}`, wants, "http://scenario", "desktop")
	require.Equal(t, "runtime_required_surface_hidden", hidden[0].Code)

	invalid := readinessSurfaceFindings(`{"experienceSurfaces":[{"id":"results","state":"loading","visible":true}]}`, wants, "http://scenario", "desktop")
	require.Equal(t, "runtime_required_surface_unsettled", invalid[0].Code)

	ready := readinessSurfaceFindings(`{"experienceSurfaces":[{"id":"results","state":"ready","visible":true}]}`, wants, "http://scenario", "desktop")
	require.Empty(t, ready)
}

func TestReadinessSurfaceFindingsAggregatesRequiredAndOptionalFailures(t *testing.T) {
	wants := []requiredSurface{
		{id: "results", required: true, states: map[string]bool{"ready": true, "error": true}},
		{id: "activity", required: false, states: map[string]bool{"ready": true, "error": true}},
	}
	partial := readinessSurfaceFindings(`{"experienceSurfaces":[{"id":"results","state":"ready","visible":true},{"id":"activity","state":"error","visible":true}]}`, wants, "http://scenario", "desktop")
	require.Len(t, partial, 1)
	require.Equal(t, "runtime_page_partial", partial[0].Code)

	failure := readinessSurfaceFindings(`{"experienceSurfaces":[{"id":"results","state":"error","visible":true},{"id":"activity","state":"ready","visible":true}]}`, wants, "http://scenario", "desktop")
	require.Len(t, failure, 1)
	require.Equal(t, "runtime_page_required_surface_error", failure[0].Code)
}

func TestReadinessProfilePrefersConcreteStateSetupRoutes(t *testing.T) {
	var profile readinessProfile
	require.NoError(t, json.Unmarshal([]byte(`{"pages":[{"routes":["/assets/:id"],"runtimeRoutes":["/assets/Button?tab=preview"],"regions":[]}]}`), &profile))
	require.Equal(t, []string{"/assets/Button?tab=preview"}, profile.routes())
}

func TestReadinessProfileSkipsUnresolvedTemplatesAndReportsGap(t *testing.T) {
	var profile readinessProfile
	require.NoError(t, json.Unmarshal([]byte(`{"pages":[{"routes":["/design/:scenario"],"regions":[]},{"routes":["/design/:scenario/:page"],"runtimeRoutes":["/design/switchboard/dashboard"],"regions":[]}]}`), &profile))
	require.Equal(t, []string{"/design/switchboard", "/design/switchboard/dashboard"}, profile.routes())
	require.Empty(t, profile.unresolvedRouteTemplates())
}

func TestReadinessProfileExpandsTemplatesFromConcreteSiblingRoutes(t *testing.T) {
	var profile readinessProfile
	require.NoError(t, json.Unmarshal([]byte(`{"pages":[{"routes":["/assets/:id"],"runtimeRoutes":["/assets/react-component-library%3AButton?tab=preview"]},{"routes":["/assets/:id/preview"]},{"routes":["/design/:scenario"]},{"routes":["/design/:scenario/:page"],"runtimeRoutes":["/design/switchboard/dashboard"]}]}`), &profile))
	require.Equal(t, []string{
		"/assets/react-component-library%3AButton?tab=preview",
		"/assets/react-component-library%3AButton/preview",
		"/design/switchboard",
		"/design/switchboard/dashboard",
	}, profile.routes())
	require.Empty(t, profile.unresolvedRouteTemplates())
}

func TestReadinessProfileReportsTemplatesWithoutConcreteSibling(t *testing.T) {
	var profile readinessProfile
	require.NoError(t, json.Unmarshal([]byte(`{"pages":[{"routes":["/design/:scenario"],"regions":[]}]}`), &profile))
	require.Equal(t, []string{"/"}, fallbackRoutes(&profile))
	require.Equal(t, []string{"/design/:scenario"}, profile.unresolvedRouteTemplates())
}

func fallbackRoutes(profile *readinessProfile) []string {
	if routes := profile.routes(); len(routes) > 0 {
		return routes
	}
	return []string{"/"}
}
