package providercatalog_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/modelcatalog"
	"audio-tools/internal/providercatalog"
	"audio-tools/internal/sttengine"
)

// TestEveryModelHasAProviderReference is the orphan-model gate: a model in the
// model catalogue must be referenced by at least one provider whose declared
// capability matches. The test imports modelcatalog only from the test package;
// production providercatalog stays independent.
func TestEveryModelHasAProviderReference(t *testing.T) {
	models := modelcatalog.Default()
	for _, model := range models.All() {
		refs := 0
		for _, providerID := range model.ProviderIDs {
			provider, ok := providercatalog.ByID(providerID)
			require.Truef(t, ok, "model %q references unknown provider %q", model.ID, providerID)
			require.Containsf(t, provider.Capabilities, providercatalog.Capability(model.Capability),
				"model %q capability %q does not match provider %q capabilities %v",
				model.ID, model.Capability, providerID, provider.Capabilities)
			refs++
		}
		require.GreaterOrEqualf(t, refs, 1, "model %q is orphaned: no provider reference in modelcatalog", model.ID)
	}
}

// TestProviderEngineIDsMatchSTTEngineManifest keeps the STT stream-selection
// manifest and the provider catalogue in lockstep: every declared engine maps
// to a provider, and every manifest engine has a provider mapping.
func TestProviderEngineIDsMatchSTTEngineManifest(t *testing.T) {
	manifest := make(map[string]struct{})
	for _, engine := range sttengine.Default().Engines() {
		manifest[engine.ID] = struct{}{}
	}
	for _, provider := range providercatalog.All() {
		for _, engineID := range provider.EngineIDs {
			_, ok := manifest[engineID]
			require.Truef(t, ok, "provider %q declares engine %q absent from sttengine manifest", provider.ID, engineID)
		}
	}
	for engineID := range manifest {
		providerID, ok := providercatalog.ProviderIDForEngine(engineID)
		require.Truef(t, ok, "sttengine manifest engine %q has no provider mapping", engineID)
		require.NotEmpty(t, providerID)
	}
}
