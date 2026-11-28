package bootstrap

import (
	"fmt"
	"sort"

	"audio-tools/internal/capabilities"
	"audio-tools/internal/modelcatalog"
	"audio-tools/internal/providercatalog"
)

func declaredAutoTTSProviders() []providercatalog.Provider {
	providers := make([]providercatalog.Provider, 0)
	for _, provider := range providercatalog.All() {
		if autoTTSProviderEligible(provider, modelcatalog.Default()) {
			providers = append(providers, provider)
		}
	}
	sortAutoTTSProviders(providers)
	return providers
}

func sortAutoTTSProviders(providers []providercatalog.Provider) {
	sort.SliceStable(providers, func(i, j int) bool {
		return providers[i].TTSRuntime.AutoPriority < providers[j].TTSRuntime.AutoPriority
	})
}

// autoTTSProviderEligible keeps declaration priority subordinate to the
// provider/model registries. Conditional auto is a narrow fallback policy for
// explicitly opted-in planned providers; it allows trying a candidate only
// when the runtime is available and never changes model promotion status.
func autoTTSProviderEligible(provider providercatalog.Provider, models *modelcatalog.Catalog) bool {
	runtime := provider.TTSRuntime
	if runtime == nil || runtime.AutoPriority <= 0 || models == nil || provider.Tier != providercatalog.TierLocal || provider.Kind != providercatalog.KindLocalResource {
		return false
	}
	for _, model := range models.ByProvider(provider.ID) {
		if model.Capability != modelcatalog.CapabilityTTS || !model.HasRole(runtime.ModelRole) {
			continue
		}
		if model.Commercial != modelcatalog.CommercialYes {
			continue
		}
		if runtime.ConditionalAuto {
			// This opt-in is deliberately narrower than promotion: only a
			// planned provider and a candidate fast/edge model may use it.
			if provider.Status == providercatalog.StatusPlanned && runtime.ModelRole != "default" && model.Readiness == modelcatalog.ReadinessCandidate {
				return true
			}
		}
		// Ordinary auto routing remains behind active-provider/default-model
		// promotion and qualification gates.
		if provider.Status == providercatalog.StatusActive && runtime.ModelRole == "default" && model.DefaultEligible() {
			return true
		}
	}
	return false
}

func declaredTTSProviderForBackend(backendID string) (providercatalog.Provider, error) {
	for _, provider := range providercatalog.All() {
		if provider.Tier == providercatalog.TierLocal && provider.TTSRuntime != nil && (provider.TTSRuntime.BackendID == backendID || (provider.TTSRuntime.SharedRuntime != nil && provider.TTSRuntime.SharedRuntime.BackendID == backendID)) {
			return provider, nil
		}
	}
	return providercatalog.Provider{}, fmt.Errorf("audio-tools/bootstrap: no local TTS provider declares backend %q", backendID)
}

func declaredTTSBackendURL(provider providercatalog.Provider, backendID string, urls map[string]string) (string, error) {
	if provider.TTSRuntime == nil {
		return "", fmt.Errorf("audio-tools/bootstrap: provider %q has no TTS runtime declaration", provider.ID)
	}
	field := provider.TTSRuntime.URLField
	if shared := provider.TTSRuntime.SharedRuntime; shared != nil && shared.BackendID == backendID {
		field = shared.URLField
	}
	return declaredTTSURLField(provider.ID, field, urls)
}

// declaredTTSHealthURL keeps both readiness and request-liveness probes on the
// runtime endpoint declared beside the provider's URL and backend identity.
func declaredTTSHealthURL(provider providercatalog.Provider, backendID, baseURL string) (string, error) {
	if provider.TTSRuntime == nil {
		return "", fmt.Errorf("audio-tools/bootstrap: provider %q has no TTS runtime declaration", provider.ID)
	}
	runtime := provider.TTSRuntime
	if shared := runtime.SharedRuntime; shared != nil && shared.BackendID == backendID {
		return baseURL + shared.HealthPath, nil
	}
	if runtime.BackendID != backendID {
		return "", fmt.Errorf("audio-tools/bootstrap: provider %q does not declare TTS backend %q", provider.ID, backendID)
	}
	return baseURL + runtime.HealthPath, nil
}

func declaredTTSCacheBackend(provider providercatalog.Provider, backendID string) string {
	runtime := provider.TTSRuntime
	if runtime != nil && runtime.SharedRuntime != nil && runtime.SharedRuntime.BackendID == backendID {
		return runtime.SharedRuntime.CachePrefix + runtime.CacheBackend
	}
	if runtime == nil {
		return ""
	}
	return runtime.CacheBackend
}

func declaredTTSSpeedMode(provider providercatalog.Provider, backendID string) string {
	runtime := provider.TTSRuntime
	if runtime == nil {
		return ""
	}
	if shared := runtime.SharedRuntime; shared != nil && shared.BackendID == backendID && shared.SpeedMode != "" {
		return shared.SpeedMode
	}
	return runtime.SpeedMode
}

// declaredTTSURL uses a closed mapping from catalog URL fields to Env values.
// Catalog data cannot name arbitrary environment variables or trigger reflection.
func declaredTTSURL(provider providercatalog.Provider, urls map[string]string) (string, error) {
	if provider.TTSRuntime == nil {
		return "", fmt.Errorf("audio-tools/bootstrap: provider %q has no TTS runtime declaration", provider.ID)
	}
	return declaredTTSURLField(provider.ID, provider.TTSRuntime.URLField, urls)
}

func declaredTTSURLField(providerID, field string, urls map[string]string) (string, error) {
	switch field {
	case "KokoroURL":
		return urls["KokoroURL"], nil
	case "KaleURL":
		return urls["KaleURL"], nil
	case "SherpaURL":
		return urls["SherpaURL"], nil
	default:
		return "", fmt.Errorf("audio-tools/bootstrap: provider %q declares unsupported TTS URL field %q", providerID, field)
	}
}

func resolveTTSBackend(requested string, providers []providercatalog.Provider) (string, error) {
	if requested != "auto" {
		return requested, nil
	}
	if len(providers) == 0 {
		return "", fmt.Errorf("audio-tools/bootstrap: no qualified TTS providers declare auto_priority")
	}
	// Runtime selection is data-declared priority. Availability/startup belongs
	// to the first TTS request so bootstrap never starts or probes model services.
	return providers[0].TTSRuntime.BackendID, nil
}

func hasCapability(defs []capabilities.Def, id string) bool {
	for _, def := range defs {
		if def.ID == id {
			return true
		}
	}
	return false
}
