package bootstrap

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

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

	sttchain "audio-tools/internal/ai/sttchain"
	intaudio "audio-tools/internal/audio"
	"audio-tools/internal/audioformat"
	"audio-tools/internal/byok"
	"audio-tools/internal/capabilities"
	"audio-tools/internal/clock"
	intcorpus "audio-tools/internal/corpus"
	diagcore "audio-tools/internal/diagnostics"
	inteval "audio-tools/internal/eval"
	intexp "audio-tools/internal/experiment"
	exprecipe "audio-tools/internal/experiment/recipe"
	"audio-tools/internal/httpc"
	"audio-tools/internal/logx"
	"audio-tools/internal/protomap"
	"audio-tools/internal/server"
	intsession "audio-tools/internal/session"
	sttpkg "audio-tools/internal/stt"
	sttpipeline "audio-tools/internal/stt/pipeline"
	sttsession "audio-tools/internal/stt/session"
	"audio-tools/internal/sttbackend"
	"audio-tools/internal/sttcapacity"
	"audio-tools/internal/sttengine"
	intsumm "audio-tools/internal/summarize"
	inttts "audio-tools/internal/tts"
	"audio-tools/internal/usagereport"

	"github.com/vrooli/api-core/storage"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	"google.golang.org/protobuf/encoding/protojson"
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

	db, _, err := OpenDB(ctx, env)
	if err != nil {
		return nil, nil, fmt.Errorf("open db: %w", err)
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
	capsCheckers := map[string]capabilities.Checker{
		"whisper-stt": &capabilities.WhisperChecker{BaseURL: env.WhisperURL, Doer: doer},
		"kokoro-tts":  &capabilities.KokoroChecker{BaseURL: env.KokoroURL, Doer: doer},
		"ollama": &capabilities.OllamaChecker{BaseURL: env.OllamaURL, Doer: doer, ModelFn: func() string {
			return summCfg.Model
		}},
	}
	capsRegistry := capabilities.NewRegistry(capabilities.Known, capsCheckers, 30*time.Second)
	capsRegistry.SetLivenessCheckers(map[string]capabilities.Checker{
		"whisper-stt": &capabilities.ResourceChecker{URL: env.WhisperURL + "/", Doer: livenessDoer},
		"kokoro-tts":  &capabilities.ResourceChecker{URL: env.KokoroURL + "/v1/audio/voices", Doer: livenessDoer},
		"ollama":      &capabilities.ResourceChecker{URL: env.OllamaURL + "/api/tags", Doer: livenessDoer},
	})
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
	// The streaming ledger owns recoverable captured PCM until processed
	// acknowledgement. Keep it in the scenario-private runtime data directory
	// (not the repository) so a process restart can resume a turn.
	streamNamespace, err := storage.ScenarioNamespace("audio-tools")
	if err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("resolve stt session namespace: %w", err)
	}
	streamResolver, err := storage.NewResolver(storage.ResolverConfig{AppID: "vrooli", Profile: storage.ProfileAuto})
	if err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("create stt session storage resolver: %w", err)
	}
	streamPaths, err := streamResolver.Resolve(storage.Options{ScenarioID: streamNamespace})
	if err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("resolve stt session storage: %w", err)
	}
	streamLedgers, err := sttsession.NewDiskRegistry(filepath.Join(streamPaths.DataDir, "stt-session-spool"), 0)
	if err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("create stt session ledger: %w", err)
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
	} else if blobs, berr := intcorpus.NewFilesystemBlobBytes(ns); berr != nil {
		logger.Printf("corpus blob store disabled: %v", berr)
	} else {
		corpusSvc = intcorpus.NewService(intcorpus.NewSQLiteRepository(db, clock.System{}), blobs, clock.System{})
		if expBlobs, eerr := intexp.NewFilesystemBlobBytes(ns); eerr != nil {
			logger.Printf("experiment blob store disabled: %v", eerr)
		} else {
			experimentSvc = intexp.NewService(intexp.NewSQLiteRepository(db, clock.System{}), expBlobs)
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
		default:
			return nil
		}
	}
	evalDeps := newExperimentEvalDeps(logger, corpusSvc, evalProviderForEngine, sttpkg.Defaults(), speakerClient)
	if experimentSvc != nil {
		experimentMgr = intexp.NewManager(intexp.Config{
			Service: experimentSvc,
			Clock:   clock.System{},
			Logger:  logger,
			Runner: func(runCtx context.Context, exp intexp.Experiment, emit func(int, string)) (intexp.RunResult, error) {
				recipe := &experimentv1.ExperimentRecipe{}
				if len(exp.RecipeJSON) > 0 {
					if err := protojson.Unmarshal(exp.RecipeJSON, recipe); err != nil {
						return intexp.RunResult{}, fmt.Errorf("parse experiment recipe: %w", err)
					}
				}
				emit(5, "loading corpus")
				report, realized, err := runExperimentReport(runCtx, evalDeps, corpusSvc, ttsSvc, audioEngine, recipe, emit)
				if err != nil {
					return intexp.RunResult{}, err
				}
				recipeJSON, err := protojson.Marshal(recipe)
				if err != nil {
					return intexp.RunResult{}, fmt.Errorf("marshal realized recipe: %w", err)
				}
				reportProto := evalH.ReportToProto(report)
				reportJSON, err := protojson.Marshal(reportProto)
				if err != nil {
					return intexp.RunResult{}, fmt.Errorf("marshal report: %w", err)
				}
				runs, err := experimentRunsForReport(report, reportProto, realized)
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

	sttDeps := sttH.Deps{
		Chain:                  chs.STT,
		Selector:               sttpkg.NewSelectorWithRegistry(chs.STT, audioEngine, engineRegistry),
		Registry:               engineRegistry,
		Voice:                  voiceSvc,
		SpeakerResource:        speakerClient,
		Engine:                 audioEngine,
		Logger:                 logger,
		Clock:                  clock.System{},
		Usage:                  usageRecorder,
		StreamConfig:           stores.STTStream,
		SpeakerConfig:          stores.STTSpeaker,
		Wakeword:               stores.Wakeword,
		Speaker:                stores.Speaker,
		Capacity:               sttcapacity.NewCLIReporter(),
		Sessions:               streamLedgers,
		EnableStreamTestFaults: env.EnableStreamTestFaults,
	}

	diagOrch := diagcore.New(diagcore.Deps{
		STT:       chs.STT,
		TTS:       chs.TTS,
		Summarize: chs.Summarize,
		Transcode: ffmpegTranscoder{},
		Usage:     usageRecorder,
		STTConfig: func(ctx context.Context) sttpkg.StreamConfig {
			return sttH.ResolveStreamPipelineConfig(ctx, stores.STTStream)
		},
		Registry: engineRegistry,
		// Layer-2 quality smoke: grade tiny bundled fixtures (no-speech
		// safety + clean-speech WER) through the same STT chain + egress
		// policy user-facing transcription uses, so a green readiness run
		// cannot hide a hallucination-filter regression.
		QualityFixtures: diagcore.DefaultQualityFixtures(),
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
		sttH.Module(sttDeps),
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
		corpusH.Module(corpusH.Deps{Logger: logger, Clock: clock.System{}, Service: corpusSvc}),
		experimentH.Module(experimentH.Deps{Logger: logger, Manager: experimentMgr, Service: experimentSvc, EstimateClipSeconds: estimateClipSeconds(corpusSvc)}),
		diagH.Module(diagOrch, logger),
	)

	cleanup := func() error {
		if experimentMgr != nil {
			experimentMgr.Close()
		}
		return db.Close()
	}
	return srv, cleanup, nil
}

func runExperimentReport(ctx context.Context, evalDeps evalH.Deps, corpusSvc *intcorpus.Service, ttsSvc *inttts.Service, audioEngine *audioformat.Engine, recipe *experimentv1.ExperimentRecipe, emit func(int, string)) (inteval.EvalReport, map[string]any, error) {
	longForm := recipe.GetLongForm()
	augmentation := recipe.GetAugmentation()
	speaker := recipe.GetSpeaker()
	longFormEnabled := longForm != nil && (longForm.GetEnabled() || longForm.GetTargetDurationSeconds() > 0 || longForm.GetGapMs() > 0 || longForm.GetTagContains() != "" || len(longForm.GetSweepDurationsSeconds()) > 0)
	augmentationEnabled := augmentation != nil && (len(augmentation.GetNoiseTypes()) > 0 || len(augmentation.GetCompetingVoiceIds()) > 0)
	speakerEnabled := speaker != nil && speakerConfigured(speaker)
	if len(recipe.GetCells()) > 0 && (augmentationEnabled || speakerEnabled) {
		return inteval.EvalReport{}, nil, fmt.Errorf("provider-neutral evaluation cells with augmentation or speaker ablation are not executable yet; do not relabel the legacy Whisper runner as provider-neutral")
	}
	if !longFormEnabled && !augmentationEnabled && !speakerEnabled {
		progress := newExperimentProgress(evalWorkUnits(len(recipe.GetClipIds()), recipe.GetStrategies(), recipe.GetRealtimeRepeats()), emit)
		opts := inteval.EvalOptions{
			DroppedSpanThresholdWords: int(safetyThreshold(recipe)),
			LatencyTailSeconds:        int(recipe.GetLatencyTailSeconds()),
		}
		if progress != nil {
			opts = progress.options("condition default", opts)
		}
		if len(recipe.GetCells()) > 0 {
			report, err := evalH.RunReportCellsWithOptions(ctx, evalDeps, recipe.GetClipIds(), recipe.GetCells(), recipe.GetChunkMs(), opts)
			return report, map[string]any{"phase": "default", "long_form": false, "provider_cells": true}, err
		}
		report, err := evalH.RunReportWithOptions(ctx, evalDeps, recipe.GetClipIds(), recipe.GetStrategies(), recipe.GetRealtimeRepeats(), recipe.GetChunkMs(), opts)
		return report, map[string]any{"phase": "default", "long_form": false}, err
	}
	if corpusSvc == nil {
		return inteval.EvalReport{}, nil, fmt.Errorf("experiment recipe materialization requires corpus service")
	}
	tagContains := ""
	if longForm != nil {
		tagContains = longForm.GetTagContains()
	}
	clips, err := loadRecipeClips(ctx, corpusSvc, recipe.GetClipIds(), tagContains)
	if err != nil {
		return inteval.EvalReport{}, nil, err
	}
	evalClips := make([]inteval.Clip, 0, len(clips))
	realized := map[string]any{"phase": "materialized", "long_form": false}
	realtimeConcurrency := 0
	if longFormEnabled {
		if longForm.GapMs <= 0 {
			longForm.GapMs = exprecipe.DefaultGapMs
		}
		if recipe.Seed == 0 {
			recipe.Seed = 1
		}
		// A non-empty sweep turns one experiment into N length-bucketed inputs,
		// one per listed duration, so the metric-vs-length curve is populated
		// from a single run. Each input is seeded distinctly (base seed +
		// duration) so the orderings differ; a single (non-sweep) input keeps
		// the recipe seed verbatim for backwards-compatible reproducibility.
		sweep := longForm.GetSweepDurationsSeconds()
		isSweep := len(sweep) > 0
		durations := sweep
		if !isSweep {
			durations = []int32{longForm.GetTargetDurationSeconds()}
		}
		recipe.RealizedClipIds = recipe.RealizedClipIds[:0]
		var totalDurationMs int64
		sweepRealized := make([]map[string]any, 0, len(durations))
		for _, durSec := range durations {
			seed := recipe.GetSeed()
			if isSweep {
				seed = recipe.GetSeed() + int64(durSec)
			}
			synthetic, longRealized, err := exprecipe.Build(exprecipe.Spec{
				Seed:                  seed,
				TargetDurationSeconds: int(durSec),
				GapMs:                 int(longForm.GetGapMs()),
			}, clips)
			if err != nil {
				return inteval.EvalReport{}, nil, err
			}
			if isSweep {
				synthetic.ID = fmt.Sprintf("long-form-%ds", durSec)
			}
			recipe.RealizedClipIds = append(recipe.RealizedClipIds, longRealized.ClipIDs...)
			recipe.RealizedReference = longRealized.Reference
			totalDurationMs += longRealized.DurationMs
			evalClips = append(evalClips, synthetic)
			sweepRealized = append(sweepRealized, map[string]any{
				"input_id":             synthetic.ID,
				"target_duration_sec":  durSec,
				"realized_duration_ms": longRealized.DurationMs,
				"clip_count":           len(longRealized.ClipIDs),
			})
		}
		recipe.RealizedDurationMs = totalDurationMs
		realized = map[string]any{
			"phase":                "long_form",
			"long_form":            true,
			"gap_ms":               longForm.GetGapMs(),
			"sweep":                isSweep,
			"input_count":          len(durations),
			"realized_duration_ms": totalDurationMs,
		}
		if isSweep {
			realized["sweep_durations_sec"] = sweep
			realized["sweep_inputs"] = sweepRealized
			realized["realtime_concurrency"] = 1
			realtimeConcurrency = 1
		} else {
			realized["target_duration_sec"] = longForm.GetTargetDurationSeconds()
		}
	} else {
		for _, c := range clips {
			evalClips = append(evalClips, inteval.Clip{
				ID:         c.ID,
				PCM:        c.PCM,
				SampleRate: c.SampleRate,
				Reference:  c.Reference,
				Format:     c.Format,
			})
		}
		realized["clip_count"] = len(evalClips)
	}
	var augGroups []exprecipe.ConditionGroup
	if augmentationEnabled {
		augmented, conditions, err := exprecipe.ApplyAugmentation(ctx, evalClips, exprecipe.AugmentationSpec{
			Seed:            recipe.GetSeed(),
			NoiseTypes:      augmentation.GetNoiseTypes(),
			SNRDB:           augmentation.GetSnrDb(),
			CompetingVoices: augmentation.GetCompetingVoiceIds(),
			CompetingText:   augmentation.GetCompetingText(),
			SynthesizeVoice: synthesizeCanonicalVoice(ttsSvc, audioEngine),
		})
		if err != nil {
			return inteval.EvalReport{}, nil, err
		}
		evalClips = augmented
		augGroups = exprecipe.GroupClipsByAugCondition(evalClips)
		recipe.RealizedAugmentationConditions = recipe.RealizedAugmentationConditions[:0]
		for _, c := range conditions {
			recipe.RealizedAugmentationConditions = append(recipe.RealizedAugmentationConditions, &experimentv1.AugmentationCondition{
				Id: c.ID, Kind: c.Kind, Source: c.Source, SnrDb: c.SNRDB, Skipped: c.Skipped, Note: c.Note,
			})
		}
		realized["augmentation"] = map[string]any{
			"noise_types":       augmentation.GetNoiseTypes(),
			"snr_db":            augmentation.GetSnrDb(),
			"competing_voices":  augmentation.GetCompetingVoiceIds(),
			"condition_count":   len(conditions),
			"evaluated_inputs":  len(evalClips),
			"row_count":         len(augGroups),
			"skipped_condition": countSkippedConditions(conditions),
		}
	}
	conditions := applySpeakerResourceAvailability(ctx, buildSpeakerConditions(speaker), evalDeps.SpeakerResource)
	recipe.RealizedSpeakerConditions = recipe.RealizedSpeakerConditions[:0]
	for _, cond := range conditions {
		recipe.RealizedSpeakerConditions = append(recipe.RealizedSpeakerConditions, &experimentv1.SpeakerCondition{
			Id:                  cond.ID,
			ExtractionEnabled:   cond.ExtractionEnabled,
			VerificationEnabled: cond.VerificationEnabled,
			VerificationMode:    cond.VerificationMode,
			Skipped:             cond.Skipped,
			Note:                cond.Note,
		})
	}
	realized["speaker"] = map[string]any{
		"enabled":                      speakerEnabled,
		"target_profile":               speaker.GetTargetProfileId(),
		"ablation_enabled":             speaker.GetAblationEnabled(),
		"condition_count":              len(conditions),
		"dropped_span_threshold_words": safetyThreshold(recipe),
	}
	if len(recipe.GetCells()) > 0 {
		if len(augGroups) != 0 || speakerEnabled {
			return inteval.EvalReport{}, nil, fmt.Errorf("provider-neutral evaluation cells with augmentation or speaker ablation are not executable yet; do not relabel the legacy Whisper runner as provider-neutral")
		}
		progress := newExperimentProgress(cellExperimentWorkUnits(evalClips, recipe.GetCells()), emit)
		opts := inteval.EvalOptions{
			LatencyTailSeconds:        int(recipe.GetLatencyTailSeconds()),
			DroppedSpanThresholdWords: int(safetyThreshold(recipe)),
			RealtimeConcurrency:       realtimeConcurrency,
		}
		if progress != nil {
			opts = progress.options("provider cell", opts)
		}
		report, err := evalH.RunReportForCells(ctx, evalDeps, evalClips, recipe.GetCells(), recipe.GetChunkMs(), opts)
		if err == nil {
			if warning, ok := longFormSourceDurationWarning(longForm, clips); ok {
				report.Warnings = appendUniqueReportWarnings(report.Warnings, warning)
			}
		}
		realized["provider_cells"] = true
		return report, realized, err
	}
	progress := newExperimentProgress(experimentWorkUnits(evalClips, augGroups, recipe.GetStrategies(), recipe.GetRealtimeRepeats(), conditions, speakerConfigured(speaker)), emit)
	report, err := runAugmentationSpeakerConditionReports(ctx, evalDeps, evalClips, augGroups, recipe.GetStrategies(), recipe.GetRealtimeRepeats(), recipe.GetChunkMs(), recipe.GetLatencyTailSeconds(), safetyThreshold(recipe), realtimeConcurrency, conditions, speakerConfigured(speaker), progress)
	if err == nil {
		if warning, ok := longFormSourceDurationWarning(longForm, clips); ok {
			report.Warnings = appendUniqueReportWarnings(report.Warnings, warning)
		}
		// Cross-condition ingress attribution: pair extraction-on rows with
		// their extraction-off siblings so target-speaker-extraction word loss
		// is no longer invisible. Runs exactly once on the fully-assembled set.
		inteval.AttributeIngressByAblation(&report)
	}
	return report, realized, err
}

func safetyThreshold(recipe *experimentv1.ExperimentRecipe) int32 {
	if recipe.GetDroppedSpanThresholdWords() > 0 {
		return recipe.GetDroppedSpanThresholdWords()
	}
	return int32(inteval.DefaultDroppedSpanThresholdWords)
}

type experimentProgress struct {
	total     int64
	completed atomic.Int64
	emit      func(int, string)
}

func newExperimentProgress(total int64, emit func(int, string)) *experimentProgress {
	if total <= 0 || emit == nil {
		return nil
	}
	return &experimentProgress{total: total, emit: emit}
}

func (p *experimentProgress) options(scope string, opts inteval.EvalOptions) inteval.EvalOptions {
	opts.ProgressScope = scope
	opts.Progress = p.step
	return opts
}

func (p *experimentProgress) step(update inteval.EvalProgress) {
	done := p.completed.Add(1)
	if done > p.total {
		done = p.total
	}
	progress := 5 + int(done*85/p.total)
	if progress > 94 {
		progress = 94
	}
	p.emit(progress, experimentProgressMessage(done, p.total, update))
}

func experimentProgressMessage(done, total int64, update inteval.EvalProgress) string {
	scope := strings.TrimSpace(update.Scope)
	if scope == "" {
		scope = "condition default"
	}
	phase := update.Phase
	if phase == "" {
		phase = "eval"
	}
	strategy := update.Strategy
	if strategy == "" {
		strategy = "strategy"
	}
	clip := update.ClipID
	if clip == "" {
		clip = "clip"
	}
	message := fmt.Sprintf("running %d/%d %s %s: strategy %s %d/%d, clip %s %d/%d",
		done, total, scope, phase, strategy, update.StrategyIndex, update.StrategyTotal, clip, update.ClipIndex, update.ClipTotal)
	if update.RepeatTotal > 0 {
		message += fmt.Sprintf(", repeat %d/%d", update.RepeatIndex, update.RepeatTotal)
	}
	return message
}

func experimentWorkUnits(clips []inteval.Clip, augGroups []exprecipe.ConditionGroup, strategies []*evalv1.EvalStrategy, realtimeRepeats int32, speakerConditions []speakerEvalCondition, speakerEnabled bool) int64 {
	if len(augGroups) == 0 {
		return evalWorkUnits(len(clips), strategies, realtimeRepeats) * int64(evaluatedSpeakerConditionCount(speakerConditions, speakerEnabled))
	}
	var total int64
	for _, group := range augGroups {
		total += evalWorkUnits(len(group.Clips), strategies, realtimeRepeats) * int64(evaluatedSpeakerConditionCount(speakerConditions, speakerEnabled))
	}
	return total
}

func cellExperimentWorkUnits(clips []inteval.Clip, cells []*experimentv1.EvaluationCell) int64 {
	if len(clips) == 0 || len(cells) == 0 {
		return 0
	}
	var total int64
	for _, cell := range cells {
		if cell == nil || cell.GetRepeatCount() < 1 {
			continue
		}
		total += int64(len(clips)) * int64(cell.GetRepeatCount())
	}
	return total
}

func evaluatedSpeakerConditionCount(conditions []speakerEvalCondition, speakerEnabled bool) int {
	if !speakerEnabled || len(conditions) == 0 {
		return 1
	}
	count := 0
	for _, cond := range conditions {
		if !cond.Skipped {
			count++
		}
	}
	return count
}

func evalWorkUnits(clipCount int, strategies []*evalv1.EvalStrategy, realtimeRepeats int32) int64 {
	if clipCount <= 0 {
		return 0
	}
	strategyCount := len(strategies)
	if strategyCount == 0 {
		strategyCount = 3
	}
	repeats := int(realtimeRepeats)
	if repeats < 0 {
		repeats = 0
	}
	return int64(clipCount * strategyCount * (1 + repeats))
}

func estimateClipSeconds(corpusSvc *intcorpus.Service) func(context.Context, []string) (int32, error) {
	if corpusSvc == nil {
		return nil
	}
	return func(ctx context.Context, clipIDs []string) (int32, error) {
		var clips []intcorpus.Clip
		var err error
		if len(clipIDs) == 0 {
			clips, err = corpusSvc.ListClips(ctx, intcorpus.ListFilter{})
			if err != nil {
				return 0, err
			}
		} else {
			clips = make([]intcorpus.Clip, 0, len(clipIDs))
			for _, id := range clipIDs {
				clip, err := corpusSvc.GetClip(ctx, id)
				if err != nil {
					return 0, err
				}
				clips = append(clips, clip)
			}
		}
		var totalMs int64
		for _, clip := range clips {
			if clip.DurationMs > 0 {
				totalMs += clip.DurationMs
			}
		}
		if totalMs <= 0 {
			return 0, nil
		}
		return int32((totalMs + 999) / 1000), nil
	}
}

func newExperimentEvalDeps(logger logx.Logger, corpusSvc *intcorpus.Service, evalProviderForEngine func(string) sttchain.Provider, defaults sttpkg.StreamConfig, speakerClient *sttpipeline.SpeakerClient) evalH.Deps {
	return evalH.Deps{
		Logger: logger,
		Clock:  clock.System{},
		Corpus: corpusSvc,
		NewProvider: func() sttchain.Provider {
			return evalProviderForEngine("whisper-local")
		},
		NewProviderForEngine: evalProviderForEngine,
		Defaults:             defaults,
		SpeakerResource:      speakerClient,
	}
}

func experimentRunsForReport(report inteval.EvalReport, reportProto *evalv1.EvalReport, realized map[string]any) ([]intexp.Run, error) {
	protoRows := reportProto.GetPerStrategy()
	if len(report.PerStrategy) != len(protoRows) {
		return nil, fmt.Errorf("experiment report row mismatch: internal=%d proto=%d", len(report.PerStrategy), len(protoRows))
	}
	runs := make([]intexp.Run, 0, len(protoRows))
	for i, strategyReport := range protoRows {
		condition, err := experimentRunConditionJSON(report.PerStrategy[i], realized)
		if err != nil {
			return nil, fmt.Errorf("marshal strategy condition: %w", err)
		}
		runs = append(runs, intexp.Run{
			Strategy:      strategyReport.GetStrategy(),
			ConditionJSON: condition,
		})
	}
	return runs, nil
}

func experimentRunConditionJSON(row inteval.StrategyReport, realized map[string]any) ([]byte, error) {
	baseStrategy := row.BaseStrategy
	if baseStrategy == "" {
		baseStrategy = row.Strategy
	}
	return json.Marshal(map[string]any{
		"strategy":        string(row.Strategy),
		"base_strategy":   string(baseStrategy),
		"label":           row.Label,
		"engine_id":       row.EngineID,
		"policy_profile":  row.PolicyProfile,
		"replay_lane":     row.ReplayLane,
		"fault_profile":   row.FaultProfile,
		"condition_group": row.ConditionGroup,
		"speaker": map[string]any{
			"extraction_enabled":   row.ExtractionEnabled,
			"verification_enabled": row.VerificationEnabled,
		},
		"realized": realized,
	})
}

func longFormSourceDurationWarning(longForm *experimentv1.LongFormRecipe, clips []exprecipe.Clip) (inteval.ReportWarning, bool) {
	if longForm == nil {
		return inteval.ReportWarning{}, false
	}
	targets := longForm.GetSweepDurationsSeconds()
	if len(targets) == 0 && longForm.GetTargetDurationSeconds() > 0 {
		targets = []int32{longForm.GetTargetDurationSeconds()}
	}
	var maxTargetSec int32
	for _, target := range targets {
		if target > maxTargetSec {
			maxTargetSec = target
		}
	}
	if maxTargetSec <= 0 {
		return inteval.ReportWarning{}, false
	}
	sourceMs := sourceDurationMs(clips)
	targetMs := int64(maxTargetSec) * int64(time.Second/time.Millisecond)
	if sourceMs <= 0 || sourceMs >= targetMs {
		return inteval.ReportWarning{}, false
	}
	return inteval.ReportWarning{
		Code:     "source_audio_under_target",
		Severity: "warning",
		Message:  fmt.Sprintf("The selected corpus provides %.1f seconds of unique source audio, below the requested long-form target of %d seconds; generated inputs may reuse clips.", float64(sourceMs)/1000, maxTargetSec),
	}, true
}

func sourceDurationMs(clips []exprecipe.Clip) int64 {
	var total int64
	for _, clip := range clips {
		sampleRate := clip.SampleRate
		if sampleRate <= 0 {
			sampleRate = exprecipe.CanonicalSampleRate
		}
		if sampleRate <= 0 {
			continue
		}
		total += int64(float64(len(clip.PCM)) / float64(sampleRate*2) * float64(time.Second/time.Millisecond))
	}
	return total
}

type speakerEvalCondition struct {
	ID                  string
	Config              sttpipeline.SpeakerConfig
	ExtractionEnabled   bool
	VerificationEnabled bool
	VerificationMode    sttv1.SpeakerMode
	Skipped             bool
	Note                string
}

func speakerConfigured(s *experimentv1.SpeakerExperimentRecipe) bool {
	return s.GetTargetProfileId() != "" || s.GetExtractionEnabled() || s.GetVerificationEnabled() || s.GetAblationEnabled()
}

func buildSpeakerConditions(s *experimentv1.SpeakerExperimentRecipe) []speakerEvalCondition {
	if s == nil || !speakerConfigured(s) {
		return []speakerEvalCondition{{ID: "speaker_off", Note: "speaker extraction and verification disabled"}}
	}
	if !s.GetAblationEnabled() {
		return []speakerEvalCondition{speakerConditionFromRecipe("speaker_recipe", s, s.GetExtractionEnabled(), s.GetVerificationEnabled())}
	}
	return []speakerEvalCondition{
		speakerConditionFromRecipe("extract_off_verify_off", s, false, false),
		speakerConditionFromRecipe("extract_on_verify_off", s, true, false),
		speakerConditionFromRecipe("extract_off_verify_on", s, false, true),
		speakerConditionFromRecipe("extract_on_verify_on", s, true, true),
	}
}

func applySpeakerResourceAvailability(ctx context.Context, conditions []speakerEvalCondition, client *sttpipeline.SpeakerClient) []speakerEvalCondition {
	needsResource := false
	for _, cond := range conditions {
		if cond.Skipped {
			continue
		}
		if cond.ExtractionEnabled || cond.VerificationEnabled {
			needsResource = true
			break
		}
	}
	if !needsResource {
		return conditions
	}
	note := ""
	if client == nil {
		note = "speaker condition skipped: speaker resource is not configured"
	} else if ready, err := client.Ready(ctx); err != nil {
		note = "speaker condition skipped: speaker resource unreachable: " + err.Error()
	} else if (ready.Status != "ready" && ready.Status != "ok") || !ready.ModelLoaded || !ready.ProfileStoreOK || !ready.TempDirOK {
		note = fmt.Sprintf("speaker condition skipped: speaker resource not ready (status=%s model_loaded=%t profile_store_ok=%t temp_dir_ok=%t)", ready.Status, ready.ModelLoaded, ready.ProfileStoreOK, ready.TempDirOK)
	}
	if note == "" {
		return conditions
	}
	for i := range conditions {
		if conditions[i].Skipped || (!conditions[i].ExtractionEnabled && !conditions[i].VerificationEnabled) {
			continue
		}
		conditions[i].Skipped = true
		conditions[i].Note = note
	}
	return conditions
}

func speakerConditionFromRecipe(id string, s *experimentv1.SpeakerExperimentRecipe, extraction bool, verification bool) speakerEvalCondition {
	mode := s.GetVerificationMode()
	modeStr := protomap.SpeakerModeFromProto(mode)
	if modeStr == "" || modeStr == "off" {
		modeStr = "filter"
	}
	threshold := s.GetThreshold()
	if threshold == 0 {
		threshold = sttpipeline.DefaultSpeakerConfig().Threshold
	}
	enabled := extraction || verification
	cfg := sttpipeline.DefaultSpeakerConfig()
	cfg.Enabled = enabled
	cfg.ProfileIDs = nil
	if s.GetTargetProfileId() != "" {
		cfg.ProfileIDs = []string{s.GetTargetProfileId()}
	}
	cfg.Threshold = threshold
	cfg.Mode = modeStr
	cfg.RejectBehavior = "drop"
	cfg.FallbackWithoutVerification = s.GetFallbackWithoutVerification()
	cfg.ExtractionEnabled = extraction
	note := "speaker stages disabled"
	if enabled {
		note = fmt.Sprintf("profile=%s extraction=%t verification=%t mode=%s", s.GetTargetProfileId(), extraction, verification, modeStr)
	}
	if enabled && s.GetTargetProfileId() == "" {
		return speakerEvalCondition{
			ID:                  id,
			Config:              cfg,
			ExtractionEnabled:   extraction,
			VerificationEnabled: verification,
			VerificationMode:    mode,
			Skipped:             true,
			Note:                "speaker condition skipped: target profile id is required",
		}
	}
	return speakerEvalCondition{
		ID:                  id,
		Config:              cfg,
		ExtractionEnabled:   extraction,
		VerificationEnabled: verification,
		VerificationMode:    mode,
		Note:                note,
	}
}

func runSpeakerConditionReportsWithOptions(ctx context.Context, evalDeps evalH.Deps, clips []inteval.Clip, strategies []*evalv1.EvalStrategy, realtimeRepeats, chunkMs int32, opts inteval.EvalOptions, conditions []speakerEvalCondition, progressScope string, progress *experimentProgress) (inteval.EvalReport, error) {
	if len(conditions) == 0 {
		if progress != nil {
			opts = progress.options(conditionScope(progressScope, "condition default"), opts)
		}
		return evalH.RunReportForClipsWithOptions(ctx, evalDeps, clips, strategies, realtimeRepeats, chunkMs, opts)
	}
	var combined inteval.EvalReport
	haveMetadata := false
	for _, cond := range conditions {
		if cond.Skipped {
			combined.Warnings = appendUniqueReportWarnings(combined.Warnings, speakerConditionSkippedWarning(cond))
			continue
		}
		condOpts := opts
		if progress != nil {
			condOpts = progress.options(conditionScope(progressScope, "speaker "+cond.ID), opts)
		}
		deps := evalDeps
		if cond.ExtractionEnabled || cond.VerificationEnabled {
			cfg := cond.Config
			deps.SpeakerConfig = &cfg
			deps.SpeakerExtractionEnabled = cond.ExtractionEnabled
			deps.SpeakerVerificationEnabled = cond.VerificationEnabled
		} else {
			deps.SpeakerConfig = nil
			deps.SpeakerExtractionEnabled = false
			deps.SpeakerVerificationEnabled = false
		}
		report, err := evalH.RunReportForClipsWithOptions(ctx, deps, clips, strategies, realtimeRepeats, chunkMs, condOpts)
		if err != nil {
			return inteval.EvalReport{}, err
		}
		if !haveMetadata {
			combined = report
			combined.PerStrategy = nil
			haveMetadata = true
		} else {
			combined.QualityMeasured = combined.QualityMeasured || report.QualityMeasured
			combined.LatencyMeasured = combined.LatencyMeasured || report.LatencyMeasured
			combined.Warnings = appendUniqueReportWarnings(combined.Warnings, report.Warnings...)
		}
		for _, row := range report.PerStrategy {
			row.BaseStrategy = row.Strategy
			row.ExtractionEnabled = cond.ExtractionEnabled
			row.VerificationEnabled = cond.VerificationEnabled
			row.Strategy = sttchain.StrategyKind(fmt.Sprintf("%s/%s", row.Strategy, cond.ID))
			if row.Label == "" {
				row.Label = string(row.Strategy)
			} else {
				row.Label = fmt.Sprintf("%s / %s", row.Label, cond.ID)
			}
			combined.PerStrategy = append(combined.PerStrategy, row)
		}
	}
	if len(combined.PerStrategy) == 0 {
		combined.Summary = inteval.EvalReportSummary{
			Recommendation:  "No speaker experiment conditions were evaluated.",
			Confidence:      "low",
			ConfidenceNotes: []string{"Every requested speaker condition was skipped before evaluation."},
		}
		return combined, nil
	}
	return combined, nil
}

func speakerConditionSkippedWarning(cond speakerEvalCondition) inteval.ReportWarning {
	message := cond.Note
	if message == "" {
		message = "speaker condition skipped"
	}
	return inteval.ReportWarning{
		Code:     "speaker_condition_skipped",
		Severity: "warning",
		Message:  fmt.Sprintf("%s: %s", cond.ID, message),
	}
}

func runAugmentationSpeakerConditionReports(ctx context.Context, evalDeps evalH.Deps, clips []inteval.Clip, augGroups []exprecipe.ConditionGroup, strategies []*evalv1.EvalStrategy, realtimeRepeats, chunkMs, latencyTailSeconds, droppedSpanThresholdWords int32, realtimeConcurrency int, speakerConditions []speakerEvalCondition, speakerEnabled bool, progress *experimentProgress) (inteval.EvalReport, error) {
	if len(augGroups) == 0 {
		return runSpeakerConditionReportsWithOptions(ctx, evalDeps, clips, strategies, realtimeRepeats, chunkMs, inteval.EvalOptions{
			LatencyTailSeconds:        int(latencyTailSeconds),
			DroppedSpanThresholdWords: int(droppedSpanThresholdWords),
			RealtimeConcurrency:       realtimeConcurrency,
		}, speakerConditions, "", progress)
	}
	var combined inteval.EvalReport
	haveMetadata := false
	for _, group := range augGroups {
		report, err := runSpeakerConditionReportsForAugmentation(ctx, evalDeps, group.Clips, strategies, realtimeRepeats, chunkMs, latencyTailSeconds, droppedSpanThresholdWords, realtimeConcurrency, speakerConditions, speakerEnabled, group.ID, progress)
		if err != nil {
			return inteval.EvalReport{}, err
		}
		if !haveMetadata {
			combined = report
			combined.PerStrategy = nil
			haveMetadata = true
		} else {
			combined.QualityMeasured = combined.QualityMeasured || report.QualityMeasured
			combined.LatencyMeasured = combined.LatencyMeasured || report.LatencyMeasured
			combined.Warnings = appendUniqueReportWarnings(combined.Warnings, report.Warnings...)
		}
		for _, row := range report.PerStrategy {
			row.ConditionGroup = group.ID
			if row.BaseStrategy == "" {
				row.BaseStrategy = row.Strategy
			}
			row.Strategy = sttchain.StrategyKind(fmt.Sprintf("%s/%s", row.Strategy, group.ID))
			if row.Label == "" {
				row.Label = string(row.Strategy)
			} else {
				row.Label = fmt.Sprintf("%s / %s", row.Label, group.ID)
			}
			combined.PerStrategy = append(combined.PerStrategy, row)
		}
	}
	if len(combined.PerStrategy) == 0 {
		return inteval.EvalReport{}, fmt.Errorf("all augmentation experiment conditions were skipped")
	}
	return combined, nil
}

func runSpeakerConditionReportsForAugmentation(ctx context.Context, evalDeps evalH.Deps, clips []inteval.Clip, strategies []*evalv1.EvalStrategy, realtimeRepeats, chunkMs, latencyTailSeconds, droppedSpanThresholdWords int32, realtimeConcurrency int, speakerConditions []speakerEvalCondition, speakerEnabled bool, augmentationID string, progress *experimentProgress) (inteval.EvalReport, error) {
	if !speakerEnabled {
		opts := inteval.EvalOptions{
			DroppedSpanThresholdWords: int(droppedSpanThresholdWords),
			LatencyTailSeconds:        int(latencyTailSeconds),
			RealtimeConcurrency:       realtimeConcurrency,
		}
		if progress != nil {
			opts = progress.options("augmentation "+augmentationID, opts)
		}
		return evalH.RunReportForClipsWithOptions(ctx, evalDeps, clips, strategies, realtimeRepeats, chunkMs, opts)
	}
	return runSpeakerConditionReportsWithOptions(ctx, evalDeps, clips, strategies, realtimeRepeats, chunkMs, inteval.EvalOptions{
		DroppedSpanThresholdWords: int(droppedSpanThresholdWords),
		LatencyTailSeconds:        int(latencyTailSeconds),
		RealtimeConcurrency:       realtimeConcurrency,
	}, speakerConditions, "augmentation "+augmentationID, progress)
}

func conditionScope(prefix, scope string) string {
	if prefix == "" {
		return scope
	}
	if scope == "" {
		return prefix
	}
	return prefix + " / " + scope
}

func appendUniqueReportWarnings(dst []inteval.ReportWarning, src ...inteval.ReportWarning) []inteval.ReportWarning {
	if len(src) == 0 {
		return dst
	}
	seen := make(map[string]struct{}, len(dst)+len(src))
	for _, warning := range dst {
		seen[reportWarningKey(warning)] = struct{}{}
	}
	for _, warning := range src {
		key := reportWarningKey(warning)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		dst = append(dst, warning)
	}
	return dst
}

func reportWarningKey(warning inteval.ReportWarning) string {
	return warning.Severity + "\x00" + warning.Code + "\x00" + warning.Message
}

func synthesizeCanonicalVoice(ttsSvc *inttts.Service, audioEngine *audioformat.Engine) func(context.Context, string, string) ([]byte, error) {
	return func(ctx context.Context, voice string, text string) ([]byte, error) {
		if ttsSvc == nil || audioEngine == nil {
			return nil, fmt.Errorf("TTS synthesis is not configured")
		}
		out, err := ttsSvc.Synthesize(ctx, inttts.SynthesizeInput{
			Input:          text,
			Voice:          voice,
			ResponseFormat: string(audioformat.OutputWAV),
			Speed:          1,
		})
		if err != nil {
			return nil, err
		}
		codec, err := audioformat.Detect(audioformat.CodecUnknown, out.Audio)
		if err != nil {
			return nil, err
		}
		pcm, err := audioEngine.Normalize(ctx, codec, out.Audio)
		if err != nil {
			return nil, err
		}
		return pcm, nil
	}
}

func countSkippedConditions(conditions []exprecipe.Condition) int {
	var n int
	for _, c := range conditions {
		if c.Skipped {
			n++
		}
	}
	return n
}

func loadRecipeClips(ctx context.Context, corpusSvc *intcorpus.Service, clipIDs []string, tagContains string) ([]exprecipe.Clip, error) {
	var metas []intcorpus.Clip
	if len(clipIDs) == 0 {
		all, err := corpusSvc.ListClips(ctx, intcorpus.ListFilter{TagContains: tagContains})
		if err != nil {
			return nil, err
		}
		metas = all
	} else {
		for _, id := range clipIDs {
			c, err := corpusSvc.GetClip(ctx, id)
			if err != nil {
				return nil, err
			}
			if tagContains != "" && !clipMatchesTag(c, tagContains) {
				continue
			}
			metas = append(metas, c)
		}
	}
	clips := make([]exprecipe.Clip, 0, len(metas))
	for _, meta := range metas {
		audio, _, err := corpusSvc.GetClipAudio(ctx, meta.ID)
		if err != nil {
			return nil, fmt.Errorf("load audio for clip %q: %w", meta.ID, err)
		}
		clips = append(clips, exprecipe.Clip{
			ID:         meta.ID,
			PCM:        audio,
			SampleRate: meta.SampleRateHz,
			Reference:  meta.ReferenceText,
			Format:     meta.Format,
		})
	}
	return clips, nil
}

func clipMatchesTag(c intcorpus.Clip, needle string) bool {
	for _, tag := range c.Tags {
		if strings.Contains(tag, needle) {
			return true
		}
	}
	return false
}

// ffmpegTranscoder adapts internal/audio.TranscodeOpts to the
// diagcore.Transcoder seam without dragging diagnostics into the audio
// package's import graph.
type ffmpegTranscoder struct{}

func (ffmpegTranscoder) Transcode(ctx context.Context, audio []byte, outputFormat string) ([]byte, error) {
	return intaudio.TranscodeOpts(ctx, audio, outputFormat, 0, 0, 0)
}
