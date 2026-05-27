package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"sync/atomic"
	"time"

	audioH "audio-tools/handlers/audio"
	diagH "audio-tools/handlers/diagnostics"
	healthH "audio-tools/handlers/health"
	hsH "audio-tools/handlers/health_status"
	plH "audio-tools/handlers/provider_lifecycle"
	sessionH "audio-tools/handlers/session"
	settingsH "audio-tools/handlers/settings"
	sttH "audio-tools/handlers/stt"
	summarizeH "audio-tools/handlers/summarize"
	ttsH "audio-tools/handlers/tts"
	usageH "audio-tools/handlers/usage"

	intaudio "audio-tools/internal/audio"
	"audio-tools/internal/audioformat"
	"audio-tools/internal/byok"
	"audio-tools/internal/capabilities"
	"audio-tools/internal/clock"
	diagcore "audio-tools/internal/diagnostics"
	"audio-tools/internal/httpc"
	"audio-tools/internal/logx"
	"audio-tools/internal/server"
	intsession "audio-tools/internal/session"
	sttpkg "audio-tools/internal/stt"
	sttpipeline "audio-tools/internal/stt/pipeline"
	"audio-tools/internal/sttengine"
	intsumm "audio-tools/internal/summarize"
	inttts "audio-tools/internal/tts"
	"audio-tools/internal/usagereport"
)

// Deps is the bundle of everything Build constructs. main.go does not
// read it (it only needs the *server.Server and a cleanup func); the
// struct is exported for tests that want to introspect individual
// singletons without re-running Build.
type Deps struct {
	Env     Env
	DB      *sql.DB
	DSN     string
	Stores  Stores
	Chains  Chains
	BYOK    byok.Registries
	Voice   *sttpipeline.Service
	TTS     *inttts.Service
	Summ    *intsumm.Summarizer
	Cache   *inttts.Cache
	Session *intsession.Registry
	Usage   *usagereport.AsyncRecorder
	Cleanup func() error
}

