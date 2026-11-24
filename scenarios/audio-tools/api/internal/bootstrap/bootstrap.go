package bootstrap

import (
	"context"
	"fmt"
	"log"

	"audio-tools/internal/byok"
	"audio-tools/internal/capabilities"
	"audio-tools/internal/duplex"
	"audio-tools/internal/logx"
	"audio-tools/internal/server"
	intsession "audio-tools/internal/session"
	sttsession "audio-tools/internal/stt/session"
	intsumm "audio-tools/internal/summarize"
	inttts "audio-tools/internal/tts"
	"audio-tools/internal/usagereport"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

type accountCreditCredentials struct{}

func (accountCreditCredentials) Resolve(ctx context.Context, identity, field string) (string, error) {
	return credentialauthority.ServiceCredential(identity, field)(ctx)
}

// Deps is the bundle of everything Build constructs. main.go does not
// read it (it only needs the *server.Server and a cleanup func); the
// struct is exported for tests that want to introspect individual
// singletons without re-running Build.
type Deps struct {
	Env       Env
	DB        *database.RoutedDB
	FileRoots *filerouting.RoutedRoots
	DSN       string
	Stores    Stores
	Chains    Chains
	BYOK      byok.Registries
	TTS       *inttts.Service
	Summ      *intsumm.Summarizer
	Cache     *inttts.Cache
	Session   *intsession.Registry
	Usage     *usagereport.AsyncRecorder
	Cleanup   func() error
}

// Build wires the entire audio-tools API and returns the composed
// *server.Server along with a cleanup closure (currently: close DB).
// BuildWithDeps constructs the server and exposes the constructed singletons
// to the process composition root. In particular, main uses DB to mount the
// dev-only routed-pool service before requests reach the scenario handler.
func BuildWithDeps(ctx context.Context) (*server.Server, *Deps, func() error, error) {
	env := Load()
	ttsPlan, env, err := resolveTTSStartupPlan(ctx, env)
	if err != nil {
		return nil, nil, nil, err
	}

	db, _, err := OpenDB(ctx, env)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open db: %w", err)
	}
	stores, err := BuildStores(db, env)
	if err != nil {
		_ = db.Close()
		return nil, nil, nil, fmt.Errorf("build stores: %w", err)
	}
	byokCheckers, err := declaredBYOKCredentialCheckers(stores.BYOK.List, stores.BYOK.Get)
	if err != nil {
		_ = db.Close()
		return nil, nil, nil, fmt.Errorf("build BYOK credential checkers: %w", err)
	}

	logger := logx.Std{L: log.Default()}

	capabilityDeps, err := buildCapabilitySubsystem(env, ttsPlan.url, ttsPlan.backend, ttsPlan.provider, byokCheckers)
	if err != nil {
		_ = db.Close()
		return nil, nil, nil, err
	}
	capsRegistry := capabilityDeps.registry
	livenessCheckers := capabilityDeps.livenessCheckers
	summCfg := capabilityDeps.summarizeConfig
	speech, err := buildSpeechSubsystem(env, ttsPlan, logger, capsRegistry, capabilityDeps.doer, summCfg)
	if err != nil {
		_ = db.Close()
		return nil, nil, nil, err
	}
	audioEngine, engineRegistry := speech.audioEngine, speech.engineRegistry
	speakerClient, whisperSvc := speech.speakerClient, speech.whisper
	ttsCache, ttsSvc, summarizer := speech.cache, speech.tts, speech.summarizer
	byokRegistries, chs := speech.byok, speech.chains
	providerRoutingLimits := providerRoutingUpperBounds(env)
	if err := applyStoredProviderRouting(ctx, stores.ProviderConfig, chs.Coordinator, env); err != nil {
		_ = db.Close()
		return nil, nil, nil, fmt.Errorf("load persisted provider routing: %w", err)
	}

	// BYOK provider health reports credential presence without exposing the
	// encrypted value. Browser providers are always available in the client;
	// they remain the labelled last-resort tier rather than a server probe.
	for id, checker := range byokCheckers {
		livenessCheckers[id] = checker
	}
	livenessCheckers["browser-stt"] = &capabilities.StaticChecker{Available: func() (bool, string) { return true, "browser speech input is available as a last resort" }}
	livenessCheckers["browser-tts"] = &capabilities.StaticChecker{Available: func() (bool, string) { return true, "browser speech output is available as a last resort" }}
	// Time the liveness checks at the registry seam so the health response can
	// report a measured latency instead of a permanent zero. Readiness checks
	// keep their own path; only the liveness tier feeds Get/Refresh/Stream.
	livenessLatency := capabilities.NewLatencyRecorder()
	capsRegistry.SetLivenessCheckers(livenessLatency.WrapAll(livenessCheckers))

	sessionRegistry := intsession.NewRegistry()
	// The streaming ledger owns recoverable captured PCM until processed
	// acknowledgement. Keep it in the scenario-private runtime data directory
	// (not the repository) so a process restart can resume a turn.
	streamNamespace, err := storage.ScenarioNamespace("audio-tools")
	if err != nil {
		_ = db.Close()
		return nil, nil, nil, fmt.Errorf("resolve stt session namespace: %w", err)
	}
	streamResolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		_ = db.Close()
		return nil, nil, nil, fmt.Errorf("create stt session storage resolver: %w", err)
	}
	streamPaths, err := streamResolver.Resolve(storage.Options{ScenarioID: streamNamespace})
	if err != nil {
		_ = db.Close()
		return nil, nil, nil, fmt.Errorf("resolve stt session storage: %w", err)
	}
	fileRoots := filerouting.New(streamPaths)
	streamLedgers, err := sttsession.NewRoutedDiskRegistry(fileRoots, 0)
	if err != nil {
		_ = db.Close()
		return nil, nil, nil, fmt.Errorf("create stt session ledger: %w", err)
	}
	usageRecorder := usagereport.New(stores.Usage, logger)

	experiments := buildExperimentSubsystem(ctx, env, db, logger, whisperSvc, speakerClient, ttsSvc, audioEngine)

	compositionDeps := server.CompositionDeps{
		DB: db, Capabilities: capsRegistry, Logger: logger, CapabilityLivenessLatency: livenessLatency,
		Env: server.Env{
			OllamaURL:       env.OllamaURL,
			TTSCacheBackend: ttsPlan.cacheBackend,
			TestIsolationActive: func() bool {
				return db.HasTestPool() && fileRoots.HasTestRoots()
			},
		},
		Chains:                    server.Chains{STT: chs.STT, TTS: chs.TTS, Summarize: chs.Summarize, Coordinator: chs.Coordinator},
		ProviderRoutingLimits:     &providerRoutingLimits,
		DuplexProviderEligibility: duplex.CatalogEligibility,
		SpeakerResource:           speakerClient, AudioEngine: audioEngine, EngineRegistry: engineRegistry, Usage: usageRecorder,
		Stores:   server.CompositionStores{ProviderConfig: stores.ProviderConfig, VoiceOverrides: stores.VoiceOverrides, BYOK: stores.BYOK, STTStream: stores.STTStream, STTSpeaker: stores.STTSpeaker, Wakeword: stores.Wakeword, Speaker: stores.Speaker, TTSConfig: stores.TTSConfig, Playback: stores.Playback, Usage: stores.Usage, DuplexJournal: stores.DuplexJournal},
		Sessions: sessionRegistry, StreamLedgers: streamLedgers, Cache: ttsCache, SummarizeConfig: &summCfg,
		CorpusCounter:     experiments.corpusCounter,
		ExperimentCounter: experiments.experimentCounter, QualificationEvidence: experiments.qualificationEvidence,
	}
	if env.LPBSBaseURL != "" && env.GPTLiveMaxSessionBudgetMicros > 0 {
		tokens, tokenErr := duplex.NewAccountCreditServiceTokensWithResolver(accountCreditCredentials{}, func(callCtx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(callCtx, "scenario-authenticator")
		}, nil)
		if tokenErr == nil {
			credits, creditErr := duplex.NewAccountCreditClient(env.LPBSBaseURL, tokens, nil)
			consent, consentErr := duplex.NewGatewayConsentVerifierWithResolver(func(callCtx context.Context) (string, error) {
				return discovery.ResolveScenarioURLDefault(callCtx, "ai-gateway")
			}, nil)
			if creditErr == nil && consentErr == nil {
				compositionDeps.DuplexAccountCredits = credits
				compositionDeps.DuplexConsentVerifier = consent
				compositionDeps.DuplexMaxSessionBudgetMicros = env.GPTLiveMaxSessionBudgetMicros
			}
		}
	}
	srv := composeAPI(compositionDeps, experiments.labRuntime)

	cleanup := func() error {
		closeExperimentSubsystem(experiments)
		return db.Close()
	}
	return srv, &Deps{Env: env, DB: db, FileRoots: fileRoots, Stores: stores, Chains: chs, BYOK: byokRegistries, TTS: ttsSvc, Summ: summarizer, Cache: ttsCache, Session: sessionRegistry, Usage: usageRecorder, Cleanup: cleanup}, cleanup, nil
}

// Build is the compatibility constructor used by in-process tests and callers
// that only need the server. Production should use BuildWithDeps to mount the
// routed database's dev-only isolation endpoint.
func Build(ctx context.Context) (*server.Server, func() error, error) {
	srv, _, cleanup, err := BuildWithDeps(ctx)
	return srv, cleanup, err
}
