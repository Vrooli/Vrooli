//go:build audio_lab

package report

import (
	"context"
	"fmt"

	"audio-tools/internal/audioformat"
	intcorpus "audio-tools/internal/corpus"
	inteval "audio-tools/internal/eval"
	exprecipe "audio-tools/internal/experiment/recipe"
	sttpipeline "audio-tools/internal/stt/pipeline"
	inttts "audio-tools/internal/tts"

	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
)

func RunExperimentReport(ctx context.Context, evalDeps inteval.RunnerDeps, corpusSvc *intcorpus.Service, ttsSvc *inttts.Service, audioEngine *audioformat.Engine, recipe *experimentv1.ExperimentRecipe, emit func(int, string)) (inteval.EvalReport, map[string]any, error) {
	modes := experimentModes(recipe)
	if len(recipe.GetCells()) > 0 && (modes.augmentationEnabled || modes.speakerEnabled) {
		return inteval.EvalReport{}, nil, fmt.Errorf("provider-neutral evaluation cells with augmentation or speaker ablation are not executable yet; do not relabel the legacy Whisper runner as provider-neutral")
	}
	if !modes.longFormEnabled && !modes.augmentationEnabled && !modes.speakerEnabled {
		return runDefaultExperimentReport(ctx, evalDeps, recipe, emit)
	}
	if corpusSvc == nil {
		return inteval.EvalReport{}, nil, fmt.Errorf("experiment recipe materialization requires corpus service")
	}
	materialized, err := materializeExperimentInputs(ctx, corpusSvc, ttsSvc, audioEngine, recipe, modes.longForm, modes.longFormEnabled, modes.augmentation, modes.augmentationEnabled, modes.speaker, modes.speakerEnabled, evalDeps.SpeakerResource)
	if err != nil {
		return inteval.EvalReport{}, nil, err
	}
	return runMaterializedExperimentReport(ctx, evalDeps, recipe, emit, modes.longForm, modes.speakerEnabled, materialized)
}

type experimentReportModes struct {
	longForm            *experimentv1.LongFormRecipe
	augmentation        *experimentv1.AugmentationRecipe
	speaker             *experimentv1.SpeakerExperimentRecipe
	longFormEnabled     bool
	augmentationEnabled bool
	speakerEnabled      bool
}

func experimentModes(recipe *experimentv1.ExperimentRecipe) experimentReportModes {
	longForm := recipe.GetLongForm()
	augmentation := recipe.GetAugmentation()
	speaker := recipe.GetSpeaker()
	return experimentReportModes{
		longForm:            longForm,
		augmentation:        augmentation,
		speaker:             speaker,
		longFormEnabled:     longForm != nil && (longForm.GetEnabled() || longForm.GetTargetDurationSeconds() > 0 || longForm.GetGapMs() > 0 || longForm.GetTagContains() != "" || len(longForm.GetSweepDurationsSeconds()) > 0),
		augmentationEnabled: augmentation != nil && (len(augmentation.GetNoiseTypes()) > 0 || len(augmentation.GetCompetingVoiceIds()) > 0),
		speakerEnabled:      speaker != nil && speakerConfigured(speaker),
	}
}

func runDefaultExperimentReport(ctx context.Context, evalDeps inteval.RunnerDeps, recipe *experimentv1.ExperimentRecipe, emit func(int, string)) (inteval.EvalReport, map[string]any, error) {
	progressUnits := evalWorkUnits(len(recipe.GetClipIds()), recipe.GetStrategies(), recipe.GetRealtimeRepeats())
	if len(recipe.GetCells()) > 0 {
		progressUnits = cellRecipeWorkUnits(len(recipe.GetClipIds()), recipe.GetCells())
	}
	progress := newExperimentProgress(progressUnits, emit)
	opts := inteval.EvalOptions{DroppedSpanThresholdWords: int(safetyThreshold(recipe)), LatencyTailSeconds: int(recipe.GetLatencyTailSeconds())}
	if progress != nil {
		opts = progress.options("condition default", opts)
	}
	if len(recipe.GetCells()) > 0 {
		report, err := inteval.RunReportCellsWithOptions(ctx, evalDeps, recipe.GetClipIds(), recipe.GetCells(), recipe.GetChunkMs(), opts)
		return report, map[string]any{"phase": "default", "long_form": false, "provider_cells": true}, err
	}
	report, err := inteval.RunReportWithOptions(ctx, evalDeps, recipe.GetClipIds(), recipe.GetStrategies(), recipe.GetRealtimeRepeats(), recipe.GetChunkMs(), opts)
	return report, map[string]any{"phase": "default", "long_form": false}, err
}

type experimentMaterialization struct {
	clips               []exprecipe.Clip
	evalClips           []inteval.Clip
	realized            map[string]any
	augGroups           []exprecipe.ConditionGroup
	speakerConditions   []speakerEvalCondition
	realtimeConcurrency int
}

