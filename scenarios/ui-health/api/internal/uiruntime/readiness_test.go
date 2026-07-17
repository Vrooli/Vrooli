package uiruntime

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadinessSurfaceFindingsEnforcesRequiredSurfaceLifecycle(t *testing.T) {
	wants := []requiredSurface{{id: "results", required: true, states: map[string]bool{"ready": true, "empty": true}}}
	missing := readinessSurfaceFindings(`{"experienceSurfaces":[]}`, wants, "http://scenario", "desktop")
	require.Equal(t, "runtime_required_surface_missing", missing[0].Code)

	hidden := readinessSurfaceFindings(`{"experienceSurfaces":[{"id":"results","state":"ready","visible":false}]}`, wants, "http://scenario", "desktop")
	require.Equal(t, "runtime_required_surface_hidden", hidden[0].Code)

	invalid := readinessSurfaceFindings(`{"experienceSurfaces":[{"id":"results","state":"loading","visible":true}]}`, wants, "http://scenario", "desktop")
	require.Equal(t, "runtime_required_surface_invalid_state", invalid[0].Code)

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
