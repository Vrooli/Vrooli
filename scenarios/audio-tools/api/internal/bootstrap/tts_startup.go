package bootstrap

import (
	"context"
	"fmt"

	"audio-tools/internal/providercatalog"
	"audio-tools/internal/tts"
	"github.com/vrooli/api-core/discovery"
)

type ttsStartupPlan struct {
	providers    []providercatalog.Provider
	urls         map[string]string
	backend      string
	provider     providercatalog.Provider
	url          string
	providerID   string
	model        string
	speedMode    string
	cacheBackend string
}

func resolveTTSStartupPlan(ctx context.Context, env Env) (ttsStartupPlan, Env, error) {
	providers := declaredAutoTTSProviders()
	urls := map[string]string{"KokoroURL": env.KokoroURL, "KaleURL": env.KaleURL, "SherpaURL": env.SherpaURL}
	backend, err := resolveTTSBackend(env.TTSBackend, providers)
	if err != nil {
		return ttsStartupPlan{}, env, err
	}
	if err := tts.VerifyCatalog(tts.CanonicalAdapterMappings); err != nil {
		return ttsStartupPlan{}, env, fmt.Errorf("validate TTS voice catalog: %w", err)
	}
	provider, err := declaredTTSProviderForBackend(backend)
	if err != nil {
		return ttsStartupPlan{}, env, fmt.Errorf("unsupported AUDIO_TTS_BACKEND %q: %w", env.TTSBackend, err)
	}
	localURL, err := declaredTTSBackendURL(provider, backend, urls)
	if err != nil {
		return ttsStartupPlan{}, env, err
	}
	cacheBackend := declaredTTSCacheBackend(provider, backend)
	if env.TTSBackend == "auto" {
		cacheBackend = "auto"
	}
	if env.LPBSBaseURL == "" {
		if resolved, resolveErr := discovery.ResolveScenarioURLDefault(ctx, "landing-page-business-suite"); resolveErr == nil {
			env.LPBSBaseURL = resolved
		}
	}
	return ttsStartupPlan{
		providers: providers, urls: urls, backend: backend, provider: provider,
		url: localURL, providerID: provider.ID,
		model:     declaredTTSRuntimeModelIDForBackend(provider, backend),
		speedMode: declaredTTSSpeedMode(provider, backend), cacheBackend: cacheBackend,
	}, env, nil
}