func materializeExperimentInputs(ctx context.Context, corpusSvc *intcorpus.Service, ttsSvc *inttts.Service, audioEngine *audioformat.Engine, recipe *experimentv1.ExperimentRecipe, longForm *experimentv1.LongFormRecipe, longFormEnabled bool, augmentation *experimentv1.AugmentationRecipe, augmentationEnabled bool, speaker *experimentv1.SpeakerExperimentRecipe, speakerEnabled bool, speakerResource *sttpipeline.SpeakerClient) (*experimentMaterialization, error) {
	tagContains := ""
	if longForm != nil {
		tagContains = longForm.GetTagContains()
	}
	clips, err := loadRecipeClips(ctx, corpusSvc, recipe.GetClipIds(), tagContains)
	if err != nil {
		return nil, err
	}
	evalClips, realized, realtimeConcurrency, err := materializeBaseInputs(clips, recipe, longForm, longFormEnabled)
	if err != nil {
		return nil, err
	}
	evalClips, augGroups, err := applyExperimentAugmentation(ctx, evalClips, recipe, augmentation, augmentationEnabled, realized, ttsSvc, audioEngine)
	if err != nil {
		return nil, err
	}
	conditions := applySpeakerResourceAvailability(ctx, buildSpeakerConditions(speaker), speakerResource)
	writeSpeakerRealization(recipe, speaker, speakerEnabled, conditions, realized)
	return &experimentMaterialization{clips: clips, evalClips: evalClips, realized: realized, augGroups: augGroups, speakerConditions: conditions, realtimeConcurrency: realtimeConcurrency}, nil
}

func materializeBaseInputs(clips []exprecipe.Clip, recipe *experimentv1.ExperimentRecipe, longForm *experimentv1.LongFormRecipe, enabled bool) ([]inteval.Clip, map[string]any, int, error) {
	evalClips := make([]inteval.Clip, 0, len(clips))
	realized := map[string]any{"phase": "materialized", "long_form": false}
	if !enabled {
		for _, c := range clips {
			evalClips = append(evalClips, inteval.Clip{ID: c.ID, PCM: c.PCM, SampleRate: c.SampleRate, Reference: c.Reference, Format: c.Format, TimedReference: c.TimedReference, TimingProvenance: c.TimingProvenance})
		}
		realized["clip_count"] = len(evalClips)
		return evalClips, realized, 0, nil
	}
	if longForm.GapMs <= 0 {
		longForm.GapMs = exprecipe.DefaultGapMs
	}
	if recipe.Seed == 0 {
		recipe.Seed = 1
	}
	// A non-empty sweep creates one length-bucketed input per listed duration.
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
			seed += int64(durSec)
		}
		synthetic, longRealized, err := exprecipe.Build(exprecipe.Spec{Seed: seed, TargetDurationSeconds: int(durSec), GapMs: int(longForm.GetGapMs())}, clips)
		if err != nil {
			return nil, nil, 0, err
		}
		if isSweep {
			synthetic.ID = fmt.Sprintf("long-form-%ds", durSec)
		}
		recipe.RealizedClipIds = append(recipe.RealizedClipIds, longRealized.ClipIDs...)
		recipe.RealizedReference = longRealized.Reference
		totalDurationMs += longRealized.DurationMs
		evalClips = append(evalClips, synthetic)
		sweepRealized = append(sweepRealized, map[string]any{"input_id": synthetic.ID, "target_duration_sec": durSec, "realized_duration_ms": longRealized.DurationMs, "clip_count": len(longRealized.ClipIDs)})
	}
	recipe.RealizedDurationMs = totalDurationMs
	realized = map[string]any{"phase": "long_form", "long_form": true, "gap_ms": longForm.GetGapMs(), "sweep": isSweep, "input_count": len(durations), "realized_duration_ms": totalDurationMs}
	if isSweep {
		realized["sweep_durations_sec"] = sweep
		realized["sweep_inputs"] = sweepRealized
		realized["realtime_concurrency"] = 1
		return evalClips, realized, 1, nil
	}
	realized["target_duration_sec"] = longForm.GetTargetDurationSeconds()
	return evalClips, realized, 0, nil
}

