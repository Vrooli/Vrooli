package server

import (
	"context"
	"time"

	audioH "audio-tools/handlers/audio"
	diagH "audio-tools/handlers/diagnostics"
	healthH "audio-tools/handlers/health"
	hsH "audio-tools/handlers/health_status"
	measuresH "audio-tools/handlers/measures"
	modelsH "audio-tools/handlers/models"
	plH "audio-tools/handlers/provider_lifecycle"
	sessionH "audio-tools/handlers/session"
	settingsH "audio-tools/handlers/settings"
	sttH "audio-tools/handlers/stt"
	summarizeH "audio-tools/handlers/summarize"
	ttsH "audio-tools/handlers/tts"
	usageH "audio-tools/handlers/usage"

	"audio-tools/internal/ai/chains"
	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/ai/summarizechain"
	"audio-tools/internal/ai/ttschain"
	"audio-tools/internal/audioformat"
	"audio-tools/internal/buildidentity"
	"audio-tools/internal/byokstore"
	"audio-tools/internal/capabilities"
	"audio-tools/internal/controlplane"
	diagcore "audio-tools/internal/diagnostics"
	"audio-tools/internal/duplex"
	"audio-tools/internal/logx"
	measuretypes "audio-tools/internal/measures"
	"audio-tools/internal/modelcatalog"
	"audio-tools/internal/modelpromotion"
	"audio-tools/internal/modulekit"
	intsession "audio-tools/internal/session"
	"audio-tools/internal/store"
	sttpkg "audio-tools/internal/stt"
	sttpipeline "audio-tools/internal/stt/pipeline"
	sttsession "audio-tools/internal/stt/session"
	"audio-tools/internal/sttcapacity"
	"audio-tools/internal/sttengine"
	intsumm "audio-tools/internal/summarize"
	inttts "audio-tools/internal/tts"
	"audio-tools/internal/usagereport"

	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/database"
)

// Env is the small runtime configuration surface required by HTTP modules.
type (
	Env struct {
		OllamaURL           string
		TTSCacheBackend     string
		TestIsolationActive func() bool
	}
	Chains struct {
		STT         *sttchain.Chain
		TTS         *ttschain.Chain
		Summarize   *summarizechain.Chain
		Coordinator *chains.Coordinator
	}
	CompositionStores struct {
		ProviderConfig *store.ProviderConfigStore
		VoiceOverrides *store.VoiceOverrideStore
		BYOK           *byokstore.Store
		STTStream      *store.STTStreamConfigStore
		STTSpeaker     *store.STTSpeakerConfigStore
		Wakeword       *store.WakeWordStore
		Speaker        *store.SpeakerStore
		TTSConfig      *store.TTSConfigStore
		Playback       *store.PlaybackStore
		Usage          *store.UsageStore
		DuplexJournal  *duplex.SQLJournal
	}
	CompositionDeps struct {
		DB                        *database.RoutedDB
		Capabilities              *capabilities.Registry
		CapabilityLivenessLatency *capabilities.LatencyRecorder
		Logger                    logx.Logger
		Env                       Env
		Chains                    Chains
		ProviderRoutingLimits     *chains.Config
		SpeakerResource           *sttpipeline.SpeakerClient
		AudioEngine               *audioformat.Engine
		EngineRegistry            *sttengine.Registry
		Usage                     *usagereport.AsyncRecorder
		Stores                    CompositionStores
		Sessions                  *intsession.Registry
		StreamLedgers             *sttsession.Registry
		Cache                     *inttts.Cache
		SummarizeConfig           *intsumm.SummarizeConfig
		CorpusCounter             measuresH.CorpusCounter
		ExperimentCounter         measuretypes.ExperimentCounter
		QualificationEvidence     modelpromotion.QualificationEvidenceStore
		// Duplex admission dependencies are deliberately opt-in. Production
		// must supply the owner-wired LPBS wallet, source-bound consent verifier,
		// and the eligibility policy that reads the active catalogs.
		DuplexAccountCredits         *duplex.AccountCreditClient
		DuplexConsentVerifier        duplex.ConsentVerifier
		DuplexProviderEligibility    duplex.ProviderEligibility
		DuplexMaxSessionBudgetMicros uint64
	}
)

