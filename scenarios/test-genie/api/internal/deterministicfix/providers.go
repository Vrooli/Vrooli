package deterministicfix

import (
	"test-genie/internal/orchestrator/phases"
)

// delegatedProviderScenarios returns the provider scenario names for every
// delegated phase in the default catalog, in registration order with
// duplicates removed. These are the scenarios that may serve the shared Fix RPC.
func delegatedProviderScenarios() []string {
	catalog := phases.NewDefaultCatalog(0)
	if catalog == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, spec := range catalog.All() {
		if spec.Delegated == nil {
			continue
		}
		name := spec.Delegated.ProviderScenario
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}
