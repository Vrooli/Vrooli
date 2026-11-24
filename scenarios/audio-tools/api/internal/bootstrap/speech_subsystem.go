package bootstrap

import (
	"fmt"
	"path/filepath"
	"time"

	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/audioformat"
	"audio-tools/internal/byok"
	"audio-tools/internal/capabilities"
	"audio-tools/internal/httpc"
	"audio-tools/internal/logx"
	sttpipeline "audio-tools/internal/stt/pipeline"
	"audio-tools/internal/sttbackend"
	"audio-tools/internal/sttengine"
	intsumm "audio-tools/internal/summarize"
	inttts "audio-tools/internal/tts"
	"github.com/vrooli/api-core/storage"
)

type speechSubsystem struct {
	audioEngine    *audioformat.Engine
	engineRegistry *sttengine.Registry
	speakerClient  *sttpipeline.SpeakerClient
	whisper        *sttpipeline.WhisperTranscriber
	cache          *inttts.Cache
	tts            *inttts.Service
	byok           byok.Registries
	chains         Chains
	summarizer     *intsumm.Summarizer
}

func buildSpeechSubsystem(env Env, plan ttsStartupPlan, logger logx.Logger, capsRegistry *capabilities.Registry, doer httpc.Doer, summarizeConfig intsumm.SummarizeConfig) (speechSubsystem, error) {
	// audioEngine is the single audio-format substrate: one instance shared
	// across the STT pipeline (Whisper-container handling), the streaming
	// Segmenter (live PCM decode), the selector capability gate, and TTS
	// egress. It holds no per-session state.
	audioEngine := audioformat.New()
	// engineRegistry is the STT engine-capability manifest (single source of
	// truth). The selector derives the Local tier's eligible strategies from
	// it; the Segmenter derives the egress-gate stage set from it.
	engineRegistry := sttengine.Default()
	// Native sherpa-onnx speaker runtime client. The logical capability name
	// remains speaker-verification for API compatibility.
	// Wired here so streaming speaker isolation + enrollment reach the native
	// sherpa embedding service; nil base URL would make every verify fall back.
	speakerClient := &sttpipeline.SpeakerClient{BaseURL: env.SherpaURL, Doer: doer}
	whisperSvc := sttpipeline.NewWhisperTranscriber(
		capsRegistry,
		env.WhisperURL+"/asr?output=json",
		doer, audioEngine,
	)
	// On-demand STT backend recovery (plan L1): when a transcribe hits a down
	// whisper backend, ensure it is running and retry once. Single-flight +
	// cooldown + allowlist live in the Ensurer. STT_AUTO_ENSURE (default on)
	// lets operators disable the auto-start.
	whisperSvc.SetBackendEnsurer(sttbackend.NewCLIEnsurer(sttBackendResources(engineRegistry)), "whisper")
	whisperSvc.SetAutoEnsure(sttpipeline.AutoEnsureEnabledFromEnv())

	cacheNamespace, err := storage.ScenarioNamespace("audio-tools")
	if err != nil {
		return speechSubsystem{}, fmt.Errorf("resolve TTS cache namespace: %w", err)
	}
	cacheResolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		return speechSubsystem{}, fmt.Errorf("create TTS cache storage resolver: %w", err)
	}
	cachePaths, err := cacheResolver.Resolve(storage.Options{ScenarioID: cacheNamespace})
	if err != nil {
		return speechSubsystem{}, fmt.Errorf("resolve TTS cache storage: %w", err)
	}
	ttsCache, err := inttts.NewPersistentCache(64*1024*1024, filepath.Join(cachePaths.DataDir, "tts-cache"), 30*24*time.Hour)
	if err != nil {
		return speechSubsystem{}, fmt.Errorf("create TTS cache: %w", err)
	}
	ttsServices := make(map[string]*inttts.Service, len(plan.providers))
	var ttsSvc *inttts.Service
	if env.TTSBackend == "auto" {
		for _, provider := range plan.providers {
			baseURL, err := declaredTTSURL(provider, plan.urls)
			if err != nil {
				return speechSubsystem{}, err
			}
			model := declaredTTSRuntimeModelID(provider.ID, provider.TTSRuntime.ModelRole)
			ttsServices[provider.ID] = newLocalTTSService(baseURL, model, provider.TTSRuntime.SpeedMode, provider.TTSRuntime.CacheBackend, provider.ID, doer, logger, capsRegistry, ttsCache)
			if provider.TTSRuntime.BackendID == plan.backend {
				ttsSvc = ttsServices[provider.ID]
			}
		}
		if ttsSvc == nil {
			return speechSubsystem{}, fmt.Errorf("audio-tools/bootstrap: selected TTS backend %q has no declared service", plan.backend)
		}
	} else {
		ttsSvc = newLocalTTSService(plan.url, plan.model, plan.speedMode, plan.cacheBackend, plan.providerID, doer, logger, capsRegistry, ttsCache)
		serviceProviderID := plan.provider.ID
		ttsServices[serviceProviderID] = ttsSvc
	}

	// The production Audio Tools local summarizer submits its task through AI
	// Gateway, preserving LOCAL_ONLY and the transcript/text privacy basis. The
	// summary evaluation package keeps its explicit resource-ollama runner as an
	// experiment owner; it is not the serving caller.
	summarizer := intsumm.NewSummarizerWithTaskRunner(summarizechain.NewAIGatewayRunner())

	byokRegistries := byok.NewRegistries()
	chs := BuildChains(env, whisperSvc, ttsServices, summarizer, byokRegistries, logger)
	return speechSubsystem{
		audioEngine: audioEngine, engineRegistry: engineRegistry, speakerClient: speakerClient,
		whisper: whisperSvc, cache: ttsCache, tts: ttsSvc, byok: byokRegistries,
		chains: chs, summarizer: summarizer,
	}, nil
}

func sttBackendResources(registry *sttengine.Registry) []string {
	if registry == nil {
		return nil
	}
	resources := make([]string, 0, len(registry.Engines()))
	for _, engine := range registry.Engines() {
		if engine.Kind == sttengine.KindLocalResource && engine.Resource != "" {
			resources = append(resources, engine.Resource)
		}
	}
	return resources
}
