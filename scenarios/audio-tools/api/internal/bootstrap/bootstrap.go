package bootstrap

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync/atomic"
	"time"

	sttchain "audio-tools/internal/ai/sttchain"
	"audio-tools/internal/audioformat"
	"audio-tools/internal/blobbytes"
	"audio-tools/internal/byok"
	"audio-tools/internal/capabilities"
	intcorpus "audio-tools/internal/corpus"
	intexp "audio-tools/internal/experiment"
	"audio-tools/internal/experiment/evaldeps"
	expreport "audio-tools/internal/experiment/report"
	"audio-tools/internal/httpc"
	"audio-tools/internal/logx"
	"audio-tools/internal/server"
	intsession "audio-tools/internal/session"
	sttpkg "audio-tools/internal/stt"
	sttpipeline "audio-tools/internal/stt/pipeline"
	sttsession "audio-tools/internal/stt/session"
	"audio-tools/internal/sttbackend"
	"audio-tools/internal/sttengine"
	intsumm "audio-tools/internal/summarize"
	inttts "audio-tools/internal/tts"
	"audio-tools/internal/usagereport"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/schedule"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	"google.golang.org/protobuf/encoding/protojson"
)

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
	Voice     *sttpipeline.Service
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
	if resolved, resolveErr := discovery.ResolveScenarioURLDefault(ctx, "landing-page-business-suite"); resolveErr == nil {
		env.LPBSBaseURL = resolved
	}

	db, _, err := OpenDB(ctx, env)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open db: %w", err)
	}

	logger := logx.Std{L: log.Default()}

	// Capability singletons (Local providers). The registry must be wired
	// with concrete checkers — otherwise IsAvailable returns false for
	// every capability and the STT/TTS/Summarize chains report "all
	// providers failed" even when the underlying resources are healthy.
	doer := httpc.DefaultDoer()
	livenessDoer := httpc.LivenessDoer()
	summCfg := intsumm.DefaultSummarizeConfig()
	summCfg.Model = env.SummarizeDefaultModel
	whisperChecker := &capabilities.WhisperChecker{BaseURL: env.WhisperURL, Doer: doer}
	kokoroChecker := &capabilities.KokoroChecker{BaseURL: env.KokoroURL, Doer: doer}
	speakerChecker := &capabilities.ResourceChecker{URL: env.SherpaURL + "/ready", Doer: doer}
	ollamaChecker := &capabilities.OllamaChecker{BaseURL: env.OllamaURL, Doer: doer, ModelFn: func() string {
		return summCfg.Model
	}}
	openRouterChecker := &capabilities.OpenRouterChecker{APIKey: env.OpenRouterAPIKey, BaseURL: env.OpenRouterURL, Doer: doer}
	livenessOpenRouterChecker := &capabilities.OpenRouterChecker{APIKey: env.OpenRouterAPIKey, BaseURL: env.OpenRouterURL, Doer: livenessDoer}
	defs := capabilities.KnownForPlatform("")
	capsCheckers := map[string]capabilities.Checker{
		"whisper-stt":          whisperChecker,
		"kyutai-stt":           &capabilities.ResourceChecker{URL: env.KyutaiURL + "/ready", Doer: doer},
		"kokoro-tts":           kokoroChecker,
		"speaker-verification": speakerChecker,
		"ollama":               ollamaChecker,
		"openrouter":           openRouterChecker,
		"audio-transcode":      &capabilities.TranscodeChecker{},
		"audio-tools": capabilities.AggregateChecker{Checkers: []capabilities.Checker{
			whisperChecker, speakerChecker, kokoroChecker, ollamaChecker,
		}},
	}
	capsRegistry := capabilities.NewRegistry(defs, capsCheckers, 30*time.Second)
	livenessCheckers := map[string]capabilities.Checker{
		"whisper-stt":          &capabilities.ResourceChecker{URL: env.WhisperURL + "/", Doer: livenessDoer},
		"kyutai-stt":           &capabilities.ResourceChecker{URL: env.KyutaiURL + "/ready", Doer: livenessDoer},
		"kokoro-tts":           &capabilities.ResourceChecker{URL: env.KokoroURL + "/v1/audio/voices", Doer: livenessDoer},
		"speaker-verification": &capabilities.ResourceChecker{URL: env.SherpaURL + "/ready", Doer: livenessDoer},
		"ollama":               &capabilities.ResourceChecker{URL: env.OllamaURL + "/api/tags", Doer: livenessDoer},
		"openrouter":           livenessOpenRouterChecker,
		"audio-transcode":      &capabilities.TranscodeChecker{},
		"audio-tools": capabilities.AggregateChecker{Checkers: []capabilities.Checker{
			&capabilities.ResourceChecker{URL: env.WhisperURL + "/", Doer: livenessDoer},
			&capabilities.ResourceChecker{URL: env.SherpaURL + "/ready", Doer: livenessDoer},
			&capabilities.ResourceChecker{URL: env.KokoroURL + "/v1/audio/voices", Doer: livenessDoer},
			&capabilities.ResourceChecker{URL: env.OllamaURL + "/api/tags", Doer: livenessDoer},
		}},
	}
	capsRegistry.SetLivenessCheckers(livenessCheckers)
	skipVerifyCount := &atomic.Int64{}
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
	voiceSvc := sttpipeline.NewService(
		sttpipeline.Config{},
		"", nil, "",
		sttpipeline.SpeakerConfig{}, "",
		speakerClient,
		capsRegistry,
		skipVerifyCount,
		env.WhisperURL+"/asr?output=json",
		doer, audioEngine,
	)
	// On-demand STT backend recovery (plan L1): when a transcribe hits a down
	// whisper backend, ensure it is running and retry once. Single-flight +
	// cooldown + allowlist live in the Ensurer. STT_AUTO_ENSURE (default on)
	// lets operators disable the auto-start.
	voiceSvc.SetBackendEnsurer(sttbackend.NewCLIEnsurer(), "whisper")
	voiceSvc.SetAutoEnsure(sttpipeline.AutoEnsureEnabledFromEnv())

	ttsCache := inttts.NewCache(64 * 1024 * 1024)
	kokoroSynth := &inttts.KokoroSynthesizer{BaseURL: env.KokoroURL, Doer: httpc.DefaultDoer()}
	ttsCfg := inttts.DefaultConfig()
	ttsSvc := inttts.NewService(inttts.Deps{
		Logger:        logger,
		GetConfig:     func() inttts.Config { return ttsCfg },
		SetConfig:     func(c inttts.Config) { ttsCfg = c },
		PersistConfig: func(inttts.Config) error { return nil },
		// Ask the capability registry, which owns the Kokoro liveness probe.
		// This must observe the provider rather than trust a display cache:
		// an earlier hardcoded success let the request path use a stopped
		// resource, while a full catalogue sweep charged synthesis for
		// unrelated readiness probes.
		KokoroCapability: func(ctx context.Context) (string, string) {
			if capsRegistry.IsProviderLive(ctx, "kokoro-tts") {
				return "available", "Kokoro (Local)"
			}
			return "unavailable", "Kokoro (Local)"
		},
		SynthesizeAudio: func(ctx context.Context, in inttts.SynthesizeInput) (io.ReadCloser, string, error) {
			return kokoroSynth.Synthesize(ctx, inttts.SynthesizeRequest{
				Input:          in.Input,
				Voice:          in.Voice,
				ResponseFormat: in.ResponseFormat,
				Speed:          in.Speed,
			})
		},
		GetCache: func(key inttts.CacheKey) (inttts.SynthesizeResult, bool) {
			out, ok := ttsCache.Get(key)
			if !ok {
				return inttts.SynthesizeResult{}, false
			}
			return inttts.SynthesizeResult{Audio: out.Audio, ContentType: out.ContentType}, true
		},
		PutCache: func(key inttts.CacheKey, audio []byte, ct string) {
			ttsCache.Put(key, audio, ct)
		},
	})

	summarizer := intsumm.NewSummarizer(env.OllamaURL, httpc.DefaultDoer())

	byokRegistries := byok.NewRegistries()
	chs := BuildChains(env, voiceSvc, ttsSvc, summarizer, byokRegistries, logger)

	stores, err := BuildStores(db, env)
	if err != nil {
		_ = db.Close()
		return nil, nil, nil, fmt.Errorf("build stores: %w", err)
	}
	// BYOK provider health reports credential presence without exposing the
	// encrypted value. Browser providers are always available in the client;
	// they remain the labelled last-resort tier rather than a server probe.
	for _, entry := range []struct{ id, provider, capability string }{
		{"openai-whisper", "openai-whisper", "stt"},
		{"deepgram", "deepgram", "stt"},
		{"openai-tts", "openai-tts", "tts"},
		{"elevenlabs", "elevenlabs", "tts"},
	} {
		livenessCheckers[entry.id] = &capabilities.BYOKCredentialChecker{
			ProviderID: entry.provider,
			Capability: entry.capability,
			List:       stores.BYOK.List,
			Get:        stores.BYOK.Get,
			Probe:      capabilities.ProbeBYOKEndpoint(entry.provider),
		}
	}
	livenessCheckers["browser-stt"] = &capabilities.StaticChecker{Available: func() (bool, string) { return true, "browser speech input is available as a last resort" }}
	livenessCheckers["browser-tts"] = &capabilities.StaticChecker{Available: func() (bool, string) { return true, "browser speech output is available as a last resort" }}
	capsRegistry.SetLivenessCheckers(livenessCheckers)

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

	// Corpus + eval/experiment harnesses. Audio bytes live in the blob store under the
	// git-ignored runtime data dir; the namespace is variant-aware so a
	// shadow instance's corpus never collides with live's. Metadata lives in
	// the corpus SQLite domain. A blob-store init failure disables the
	// corpus/eval/experiment surfaces (their handlers return FailedPrecondition) rather
	// than failing the whole boot.
	var corpusSvc *intcorpus.Service
	var experimentSvc *intexp.Service
	var experimentMgr *intexp.Manager
	if ns, nerr := storage.ScenarioNamespace("audio-tools"); nerr != nil {
		logger.Printf("corpus blob store disabled: resolve namespace: %v", nerr)
	} else if blobs, berr := blobbytes.NewFilesystem(ns, "corpus-blobs", "corpus"); berr != nil {
		logger.Printf("corpus blob store disabled: %v", berr)
	} else {
		corpusSvc = intcorpus.NewService(intcorpus.NewSQLiteRepository(db, schedule.System()), blobs, schedule.System())
		if expBlobs, eerr := blobbytes.NewFilesystem(ns, "experiment-blobs", "experiment"); eerr != nil {
			logger.Printf("experiment blob store disabled: %v", eerr)
		} else {
			experimentSvc = intexp.NewService(intexp.NewSQLiteRepository(db, schedule.System()), expBlobs)
		}
	}
	// Each experiment replay receives a fresh provider. Provider-neutral cells
	// select this factory by engine id instead of relabelling a Whisper run.
	evalProviderForEngine := func(engineID string) sttchain.Provider {
		switch engineID {
		case "", "whisper-local", "eval-whisper-local":
			return sttchain.NewLocalProvider(voiceSvc)
		case "kyutai":
			return sttchain.NewKyutaiProvider(env.KyutaiURL)
		case "sherpa-streaming":
			return sttchain.NewSherpaProvider(env.SherpaURL)
		default:
			return nil
		}
	}
	evalDeps := evaldeps.New(logger, corpusSvc, evalProviderForEngine, sttpkg.Defaults(), speakerClient)
	if experimentSvc != nil {
		experimentMgr = intexp.NewManager(intexp.Config{
			Service: experimentSvc,
			Clock:   schedule.System(),
			Logger:  logger,
			Runner: func(runCtx context.Context, exp intexp.Experiment, emit func(int, string)) (intexp.RunResult, error) {
				recipe := &experimentv1.ExperimentRecipe{}
				if len(exp.RecipeJSON) > 0 {
					if err := protojson.Unmarshal(exp.RecipeJSON, recipe); err != nil {
						return intexp.RunResult{}, fmt.Errorf("parse experiment recipe: %w", err)
					}
				}
				emit(5, "loading corpus")
				report, realized, err := expreport.RunExperimentReport(runCtx, evalDeps, corpusSvc, ttsSvc, audioEngine, recipe, emit)
				if err != nil {
					return intexp.RunResult{}, err
				}
				recipeJSON, err := protojson.Marshal(recipe)
				if err != nil {
					return intexp.RunResult{}, fmt.Errorf("marshal realized recipe: %w", err)
				}
				reportProto := server.ReportToProto(report)
				reportJSON, err := protojson.Marshal(reportProto)
				if err != nil {
					return intexp.RunResult{}, fmt.Errorf("marshal report: %w", err)
				}
				runs, err := expreport.ExperimentRunsForReport(report, reportProto, realized)
				if err != nil {
					return intexp.RunResult{}, err
				}
				emit(95, "storing report")
				return intexp.RunResult{Report: reportJSON, ReportMIME: "application/json", Runs: runs, RecipeJSON: recipeJSON}, nil
			},
		})
		if err := experimentMgr.Start(ctx); err != nil {
			logger.Printf("experiment manager disabled: %v", err)
			experimentMgr = nil
			experimentSvc = nil
		}
	}

	srv := server.Compose(server.CompositionDeps{
		DB: db, Capabilities: capsRegistry, Logger: logger,
		Env: server.Env{
			OllamaURL: env.OllamaURL,
			TestIsolationActive: func() bool {
				return db.HasTestPool() && fileRoots.HasTestRoots()
			},
		},
		Chains: server.Chains{STT: chs.STT, TTS: chs.TTS, Summarize: chs.Summarize, Coordinator: chs.Coordinator},
		Voice:  voiceSvc, SpeakerResource: speakerClient, AudioEngine: audioEngine, EngineRegistry: engineRegistry, Usage: usageRecorder,
		Stores:   server.CompositionStores{ProviderConfig: stores.ProviderConfig, VoiceOverrides: stores.VoiceOverrides, BYOK: stores.BYOK, STTStream: stores.STTStream, STTSpeaker: stores.STTSpeaker, Wakeword: stores.Wakeword, Speaker: stores.Speaker, TTSConfig: stores.TTSConfig, Playback: stores.Playback, Usage: stores.Usage},
		Sessions: sessionRegistry, StreamLedgers: streamLedgers, TTS: ttsSvc, Cache: ttsCache, SummarizeConfig: &summCfg, Corpus: corpusSvc, ExperimentManager: experimentMgr, Experiments: experimentSvc,
	})

	cleanup := func() error {
		if experimentMgr != nil {
			experimentMgr.Close()
		}
		return db.Close()
	}
	return srv, &Deps{Env: env, DB: db, FileRoots: fileRoots, Stores: stores, Chains: chs, BYOK: byokRegistries, Voice: voiceSvc, TTS: ttsSvc, Summ: summarizer, Cache: ttsCache, Session: sessionRegistry, Usage: usageRecorder, Cleanup: cleanup}, cleanup, nil
}

// Build is the compatibility constructor used by in-process tests and callers
// that only need the server. Production should use BuildWithDeps to mount the
// routed database's dev-only isolation endpoint.
func Build(ctx context.Context) (*server.Server, func() error, error) {
	srv, _, cleanup, err := BuildWithDeps(ctx)
	return srv, cleanup, err
}
