package bindings

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBoundaryInventoryMatchesAdmittedRegistryBindings(t *testing.T) {
	r, err := Load(repoRoot(t))
	require.NoError(t, err)
	// Assert each exported row is exactly one callable List entry, not a raw
	// manifest row or an inferred reachability edge.
	listed := r.List("", "")
	r.SetReachabilityResolver(func(context.Context, string) (string, error) { panic("inventory must not resolve targets") })
	inventory := r.BoundaryInventory()
	want := 0
	for _, binding := range listed {
		if binding.GetScenario() != "program-runtime" && binding.GetService() != "" && binding.GetMethod() != "" {
			want++
		}
	}
	require.Positive(t, want)
	require.Len(t, inventory, want)
	require.Equal(t, inventory, r.BoundaryInventory(), "inventory order must be deterministic")
	byID := make(map[string]bool, len(listed))
	for _, binding := range listed {
		byID[binding.GetId()] = true
	}
	for _, item := range inventory {
		require.True(t, byID[item.ID], "exported ID must be admitted by Registry.List")
		binding, ok := r.Binding(item.ID)
		require.True(t, ok)
		require.Equal(t, binding.GetScenario(), item.Target)
		require.Equal(t, binding.GetService(), item.Service)
		require.Equal(t, binding.GetMethod(), item.Method)
		require.Equal(t, "program-runtime", item.Caller)
		require.Equal(t, "program-runtime-admitted-capability", item.Provenance)
		require.NotEmpty(t, item.RequestType)
		require.NotEmpty(t, item.ResponseType)
	}
}

func TestBoundaryInventoryExcludesPrivateBridgeAndLocalBindings(t *testing.T) {
	r := fixtureRegistry(t, `{"name":"program-runtime","groups":[{"name":"ops","commands":[{"name":"read","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	for _, item := range r.BoundaryInventory() {
		require.NotEqual(t, "program-runtime", item.Target)
		require.NotContains(t, item.ID, "/internal/program-runtime/")
	}
}
