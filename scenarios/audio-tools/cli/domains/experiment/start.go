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

func (h *handlers) start(ctx cliapp.RunContext) error { return h.startPrimitive().Run(ctx) }

func (h *handlers) startPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(func(ctx cliapp.OperationContext) (*experimentv1.StartExperimentResponse, error) {
		req, err := buildStartRequest(ctx)
		if err != nil {
			return nil, err
		}
		resp, err := h.client.StartExperiment(context.Background(), connect.NewRequest(req))
		if err != nil {
			return nil, cliapp.WrapAPIError("experiment-start", err, nil)
		}
		if resp.Msg.GetExperiment() == nil {
			return nil, fmt.Errorf("server returned no experiment")
		}
		return resp.Msg, nil
	}, renderStartHuman, func(ctx cliapp.RunContext, msg *experimentv1.StartExperimentResponse) error {
		return printExperimentProtoJSON(ctx.Stdout(), msg)
	})
}

func renderStartHuman(ctx cliapp.RunContext, msg *experimentv1.StartExperimentResponse) error {
	exp := msg.GetExperiment()
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
	if mode := exp.GetRecipe().GetReportRetentionMode(); mode != experimentv1.ReportRetentionMode_REPORT_RETENTION_MODE_UNSPECIFIED {
		changes = append(changes, "report_retention_mode="+strings.ToLower(strings.TrimPrefix(mode.String(), "REPORT_RETENTION_MODE_")))
	}
	if eta := estimatedSecondsLabel(msg.GetEstimatedSeconds()); eta != "" {
		changes = append(changes, eta)
	}
	if msg.GetDryRun() {
		return renderExperimentProtoMutation(ctx, msg, cliapp.MutationReport{Result: []string{"Dry run: recipe validated, nothing was enqueued.", "Re-run without --dry-run to start the experiment."}, Changes: changes})
	}
	return renderExperimentProtoMutation(ctx, msg, cliapp.MutationReport{Result: []string{fmt.Sprintf("Started experiment %s (%s).", exp.GetId(), statusLabel(exp.GetStatus()))}, Changes: changes, NextCommand: []string{fmt.Sprintf("audio-tools experiment wait %s --json", exp.GetId()), fmt.Sprintf("audio-tools experiment report %s --json", exp.GetId())}})
}

func buildStartRequest(ctx cliapp.OperationContext) (*experimentv1.StartExperimentRequest, error) {
	recipe, err := loadBaseRecipe(ctx)
	if err != nil {
		return nil, err
	}
	req := &experimentv1.StartExperimentRequest{Name: ctx.Flag("name"), Recipe: recipe}
	steps := []func(cliapp.OperationContext, *experimentv1.StartExperimentRequest) error{
		applySelectionFlags, applyBasicRecipeFlags, applyStrategyTuningFlags,
		applySeedFlag, applyLongFormFlags, applyAugmentationFlags,
		applySpeakerFlags, applyEstimatedSecondsFlag,
	}
	for _, step := range steps {
		if err := step(ctx, req); err != nil {
			return nil, err
		}
	}
	// Echo the strategies that will actually run so `start --json` is honest.
	ensureDefaultStrategies(req.Recipe)
	req.DryRun = ctx.BoolFlag("dry-run")
	return req, nil
}

func applySelectionFlags(ctx cliapp.OperationContext, req *experimentv1.StartExperimentRequest) error {
	if v := strings.TrimSpace(ctx.Flag("clip-ids")); v != "" {
		req.Recipe.ClipIds = splitCSV(v)
	}
	if v := strings.TrimSpace(ctx.Flag("strategies")); v != "" {
		req.Recipe.Strategies = strategiesFromFlag(v)
	}
	return nil
}

