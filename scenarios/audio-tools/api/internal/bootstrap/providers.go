package bootstrap

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	lpbsclients "audio-tools/integrations/lpbs/clients"
	"audio-tools/internal/ai/chains"
	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/ai/ttschain"
	"audio-tools/internal/byok"
	"audio-tools/internal/byokstore"
	"audio-tools/internal/capabilities"
	"audio-tools/internal/logx"
	"audio-tools/internal/modelcatalog"
	"audio-tools/internal/providercatalog"
	"audio-tools/internal/store"
	sttpipeline "audio-tools/internal/stt/pipeline"
	intsumm "audio-tools/internal/summarize"
	inttts "audio-tools/internal/tts"
)

// declaredBYOKCredentialCheckers builds the generic remote-credential health
// checkers from provider declarations. Probe endpoint/header facts belong to
// the catalog; bootstrap only supplies the encrypted store boundary.
func declaredBYOKCredentialCheckers(
	list func(context.Context) ([]byokstore.Credential, error),
	get func(context.Context, string, string) (string, bool, error),
) (map[string]capabilities.Checker, error) {
	checkers := make(map[string]capabilities.Checker)
	for _, provider := range providercatalog.All() {
		if provider.Status != providercatalog.StatusActive || provider.Tier != providercatalog.TierBYOK || provider.Probe == nil {
			continue
		}
		if len(provider.Capabilities) != 1 {
			return nil, fmt.Errorf("provider %q has a BYOK probe but %d capabilities; provider health currently requires one credential capability", provider.ID, len(provider.Capabilities))
		}
		capability := string(provider.Capabilities[0])
		checkers[provider.ID] = &capabilities.BYOKCredentialChecker{
			ProviderID: provider.ID,
			Capability: capability,
			List:       list,
			Get:        get,
			Probe:      capabilities.ProbeBYOKEndpoint(provider.ID),
		}
	}
	return checkers, nil
}

// Chains bundles the three provider chains plus the read-only coordinator
// that handlers use to surface availability snapshots.
type Chains struct {
	STT         *sttchain.Chain
	TTS         *ttschain.Chain
	Summarize   *summarizechain.Chain
	Coordinator *chains.Coordinator
}

type providerConfigReader interface {
	Get(context.Context) (store.ProviderConfig, error)
}

// declaredLocalSTTEngineProviders binds executable adapters to stable provider
// declarations, then projects the catalog-owned engine IDs onto those adapters.
// This keeps engine identity in catalog.seed.json instead of repeating it in
// the composition root when an engine is renamed or added to an existing
// adapter family.
func declaredLocalSTTEngineProviders(whisper, nativeStreaming, sherpa sttchain.Provider) map[string]sttchain.Provider {
	byAdapter := map[string]sttchain.Provider{
		"whisper-local": whisper,
		"vrooli-stream": nativeStreaming,
		"sherpa-stream": sherpa,
	}
	return declaredLocalSTTEngineProvidersFor(providercatalog.All(), byAdapter)
}

// declaredLocalSTTEngineProvidersFor lets a new provider reuse an implemented
// engine family by declaring its STT adapter key and engine IDs in the catalog.
func declaredLocalSTTEngineProvidersFor(declarations []providercatalog.Provider, byAdapter map[string]sttchain.Provider) map[string]sttchain.Provider {
	engines := make(map[string]sttchain.Provider)
	for _, declaration := range declarations {
		adapter, supported := byAdapter[declaration.STTEvalAdapterID]
		if !supported || adapter == nil || declaration.Status != providercatalog.StatusActive || declaration.Tier != providercatalog.TierLocal {
			continue
		}
		for _, engineID := range declaration.EngineIDs {
			engines[engineID] = adapter
		}
	}
	return engines
}