// composeServer is the transport composition boundary. Bootstrap constructs
// domain services; this function is the only place that adapts them to HTTP
// modules, keeping handler packages out of the dependency wiring file.
func Compose(d CompositionDeps) *Server {
	return compose(d, nil)
}

func compose(d CompositionDeps, labModules []modulekit.Module) *Server {
	db, capsRegistry, logger, chains := d.DB, d.Capabilities, d.Logger, d.Chains
	speakerResource, audioEngine, engineRegistry, usage := d.SpeakerResource, d.AudioEngine, d.EngineRegistry, d.Usage
	stores, sessions, streamLedgers, cache := d.Stores, d.Sessions, d.StreamLedgers, d.Cache
	summarizeConfig := d.SummarizeConfig
	engineResolver := sttengine.NewLiveResolver(controlplane.New().Run, 30*time.Second)
	engineResolver.SetAvailability(func(ctx context.Context, id string) bool {
		return chains.STT != nil && chains.STT.LocalEngineServiceable(ctx, id)
	})
	selector := sttpkg.NewSelectorWithRegistry(chains.STT, audioEngine, engineRegistry)
	selector.EngineResolver = engineResolver
	defaultCredential := func(ctx context.Context, capability string) (string, string, bool) {
		if stores.BYOK == nil {
			return "", "", false
		}
		credentials, err := stores.BYOK.List(ctx)
		if err != nil {
			return "", "", false
		}
		for _, credential := range credentials {
			if credential.Capability != capability {
				continue
			}
			key, ok, err := stores.BYOK.Get(ctx, credential.ProviderID, credential.Capability)
			if err == nil && ok && key != "" {
				return credential.ProviderID, key, true
			}
		}
		return "", "", false
	}
	sttDeps := sttH.Deps{
		// The resolver combines bounded control-plane facts with the chain's
		// provider-owned liveness signal. Its cache is shared by the picker and
		// automatic stream selection.
		Chain:             chains.STT,
		Selector:          selector,
		Registry:          engineRegistry,
		EngineResolver:    engineResolver,
		SpeakerResource:   speakerResource,
		Engine:            audioEngine,
		Logger:            logger,
		Clock:             schedule.System(),
		Usage:             usage,
		StreamConfig:      stores.STTStream,
		SpeakerConfig:     stores.STTSpeaker,
		Wakeword:          stores.Wakeword,
		Speaker:           stores.Speaker,
		Capacity:          sttcapacity.NewCLIReporter(),
		Sessions:          streamLedgers,
		DefaultCredential: defaultCredential,
	}
	diagnostics := diagcore.New(diagcore.Deps{
		STT:       chains.STT,
		TTS:       chains.TTS,
		Summarize: chains.Summarize,
		Transcode: audioformat.Transcoder{},
		Usage:     usage,
		STTConfig: func(ctx context.Context) sttpkg.StreamConfig {
			return sttH.ResolveStreamPipelineConfig(ctx, stores.STTStream)
		},
		Registry:        engineRegistry,
		QualityFixtures: diagcore.DefaultQualityFixtures(),
	})

	productModules := []modulekit.Module{
		healthH.Module(db, capsRegistry, "audio-tools-api", "1.0.0", healthH.BuildIdentity{
			BuildTime:      buildidentity.RuntimeBuildTime(),
			SourceIdentity: buildidentity.SourceIdentity,
		}),
		hsH.Module(hsH.Deps{Registry: capsRegistry, Logger: logger, Clock: schedule.System(), Latency: d.CapabilityLivenessLatency}),
		plH.Module(plH.Deps{Registry: capsRegistry, Controller: capabilities.NewCLIController(), Logger: logger, Clock: schedule.System(), InvalidateEngineCache: engineResolver.Invalidate}),
		// The model registry is static, validated data: one embedded seed, one
		// projection. It is mounted here so the CLI, UI, and operators read the
		// same facts the selector uses instead of hardcoding model ids.
		modelsH.Module(modelsH.Deps{Catalog: modelcatalog.Default(), QualificationEvidence: d.QualificationEvidence}),
		measuresH.Module(measuresH.Deps{BYOK: stores.BYOK, Corpus: d.CorpusCounter, Experiments: d.ExperimentCounter, Speakers: stores.Speaker, Wakewords: stores.Wakeword, DuplexAccounting: stores.DuplexJournal, Logger: logger, Clock: schedule.System()}),
		audioH.Module(logger),
		sessionH.Module(sessions, logger, schedule.System()),
		settingsH.Module(settingsH.Deps{Logger: logger, ProviderConfig: stores.ProviderConfig, BYOK: stores.BYOK, VoiceOverrides: stores.VoiceOverrides, Coordinator: chains.Coordinator, ProviderRoutingLimits: d.ProviderRoutingLimits}),
		sttH.Module(sttDeps),
		summarizeH.Module(chains.Summarize, func() intsumm.SummarizeConfig { return *summarizeConfig }, func(c intsumm.SummarizeConfig) { *summarizeConfig = c }, func(ctx context.Context) ([]intsumm.SummarizeModelInfo, error) {
			return intsumm.ListSummarizeModels(ctx)
		}, logger, schedule.System(), usage),
		ttsH.Module(ttsH.Deps{Chain: chains.TTS, SummarizeChain: chains.Summarize, Engine: audioEngine, Logger: logger, Clock: schedule.System(), Usage: usage, Cache: cache, CacheBackend: d.Env.TTSCacheBackend, ConfigStore: stores.TTSConfig, Playback: stores.Playback, DefaultCredential: defaultCredential}),
		usageH.Module(usageH.Deps{Logger: logger, Clock: schedule.System(), Store: stores.Usage}),
		diagH.Module(diagnostics, logger),
	}
	productModules = append(productModules, labModules...)
	composed := New(Deps{Clock: schedule.System(), Logger: logger}, productModules...)
	// Retain the provider dependency for a future admitted duplex module. This
	// constructs no route and does not resolve the key; the lazy resolver reads
	// the canonical credential authority only when Open is eventually called.
	composed.duplexProvider = &duplex.OpenAILiveProvider{Keys: duplex.NewLazyDefaultAuthorityKeyResolver()}
	// The websocket route is mounted only when the durable spend, consent,
	// catalog eligibility, and journal dependencies are all explicit. A missing
	// owner seam therefore leaves the route absent instead of weakening admission.
	if canComposeGPTLive(d) {
		credits := d.DuplexAccountCredits.WithJournal(stores.DuplexJournal)
		policy := duplex.NewAdmissionPolicy(
			duplex.NewBudget(d.DuplexMaxSessionBudgetMicros),
			d.DuplexMaxSessionBudgetMicros,
			d.DuplexProviderEligibility,
		).WithConsentVerifier(d.DuplexConsentVerifier)
		coordinator, err := duplex.NewCoordinator(composed.duplexProvider, &duplex.AccountCreditAdmission{Policy: policy, Client: credits})
		if err == nil && coordinator != nil {
			duplex.HTTPModule(coordinator)(composed.router)
		}
	}
	return composed
}

// canComposeGPTLive keeps the active catalog as an independent route-mount
// fence. An injected eligibility callback can add deployment-specific
// constraints, but cannot promote a planned provider/model pair.
func canComposeGPTLive(d CompositionDeps) bool {
	return d.Stores.DuplexJournal != nil &&
		d.DuplexAccountCredits != nil &&
		d.DuplexConsentVerifier != nil &&
		d.DuplexProviderEligibility != nil &&
		d.DuplexMaxSessionBudgetMicros > 0 &&
		duplex.CatalogEligibility("openai-gpt-live", "gpt-live-1.1") &&
		d.DuplexProviderEligibility("openai-gpt-live", "gpt-live-1.1")
}