func applyBasicRecipeFlags(ctx cliapp.OperationContext, req *experimentv1.StartExperimentRequest) error {
	flags := []struct {
		name string
		set  func(int32)
	}{
		{"realtime-repeats", func(n int32) { req.Recipe.RealtimeRepeats = n }},
		{"latency-tail-seconds", func(n int32) { req.Recipe.LatencyTailSeconds = n }},
		{"chunk-ms", func(n int32) { req.Recipe.ChunkMs = n }},
		{"dropped-span-threshold", func(n int32) { req.Recipe.DroppedSpanThresholdWords = n }},
	}
	for _, flag := range flags {
		if v := strings.TrimSpace(ctx.Flag(flag.name)); v != "" {
			n, err := parseIntFlag(flag.name, v)
			if err != nil {
				return err
			}
			flag.set(n)
		}
	}
	return nil
}

func applyStrategyTuningFlags(ctx cliapp.OperationContext, req *experimentv1.StartExperimentRequest) error {
	flags := []struct {
		name    string
		applies func(string) bool
		set     func(*evalv1.EvalStrategy, int32)
	}{
		{"overlap-max-window-ms", strategyUsesOverlap, func(s *evalv1.EvalStrategy, n int32) { s.OverlapMaxWindowMs = n }},
		{"overlap-max-stall-rejects", strategyUsesOverlap, func(s *evalv1.EvalStrategy, n int32) { s.OverlapMaxStallRejects = n }},
		{"overlap-window-ms", strategyUsesOverlap, func(s *evalv1.EvalStrategy, n int32) { s.OverlapWindowMs = n }},
		{"overlap-commit-runs", strategyUsesOverlap, func(s *evalv1.EvalStrategy, n int32) { s.OverlapCommitRuns = n }},
		{"vad-silence-ms", strategyUsesVADSilence, func(s *evalv1.EvalStrategy, n int32) { s.VadSilenceMs = n }},
	}
	for _, flag := range flags {
		if v := strings.TrimSpace(ctx.Flag(flag.name)); v != "" {
			n, err := parseNonNegativeIntFlag(flag.name, v)
			if err != nil {
				return err
			}
			ensureDefaultStrategies(req.Recipe)
			for _, strategy := range req.Recipe.Strategies {
				if flag.applies(strategy.GetKind()) {
					flag.set(strategy, n)
				}
			}
		}
	}
	return nil
}

func applySeedFlag(ctx cliapp.OperationContext, req *experimentv1.StartExperimentRequest) error {
	if v := strings.TrimSpace(ctx.Flag("seed")); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("--seed must be an integer: %q", v)
		}
		req.Recipe.Seed = n
	}
	return nil
}

func applyLongFormFlags(ctx cliapp.OperationContext, req *experimentv1.StartExperimentRequest) error {
	lf := req.Recipe.GetLongForm()
	if lf == nil {
		lf = &experimentv1.LongFormRecipe{}
	}
	if set, enabled, err := optionalBoolFlag(ctx, "long-form"); err != nil {
		return err
	} else if set {
		lf.Enabled = enabled
	}
	if v := strings.TrimSpace(ctx.Flag("target-duration-seconds")); v != "" {
		n, err := parseIntFlag("target-duration-seconds", v)
		if err != nil {
			return err
		}
		lf.Enabled = true
		lf.TargetDurationSeconds = n
	}
	if v := strings.TrimSpace(ctx.Flag("gap-ms")); v != "" {
		n, err := parseIntFlag("gap-ms", v)
		if err != nil {
			return err
		}
		lf.Enabled = true
		lf.GapMs = n
	}
	if v := strings.TrimSpace(ctx.Flag("tag-contains")); v != "" {
		lf.Enabled = true
		lf.TagContains = v
	}
	if v := strings.TrimSpace(ctx.Flag("sweep-durations")); v != "" {
		values, err := parseIntCSVFlag("sweep-durations", v)
		if err != nil {
			return err
		}
		lf.Enabled = true
		lf.SweepDurationsSeconds = values
	}
	if longFormHasContent(lf) {
		req.Recipe.LongForm = lf
	}
	return nil
}