func applyStoredProviderRouting(ctx context.Context, reader providerConfigReader, coordinator *chains.Coordinator, env Env) error {
	cfg, err := reader.Get(ctx)
	if err != nil {
		return err
	}
	if coordinator != nil {
		coordinator.Reconfigure(chains.ClampEnablement(providerConfigRouting(cfg), providerRoutingUpperBounds(env)))
	}
	return nil
}

func providerRoutingUpperBounds(env Env) chains.Config {
	return chains.Config{BYOKEnabled: env.EnableBYOK, VrooliEnabled: env.EnableVrooli, LocalEnabled: env.EnableLocal}
}

func providerConfigRouting(cfg store.ProviderConfig) chains.Config {
	return chains.Config{
		BYOKEnabled: cfg.BYOKEnabled, VrooliEnabled: cfg.VrooliEnabled, LocalEnabled: cfg.LocalEnabled,
		TTLByOK:   time.Duration(cfg.AvailTTLBYOKSeconds) * time.Second,
		TTLVrooli: time.Duration(cfg.AvailTTLVrooliSecs) * time.Second,
	}
}

// BuildChains wires the Local/BYOK providers for STT, TTS, and Summarize
// using the supplied capability singletons and per-domain BYOK registries.
func BuildChains(
	env Env,
	whisperSvc *sttpipeline.WhisperTranscriber,
	ttsServices map[string]*inttts.Service,
	summarizer *intsumm.Summarizer,
	byokRegistries byok.Registries,
	logger logx.Logger,
) Chains {
	// Keep the owned hosted tier in the same composition root as Local and
	// BYOK. The clients currently report unavailable until the shared gateway
	// contract is deployed, but wiring them here makes that state observable
	// and prevents the enable flag from becoming a dead configuration knob.
	lpbsSTT := sttchain.NewVrooliProvider(lpbsclients.NewSTTClient(env.LPBSBaseURL))
	lpbsTTS := ttschain.NewVrooliProvider(lpbsclients.NewTTSClient(env.LPBSBaseURL))
	lpbsSummarize := summarizechain.NewVrooliProvider(lpbsclients.NewSummarizeClient(env.LPBSBaseURL))
	// Local-tier engines share the Local tier but are distinct providers
	// resolved per-session by StreamStart.EngineID. Whisper (batch) also
	// serves the unary path; Kyutai (streaming) is selectable only on the
	// streaming path. Engine IDs must match internal/sttengine/manifest.json;
	// the provider catalog declares each engine ID and its STT adapter key.
	// Bootstrap binds those adapter keys to constructors here because sttchain
	// cannot import sttengine without creating a cycle through egress. An engine
	// whose resource is down fails its IsAvailable probe and is hidden in the
	// picker.
	whisperLocal := sttchain.NewLocalProvider(whisperSvc)
	kyutaiLocal := sttchain.NewKyutaiProvider(env.KyutaiURL)
	sherpaLocal := sttchain.NewSherpaProvider(env.SherpaURL)
	virtualReplay := sttchain.NewVirtualReplayProvider()
	localEngines := declaredLocalSTTEngineProviders(whisperLocal, kyutaiLocal, sherpaLocal)
	// The replay-only engine is intentionally not part of the product catalog.
	// Soak leases add it only to their isolated session.
	localEngines["virtual-replay"] = virtualReplay
	batchModels := declaredWhisperBatchModels(whisperLocal)
	for modelID, provider := range declaredVLLMBatchModels(env.RumbleVLLMURL) {
		batchModels[modelID] = provider
	}
	defaultTTSProvider := declaredQualifiedDefaultTTSProvider(providercatalog.All(), modelcatalog.Default())
	localTTSRuntimeProvider := defaultTTSProvider
	localTTSServiceProviderID := defaultTTSProvider.ID
	localTTSProviderID, localTTSModelID := defaultTTSProvider.ID, declaredTTSModelID(defaultTTSProvider.ID, defaultTTSProvider.TTSRuntime.ModelRole)
	var localTTSAdapterID string
	var localTTSVoiceMap inttts.AdapterVoiceMap
	localTTSAdapterID, localTTSVoiceMap = declaredTTSAdapter(localTTSProviderID)
	if aliases := providercatalog.RuntimeAliasesForProvider(localTTSProviderID); len(aliases) > 0 {
		localTTSProviderID = aliases[0]
	}
	if env.TTSBackend != "auto" {
		if selected, err := declaredTTSProviderForBackend(env.TTSBackend); err == nil {
			localTTSRuntimeProvider = selected
			localTTSServiceProviderID = selected.ID
			localTTSModelID = declaredTTSModelIDForBackend(selected, env.TTSBackend)
			localTTSAdapterID, localTTSVoiceMap = declaredTTSAdapter(selected.ID)
			localTTSProviderID = selected.ID
			if aliases := providercatalog.RuntimeAliasesForProvider(selected.ID); len(aliases) > 0 {
				localTTSProviderID = aliases[0]
			}
		}
	}
	var localTTS *ttschain.LocalProvider
	if env.TTSBackend == "auto" {
		// All declared auto routes share one startup slot, including future
		// compatible local TTS declarations.
		providers := declaredAutoTTSProviders()
		resources := make([]string, 0, len(providers))
		for _, provider := range providers {
			resources = append(resources, provider.ResourceSlug)
		}
		backendEnsurer := ttschain.NewCLIBackendEnsurer(resources...)
		configured := make([]*ttschain.LocalProvider, 0, len(providers))
		for _, provider := range providers {
			runtime := provider.TTSRuntime
			svc := ttsServices[provider.ID]
			if svc == nil {
				continue
			}
			modelID := declaredTTSModelID(provider.ID, runtime.ModelRole)
			adapterID, voiceMap := declaredTTSAdapter(provider.ID)
			providerID := provider.ID
			if aliases := providercatalog.RuntimeAliasesForProvider(provider.ID); len(aliases) > 0 {
				providerID = aliases[0]
			}
			local := ttschain.NewConfiguredLocalProvider(svc, nil, providerID, modelID, adapterID, voiceMap)
			local.SetCacheBackend(runtime.CacheBackend)
			local.SetBackendEnsurer(backendEnsurer, provider.ResourceSlug)
			configured = append(configured, local)
		}
		localTTS = ttschain.NewFailoverLocalProviders(configured...)
	}
	if localTTS == nil {
		localTTS = ttschain.NewConfiguredLocalProvider(ttsServices[localTTSServiceProviderID], nil, localTTSProviderID, localTTSModelID, localTTSAdapterID, localTTSVoiceMap)
		localTTS.SetCacheBackend(declaredTTSCacheBackend(localTTSRuntimeProvider, env.TTSBackend))
	}
	stt := sttchain.NewChain(sttchain.Options{
		Local:          whisperLocal,
		BatchModels:    batchModels,
		LocalEngines:   localEngines,
		BYOK:           sttchain.NewBYOKProvider(byokRegistries.STT),
		EnableLocal:    env.EnableLocal,
		LocalFirst:     true,
		EnableBYOK:     env.EnableBYOK,
		EnableVrooli:   env.EnableVrooli,
		Vrooli:         lpbsSTT,
		AvailTTLByOK:   env.AvailTTLBYOK,
		AvailTTLVrooli: env.AvailTTLVrooli,
		Logx:           logger,
	})
	tts := ttschain.NewChain(ttschain.Options{
		Local:          localTTS,
		BYOK:           ttschain.NewBYOKProvider(byokRegistries.TTS),
		EnableLocal:    env.EnableLocal,
		LocalFirst:     true,
		EnableBYOK:     env.EnableBYOK,
		EnableVrooli:   env.EnableVrooli,
		Vrooli:         lpbsTTS,
		AvailTTLByOK:   env.AvailTTLBYOK,
		AvailTTLVrooli: env.AvailTTLVrooli,
		Logx:           logger,
	})
	summ := summarizechain.NewChain(summarizechain.Options{
		Local:          summarizechain.NewLocalProvider(summarizer, env.SummarizeDefaultModel),
		BYOK:           summarizechain.NewBYOKProvider(byokRegistries.Summarize),
		EnableLocal:    env.EnableLocal,
		EnableBYOK:     env.EnableBYOK,
		EnableVrooli:   env.EnableVrooli,
		Vrooli:         lpbsSummarize,
		AvailTTLByOK:   env.AvailTTLBYOK,
		AvailTTLVrooli: env.AvailTTLVrooli,
		Logx:           logger,
	})
	return Chains{
		STT:         stt,
		TTS:         tts,
		Summarize:   summ,
		Coordinator: &chains.Coordinator{STT: stt, TTS: tts, Summarize: summ},
	}
}

