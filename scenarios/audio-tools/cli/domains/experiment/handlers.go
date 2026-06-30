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
	experimentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment/experiment_v1connect"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client experimentconnect.ExperimentServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: experimentconnect.NewExperimentServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) start(ctx cliapp.RunContext) error {
	req := &experimentv1.StartExperimentRequest{
		Name: ctx.Flag("name"),
		Recipe: &experimentv1.ExperimentRecipe{
			ClipIds:    splitCSV(ctx.Flag("clip-ids")),
			Strategies: strategiesFromFlag(ctx.Flag("strategies")),
		},
	}
	if v := strings.TrimSpace(ctx.Flag("realtime-repeats")); v != "" {
		n, err := parseIntFlag("realtime-repeats", v)
		if err != nil {
			return err
		}
		req.Recipe.RealtimeRepeats = int32(n)
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
		n, err := parseIntFlag("overlap-max-window-ms", v)
		if err != nil {
			return err
		}
		ensureDefaultStrategies(req.Recipe)
		for _, s := range req.Recipe.Strategies {
			s.OverlapMaxWindowMs = int32(n)
		}
	}
	if v := strings.TrimSpace(ctx.Flag("seed")); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("--seed must be an integer: %q", v)
		}
		req.Recipe.Seed = n
	}
	longForm := &experimentv1.LongFormRecipe{}
	if v := strings.TrimSpace(ctx.Flag("long-form")); v != "" {
		enabled, err := parseBoolFlag("long-form", v)
		if err != nil {
			return err
		}
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
	if longForm.GetEnabled() || longForm.GetTargetDurationSeconds() > 0 || longForm.GetGapMs() > 0 || longForm.GetTagContains() != "" {
		req.Recipe.LongForm = longForm
	}
	augmentation := &experimentv1.AugmentationRecipe{}
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
	if len(augmentation.GetNoiseTypes()) > 0 || len(augmentation.GetCompetingVoiceIds()) > 0 || len(augmentation.GetSnrDb()) > 0 || augmentation.GetCompetingText() != "" {
		req.Recipe.Augmentation = augmentation
	}
	speaker := &experimentv1.SpeakerExperimentRecipe{}
	if v := strings.TrimSpace(ctx.Flag("target-profile-id")); v != "" {
		speaker.TargetProfileId = v
	}
	if v := strings.TrimSpace(ctx.Flag("speaker-extraction")); v != "" {
		enabled, err := parseBoolFlag("speaker-extraction", v)
		if err != nil {
			return err
		}
		speaker.ExtractionEnabled = enabled
	}
	if v := strings.TrimSpace(ctx.Flag("speaker-verification")); v != "" {
		enabled, err := parseBoolFlag("speaker-verification", v)
		if err != nil {
			return err
		}
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
	if v := strings.TrimSpace(ctx.Flag("speaker-fallback")); v != "" {
		enabled, err := parseBoolFlag("speaker-fallback", v)
		if err != nil {
			return err
		}
		speaker.FallbackWithoutVerification = enabled
	}
	if v := strings.TrimSpace(ctx.Flag("speaker-ablation")); v != "" {
		enabled, err := parseBoolFlag("speaker-ablation", v)
		if err != nil {
			return err
		}
		speaker.AblationEnabled = enabled
	}
	if speaker.GetTargetProfileId() != "" || speaker.GetExtractionEnabled() || speaker.GetVerificationEnabled() || speaker.GetVerificationMode() != sttv1.SpeakerMode_SPEAKER_MODE_UNSPECIFIED || speaker.GetThreshold() != 0 || speaker.GetFallbackWithoutVerification() || speaker.GetAblationEnabled() {
		req.Recipe.Speaker = speaker
	}
	if v := strings.TrimSpace(ctx.Flag("estimated-seconds")); v != "" {
		n, err := parseIntFlag("estimated-seconds", v)
		if err != nil {
			return err
		}
		req.EstimatedSeconds = int32(n)
	}

	resp, err := h.client.StartExperiment(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("experiment-start", err, nil)
	}
	exp := resp.Msg.GetExperiment()
	if exp == nil {
		return fmt.Errorf("server returned no experiment")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Started experiment %s (%s).", exp.GetId(), statusLabel(exp.GetStatus()))},
		Changes: []string{
			fmt.Sprintf("name=%q", exp.GetName()),
			fmt.Sprintf("strategies=%s", strategyLabels(exp.GetRecipe().GetStrategies())),
			fmt.Sprintf("clip_ids=%s", strings.Join(exp.GetRecipe().GetClipIds(), ",")),
			fmt.Sprintf("long_form=%t", exp.GetRecipe().GetLongForm().GetEnabled()),
			fmt.Sprintf("augmentation_noise=%s", strings.Join(exp.GetRecipe().GetAugmentation().GetNoiseTypes(), ",")),
			fmt.Sprintf("augmentation_voices=%s", strings.Join(exp.GetRecipe().GetAugmentation().GetCompetingVoiceIds(), ",")),
			fmt.Sprintf("speaker_profile=%s", exp.GetRecipe().GetSpeaker().GetTargetProfileId()),
			fmt.Sprintf("speaker_ablation=%t", exp.GetRecipe().GetSpeaker().GetAblationEnabled()),
		},
		NextCommand: []string{
			fmt.Sprintf("audio-tools experiment wait %s --json", exp.GetId()),
			fmt.Sprintf("audio-tools experiment report %s --json", exp.GetId()),
		},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetExperiment(context.Background(), connect.NewRequest(&experimentv1.GetExperimentRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("experiment-get", err, nil)
	}
	return renderExperiment(ctx, resp.Msg, "get")
}

func (h *handlers) wait(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.WaitExperiment(context.Background(), connect.NewRequest(&experimentv1.WaitExperimentRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("experiment-wait", err, nil)
	}
	return renderExperiment(ctx, resp.Msg, "wait")
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	req := &experimentv1.ListExperimentsRequest{}
	if v := strings.TrimSpace(ctx.Flag("status")); v != "" {
		status, err := statusFromFlag(v)
		if err != nil {
			return err
		}
		req.Status = status
	}
	if v := strings.TrimSpace(ctx.Flag("limit")); v != "" {
		n, err := parseIntFlag("limit", v)
		if err != nil {
			return err
		}
		req.Limit = int32(n)
	}
	if v := strings.TrimSpace(ctx.Flag("offset")); v != "" {
		n, err := parseIntFlag("offset", v)
		if err != nil {
			return err
		}
		req.Offset = int32(n)
	}
	resp, err := h.client.ListExperiments(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("experiment-list", err, nil)
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	experiments := resp.Msg.GetExperiments()
	if len(experiments) == 0 {
		fmt.Fprintln(ctx.Stdout(), "No experiments found.")
		return nil
	}
	fmt.Fprintf(ctx.Stdout(), "%-38s  %-10s  %-22s  %s\n", "ID", "STATUS", "CREATED", "NAME")
	for _, exp := range experiments {
		fmt.Fprintf(ctx.Stdout(), "%-38s  %-10s  %-22s  %s\n",
			exp.GetId(), statusLabel(exp.GetStatus()), timestampLabel(exp.GetCreatedAt()), exp.GetName())
	}
	return nil
}

func (h *handlers) cancel(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.CancelExperiment(context.Background(), connect.NewRequest(&experimentv1.CancelExperimentRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("experiment-cancel", err, nil)
	}
	exp := resp.Msg.GetExperiment()
	if exp == nil {
		return fmt.Errorf("server returned no experiment")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Canceled experiment %s (%s).", exp.GetId(), statusLabel(exp.GetStatus()))},
		NextCommand: []string{
			fmt.Sprintf("audio-tools experiment get %s --json", exp.GetId()),
		},
	})
}

func (h *handlers) watch(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	stream, err := h.client.StreamExperimentEvents(context.Background(), connect.NewRequest(&experimentv1.StreamExperimentEventsRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("experiment-watch", err, nil)
	}
	defer stream.Close()
	for stream.Receive() {
		ev := stream.Msg()
		if ctx.JSON() {
			if err := cliapp.PrintProtoJSON(ctx.Stdout(), ev); err != nil {
				return err
			}
			continue
		}
		fmt.Fprintf(ctx.Stdout(), "%s %3d%% %s\n", statusLabel(ev.GetStatus()), ev.GetProgress(), ev.GetMessage())
	}
	if err := stream.Err(); err != nil {
		return cliapp.WrapAPIError("experiment-watch receive", err, nil)
	}
	return nil
}

func (h *handlers) report(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetExperimentReport(context.Background(), connect.NewRequest(&experimentv1.GetExperimentReportRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError("experiment-report", err, nil)
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	exp := resp.Msg.GetExperiment()
	fmt.Fprintf(ctx.Stdout(), "Experiment %s (%s)\n", exp.GetId(), statusLabel(exp.GetStatus()))
	printReportTable(ctx, resp.Msg.GetReport())
	return nil
}

func (h *handlers) compare(ctx cliapp.RunContext) error {
	ids := splitCSV(ctx.Positional("ids"))
	if len(ids) < 2 {
		return fmt.Errorf("compare requires at least two comma-separated experiment ids")
	}
	resp, err := h.client.CompareExperiments(context.Background(), connect.NewRequest(&experimentv1.CompareExperimentsRequest{Ids: ids}))
	if err != nil {
		return cliapp.WrapAPIError("experiment-compare", err, nil)
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), resp.Msg)
	}
	printComparison(ctx, resp.Msg.GetExperiments())
	return nil
}

func renderExperiment(ctx cliapp.RunContext, msg interface {
	GetExperiment() *experimentv1.Experiment
	GetRuns() []*experimentv1.ExperimentRun
}, op string,
) error {
	switch m := msg.(type) {
	case *experimentv1.GetExperimentResponse:
		if ctx.JSON() {
			return cliapp.PrintProtoJSON(ctx.Stdout(), m)
		}
	case *experimentv1.WaitExperimentResponse:
		if ctx.JSON() {
			return cliapp.PrintProtoJSON(ctx.Stdout(), m)
		}
	}
	exp := msg.GetExperiment()
	if exp == nil {
		return fmt.Errorf("server returned no experiment")
	}
	runs := msg.GetRuns()
	report := cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Experiment %s is %s.", exp.GetId(), statusLabel(exp.GetStatus())),
			fmt.Sprintf("name=%q created=%s started=%s finished=%s", exp.GetName(), timestampLabel(exp.GetCreatedAt()), timestampLabel(exp.GetStartedAt()), timestampLabel(exp.GetFinishedAt())),
		},
		ResultsHeading: "Runs",
		RetrievalHints: []string{
			fmt.Sprintf("audio-tools experiment report %s --json", exp.GetId()),
			fmt.Sprintf("audio-tools experiment %s %s --json", op, exp.GetId()),
		},
	}
	for _, run := range runs {
		report.Results = append(report.Results, fmt.Sprintf("%s metrics=%s condition=%s", run.GetStrategy(), run.GetMetricsJson(), run.GetConditionJson()))
	}
	if len(runs) == 0 {
		report.Results = append(report.Results, "(none yet)")
	}
	return ctx.RenderList(report)
}

func printReportTable(ctx cliapp.RunContext, report *evalv1.EvalReport) {
	if report == nil || len(report.GetPerStrategy()) == 0 {
		fmt.Fprintln(ctx.Stdout(), "No report rows available.")
		return
	}
	fmt.Fprintf(ctx.Stdout(), "%-24s  %6s  %6s  %6s  %8s  %9s  %9s  %8s  %8s  %8s\n",
		"STRATEGY", "WER%", "CALLS", "RTF", "AUDIO_S", "LAT_P50", "LAT_P95", "REVISES", "SAFE", "MAXDROP")
	for _, s := range report.GetPerStrategy() {
		lat50, lat95 := "-", "-"
		if report.GetLatencyMeasured() {
			lat50 = fmt.Sprintf("%.0fms", s.GetFinalizationLatencyP50Ms())
			lat95 = fmt.Sprintf("%.0fms", s.GetFinalizationLatencyP95Ms())
		}
		safe := "pass"
		if s.GetSafety() != nil && !s.GetSafety().GetPassed() {
			safe = "fail"
		}
		maxDrop := int32(0)
		if s.GetSafety() != nil {
			maxDrop = s.GetSafety().GetMaxDroppedSpanWords()
		}
		fmt.Fprintf(ctx.Stdout(), "%-24s  %6.1f  %6d  %6.2f  %8.2f  %9s  %9s  %8d  %8s  %8d\n",
			s.GetLabel(), s.GetWer()*100, s.GetWhisperCalls(), s.GetRtf(),
			s.GetWhisperAudioSeconds(), lat50, lat95, s.GetPartialRevisions(), safe, maxDrop)
	}
}

func printComparison(ctx cliapp.RunContext, experiments []*experimentv1.ComparedExperiment) {
	if len(experiments) == 0 {
		fmt.Fprintln(ctx.Stdout(), "No experiments returned.")
		return
	}
	fmt.Fprintf(ctx.Stdout(), "%-38s  %-10s  %-24s  %8s  %8s  %8s\n", "ID", "STATUS", "STRATEGY", "WER%", "CALLS", "RTF")
	for _, ce := range experiments {
		exp := ce.GetExperiment()
		for _, row := range ce.GetReport().GetPerStrategy() {
			fmt.Fprintf(ctx.Stdout(), "%-38s  %-10s  %-24s  %8.2f  %8d  %8.2f\n",
				exp.GetId(), statusLabel(exp.GetStatus()), row.GetLabel(), row.GetWer()*100, row.GetWhisperCalls(), row.GetRtf())
		}
	}
	if len(experiments) >= 2 {
		fmt.Fprintln(ctx.Stdout(), "\nUse --json for full recipe fields and per-experiment report payloads.")
	}
}

func strategiesFromFlag(s string) []*evalv1.EvalStrategy {
	kinds := splitCSV(s)
	out := make([]*evalv1.EvalStrategy, 0, len(kinds))
	for _, kind := range kinds {
		out = append(out, &evalv1.EvalStrategy{Kind: kind, OverlapMaxStallRejects: -1})
	}
	return out
}

func ensureDefaultStrategies(recipe *experimentv1.ExperimentRecipe) {
	if recipe == nil || len(recipe.GetStrategies()) > 0 {
		return
	}
	recipe.Strategies = []*evalv1.EvalStrategy{
		{Kind: "batch", OverlapMaxStallRejects: -1},
		{Kind: "vad_segment", OverlapMaxStallRejects: -1},
		{Kind: "overlap_agree", OverlapMaxStallRejects: -1},
	}
}

func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func parseIntFlag(name, value string) (int, error) {
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("--%s must be an integer: %q", name, value)
	}
	return n, nil
}

func parseBoolFlag(name, value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("--%s must be true or false: %q", name, value)
	}
}

func parseFloatCSVFlag(name string, value string) ([]float64, error) {
	parts := splitCSV(value)
	out := make([]float64, 0, len(parts))
	for _, part := range parts {
		n, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return nil, fmt.Errorf("--%s must be comma-separated numbers: %q", name, value)
		}
		out = append(out, n)
	}
	return out, nil
}

func speakerModeFromFlag(s string) (sttv1.SpeakerMode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "off":
		return sttv1.SpeakerMode_SPEAKER_MODE_OFF, nil
	case "filter":
		return sttv1.SpeakerMode_SPEAKER_MODE_FILTER, nil
	case "advisory":
		return sttv1.SpeakerMode_SPEAKER_MODE_ADVISORY, nil
	default:
		return sttv1.SpeakerMode_SPEAKER_MODE_UNSPECIFIED, fmt.Errorf("--speaker-mode must be off|filter|advisory: %q", s)
	}
}