func applyAugmentationFlags(ctx cliapp.OperationContext, req *experimentv1.StartExperimentRequest) error {
	a := req.Recipe.GetAugmentation()
	if a == nil {
		a = &experimentv1.AugmentationRecipe{}
	}
	if v := strings.TrimSpace(ctx.Flag("noise-types")); v != "" {
		a.NoiseTypes = splitCSV(v)
	}
	if v := strings.TrimSpace(ctx.Flag("snr-db")); v != "" {
		values, err := parseFloatCSVFlag("snr-db", v)
		if err != nil {
			return err
		}
		a.SnrDb = values
	}
	if v := strings.TrimSpace(ctx.Flag("competing-voices")); v != "" {
		a.CompetingVoiceIds = splitCSV(v)
	}
	if v := strings.TrimSpace(ctx.Flag("competing-text")); v != "" {
		a.CompetingText = v
	}
	if augmentationHasContent(a) {
		req.Recipe.Augmentation = a
	}
	return nil
}

func applySpeakerFlags(ctx cliapp.OperationContext, req *experimentv1.StartExperimentRequest) error {
	s := req.Recipe.GetSpeaker()
	if s == nil {
		s = &experimentv1.SpeakerExperimentRecipe{}
	}
	if v := strings.TrimSpace(ctx.Flag("target-profile-id")); v != "" {
		s.TargetProfileId = v
	}
	if err := applySpeakerBoolFlags(ctx, s, true); err != nil {
		return err
	}
	if v := strings.TrimSpace(ctx.Flag("speaker-mode")); v != "" {
		mode, err := speakerModeFromFlag(v)
		if err != nil {
			return err
		}
		s.VerificationMode = mode
	}
	if v := strings.TrimSpace(ctx.Flag("speaker-threshold")); v != "" {
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("--speaker-threshold must be a number: %q", v)
		}
		s.Threshold = n
	}
	if err := applySpeakerBoolFlags(ctx, s, false); err != nil {
		return err
	}
	if speakerHasContent(s) {
		req.Recipe.Speaker = s
	}
	if (s.GetExtractionEnabled() || s.GetVerificationEnabled() || s.GetAblationEnabled()) && strings.TrimSpace(s.GetTargetProfileId()) == "" {
		return fmt.Errorf("speaker experiments require --target-profile-id; enroll a profile with `audio-tools stt speaker-enroll --file <clip> --activate true` and list ids with `audio-tools stt speaker-status`")
	}
	return nil
}

func applySpeakerBoolFlags(ctx cliapp.OperationContext, s *experimentv1.SpeakerExperimentRecipe, initial bool) error {
	flags := []struct {
		name string
		set  func(bool)
	}{}
	if initial {
		flags = append(flags, struct {
			name string
			set  func(bool)
		}{"speaker-extraction", func(v bool) { s.ExtractionEnabled = v }}, struct {
			name string
			set  func(bool)
		}{"speaker-verification", func(v bool) { s.VerificationEnabled = v }})
	} else {
		flags = append(flags, struct {
			name string
			set  func(bool)
		}{"speaker-fallback", func(v bool) { s.FallbackWithoutVerification = v }}, struct {
			name string
			set  func(bool)
		}{"speaker-ablation", func(v bool) { s.AblationEnabled = v }})
	}
	for _, flag := range flags {
		if set, value, err := optionalBoolFlag(ctx, flag.name); err != nil {
			return err
		} else if set {
			flag.set(value)
		}
	}
	return nil
}

func applyEstimatedSecondsFlag(ctx cliapp.OperationContext, req *experimentv1.StartExperimentRequest) error {
	if v := strings.TrimSpace(ctx.Flag("estimated-seconds")); v != "" {
		n, err := parseIntFlag("estimated-seconds", v)
		if err != nil {
			return err
		}
		req.EstimatedSeconds = n
	}
	return nil
}

func strategiesFromFlag(s string) []*evalv1.EvalStrategy {
	kinds := splitCSV(s)
	out := make([]*evalv1.EvalStrategy, 0, len(kinds))
	for _, kind := range kinds {
		out = append(out, &evalv1.EvalStrategy{Kind: kind, OverlapMaxStallRejects: -1})
	}
	return out
}
