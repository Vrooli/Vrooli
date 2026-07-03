package experiment

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
)

func (h *handlers) start(ctx cliapp.RunContext) error {
	recipe, err := loadBaseRecipe(ctx)
	if err != nil {
		return err
	}
	req := &experimentv1.StartExperimentRequest{
		Name:   ctx.Flag("name"),
		Recipe: recipe,
	}
	// Individual flags override any field the base recipe (--recipe-json /
	// --recipe-file) supplied; only override when the flag was passed.
	if v := strings.TrimSpace(ctx.Flag("clip-ids")); v != "" {
		req.Recipe.ClipIds = splitCSV(v)
	}
	if v := strings.TrimSpace(ctx.Flag("strategies")); v != "" {
		req.Recipe.Strategies = strategiesFromFlag(v)
	}
	if v := strings.TrimSpace(ctx.Flag("realtime-repeats")); v != "" {
		n, err := parseIntFlag("realtime-repeats", v)
		if err != nil {
			return err
		}
		req.Recipe.RealtimeRepeats = int32(n)
	}
	if v := strings.TrimSpace(ctx.Flag("latency-tail-seconds")); v != "" {
		n, err := parseIntFlag("latency-tail-seconds", v)
		if err != nil {
			return err
		}
		req.Recipe.LatencyTailSeconds = int32(n)
	}
	if v := strings.TrimSpace(ctx.Flag("chunk-ms")); v != "" {
		n, err := parseIntFlag("chunk-ms", v)
		if err != nil {
			return err
		}
		req.Recipe.ChunkMs = int32(n)
	}
	if v := strings.TrimSpace(ctx.Flag("dropped-span-threshold")); v != "" {
		n, err := parseIntFlag("dropped-span-threshold", v)
		if err != nil {
			return err
		}
		req.Recipe.DroppedSpanThresholdWords = int32(n)
	}
	if v := strings.TrimSpace(ctx.Flag("overlap-max-window-ms")); v != "" {
		n, err := parseNonNegativeIntFlag("overlap-max-window-ms", v)
		if err != nil {
			return err
		}
		ensureDefaultStrategies(req.Recipe)
		for _, s := range req.Recipe.Strategies {
			if strategyUsesOverlap(s.GetKind()) {
				s.OverlapMaxWindowMs = int32(n)
			}
		}
	}
	if v := strings.TrimSpace(ctx.Flag("overlap-max-stall-rejects")); v != "" {
		n, err := parseNonNegativeIntFlag("overlap-max-stall-rejects", v)
		if err != nil {
			return err
		}
		ensureDefaultStrategies(req.Recipe)
		for _, s := range req.Recipe.Strategies {
			if strategyUsesOverlap(s.GetKind()) {
				s.OverlapMaxStallRejects = int32(n)
			}
		}
	}
	if v := strings.TrimSpace(ctx.Flag("overlap-window-ms")); v != "" {
		n, err := parseNonNegativeIntFlag("overlap-window-ms", v)
		if err != nil {
			return err
		}
		ensureDefaultStrategies(req.Recipe)
		for _, s := range req.Recipe.Strategies {
			if strategyUsesOverlap(s.GetKind()) {
				s.OverlapWindowMs = int32(n)
			}
		}
	}
	if v := strings.TrimSpace(ctx.Flag("overlap-commit-runs")); v != "" {
		n, err := parseNonNegativeIntFlag("overlap-commit-runs", v)
		if err != nil {
			return err
		}
		ensureDefaultStrategies(req.Recipe)
		for _, s := range req.Recipe.Strategies {
			if strategyUsesOverlap(s.GetKind()) {
				s.OverlapCommitRuns = int32(n)
			}
		}
	}
	if v := strings.TrimSpace(ctx.Flag("vad-silence-ms")); v != "" {
		n, err := parseNonNegativeIntFlag("vad-silence-ms", v)
		if err != nil {
			return err
		}
		ensureDefaultStrategies(req.Recipe)
		for _, s := range req.Recipe.Strategies {
			if strategyUsesVADSilence(s.GetKind()) {
				s.VadSilenceMs = int32(n)
			}
		}
	}
	if v := strings.TrimSpace(ctx.Flag("seed")); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("--seed must be an integer: %q", v)
		}
		req.Recipe.Seed = n
	}
	longForm := req.Recipe.GetLongForm()
	if longForm == nil {
		longForm = &experimentv1.LongFormRecipe{}
	}
	if set, enabled, err := optionalBoolFlag(ctx, "long-form"); err != nil {
		return err
	} else if set {
		longForm.Enabled = enabled
	}
	if v := strings.TrimSpace(ctx.Flag("target-duration-seconds")); v != "" {
		n, err := parseIntFlag("target-duration-seconds", v)
		if err != nil {
			return err
		}
		longForm.Enabled = true
		longForm.TargetDurationSeconds = int32(n)
	}
	if v := strings.TrimSpace(ctx.Flag("gap-ms")); v != "" {
		n, err := parseIntFlag("gap-ms", v)
		if err != nil {
			return err
		}
		longForm.Enabled = true
		longForm.GapMs = int32(n)
	}
	if v := strings.TrimSpace(ctx.Flag("tag-contains")); v != "" {
		longForm.Enabled = true
		longForm.TagContains = v
	}
	if v := strings.TrimSpace(ctx.Flag("sweep-durations")); v != "" {
		values, err := parseIntCSVFlag("sweep-durations", v)
		if err != nil {
			return err
		}
		longForm.Enabled = true
		longForm.SweepDurationsSeconds = values
	}
	if longFormHasContent(longForm) {
		req.Recipe.LongForm = longForm
	}
	augmentation := req.Recipe.GetAugmentation()
	if augmentation == nil {
		augmentation = &experimentv1.AugmentationRecipe{}
	}
	if v := strings.TrimSpace(ctx.Flag("noise-types")); v != "" {
		augmentation.NoiseTypes = splitCSV(v)
	}
	if v := strings.TrimSpace(ctx.Flag("snr-db")); v != "" {
		values, err := parseFloatCSVFlag("snr-db", v)
		if err != nil {
			return err
		}
		augmentation.SnrDb = values
	}
	if v := strings.TrimSpace(ctx.Flag("competing-voices")); v != "" {
		augmentation.CompetingVoiceIds = splitCSV(v)
	}
	if v := strings.TrimSpace(ctx.Flag("competing-text")); v != "" {
		augmentation.CompetingText = v
	}
	if augmentationHasContent(augmentation) {
		req.Recipe.Augmentation = augmentation
	}
	speaker := req.Recipe.GetSpeaker()
	if speaker == nil {
		speaker = &experimentv1.SpeakerExperimentRecipe{}
	}
	if v := strings.TrimSpace(ctx.Flag("target-profile-id")); v != "" {
		speaker.TargetProfileId = v
	}
	if set, enabled, err := optionalBoolFlag(ctx, "speaker-extraction"); err != nil {
		return err
	} else if set {
		speaker.ExtractionEnabled = enabled
	}
	if set, enabled, err := optionalBoolFlag(ctx, "speaker-verification"); err != nil {
		return err
	} else if set {
		speaker.VerificationEnabled = enabled
	}
	if v := strings.TrimSpace(ctx.Flag("speaker-mode")); v != "" {
		mode, err := speakerModeFromFlag(v)
		if err != nil {
			return err
		}
		speaker.VerificationMode = mode
	}
	if v := strings.TrimSpace(ctx.Flag("speaker-threshold")); v != "" {
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("--speaker-threshold must be a number: %q", v)
		}
		speaker.Threshold = n
	}
	if set, enabled, err := optionalBoolFlag(ctx, "speaker-fallback"); err != nil {
		return err
	} else if set {
		speaker.FallbackWithoutVerification = enabled
	}
	if set, enabled, err := optionalBoolFlag(ctx, "speaker-ablation"); err != nil {
		return err
	} else if set {
		speaker.AblationEnabled = enabled
	}
	if speakerHasContent(speaker) {
		req.Recipe.Speaker = speaker
	}
	// Local guard: target-speaker extraction/verification/ablation cannot run
	// without an enrolled profile, so fail before submitting an unrunnable job.
	if (speaker.GetExtractionEnabled() || speaker.GetVerificationEnabled() || speaker.GetAblationEnabled()) && strings.TrimSpace(speaker.GetTargetProfileId()) == "" {
		return fmt.Errorf("speaker experiments require --target-profile-id; enroll a profile with `audio-tools stt speaker-enroll --file <clip> --activate true` and list ids with `audio-tools stt speaker-status`")
	}
	if v := strings.TrimSpace(ctx.Flag("estimated-seconds")); v != "" {
		n, err := parseIntFlag("estimated-seconds", v)
		if err != nil {
			return err
		}
		req.EstimatedSeconds = int32(n)
	}
	// Echo the strategies that will actually run (the server applies the same
	// default trio at run time when none are given) so `start --json` is honest.
	ensureDefaultStrategies(req.Recipe)

	req.DryRun = ctx.BoolFlag("dry-run")

	resp, err := h.client.StartExperiment(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("experiment-start", err, nil)
	}
	exp := resp.Msg.GetExperiment()
	if exp == nil {
		return fmt.Errorf("server returned no experiment")
	}
	changes := []string{
		fmt.Sprintf("name=%q", exp.GetName()),
		fmt.Sprintf("strategies=%s", strategyLabels(exp.GetRecipe().GetStrategies())),
		fmt.Sprintf("clip_ids=%s", strings.Join(exp.GetRecipe().GetClipIds(), ",")),
		fmt.Sprintf("long_form=%t", exp.GetRecipe().GetLongForm().GetEnabled()),
		fmt.Sprintf("long_form_sweep=%s", int32sCSV(exp.GetRecipe().GetLongForm().GetSweepDurationsSeconds())),
		fmt.Sprintf("augmentation_noise=%s", strings.Join(exp.GetRecipe().GetAugmentation().GetNoiseTypes(), ",")),
		fmt.Sprintf("augmentation_voices=%s", strings.Join(exp.GetRecipe().GetAugmentation().GetCompetingVoiceIds(), ",")),
		fmt.Sprintf("speaker_profile=%s", exp.GetRecipe().GetSpeaker().GetTargetProfileId()),
		fmt.Sprintf("speaker_ablation=%t", exp.GetRecipe().GetSpeaker().GetAblationEnabled()),
	}
	if eta := estimatedSecondsLabel(resp.Msg.GetEstimatedSeconds()); eta != "" {
		changes = append(changes, eta)
	}
	if resp.Msg.GetDryRun() {
		// Nothing was enqueued: report the validated preview and how to commit it.
		return renderExperimentProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
			Result:  []string{"Dry run: recipe validated, nothing was enqueued.", "Re-run without --dry-run to start the experiment."},
			Changes: changes,
		})
	}
	return renderExperimentProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Started experiment %s (%s).", exp.GetId(), statusLabel(exp.GetStatus()))},
		Changes: changes,
		NextCommand: []string{
			fmt.Sprintf("audio-tools experiment wait %s --json", exp.GetId()),
			fmt.Sprintf("audio-tools experiment report %s --json", exp.GetId()),
		},
	})
}

func strategiesFromFlag(s string) []*evalv1.EvalStrategy {
	kinds := splitCSV(s)
	out := make([]*evalv1.EvalStrategy, 0, len(kinds))
	for _, kind := range kinds {
		out = append(out, &evalv1.EvalStrategy{Kind: kind, OverlapMaxStallRejects: -1})
	}
	return out
}