// declaredTTSModelID binds result and usage identity to the model registry.
// Resource adapters retain ownership of their HTTP protocol model names.
func declaredTTSModelID(providerID, role string) string {
	for _, model := range modelcatalog.Default().ByProvider(providerID) {
		if model.Capability != modelcatalog.CapabilityTTS {
			continue
		}
		if role == "default" && model.Readiness != modelcatalog.ReadinessAvailable {
			continue
		}
		if model.HasRole(role) {
			return model.ID
		}
	}
	panic("audio-tools/bootstrap: no TTS model declares role " + role + " for provider " + providerID)
}

func declaredTTSRuntimeModelID(providerID, role string) string {
	for _, model := range modelcatalog.Default().ByProvider(providerID) {
		if model.Capability == modelcatalog.CapabilityTTS && model.HasRole(role) && model.RuntimeModelID != "" {
			if role == "default" && !model.DefaultEligible() {
				continue
			}
			return model.RuntimeModelID
		}
	}
	panic("audio-tools/bootstrap: no TTS runtime model declares role " + role + " for provider " + providerID)
}

// declaredTTSModelRole selects the model role attached to the runtime actually
// serving this provider. Alternate runtimes can use a different artifact while
// retaining provider and voice identity.
func declaredTTSModelRole(provider providercatalog.Provider, backendID string) string {
	if provider.TTSRuntime == nil {
		return ""
	}
	if shared := provider.TTSRuntime.SharedRuntime; shared != nil && shared.BackendID == backendID {
		return shared.ModelRole
	}
	if provider.TTSRuntime.BackendID == backendID {
		return provider.TTSRuntime.ModelRole
	}
	return ""
}