func applyExperimentAugmentation(ctx context.Context, evalClips []inteval.Clip, recipe *experimentv1.ExperimentRecipe, augmentation *experimentv1.AugmentationRecipe, enabled bool, realized map[string]any, ttsSvc *inttts.Service, audioEngine *audioformat.Engine) ([]inteval.Clip, []exprecipe.ConditionGroup, error) {
	if !enabled {
		return evalClips, nil, nil
	}
	augmented, conditions, err := exprecipe.ApplyAugmentation(ctx, evalClips, exprecipe.AugmentationSpec{Seed: recipe.GetSeed(), NoiseTypes: augmentation.GetNoiseTypes(), SNRDB: augmentation.GetSnrDb(), CompetingVoices: augmentation.GetCompetingVoiceIds(), CompetingText: augmentation.GetCompetingText(), SynthesizeVoice: synthesizeCanonicalVoice(ttsSvc, audioEngine)})
	if err != nil {
		return nil, nil, err
	}
	evalClips = augmented
	groups := exprecipe.GroupClipsByAugCondition(evalClips)
	recipe.RealizedAugmentationConditions = recipe.RealizedAugmentationConditions[:0]
	for _, c := range conditions {
		recipe.RealizedAugmentationConditions = append(recipe.RealizedAugmentationConditions, &experimentv1.AugmentationCondition{Id: c.ID, Kind: c.Kind, Source: c.Source, SnrDb: c.SNRDB, Skipped: c.Skipped, Note: c.Note})
	}
	realized["augmentation"] = map[string]any{"noise_types": augmentation.GetNoiseTypes(), "snr_db": augmentation.GetSnrDb(), "competing_voices": augmentation.GetCompetingVoiceIds(), "condition_count": len(conditions), "evaluated_inputs": len(evalClips), "row_count": len(groups), "skipped_condition": countSkippedConditions(conditions)}
	return evalClips, groups, nil
}

func writeSpeakerRealization(recipe *experimentv1.ExperimentRecipe, speaker *experimentv1.SpeakerExperimentRecipe, enabled bool, conditions []speakerEvalCondition, realized map[string]any) {
	recipe.RealizedSpeakerConditions = recipe.RealizedSpeakerConditions[:0]
	for _, cond := range conditions {
		recipe.RealizedSpeakerConditions = append(recipe.RealizedSpeakerConditions, &experimentv1.SpeakerCondition{Id: cond.ID, ExtractionEnabled: cond.ExtractionEnabled, VerificationEnabled: cond.VerificationEnabled, VerificationMode: cond.VerificationMode, Skipped: cond.Skipped, Note: cond.Note})
	}
	realized["speaker"] = map[string]any{"enabled": enabled, "target_profile": speaker.GetTargetProfileId(), "ablation_enabled": speaker.GetAblationEnabled(), "condition_count": len(conditions), "dropped_span_threshold_words": safetyThreshold(recipe)}
}

func runMaterializedExperimentReport(ctx context.Context, evalDeps inteval.RunnerDeps, recipe *experimentv1.ExperimentRecipe, emit func(int, string), longForm *experimentv1.LongFormRecipe, speakerEnabled bool, materialized *experimentMaterialization) (inteval.EvalReport, map[string]any, error) {
	if len(recipe.GetCells()) > 0 {
		if len(materialized.augGroups) != 0 || speakerEnabled {
			return inteval.EvalReport{}, nil, fmt.Errorf("provider-neutral evaluation cells with augmentation or speaker ablation are not executable yet; do not relabel the legacy Whisper runner as provider-neutral")
		}
		progress := newExperimentProgress(cellExperimentWorkUnits(materialized.evalClips, recipe.GetCells()), emit)
		opts := inteval.EvalOptions{LatencyTailSeconds: int(recipe.GetLatencyTailSeconds()), DroppedSpanThresholdWords: int(safetyThreshold(recipe)), RealtimeConcurrency: materialized.realtimeConcurrency}
		if progress != nil {
			opts = progress.options("provider cell", opts)
		}
		report, err := inteval.RunReportForCells(ctx, evalDeps, materialized.evalClips, recipe.GetCells(), recipe.GetChunkMs(), opts)
		if err == nil {
			if warning, ok := longFormSourceDurationWarning(longForm, materialized.clips); ok {
				report.Warnings = appendUniqueReportWarnings(report.Warnings, warning)
			}
		}
		materialized.realized["provider_cells"] = true
		return report, materialized.realized, err
	}
	progress := newExperimentProgress(experimentWorkUnits(materialized.evalClips, materialized.augGroups, recipe.GetStrategies(), recipe.GetRealtimeRepeats(), materialized.speakerConditions, speakerConfigured(recipe.GetSpeaker())), emit)
	report, err := runAugmentationSpeakerConditionReports(ctx, evalDeps, materialized.evalClips, materialized.augGroups, recipe.GetStrategies(), recipe.GetRealtimeRepeats(), recipe.GetChunkMs(), recipe.GetLatencyTailSeconds(), safetyThreshold(recipe), materialized.realtimeConcurrency, materialized.speakerConditions, speakerConfigured(recipe.GetSpeaker()), progress)
	if err == nil {
		if warning, ok := longFormSourceDurationWarning(longForm, materialized.clips); ok {
			report.Warnings = appendUniqueReportWarnings(report.Warnings, warning)
		}
		// Attribution runs once, after all condition rows have been assembled.
		inteval.AttributeIngressByAblation(&report)
	}
	return report, materialized.realized, err
}

func safetyThreshold(recipe *experimentv1.ExperimentRecipe) int32 {
	if recipe.GetDroppedSpanThresholdWords() > 0 {
		return recipe.GetDroppedSpanThresholdWords()
	}
	return int32(inteval.DefaultDroppedSpanThresholdWords)
}