// Build wires the entire audio-tools API and returns the composed
// *server.Server along with a cleanup closure (currently: close DB).
func Build(ctx context.Context) (*server.Server, func() error, error) {
	env := Load()

	db, dsn, err := OpenDB(ctx, env)
	if err != nil {
		return nil, nil, fmt.Errorf("open db: %w", err)
	}

	logger := logx.Std{L: log.Default()}

	// Capability singletons (Local providers). The registry must be wired
	// with concrete checkers — otherwise IsAvailable returns false for
	// every capability and the STT/TTS/Summarize chains report "all
	// providers failed" even when the underlying resources are healthy.
	doer := httpc.DefaultDoer()
	summCfg := intsumm.DefaultSummarizeConfig()
	summCfg.Model = env.SummarizeDefaultModel
	capsCheckers := map[string]capabilities.Checker{
		"whisper-stt": &capabilities.WhisperChecker{BaseURL: env.WhisperURL, Doer: doer},
		"kokoro-tts":  &capabilities.KokoroChecker{BaseURL: env.KokoroURL, Doer: doer},
		"ollama": &capabilities.OllamaChecker{BaseURL: env.OllamaURL, Doer: doer, ModelFn: func() string {
			return summCfg.Model
		}},
	}
	capsRegistry := capabilities.NewRegistry(capabilities.Known, capsCheckers, 30*time.Second)
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
	// Speaker-verification resource client (resources/speaker-verification).
	// Wired here so streaming speaker isolation + enrollment reach the real
	// ECAPA embedding service; nil base URL would make every verify fall back.
	speakerClient := &sttpipeline.SpeakerClient{BaseURL: env.SpeakerURL, Doer: doer}
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

	ttsCache := inttts.NewCache(64 * 1024 * 1024)
	kokoroSynth := &inttts.KokoroSynthesizer{BaseURL: env.KokoroURL, Doer: httpc.DefaultDoer()}
	ttsCfg := inttts.DefaultConfig()
	ttsSvc := inttts.NewService(inttts.Deps{
		Logger:        logger,
		GetConfig:     func() inttts.Config { return ttsCfg },
		SetConfig:     func(c inttts.Config) { ttsCfg = c },
		PersistConfig: func(inttts.Config) error { return nil },
		KokoroCapability: func(ctx context.Context) (string, string) {
			return "available", "Kokoro (Local)"
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
		return nil, nil, fmt.Errorf("build stores: %w", err)
	}

	sessionRegistry := intsession.NewRegistry()
	usageRecorder := usagereport.New(stores.Usage, logger)

	diagOrch := diagcore.New(diagcore.Deps{
		STT:       chs.STT,
		TTS:       chs.TTS,
		Summarize: chs.Summarize,
		Transcode: ffmpegTranscoder{},
		Usage:     usageRecorder,
	})

	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: logger},
		healthH.Module(db, capsRegistry, "audio-tools-api", "1.0.0"),
		hsH.Module(hsH.Deps{Registry: capsRegistry, Logger: logger, Clock: clock.System{}}),
		plH.Module(plH.Deps{
			Registry:   capsRegistry,
			Controller: capabilities.NewCLIController(),
			Logger:     logger,
			Clock:      clock.System{},
		}),
		audioH.Module(logger),
		sessionH.Module(sessionRegistry, logger, clock.System{}),
		settingsH.Module(settingsH.Deps{
			Logger:         logger,
			ProviderConfig: stores.ProviderConfig,
			BYOK:           stores.BYOK,
			VoiceOverrides: stores.VoiceOverrides,
			Coordinator:    chs.Coordinator,
		}),
		sttH.Module(sttH.Deps{
			Chain:           chs.STT,
			Selector:        sttpkg.NewSelectorWithRegistry(chs.STT, audioEngine, engineRegistry),
			Registry:        engineRegistry,
			Voice:           voiceSvc,
			SpeakerResource: speakerClient,
			Engine:          audioEngine,
			Logger:          logger,
			Clock:           clock.System{},
			Usage:           usageRecorder,
			StreamConfig:    stores.STTStream,
			Wakeword:        stores.Wakeword,
			Speaker:         stores.Speaker,
		}),
		summarizeH.Module(
			chs.Summarize,
			func() intsumm.SummarizeConfig { return summCfg },
			func(c intsumm.SummarizeConfig) { summCfg = c },
			func(ctx context.Context) ([]intsumm.SummarizeModelInfo, error) {
				return intsumm.ListSummarizeModels(ctx, env.OllamaURL, doer)
			},
			logger,
			clock.System{},
			usageRecorder,
		),
		ttsH.Module(ttsH.Deps{
			Chain:          chs.TTS,
			SummarizeChain: chs.Summarize,
			TTSService:     ttsSvc,
			Engine:         audioEngine,
			Logger:         logger,
			Clock:          clock.System{},
			Usage:          usageRecorder,
			Cache:          ttsCache,
			ConfigStore:    stores.TTSConfig,
			Playback:       stores.Playback,
		}),
		usageH.Module(usageH.Deps{Logger: logger, Clock: clock.System{}, Store: stores.Usage}),
		diagH.Module(diagOrch, logger),
	)

	_ = dsn // retained for diagnostics if Deps is exposed; not used here.
	cleanup := func() error { return db.Close() }
	return srv, cleanup, nil
}

// ffmpegTranscoder adapts internal/audio.TranscodeOpts to the
// diagcore.Transcoder seam without dragging diagnostics into the audio
// package's import graph.
type ffmpegTranscoder struct{}

func (ffmpegTranscoder) Transcode(ctx context.Context, audio []byte, outputFormat string) ([]byte, error) {
	return intaudio.TranscodeOpts(ctx, audio, outputFormat, 0, 0, 0)
}