func declaredTTSModelIDForBackend(provider providercatalog.Provider, backendID string) string {
	role := declaredTTSModelRole(provider, backendID)
	if role == "" {
		panic("audio-tools/bootstrap: provider " + provider.ID + " does not declare TTS backend " + backendID)
	}
	return declaredTTSModelID(provider.ID, role)
}

func declaredTTSRuntimeModelIDForBackend(provider providercatalog.Provider, backendID string) string {
	role := declaredTTSModelRole(provider, backendID)
	if role == "" {
		panic("audio-tools/bootstrap: provider " + provider.ID + " does not declare TTS backend " + backendID)
	}
	return declaredTTSRuntimeModelID(provider.ID, role)
}

func declaredTTSAdapter(providerID string) (string, inttts.AdapterVoiceMap) {
	provider, ok := providercatalog.ByID(providerID)
	if !ok || provider.AdapterID == "" {
		panic("audio-tools/bootstrap: TTS provider has no declared adapter_id: " + providerID)
	}
	mapping, ok := inttts.MappingForAdapter(provider.AdapterID)
	if !ok {
		panic("audio-tools/bootstrap: no voice mapping for declared TTS adapter " + provider.AdapterID)
	}
	return provider.AdapterID, mapping
}

// declaredTTSProviderForResource keeps bootstrap's backend-to-resource choice
// stable while sourcing the actual provider identity from the provider catalog.
func declaredTTSProviderForResource(resourceSlug string) providercatalog.Provider {
	for _, provider := range providercatalog.All() {
		if provider.Tier == providercatalog.TierLocal && provider.ResourceSlug == resourceSlug && slices.Contains(provider.Capabilities, providercatalog.CapabilityTTS) {
			return provider
		}
	}
	panic("audio-tools/bootstrap: no local TTS provider declares resource " + resourceSlug)
}

