package bootstrap

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	// evalProvider builds a fresh Local (Whisper) provider per replay; the
	// eval handler wraps each in a metered decorator.
	evalProvider := func() sttchain.Provider { return sttchain.NewLocalProvider(voiceSvc) }
	evalDeps := evalH.Deps{
		Logger:      logger,
		Clock:       clock.System{},
		Corpus:      corpusSvc,
		NewProvider: evalProvider,
		Defaults:    sttpkg.Defaults(),
	}
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
				report, realized, err := runExperimentReport(runCtx, evalDeps, corpusSvc, ttsSvc, audioEngine, recipe)
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
				runs := make([]intexp.Run, 0, len(reportProto.GetPerStrategy()))
				for _, strategyReport := range reportProto.GetPerStrategy() {
					metrics, err := protojson.Marshal(strategyReport)
					if err != nil {
						return intexp.RunResult{}, fmt.Errorf("marshal strategy metrics: %w", err)
					}
					condition, _ := json.Marshal(realized)
					runs = append(runs, intexp.Run{
						Strategy:      strategyReport.GetStrategy(),
						ConditionJSON: condition,
						MetricsJSON:   metrics,
					})
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
			SpeakerConfig:   stores.STTSpeaker,
			Wakeword:        stores.Wakeword,
			Speaker:         stores.Speaker,
			Capacity:        sttcapacity.NewCLIReporter(),
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
		corpusH.Module(corpusH.Deps{Logger: logger, Clock: clock.System{}, Service: corpusSvc}),
		evalH.Module(evalDeps),
		experimentH.Module(experimentH.Deps{Logger: logger, Manager: experimentMgr, Service: experimentSvc}),
		diagH.Module(diagOrch, logger),
	)

	_ = dsn // retained for diagnostics if Deps is exposed; not used here.
	cleanup := func() error {
		if experimentMgr != nil {
			experimentMgr.Close()
		}
		return db.Close()
	}
	return srv, cleanup, nil
}

func runExperimentReport(ctx context.Context, evalDeps evalH.Deps, corpusSvc *intcorpus.Service, ttsSvc *inttts.Service, audioEngine *audioformat.Engine, recipe *experimentv1.ExperimentRecipe) (inteval.EvalReport, map[string]any, error) {
	longForm := recipe.GetLongForm()
	augmentation := recipe.GetAugmentation()
	speaker := recipe.GetSpeaker()
	longFormEnabled := longForm != nil && (longForm.GetEnabled() || longForm.GetTargetDurationSeconds() > 0 || longForm.GetGapMs() > 0 || longForm.GetTagContains() != "" || len(longForm.GetSweepDurationsSeconds()) > 0)
	augmentationEnabled := augmentation != nil && (len(augmentation.GetNoiseTypes()) > 0 || len(augmentation.GetCompetingVoiceIds()) > 0)
	speakerEnabled := speaker != nil && speakerConfigured(speaker)
	if !longFormEnabled && !augmentationEnabled && !speakerEnabled {
		report, err := evalH.RunReportWithOptions(ctx, evalDeps, recipe.GetClipIds(), recipe.GetStrategies(), recipe.GetRealtimeRepeats(), recipe.GetChunkMs(), inteval.EvalOptions{
			DroppedSpanThresholdWords: int(safetyThreshold(recipe)),
			LatencyTailSeconds:        int(recipe.GetLatencyTailSeconds()),
		})
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
	conditions := buildSpeakerConditions(speaker)
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
	report, err := runAugmentationSpeakerConditionReports(ctx, evalDeps, evalClips, augGroups, recipe.GetStrategies(), recipe.GetRealtimeRepeats(), recipe.GetChunkMs(), recipe.GetLatencyTailSeconds(), safetyThreshold(recipe), conditions, speakerConfigured(speaker))
	if err == nil {
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

func runSpeakerConditionReportsWithOptions(ctx context.Context, evalDeps evalH.Deps, clips []inteval.Clip, strategies []*evalv1.EvalStrategy, realtimeRepeats, chunkMs int32, opts inteval.EvalOptions, conditions []speakerEvalCondition) (inteval.EvalReport, error) {
	if len(conditions) == 0 {
		return evalH.RunReportForClipsWithOptions(ctx, evalDeps, clips, strategies, realtimeRepeats, chunkMs, opts)
	}
	var combined inteval.EvalReport
	haveMetadata := false
	for _, cond := range conditions {
		if cond.Skipped {
			continue
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
		report, err := evalH.RunReportForClipsWithOptions(ctx, deps, clips, strategies, realtimeRepeats, chunkMs, opts)
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
		return inteval.EvalReport{}, fmt.Errorf("all speaker experiment conditions were skipped")
	}
	return combined, nil
}

func runAugmentationSpeakerConditionReports(ctx context.Context, evalDeps evalH.Deps, clips []inteval.Clip, augGroups []exprecipe.ConditionGroup, strategies []*evalv1.EvalStrategy, realtimeRepeats, chunkMs, latencyTailSeconds, droppedSpanThresholdWords int32, speakerConditions []speakerEvalCondition, speakerEnabled bool) (inteval.EvalReport, error) {
	if len(augGroups) == 0 {
		return runSpeakerConditionReportsWithOptions(ctx, evalDeps, clips, strategies, realtimeRepeats, chunkMs, inteval.EvalOptions{
			LatencyTailSeconds:        int(latencyTailSeconds),
			DroppedSpanThresholdWords: int(droppedSpanThresholdWords),
		}, speakerConditions)
	}
	var combined inteval.EvalReport
	haveMetadata := false
	for _, group := range augGroups {
		report, err := runSpeakerConditionReportsForAugmentation(ctx, evalDeps, group.Clips, strategies, realtimeRepeats, chunkMs, latencyTailSeconds, droppedSpanThresholdWords, speakerConditions, speakerEnabled)
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

func runSpeakerConditionReportsForAugmentation(ctx context.Context, evalDeps evalH.Deps, clips []inteval.Clip, strategies []*evalv1.EvalStrategy, realtimeRepeats, chunkMs, latencyTailSeconds, droppedSpanThresholdWords int32, speakerConditions []speakerEvalCondition, speakerEnabled bool) (inteval.EvalReport, error) {
	if !speakerEnabled {
		return evalH.RunReportForClipsWithOptions(ctx, evalDeps, clips, strategies, realtimeRepeats, chunkMs, inteval.EvalOptions{
			DroppedSpanThresholdWords: int(droppedSpanThresholdWords),
			LatencyTailSeconds:        int(latencyTailSeconds),
		})
	}
	return runSpeakerConditionReportsWithOptions(ctx, evalDeps, clips, strategies, realtimeRepeats, chunkMs, inteval.EvalOptions{
		DroppedSpanThresholdWords: int(droppedSpanThresholdWords),
		LatencyTailSeconds:        int(latencyTailSeconds),
	}, speakerConditions)
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
