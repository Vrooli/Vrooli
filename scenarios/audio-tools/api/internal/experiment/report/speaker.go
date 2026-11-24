//go:build audio_lab

package report

import (
	"context"
	"fmt"

	"audio-tools/internal/ai/sttchain"
	inteval "audio-tools/internal/eval"
	"audio-tools/internal/protomap"
	"audio-tools/internal/stt/egress"
	"audio-tools/internal/stt/ingress"
	sttpipeline "audio-tools/internal/stt/pipeline"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
)

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
	note := speakerResourceReadinessNote(ctx, client)
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

func speakerResourceReadinessNote(ctx context.Context, client *sttpipeline.SpeakerClient) string {
	if client == nil {
		return "speaker condition skipped: speaker resource is not configured"
	}
	ready, err := client.Ready(ctx)
	if err != nil {
		return "speaker condition skipped: speaker resource unreachable: " + err.Error()
	}
	if (ready.Status != "ready" && ready.Status != "ok") || !ready.ModelLoaded || !ready.ProfileStoreOK || !ready.TempDirOK {
		return fmt.Sprintf("speaker condition skipped: speaker resource not ready (status=%s model_loaded=%t profile_store_ok=%t temp_dir_ok=%t)", ready.Status, ready.ModelLoaded, ready.ProfileStoreOK, ready.TempDirOK)
	}
	return ""
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
		deps := speakerConditionRunnerDeps(evalDeps, cond)
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

func speakerConditionRunnerDeps(evalDeps inteval.RunnerDeps, cond speakerEvalCondition) inteval.RunnerDeps {
	deps := evalDeps
	if cond.ExtractionEnabled || cond.VerificationEnabled {
		cfg := cond.Config
		deps.SpeakerConfig = &cfg
		deps.SpeakerExtractionEnabled = cond.ExtractionEnabled
		deps.SpeakerVerificationEnabled = cond.VerificationEnabled
		deps.NewSpeakerIsolation = func() egress.SpeakerIsolation {
			if !cond.VerificationEnabled || deps.NewSpeakerIsolationForConfig == nil {
				return nil
			}
			return deps.NewSpeakerIsolationForConfig(cfg, deps.SpeakerResource)
		}
		deps.NewSpeakerExtraction = func() ingress.TargetExtractor {
			if !cond.ExtractionEnabled || deps.NewSpeakerExtractionForConfig == nil {
				return nil
			}
			return deps.NewSpeakerExtractionForConfig(cfg, deps.SpeakerResource)
		}
		return deps
	}
	deps.SpeakerConfig = nil
	deps.SpeakerExtractionEnabled = false
	deps.SpeakerVerificationEnabled = false
	deps.NewSpeakerIsolation = func() egress.SpeakerIsolation { return nil }
	deps.NewSpeakerExtraction = func() ingress.TargetExtractor { return nil }
	return deps
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
