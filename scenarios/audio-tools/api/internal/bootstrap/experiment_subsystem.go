//go:build audio_lab

package bootstrap

import (
	"context"
	"fmt"

	sttchain "audio-tools/internal/ai/sttchain"
	"audio-tools/internal/audioformat"
	"audio-tools/internal/blobbytes"
	"audio-tools/internal/buildidentity"
	intcorpus "audio-tools/internal/corpus"
	intexp "audio-tools/internal/experiment"
	"audio-tools/internal/experiment/evaldeps"
	expreport "audio-tools/internal/experiment/report"
	"audio-tools/internal/logx"
	"audio-tools/internal/providercatalog"
	sttpkg "audio-tools/internal/stt"
	sttpipeline "audio-tools/internal/stt/pipeline"
	inttts "audio-tools/internal/tts"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/api-core/storage"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	"google.golang.org/protobuf/encoding/protojson"
)

func buildExperimentSubsystem(ctx context.Context, env Env, db *database.RoutedDB, logger logx.Logger, whisperSvc *sttpipeline.WhisperTranscriber, speakerClient *sttpipeline.SpeakerClient, ttsSvc *inttts.Service, audioEngine *audioformat.Engine) experimentSubsystem {
	if !env.EnableLabAPI {
		return experimentSubsystem{experimentCounter: emptyExperimentCounter{}}
	}
	// The lab API keeps aggregate projections colocated with the metadata
	// repository while the product build has no dependency on these packages.
	corpusRepo := intcorpus.NewSQLiteRepository(db, schedule.System())
	experimentRepo := intexp.NewSQLiteRepository(db, schedule.System())
	out := experimentSubsystem{
		corpusCounter:         corpusCounterProjection{repo: corpusRepo},
		experimentCounter:     experimentRepo,
		qualificationEvidence: experimentRepo,
	}

	// Lab artifacts use the variant-aware runtime namespace so a shadow corpus
	// never collides with live. Blob-store initialization failure disables lab
	// execution without failing the product runtime.
	var corpusSvc *intcorpus.Service
	var experimentSvc *intexp.Service
	var experimentMgr *intexp.Manager
	if ns, nerr := storage.ScenarioNamespace("audio-tools"); nerr != nil {
		logger.Printf("corpus blob store disabled: resolve namespace: %v", nerr)
	} else if blobs, berr := blobbytes.NewFilesystem(ns, "corpus-blobs", "corpus"); berr != nil {
		logger.Printf("corpus blob store disabled: %v", berr)
	} else {
		corpusSvc = intcorpus.NewService(corpusRepo, blobs, schedule.System())
		if expBlobs, eerr := blobbytes.NewFilesystem(ns, "experiment-blobs", "experiment"); eerr != nil {
			logger.Printf("experiment blob store disabled: %v", eerr)
		} else {
			experimentSvc = intexp.NewService(experimentRepo, expBlobs)
		}
	}
	// Each experiment replay receives a fresh provider. Provider-neutral cells
	// select this factory by engine id instead of relabelling a Whisper run.
	evalProviderForEngine := evalProviderFactory(env, whisperSvc)
	evalDeps := evaldeps.New(logger, corpusSvc, evalProviderForEngine, sttpkg.Defaults(), speakerClient)
	// The async worker belongs to the supervisor-selected lab runtime. Starting
	// it in product mode would consume resources with no mounted lab API.
	if shouldStartLabWorker(env.EnableLabAPI, experimentSvc != nil) {
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
				var qualityEvidence []intexp.QualificationEvidence
				if corpusSvc != nil {
					clips, listErr := corpusSvc.ListClips(runCtx, intcorpus.ListFilter{})
					if listErr != nil {
						return intexp.RunResult{}, fmt.Errorf("load corpus provenance for quality evidence: %w", listErr)
					}
					qualityEvidence = expreport.QualityEvidenceForReport(clips, report, buildidentity.SourceIdentity)
				}
				metricsOnly := recipe.GetReportRetentionMode() == experimentv1.ReportRetentionMode_REPORT_RETENTION_MODE_METRICS_ONLY
				persistedRecipe := recipeForCompletedExperiment(recipe, metricsOnly)
				recipeJSON, err := protojson.Marshal(persistedRecipe)
				if err != nil {
					return intexp.RunResult{}, fmt.Errorf("marshal realized recipe: %w", err)
				}
				reportProto, reportJSON, err := reportForPersistence(report, recipe.GetReportRetentionMode())
				if err != nil {
					return intexp.RunResult{}, fmt.Errorf("marshal report: %w", err)
				}
				var runs []intexp.Run
				if metricsOnly {
					runs, err = expreport.ExperimentRunsForMetricsOnlyReport(report, reportProto)
				} else {
					runs, err = expreport.ExperimentRunsForReport(report, reportProto, realized)
				}
				if err != nil {
					return intexp.RunResult{}, err
				}
				emit(95, "storing report")
				return intexp.RunResult{Report: reportJSON, ReportMIME: "application/json", Runs: runs, RecipeJSON: recipeJSON, QualificationEvidence: qualityEvidence}, nil
			},
		})
		if err := experimentMgr.Start(ctx); err != nil {
			logger.Printf("experiment manager disabled: %v", err)
			experimentMgr = nil
			experimentSvc = nil
		}
	}

	out.labRuntime = &labRuntimeDeps{Corpus: corpusSvc, Experiments: experimentSvc, ExperimentManager: experimentMgr}
	return out
}

// evalProviderFactory binds executable adapters to provider declarations, then
// lets the catalogue resolve engine IDs. Engine renames or additions for an
// existing provider therefore do not require edits here.
func evalProviderFactory(env Env, whisperSvc *sttpipeline.WhisperTranscriber) func(string) sttchain.Provider {
	// Adapter implementations are registered once. Providers and engines bind
	// to these stable adapter keys in providercatalog, so adding another engine
	// which speaks an existing protocol is a catalogue-only change.
	adapters := map[string]func() sttchain.Provider{
		"whisper-local": func() sttchain.Provider { return sttchain.NewLocalProvider(whisperSvc) },
		"vrooli-stream": func() sttchain.Provider { return sttchain.NewKyutaiProvider(env.KyutaiURL) },
		"sherpa-stream": func() sttchain.Provider { return sttchain.NewSherpaProvider(env.SherpaURL) },
	}
	return func(engineID string) sttchain.Provider {
		if engineID == "" || engineID == "eval-whisper-local" {
			return sttchain.NewLocalProvider(whisperSvc)
		}
		providerID, ok := providercatalog.ProviderIDForEngine(engineID)
		if !ok {
			return nil
		}
		provider, ok := providercatalog.Default().ByID(providerID)
		if !ok || provider.STTEvalAdapterID == "" {
			return nil
		}
		factory := adapters[provider.STTEvalAdapterID]
		if factory == nil {
			return nil
		}
		return factory()
	}
}
