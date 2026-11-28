//go:build audio_lab

package bootstrap

import (
	"slices"
	"testing"

	"audio-tools/internal/providercatalog"
)

func TestExperimentProviderFactoryCoversDeclaredLocalSTTEngines(t *testing.T) {
	factory := evalProviderFactory(Env{}, nil)
	for _, provider := range providercatalog.Default().Active() {
		if provider.Kind != providercatalog.KindLocalResource || !slices.Contains(provider.Capabilities, providercatalog.CapabilitySTT) {
			continue
		}
		if provider.STTEvalAdapterID == "" {
			t.Errorf("active local STT provider %q has no declared experiment adapter", provider.ID)
			continue
		}
		for _, engineID := range provider.EngineIDs {
			resolved, ok := providercatalog.ProviderIDForEngine(engineID)
			if !ok || resolved != provider.ID {
				t.Errorf("catalog engine %q resolved to %q, want provider %q", engineID, resolved, provider.ID)
				continue
			}
			if factory(engineID) == nil {
				t.Errorf("declared engine %q for provider %q has no experiment adapter", engineID, provider.ID)
				continue
			}
		}
	}
	if factory("unregistered-engine") != nil {
		t.Fatal("unknown engine unexpectedly selected an experiment adapter")
	}
}
