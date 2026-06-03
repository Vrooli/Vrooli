package providers_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"

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

// TestSeedIDsCoverAllProviders pins the full registered leaf set so a dropped or
// renamed seed surfaces immediately. Phase 3/4 shipped the first three live
// leaves; Phase 8 adds the remaining live leaves (swarm backlog/initiative,
// prompt-manager skill/action, ui-health widgets, knowledge-observatory docs)
// and the capability_gap stubs (the Track-A checklist from ai-search-routing.md).
func TestSeedIDsCoverAllProviders(t *testing.T) {
	got := providers.SeedIDs()
	require.Equal(t, []string{
		// capability_gap stubs (Phase 8) — sorted in with the rest by id.
		"agent-manager.runs",
		"architecture-cartographer.domain-map",
		// live leaves (Phase 3/4 + Phase 8).
		"cli-health.commands",
		"code-reference.code",
		"command-center.metrics",
		"contract-registry.contracts",
		"git-control-tower.git-provenance",
		"knowledge-observatory.docs",
		"product-manager-agent.requirements",
		"prompt-manager.action",
		"prompt-manager.skill",
		"scenario-dependency-analyzer.resources",
		"scenario-dependency-analyzer.scenarios",
		"swarm-manager.backlog",
		"swarm-manager.initiative",
		"swarm-manager.records",
		"ui-health.surfaces",
		"ui-health.widgets",
		"vrooli-onboarding.config",
	}, got)
}

// TestSeedStatesMatchLiveVsGap pins which leaves are live (ACTIVE, callable
// endpoint) versus tracked gaps (CAPABILITY_GAP, no endpoint). This guards the
// invariant that a gap stub is a TODO row, never a live provider the router
// would fan out to.
func TestSeedStatesMatchLiveVsGap(t *testing.T) {
	wantGaps := map[string]bool{
		"agent-manager.runs":                     true,
		"architecture-cartographer.domain-map":   true,
		"code-reference.code":                    true,
		"command-center.metrics":                 true,
		"contract-registry.contracts":            true,
		"git-control-tower.git-provenance":       true,
		"product-manager-agent.requirements":     true,
		"scenario-dependency-analyzer.resources": true,
		"scenario-dependency-analyzer.scenarios": true,
		"vrooli-onboarding.config":               true,
	}
	for id, d := range providers.Seeds() {
		internalregistry.Normalize(d)
		if wantGaps[id] {
			require.Equal(t, registryv1.ProviderState_PROVIDER_STATE_CAPABILITY_GAP, d.GetState(), "%s must be a gap stub", id)
			require.Nil(t, d.GetEndpoint(), "%s gap stub must carry no endpoint", id)
			require.NotEmpty(t, d.GetIntendedHome(), "%s gap stub must declare intended_home", id)
		} else {
			require.Equal(t, registryv1.ProviderState_PROVIDER_STATE_ACTIVE, d.GetState(), "%s must be a live provider", id)
			require.NotNil(t, d.GetEndpoint(), "%s live provider must carry an endpoint", id)
		}
	}
}
