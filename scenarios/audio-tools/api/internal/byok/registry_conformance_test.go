package byok_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/byok"
	"audio-tools/internal/providercatalog"
)

// TestRegistriesMatchProviderCatalog closes both directions of the BYOK
// provider/adapter seam: every registered adapter names an active credential
// provider that declares the capability, and every active credential provider
// that declares the capability has an adapter. A new BYOK provider in the
// catalog without an adapter (or a retired provider with a lingering adapter)
// fails here instead of silently never being selected.
func TestRegistriesMatchProviderCatalog(t *testing.T) {
	registries := byok.NewRegistries()

	capabilities := map[string]providercatalog.Capability{
		"stt":       providercatalog.CapabilitySTT,
		"tts":       providercatalog.CapabilityTTS,
		"summarize": providercatalog.CapabilitySummarize,
	}
	adapterIDs := map[string]map[string]bool{
		"stt":       keys(registries.STT),
		"tts":       keys(registries.TTS),
		"summarize": keys(registries.Summarize),
	}

	for capability, ids := range adapterIDs {
		for providerID := range ids {
			provider, ok := providercatalog.ByID(providerID)
			require.Truef(t, ok, "%s adapter %q has no provider catalog record", capability, providerID)
			require.Equalf(t, providercatalog.StatusActive, provider.Status, "%s adapter %q is not an active provider", capability, providerID)
			require.Truef(t, provider.Kind == providercatalog.KindBYOKAPI || provider.Kind == providercatalog.KindGateway,
				"%s adapter %q is not a credential provider (kind=%s)", capability, providerID, provider.Kind)
			require.Truef(t, declares(provider, capabilities[capability]),
				"%s adapter %q does not declare capability %s", capability, providerID, capability)
		}
	}

	for capability, providerCapability := range capabilities {
		for _, provider := range providercatalog.ByCapability(providerCapability) {
			if provider.Kind != providercatalog.KindBYOKAPI && provider.Kind != providercatalog.KindGateway {
				continue
			}
			require.Truef(t, adapterIDs[capability][provider.ID],
				"active %s provider %q has no BYOK adapter", capability, provider.ID)
		}
	}
}

func keys[T any](m map[string]T) map[string]bool {
	out := make(map[string]bool, len(m))
	for id := range m {
		out[id] = true
	}
	return out
}

func declares(provider providercatalog.Provider, capability providercatalog.Capability) bool {
	for _, declared := range provider.Capabilities {
		if declared == capability {
			return true
		}
	}
	return false
}