// declaredQualifiedDefaultTTSProvider selects the active local provider whose
// declared default role resolves to a currently eligible TTS model. Bootstrap
// can therefore use the catalog's qualified default without naming a specific
// resource/provider in runtime wiring.
func declaredQualifiedDefaultTTSProvider(providers []providercatalog.Provider, models *modelcatalog.Catalog) providercatalog.Provider {
	for _, provider := range providers {
		if provider.Status != providercatalog.StatusActive || provider.Tier != providercatalog.TierLocal || provider.TTSRuntime == nil || provider.TTSRuntime.ModelRole != "default" {
			continue
		}
		if !slices.Contains(provider.Capabilities, providercatalog.CapabilityTTS) || models == nil {
			continue
		}
		for _, model := range models.ByProvider(provider.ID) {
			if model.Capability == modelcatalog.CapabilityTTS && model.HasRole("default") && model.DefaultEligible() {
				return provider
			}
		}
	}
	panic("audio-tools/bootstrap: no active local TTS provider declares an eligible default model")
}

// declaredVLLMBatchModels builds opt-in batch routes from the provider/model
// catalogs. An endpoint is configuration only: the backend remains unavailable
// until its read-only health and served-model probes succeed.
func declaredVLLMBatchModels(endpoint string) map[string]sttchain.Provider {
	return declaredBatchModels(modelcatalog.Default(), endpoint)
}

func declaredBatchModels(catalog *modelcatalog.Catalog, endpoint string) map[string]sttchain.Provider {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" || catalog == nil {
		return nil
	}
	routes := make(map[string]sttchain.Provider)
	for _, model := range catalog.All() {
		factory := batchAdapterFactories[model.BatchAdapterID]
		if factory == nil {
			continue
		}
		for _, providerID := range model.ProviderIDs {
			provider, ok := providercatalog.ByID(providerID)
			if !ok || provider.Tier != providercatalog.TierLocal {
				continue
			}
			backend := factory(endpoint, model)
			routes[model.ID] = sttchain.NewBatchProvider(provider.ID, backend, nil)
			break
		}
	}
	return routes
}

// batchAdapterFactories is the protocol registry. Model and provider records
// choose an implemented adapter by declaration; adding another model on an
// existing protocol does not require another model-id branch here.
var batchAdapterFactories = map[string]func(string, modelcatalog.Model) sttchain.BatchBackend{
	"vllm-openai-chat-audio": func(endpoint string, model modelcatalog.Model) sttchain.BatchBackend {
		backend := sttchain.NewRumbleVLLMBackend(endpoint, nil)
		backend.ServedModel = model.RuntimeModelID
		return backend
	},
}

// declaredWhisperBatchModels binds an explicit model id only when it matches
// the model currently loaded by the existing Whisper provider.
func declaredWhisperBatchModels(whisper *sttchain.LocalProvider) map[string]sttchain.Provider {
	if whisper == nil {
		return nil
	}
	routes := make(map[string]sttchain.Provider)
	for _, provider := range providercatalog.All() {
		if provider.Status != providercatalog.StatusActive || provider.Tier != providercatalog.TierLocal {
			continue
		}
		for _, model := range modelcatalog.Default().ByProvider(provider.ID) {
			if model.Runtime != "whisper.cpp" {
				continue
			}
			if model.ID == whisper.Model() {
				routes[model.ID] = whisper
			}
		}
	}
	return routes
}
