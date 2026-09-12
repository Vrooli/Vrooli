package report

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/audioformat"
	intcorpus "audio-tools/internal/corpus"
	inteval "audio-tools/internal/eval"
	intexp "audio-tools/internal/experiment"
	exprecipe "audio-tools/internal/experiment/recipe"
	"audio-tools/internal/protomap"
	"audio-tools/internal/stt/egress"
	"audio-tools/internal/stt/ingress"
	sttpipeline "audio-tools/internal/stt/pipeline"
	inttts "audio-tools/internal/tts"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

func RunExperimentReport(ctx context.Context, evalDeps inteval.RunnerDeps, corpusSvc *intcorpus.Service, ttsSvc *inttts.Service, audioEngine *audioformat.Engine, recipe *experimentv1.ExperimentRecipe, emit func(int, string)) (inteval.EvalReport, map[string]any, error) {
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
			report, err := inteval.RunReportCellsWithOptions(ctx, evalDeps, recipe.GetClipIds(), recipe.GetCells(), recipe.GetChunkMs(), opts)
			return report, map[string]any{"phase": "default", "long_form": false, "provider_cells": true}, err
		}
		report, err := inteval.RunReportWithOptions(ctx, evalDeps, recipe.GetClipIds(), recipe.GetStrategies(), recipe.GetRealtimeRepeats(), recipe.GetChunkMs(), opts)
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
		report, err := inteval.RunReportForCells(ctx, evalDeps, evalClips, recipe.GetCells(), recipe.GetChunkMs(), opts)
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

// EstimateClipSeconds exposes the report queue-budget estimator to composition.
func EstimateClipSeconds(corpusSvc *intcorpus.Service) func(context.Context, []string) (int32, error) {
	return estimateClipSeconds(corpusSvc)
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

// ExperimentRunsForReport produces persistence rows from a realized report.
func ExperimentRunsForReport(report inteval.EvalReport, reportProto *evalv1.EvalReport, realized map[string]any) ([]intexp.Run, error) {
	return experimentRunsForReport(report, reportProto, realized)
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

func runSpeakerConditionReportsWithOptions(ctx context.Context, evalDeps inteval.RunnerDeps, clips []inteval.Clip, strategies []*evalv1.EvalStrategy, realtimeRepeats, chunkMs int32, opts inteval.EvalOptions, conditions []speakerEvalCondition, progressScope string, progress *experimentProgress) (inteval.EvalReport, error) {
	if len(conditions) == 0 {
		if progress != nil {
			opts = progress.options(conditionScope(progressScope, "condition default"), opts)
		}
		return inteval.RunReportForClipsWithOptions(ctx, evalDeps, clips, strategies, realtimeRepeats, chunkMs, opts)
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
			deps.NewSpeakerIsolation = func() egress.SpeakerIsolation {
				if !cond.VerificationEnabled {
					return nil
				}
				if deps.NewSpeakerIsolationForConfig == nil {
					return nil
				}
				return deps.NewSpeakerIsolationForConfig(cfg, deps.SpeakerResource)
			}
			deps.NewSpeakerExtraction = func() ingress.TargetExtractor {
				if !cond.ExtractionEnabled {
					return nil
				}
				if deps.NewSpeakerExtractionForConfig == nil {
					return nil
				}
				return deps.NewSpeakerExtractionForConfig(cfg, deps.SpeakerResource)
			}
		} else {
			deps.SpeakerConfig = nil
			deps.SpeakerExtractionEnabled = false
			deps.SpeakerVerificationEnabled = false
			deps.NewSpeakerIsolation = func() egress.SpeakerIsolation { return nil }
			deps.NewSpeakerExtraction = func() ingress.TargetExtractor { return nil }
		}
		report, err := inteval.RunReportForClipsWithOptions(ctx, deps, clips, strategies, realtimeRepeats, chunkMs, condOpts)
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

func runAugmentationSpeakerConditionReports(ctx context.Context, evalDeps inteval.RunnerDeps, clips []inteval.Clip, augGroups []exprecipe.ConditionGroup, strategies []*evalv1.EvalStrategy, realtimeRepeats, chunkMs, latencyTailSeconds, droppedSpanThresholdWords int32, realtimeConcurrency int, speakerConditions []speakerEvalCondition, speakerEnabled bool, progress *experimentProgress) (inteval.EvalReport, error) {
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

func runSpeakerConditionReportsForAugmentation(ctx context.Context, evalDeps inteval.RunnerDeps, clips []inteval.Clip, strategies []*evalv1.EvalStrategy, realtimeRepeats, chunkMs, latencyTailSeconds, droppedSpanThresholdWords int32, realtimeConcurrency int, speakerConditions []speakerEvalCondition, speakerEnabled bool, augmentationID string, progress *experimentProgress) (inteval.EvalReport, error) {
	if !speakerEnabled {
		opts := inteval.EvalOptions{
			DroppedSpanThresholdWords: int(droppedSpanThresholdWords),
			LatencyTailSeconds:        int(latencyTailSeconds),
			RealtimeConcurrency:       realtimeConcurrency,
		}
		if progress != nil {
			opts = progress.options("augmentation "+augmentationID, opts)
		}
		return inteval.RunReportForClipsWithOptions(ctx, evalDeps, clips, strategies, realtimeRepeats, chunkMs, opts)
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
