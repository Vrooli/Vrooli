//go:build audio_lab

package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"audio-tools/internal/database"
	inteval "audio-tools/internal/eval"
	intexp "audio-tools/internal/experiment"
	expreport "audio-tools/internal/experiment/report"
	"audio-tools/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/scheduletest"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
	experiment "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestReportForPersistence_MetricsOnlyIsTranscriptFreeAndKeepsMetrics(t *testing.T) {
	const secret = "PRIVATE-EVAL-TRANSCRIPT-SENTINEL"
	report := inteval.EvalReport{
		QualityMeasured: true, LatencyMeasured: true,
		Summary:  inteval.EvalReportSummary{Recommendation: secret, Reasons: []string{secret}},
		Warnings: []inteval.ReportWarning{{Message: secret}},
		PerStrategy: []inteval.StrategyReport{{
			Strategy: "vad_segment", Label: secret, EngineID: "engine-a", ModelID: "model-b", PolicyProfile: "policy-c", ReplayLane: "realtime", FaultProfile: "clean",
			WER: .25, EditCounts: inteval.EditCounts{Substitutions: 1}, RefWords: 4, RTF: .5,
			FinalizationLatencyP50Ms: 120, FinalizationLatencyP95Ms: 240,
			Reasons: []string{secret}, Safety: inteval.SafetyGateReport{Reasons: []string{secret}, RetractionEvents: []inteval.RetractionEvent{{PreviousText: secret, CurrentText: secret}}},
			Scaling: inteval.ScalingAnalysis{Points: []inteval.ScalingPoint{{ClipID: secret, WER: .1}}},
			PerClip: []inteval.ClipResult{{ClipID: secret, Reference: secret, Hypothesis: secret, NormalizedReference: secret, NormalizedHypothesis: secret,
				EditOperations: []inteval.EditOperation{{ReferenceToken: secret, HypothesisToken: secret}}, CommitTimeline: []inteval.CommitState{{Text: secret}},
				WER: inteval.WERResult{EditCounts: inteval.EditCounts{Substitutions: 1}, RefWords: 4, HypWords: 3}, AudioDurationMs: 8000, RTF: .5, ProviderLatencyMs: 70,
				TimestampDrift:     &inteval.TimestampDriftResult{ReferenceWords: 4, MatchedWords: 3, Coverage: .75, StartAbsP95Ms: 40},
				TimestampReference: &inteval.TimingReferenceProvenance{Status: "human_reviewed", SourceReference: "corpus-v2", Reviewer: "reviewer-5", ReviewMethod: "dual_review", ReviewedAt: "2026-11-22T12:00:00Z"},
			}},
		}},
	}

	metricsProto, metricsJSON, err := reportForPersistence(report, experiment.ReportRetentionMode_REPORT_RETENTION_MODE_METRICS_ONLY)
	require.NoError(t, err)
	payload := string(metricsJSON)
	require.NotContains(t, payload, secret)
	require.Equal(t, "metrics_only", metricsProto.GetRetentionMode())
	require.Equal(t, .25, metricsProto.GetPerStrategy()[0].GetWer())
	require.Equal(t, 120.0, metricsProto.GetPerStrategy()[0].GetFinalizationLatencyP50Ms())
	clip := metricsProto.GetPerStrategy()[0].GetPerClip()[0]
	require.Equal(t, .25, clip.GetWer())
	require.Equal(t, 40.0, clip.GetTimestampDrift().GetStartAbsP95Ms())
	require.Equal(t, "reviewer-5", clip.GetTimestampReference().GetReviewer())
	require.NotContains(t, payload, "reference\"")
	require.NotContains(t, payload, "hypothesis\"")
	require.NotContains(t, payload, "diagnostic")
	require.True(t, strings.Contains(payload, "engine-a"))

	fullProto, fullJSON, err := reportForPersistence(report, experiment.ReportRetentionMode_REPORT_RETENTION_MODE_UNSPECIFIED)
	require.NoError(t, err)
	require.Empty(t, fullProto.GetRetentionMode(), "legacy default remains unmarked")
	require.Contains(t, string(fullJSON), secret, "unspecified mode must preserve historical full-report retention")
}