func statusFromFlag(s string) (experimentv1.ExperimentStatus, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "queued":
		return experimentv1.ExperimentStatus_EXPERIMENT_STATUS_QUEUED, nil
	case "running":
		return experimentv1.ExperimentStatus_EXPERIMENT_STATUS_RUNNING, nil
	case "succeeded", "success":
		return experimentv1.ExperimentStatus_EXPERIMENT_STATUS_SUCCEEDED, nil
	case "failed", "failure":
		return experimentv1.ExperimentStatus_EXPERIMENT_STATUS_FAILED, nil
	case "canceled", "cancelled":
		return experimentv1.ExperimentStatus_EXPERIMENT_STATUS_CANCELED, nil
	default:
		return experimentv1.ExperimentStatus_EXPERIMENT_STATUS_UNSPECIFIED, fmt.Errorf("--status must be queued|running|succeeded|failed|canceled: %q", s)
	}
}

func statusLabel(s experimentv1.ExperimentStatus) string {
	switch s {
	case experimentv1.ExperimentStatus_EXPERIMENT_STATUS_QUEUED:
		return "queued"
	case experimentv1.ExperimentStatus_EXPERIMENT_STATUS_RUNNING:
		return "running"
	case experimentv1.ExperimentStatus_EXPERIMENT_STATUS_SUCCEEDED:
		return "succeeded"
	case experimentv1.ExperimentStatus_EXPERIMENT_STATUS_FAILED:
		return "failed"
	case experimentv1.ExperimentStatus_EXPERIMENT_STATUS_CANCELED:
		return "canceled"
	default:
		return "unspecified"
	}
}

func timestampLabel(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return "-"
	}
	return ts.AsTime().Format("2006-01-02T15:04:05Z07:00")
}

func strategyLabels(strategies []*evalv1.EvalStrategy) string {
	if len(strategies) == 0 {
		return "default"
	}
	out := make([]string, 0, len(strategies))
	for _, s := range strategies {
		out = append(out, s.GetKind())
	}
	return strings.Join(out, ",")
}
