package server

import (
	"context"

	audioH "audio-tools/handlers/audio"
	corpusH "audio-tools/handlers/corpus"
	diagH "audio-tools/handlers/diagnostics"
	evalH "audio-tools/handlers/eval"
	experimentH "audio-tools/handlers/experiment"
	healthH "audio-tools/handlers/health"
	hsH "audio-tools/handlers/health_status"
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
	"audio-tools/internal/byokstore"
	"audio-tools/internal/capabilities"
	"audio-tools/internal/clock"
	intcorpus "audio-tools/internal/corpus"
	diagcore "audio-tools/internal/diagnostics"
	inteval "audio-tools/internal/eval"
	intexp "audio-tools/internal/experiment"
	expreport "audio-tools/internal/experiment/report"
	"audio-tools/internal/httpc"
	"audio-tools/internal/logx"
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

	"github.com/vrooli/api-core/database"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
)

// Env is the small runtime configuration surface required by HTTP modules.
type (
	Env struct {
		OllamaURL            string
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
	}
	CompositionDeps struct {
		DB                *database.RoutedDB
		Capabilities      *capabilities.Registry
		Logger            logx.Logger
		Env               Env
		Chains            Chains
		Voice             *sttpipeline.Service
		SpeakerResource   *sttpipeline.SpeakerClient
		AudioEngine       *audioformat.Engine
		EngineRegistry    *sttengine.Registry
		Usage             *usagereport.AsyncRecorder
		Stores            CompositionStores
		Sessions          *intsession.Registry
		StreamLedgers     *sttsession.Registry
		TTS               *inttts.Service
		Cache             *inttts.Cache
		SummarizeConfig   *intsumm.SummarizeConfig
		Corpus            *intcorpus.Service
		ExperimentManager *intexp.Manager
		Experiments       *intexp.Service
	}
)

// composeServer is the transport composition boundary. Bootstrap constructs
// domain services; this function is the only place that adapts them to HTTP
// modules, keeping handler packages out of the dependency wiring file.
func Compose(d CompositionDeps) *Server {
	db, capsRegistry, logger, env, chains := d.DB, d.Capabilities, d.Logger, d.Env, d.Chains
	voice, speakerResource, audioEngine, engineRegistry, usage := d.Voice, d.SpeakerResource, d.AudioEngine, d.EngineRegistry, d.Usage
	stores, sessions, streamLedgers, tts, cache := d.Stores, d.Sessions, d.StreamLedgers, d.TTS, d.Cache
	summarizeConfig, corpus, experimentManager, experiments := d.SummarizeConfig, d.Corpus, d.ExperimentManager, d.Experiments
	sttDeps := sttH.Deps{
		Chain:                  chains.STT,
		Selector:               sttpkg.NewSelectorWithRegistry(chains.STT, audioEngine, engineRegistry),
		Registry:               engineRegistry,
		Voice:                  voice,
		SpeakerResource:        speakerResource,
		Engine:                 audioEngine,
		Logger:                 logger,
		Clock:                  clock.System{},
		Usage:                  usage,
		StreamConfig:           stores.STTStream,
		SpeakerConfig:          stores.STTSpeaker,
		Wakeword:               stores.Wakeword,
		Speaker:                stores.Speaker,
		Capacity:               sttcapacity.NewCLIReporter(),
		Sessions:               streamLedgers,
		TestIsolationActive:    env.TestIsolationActive,
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

	return New(
		Deps{Clock: clock.System{}, Logger: logger},
		healthH.Module(db, capsRegistry, "audio-tools-api", "1.0.0"),
		hsH.Module(hsH.Deps{Registry: capsRegistry, Logger: logger, Clock: clock.System{}}),
		plH.Module(plH.Deps{Registry: capsRegistry, Controller: capabilities.NewCLIController(), Logger: logger, Clock: clock.System{}}),
		audioH.Module(logger),
		sessionH.Module(sessions, logger, clock.System{}),
		settingsH.Module(settingsH.Deps{Logger: logger, ProviderConfig: stores.ProviderConfig, BYOK: stores.BYOK, VoiceOverrides: stores.VoiceOverrides, Coordinator: chains.Coordinator}),
		sttH.Module(sttDeps),
		summarizeH.Module(chains.Summarize, func() intsumm.SummarizeConfig { return *summarizeConfig }, func(c intsumm.SummarizeConfig) { *summarizeConfig = c }, func(ctx context.Context) ([]intsumm.SummarizeModelInfo, error) {
			return intsumm.ListSummarizeModels(ctx, env.OllamaURL, httpc.DefaultDoer())
		}, logger, clock.System{}, usage),
		ttsH.Module(ttsH.Deps{Chain: chains.TTS, SummarizeChain: chains.Summarize, TTSService: tts, Engine: audioEngine, Logger: logger, Clock: clock.System{}, Usage: usage, Cache: cache, ConfigStore: stores.TTSConfig, Playback: stores.Playback}),
		usageH.Module(usageH.Deps{Logger: logger, Clock: clock.System{}, Store: stores.Usage}),
		corpusH.Module(corpusH.Deps{Logger: logger, Clock: clock.System{}, Service: corpus}),
		experimentH.Module(experimentH.Deps{Logger: logger, Manager: experimentManager, Service: experiments, EstimateClipSeconds: expreport.EstimateClipSeconds(corpus)}),
		diagH.Module(diagnostics, logger),
	)
}

// reportToProto keeps the legacy eval report mapper at the transport boundary
// while the experiment runner remains an internal domain concern.
func ReportToProto(report inteval.EvalReport) *evalv1.EvalReport { return evalH.ReportToProto(report) }
