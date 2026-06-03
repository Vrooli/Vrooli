package providers_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"search-hub/internal/providers"
	internalregistry "search-hub/internal/registry"
)

// TestSeedsAreValidDescriptors guards every embedded provider seed: it must
// parse (Seeds() panics otherwise) and — after Normalize — pass the same
// Validate gate the registry applies on RegisterProvider. This means an
// operator (or the Phase 8 bulk-registration step) can register any shipped
// seed and it will be accepted, with no drift between "the bytes we embed" and
// "what the registry considers valid".
func TestSeedsAreValidDescriptors(t *testing.T) {
	seeds := providers.Seeds()
	require.NotEmpty(t, seeds, "at least the cli-health.commands seed must ship")

	for id, d := range seeds {
		require.Equal(t, id, d.GetProviderId(), "seed map key must equal the descriptor's provider_id")
		internalregistry.Normalize(d)
		require.NoErrorf(t, internalregistry.Validate(d), "embedded seed %q must be a valid registerable descriptor", id)
	}
}

// TestSeedIDsCoverLivePhase4Providers pins the live leaves Phase 3/4 register so
// a dropped or renamed seed surfaces immediately. Phase 8 adds the rest
// (ui-health.widgets, swarm-manager.backlog/.initiative, knowledge-observatory,
// prompt-manager); this list grows then.
func TestSeedIDsCoverLivePhase4Providers(t *testing.T) {
	got := providers.SeedIDs()
	require.Equal(t, []string{
		"cli-health.commands",
		"swarm-manager.records",
		"ui-health.surfaces",
	}, got)
}
