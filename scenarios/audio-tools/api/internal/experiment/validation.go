package experiment

import (
	"errors"
	"fmt"
	"math"
	"strings"

	exprecipe "audio-tools/internal/experiment/recipe"

	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
)

// ErrSpeakerProfileRequired names the policy failure distinct from malformed
// input: callers render it as a failed precondition with enrollment guidance.
var ErrSpeakerProfileRequired = errors.New("speaker experiments require a target_profile_id; enroll a voice profile with `audio-tools stt speaker-enroll --file <clip> --activate true` and list profile ids with `audio-tools stt speaker-status`")

const (
	minSNRDB = -80.0
	maxSNRDB = 80.0
)

// ValidateRecipe rejects experiment inputs that would otherwise queue only to
// fail or skip every condition. It belongs to the experiment domain so every
// transport can enforce the same policy before scheduling work.
func ValidateRecipe(recipe *experimentv1.ExperimentRecipe) error {
	if recipe.GetRealtimeRepeats() < 0 {
		return errors.New("realtime_repeats must be non-negative")
	}
	if recipe.GetChunkMs() < 0 {
		return errors.New("chunk_ms must be non-negative")
	}
	if recipe.GetSeed() < 0 {
		return errors.New("seed must be non-negative")
	}
	if recipe.GetDroppedSpanThresholdWords() < 0 {
		return errors.New("dropped_span_threshold_words must be non-negative")
	}
	if recipe.GetLatencyTailSeconds() < 0 {
		return errors.New("latency_tail_seconds must be non-negative")
	}
	for i, strategy := range recipe.GetStrategies() {
		switch strategy.GetKind() {
		case "", "batch", "vad_segment", "overlap_agree":
		default:
			return fmt.Errorf("strategies[%d].kind %q is not supported", i, strategy.GetKind())
		}
		if strategy.GetOverlapWindowMs() < 0 {
			return fmt.Errorf("strategies[%d].overlap_window_ms must be non-negative", i)
		}
		if strategy.GetOverlapCommitRuns() < 0 {
			return fmt.Errorf("strategies[%d].overlap_commit_runs must be non-negative", i)
		}
		if strategy.GetVadSilenceMs() < 0 {
			return fmt.Errorf("strategies[%d].vad_silence_ms must be non-negative", i)
		}
		if strategy.GetOverlapMaxWindowMs() < 0 {
			return fmt.Errorf("strategies[%d].overlap_max_window_ms must be non-negative", i)
		}
	}
	for i, cell := range recipe.GetCells() {
		if strings.TrimSpace(cell.GetEngineId()) == "" {
			return fmt.Errorf("cells[%d].engine_id is required", i)
		}
		switch cell.GetStrategy() {
		case "batch", "buffered", "buffered_fallback", "vad_segment", "vad", "overlap_agree", "overlap", "passthrough":
		default:
			return fmt.Errorf("cells[%d].strategy %q is not supported", i, cell.GetStrategy())
		}
		switch cell.GetReplayLane() {
		case experimentv1.ReplayLane_REPLAY_LANE_DETERMINISTIC, experimentv1.ReplayLane_REPLAY_LANE_REALTIME, experimentv1.ReplayLane_REPLAY_LANE_PRODUCT_PATH:
		default:
			return fmt.Errorf("cells[%d].replay_lane is required", i)
		}
		if cell.GetRepeatCount() < 1 {
			return fmt.Errorf("cells[%d].repeat_count must be at least 1", i)
		}
		if cell.GetReplayLane() == experimentv1.ReplayLane_REPLAY_LANE_REALTIME && recipe.GetLatencyTailSeconds() > 0 {
			return fmt.Errorf("cells[%d] realtime evidence cannot use latency_tail_seconds", i)
		}
	}
	if longForm := recipe.GetLongForm(); longForm != nil {
		if longForm.GetTargetDurationSeconds() < 0 {
			return errors.New("long_form.target_duration_seconds must be non-negative")
		}
		if longForm.GetGapMs() < 0 {
			return errors.New("long_form.gap_ms must be non-negative")
		}
		for i, duration := range longForm.GetSweepDurationsSeconds() {
			if duration < 0 {
				return fmt.Errorf("long_form.sweep_durations_seconds[%d] must be non-negative", i)
			}
		}
	}
	if augmentation := recipe.GetAugmentation(); augmentation != nil {
		for i, snr := range augmentation.GetSnrDb() {
			if math.IsNaN(snr) || math.IsInf(snr, 0) {
				return fmt.Errorf("augmentation.snr_db[%d] must be a finite number", i)
			}
			if snr < minSNRDB || snr > maxSNRDB {
				return fmt.Errorf("augmentation.snr_db[%d]=%g is out of range (%g..%g dB)", i, snr, minSNRDB, maxSNRDB)
			}
		}
		for i, noiseType := range augmentation.GetNoiseTypes() {
			if strings.TrimSpace(noiseType) != "" && !exprecipe.IsKnownNoiseType(noiseType) {
				return fmt.Errorf("augmentation.noise_types[%d] %q is not a supported noise bed; use one of: %s", i, noiseType, strings.Join(exprecipe.KnownNoiseTypes(), ", "))
			}
		}
	}
	speaker := recipe.GetSpeaker()
	if speaker != nil && (speaker.GetExtractionEnabled() || speaker.GetVerificationEnabled() || speaker.GetAblationEnabled()) && speaker.GetTargetProfileId() == "" {
		return ErrSpeakerProfileRequired
	}
	return nil
}
