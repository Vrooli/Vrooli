//go:build audio_lab

package report

import (
	"context"
	"fmt"
	"strings"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/audioformat"
	intcorpus "audio-tools/internal/corpus"
	inteval "audio-tools/internal/eval"
	exprecipe "audio-tools/internal/experiment/recipe"
	inttts "audio-tools/internal/tts"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
)

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