func TestMetricsOnlyRetentionLifecycleScopesTranscriptRemovalToReportArtifacts(t *testing.T) {
	const (
		recipeText         = "RECIPE-CONFIG-SENTINEL"
		generatedReference = "GENERATED-REFERENCE-SENTINEL"
		reportText         = "REPORT-TRANSCRIPT-DIAGNOSTIC-SENTINEL"
	)
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db,
		apidb.SchemaProviderFunc(database.SystemSchema),
		apidb.SchemaProviderFunc(intexp.Schema),
	))
	clock := scheduletest.New(time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC))
	repo := intexp.NewSQLiteRepository(db, clock)
	blobs := mocks.NewFakeBlobStore()
	service := intexp.NewService(repo, blobs)

	recipe := &experiment.ExperimentRecipe{
		ReportRetentionMode: experiment.ReportRetentionMode_REPORT_RETENTION_MODE_METRICS_ONLY,
		RealizedReference:   generatedReference,
		LongForm:            &experiment.LongFormRecipe{Enabled: true, TagContains: recipeText},
		Augmentation:        &experiment.AugmentationRecipe{CompetingText: recipeText},
		Cells:               []*experiment.EvaluationCell{{EngineId: "engine-a", Strategy: "batch", Label: recipeText}},
	}
	recipeJSON, err := protojson.Marshal(recipe)
	require.NoError(t, err)
	report := inteval.EvalReport{QualityMeasured: true, LatencyMeasured: true, PerStrategy: []inteval.StrategyReport{{
		Strategy: "batch", Label: reportText, EngineID: "engine-a", ModelID: "model-a", PolicyProfile: "local", ReplayLane: "realtime",
		WER: .25, EditCounts: inteval.EditCounts{Substitutions: 1}, RefWords: 4, FinalizationLatencyP95Ms: 210,
		Safety: inteval.SafetyGateReport{Reasons: []string{reportText}},
		PerClip: []inteval.ClipResult{{ClipID: reportText, Reference: reportText, Hypothesis: reportText, NormalizedReference: reportText, NormalizedHypothesis: reportText,
			WER:                inteval.WERResult{EditCounts: inteval.EditCounts{Substitutions: 1}, RefWords: 4, HypWords: 3},
			EditOperations:     []inteval.EditOperation{{ReferenceToken: reportText, HypothesisToken: reportText}},
			CommitTimeline:     []inteval.CommitState{{Text: reportText}},
			TimestampReference: &inteval.TimingReferenceProvenance{Status: "human_reviewed", SourceReference: "corpus-v1", Reviewer: "reviewer-a", ReviewMethod: "double_check", ReviewedAt: "2026-11-22T12:00:00Z"},
		}},
	}}}
	reportProto, reportJSON, err := reportForPersistence(report, recipe.GetReportRetentionMode())
	require.NoError(t, err)
	runs, err := expreport.ExperimentRunsForMetricsOnlyReport(report, reportProto)
	require.NoError(t, err)
	completedRecipeJSON, err := protojson.Marshal(recipeForCompletedExperiment(recipe, true))
	require.NoError(t, err)

	started := make(chan struct{})
	release := make(chan struct{})
	mgr := intexp.NewManager(intexp.Config{Service: service, Clock: clock, Runner: func(_ context.Context, submitted intexp.Experiment, _ func(int, string)) (intexp.RunResult, error) {
		if submitted.Name != "metrics-only-success" {
			return intexp.RunResult{}, errors.New("expected evaluation failure")
		}
		close(started)
		<-release
		return intexp.RunResult{Report: reportJSON, RecipeJSON: completedRecipeJSON, Runs: runs}, nil
	}})
	require.NoError(t, mgr.Start(ctx))
	t.Cleanup(mgr.Close)

	submitted, err := mgr.Submit(ctx, intexp.SubmitSpec{Name: "metrics-only-success", RecipeJSON: recipeJSON})
	require.NoError(t, err)
	<-started
	queued, err := repo.GetExperiment(ctx, submitted.ID)
	require.NoError(t, err)
	require.Contains(t, string(queued.RecipeJSON), recipeText, "queued recipe retains configured inputs for execution")
	require.Contains(t, string(submitted.RecipeJSON), recipeText, "Submit returns the recipe as declared")
	close(release)

	done, err := mgr.Wait(ctx, submitted.ID)
	require.NoError(t, err)
	require.Equal(t, intexp.StatusSucceeded, done.Status)
	stored, err := repo.GetExperiment(ctx, submitted.ID)
	require.NoError(t, err)
	require.Equal(t, done.RecipeJSON, stored.RecipeJSON)
	require.NotContains(t, string(stored.RecipeJSON), generatedReference, "completed recipe drops generated long-form reference")
	require.Contains(t, string(stored.RecipeJSON), recipeText, "submitted config remains available")

	persistedReport, err := service.GetReport(ctx, stored)
	require.NoError(t, err)
	require.NotContains(t, string(persistedReport), reportText)
	require.Contains(t, string(persistedReport), "engine-a")
	require.Contains(t, string(persistedReport), "210")
	var decoded evalv1.EvalReport
	require.NoError(t, protojson.Unmarshal(persistedReport, &decoded))
	require.Equal(t, "metrics_only", decoded.GetRetentionMode())
	require.Equal(t, .25, decoded.GetPerStrategy()[0].GetWer())
	persistedRuns, err := service.ListRuns(ctx, stored.ID)
	require.NoError(t, err)
	require.Len(t, persistedRuns, 1)
	require.NotContains(t, string(persistedRuns[0].ConditionJSON), recipeText)
	require.NotContains(t, string(persistedRuns[0].ConditionJSON), reportText)
	require.Contains(t, string(persistedRuns[0].ConditionJSON), "engine-a")

	failedSubmit, err := mgr.Submit(ctx, intexp.SubmitSpec{Name: "metrics-only-failure", RecipeJSON: recipeJSON})
	require.NoError(t, err)
	failureDone, err := mgr.Wait(ctx, failedSubmit.ID)
	require.NoError(t, err)
	require.Equal(t, intexp.StatusFailed, failureDone.Status)
	require.Empty(t, failureDone.ResultRef)
	require.Contains(t, string(failureDone.RecipeJSON), recipeText, "failed run retains caller recipe; no report was created")
	_, err = service.GetReport(ctx, failureDone)
	require.Error(t, err)
}
