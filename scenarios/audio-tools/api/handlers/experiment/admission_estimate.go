//go:build audio_lab

package experiment

import (
	"context"
	"fmt"

	"audio-tools/internal/protoint"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
)

func (h *connectHandler) estimateExperimentSeconds(ctx context.Context, recipe *experimentv1.ExperimentRecipe, clientOverride int32) (int32, error) {
	if recipe == nil {
		if clientOverride > experimentMaxEstimatedSeconds {
			return 0, experimentDurationLimitError(int64(clientOverride))
		}
		return clientOverride, nil
	}
	durationSeconds, err := h.estimateRecipeSeconds(ctx, recipe)
	if err != nil {
		return 0, err
	}
	if durationSeconds <= 0 {
		if clientOverride > experimentMaxEstimatedSeconds {
			return 0, experimentDurationLimitError(int64(clientOverride))
		}
		return clientOverride, nil
	}
	total := estimateRecipeWorkSeconds(recipe, durationSeconds)
	// The recipe-derived duration is authoritative. A caller may reserve more
	// time for a known external setup cost, but it may not shrink the derived
	// budget and turn a long run into a misleadingly short one.
	if clientOverride > 0 && int64(clientOverride) > total {
		total = int64(clientOverride)
	}
	if total > int64(experimentMaxEstimatedSeconds) {
		return 0, experimentDurationLimitError(total)
	}
	return protoint.FromInt64(total), nil
}

func (h *connectHandler) estimateRecipeSeconds(ctx context.Context, recipe *experimentv1.ExperimentRecipe) (int32, error) {
	durationSeconds := estimatedRecipeDurationSeconds(recipe)
	if durationSeconds > 0 || h.deps.EstimateClipSeconds == nil {
		return durationSeconds, nil
	}
	return h.deps.EstimateClipSeconds(ctx, recipe.GetClipIds())
}

func estimateRecipeWorkSeconds(recipe *experimentv1.ExperimentRecipe, durationSeconds int32) int64 {
	conditions := estimatedConditionCount(recipe)
	// Cells are independent provider/strategy/lane work units. Their
	// repeat_count must contribute to admission cost; counting only the number
	// of cells would let a repeated real-time recipe understate its duration.
	var total int64
	if cells := recipe.GetCells(); len(cells) > 0 {
		for _, cell := range cells {
			total += estimateCellWorkSeconds(cell, durationSeconds, conditions)
		}
	} else {
		total = estimateLegacyRecipeWorkSeconds(recipe, durationSeconds, conditions)
	}
	if total < 1 {
		total = 1
	}
	return total
}

func estimateCellWorkSeconds(cell *experimentv1.EvaluationCell, durationSeconds int32, conditions int) int64 {
	repeats := cell.GetRepeatCount()
	if repeats < 1 {
		repeats = 1
	}
	if cell.GetReplayLane() != experimentv1.ReplayLane_REPLAY_LANE_REALTIME {
		return int64(durationSeconds) * int64(repeats) * int64(conditions)
	}
	// Realtime cells reserve a separate unpaced quality pass before latency repeats.
	qualitySeconds := int64(durationSeconds) / 4
	if qualitySeconds < 1 {
		qualitySeconds = 1
	}
	return (int64(durationSeconds)*int64(repeats) + qualitySeconds) * int64(conditions)
}

func estimateLegacyRecipeWorkSeconds(recipe *experimentv1.ExperimentRecipe, durationSeconds int32, conditions int) int64 {
	strategies := len(recipe.GetStrategies())
	if strategies == 0 {
		strategies = 3
	}
	repeats := int(recipe.GetRealtimeRepeats()) + 1
	if repeats < 1 {
		repeats = 1
	}
	return int64(durationSeconds) * int64(strategies*repeats) * int64(conditions)
}

func experimentDurationLimitError(seconds int64) error {
	return fmt.Errorf("estimated experiment duration %ds exceeds the %ds qualification ceiling; split the matrix into explicit provider cells", seconds, experimentMaxEstimatedSeconds)
}

func estimatedRecipeDurationSeconds(recipe *experimentv1.ExperimentRecipe) int32 {
	if recipe.GetRealizedDurationMs() > 0 {
		return protoint.FromInt64((recipe.GetRealizedDurationMs() + 999) / 1000)
	}
	longForm := recipe.GetLongForm()
	if longForm == nil {
		return 0
	}
	if sweep := longForm.GetSweepDurationsSeconds(); len(sweep) > 0 {
		var total int32
		for _, seconds := range sweep {
			if seconds > 0 {
				total += seconds
			}
		}
		return total
	}
	if longForm.GetEnabled() && longForm.GetTargetDurationSeconds() > 0 {
		return longForm.GetTargetDurationSeconds()
	}
	return 0
}

func estimatedConditionCount(recipe *experimentv1.ExperimentRecipe) int {
	count := 1
	if realized := len(recipe.GetRealizedAugmentationConditions()); realized > 0 {
		count *= realized
	} else {
		count *= augmentationConditionCount(recipe.GetAugmentation())
	}
	if realized := len(recipe.GetRealizedSpeakerConditions()); realized > 0 {
		count *= realized
	} else {
		count *= speakerConditionCount(recipe.GetSpeaker())
	}
	if count < 1 {
		return 1
	}
	return count
}

func augmentationConditionCount(augmentation *experimentv1.AugmentationRecipe) int {
	if augmentation == nil {
		return 1
	}
	noise := len(augmentation.GetNoiseTypes())
	snr := len(augmentation.GetSnrDb())
	voices := len(augmentation.GetCompetingVoiceIds())
	count := 1
	if noise > 0 {
		count *= noise
	}
	if snr > 0 {
		count *= snr
	}
	if voices > 0 {
		count *= voices
	}
	return count
}

func speakerConditionCount(speaker *experimentv1.SpeakerExperimentRecipe) int {
	if speaker != nil && speaker.GetAblationEnabled() {
		return 4
	}
	return 1
}
