package experiment

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
	experimentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment"
	experimentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/experiment/experiment_v1connect"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client experimentconnect.ExperimentServiceClient
}

var experimentProtoJSONOptions = protojson.MarshalOptions{
	UseProtoNames:     true,
	Multiline:         true,
	Indent:            "  ",
	EmitDefaultValues: true,
	EmitUnpopulated:   false,
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: experimentconnect.NewExperimentServiceClient(httpClient, baseURL),
	}
}

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
	return renderExperimentProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Started experiment %s (%s).", exp.GetId(), statusLabel(exp.GetStatus()))},
		Changes: changes,
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
	if experimentTerminal(resp.Msg.GetExperiment()) && resp.Msg.GetExperiment().GetResultRef() != "" {
		reportResp, reportErr := h.client.GetExperimentReport(context.Background(), connect.NewRequest(&experimentv1.GetExperimentReportRequest{Id: id}))
		if reportErr == nil && reportResp != nil && reportResp.Msg != nil && reportResp.Msg.GetReport() != nil {
			if ctx.JSON() {
				return printExperimentProtoJSON(ctx.Stdout(), reportResp.Msg)
			}
			return renderReport(ctx, reportResp.Msg)
		}
		fmt.Fprintf(ctx.Stderr(), "Report is not available yet; use `audio-tools experiment report %s` to re-attach.\n", id)
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
		return printExperimentProtoJSON(ctx.Stdout(), resp.Msg)
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
	printExperimentErrors(ctx, experiments)
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
	return renderExperimentProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
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
			if err := printExperimentProtoJSON(ctx.Stdout(), ev); err != nil {
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
		return printExperimentProtoJSON(ctx.Stdout(), resp.Msg)
	}
	return renderReport(ctx, resp.Msg)
}

func (h *handlers) compare(ctx cliapp.RunContext) error {
	// Accept both space-separated positionals (`compare a b`) and comma-separated
	// forms (`compare a,b`), flatten, and dedupe while preserving order.
	ids := dedupeStrings(flattenCSV(ctx.Positionals("ids")))
	if len(ids) < 2 {
		return fmt.Errorf("compare requires at least two experiment ids (space- or comma-separated)")
	}
	resp, err := h.client.CompareExperiments(context.Background(), connect.NewRequest(&experimentv1.CompareExperimentsRequest{Ids: ids}))
	if err != nil {
		return cliapp.WrapAPIError("experiment-compare", err, nil)
	}
	if ctx.JSON() {
		return printExperimentProtoJSON(ctx.Stdout(), resp.Msg)
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
			return printExperimentProtoJSON(ctx.Stdout(), m)
		}
	case *experimentv1.WaitExperimentResponse:
		if ctx.JSON() {
			return printExperimentProtoJSON(ctx.Stdout(), m)
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
	if errMsg := strings.TrimSpace(exp.GetError()); errMsg != "" {
		report.Summary = append(report.Summary, fmt.Sprintf("error=%s", errMsg))
	}
	for _, run := range runs {
		report.Results = append(report.Results, formatRunStatus(run))
	}
	if len(runs) == 0 {
		report.Results = append(report.Results, "(none yet)")
	}
	return ctx.RenderList(report)
}

func printExperimentProtoJSON(w io.Writer, msg proto.Message) error {
	body, err := experimentProtoJSONOptions.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal experiment proto json: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		return err
	}
	if len(body) == 0 || body[len(body)-1] != '\n' {
		_, err = w.Write([]byte{'\n'})
	}
	return err
}

func renderExperimentProtoMutation(ctx cliapp.RunContext, payload proto.Message, human cliapp.MutationReport) error {
	if ctx.JSON() {
		return printExperimentProtoJSON(ctx.Stdout(), payload)
	}
	return ctx.RenderMutation(human)
}

func renderReport(ctx cliapp.RunContext, msg *experimentv1.GetExperimentReportResponse) error {
	if msg == nil || msg.GetExperiment() == nil {
		return fmt.Errorf("server returned no experiment")
	}
	exp := msg.GetExperiment()
	fmt.Fprintf(ctx.Stdout(), "Experiment %s (%s)\n", exp.GetId(), statusLabel(exp.GetStatus()))
	if errMsg := strings.TrimSpace(exp.GetError()); errMsg != "" {
		fmt.Fprintf(ctx.Stdout(), "Error: %s\n", errMsg)
	}
	printReportTable(ctx, msg.GetReport())
	return nil
}

func printReportTable(ctx cliapp.RunContext, report *evalv1.EvalReport) {
	if report == nil || len(report.GetPerStrategy()) == 0 {
		fmt.Fprintln(ctx.Stdout(), "No report rows available.")
		return
	}
	printReportSummary(ctx, report)
	fmt.Fprintf(ctx.Stdout(), "%-34s  %6s  %6s  %6s  %8s  %9s  %9s  %8s  %8s  %8s  %12s\n",
		"STRATEGY", "WER%", "CALLS", "RTF", "AUDIO_S", "LAT_P50", "LAT_P95", "REVISES", "SAFE", "MAXDROP", "VERDICT")
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
		verdict := s.GetVerdict()
		if verdict == "" {
			verdict = "-"
		}
		fmt.Fprintf(ctx.Stdout(), "%-34s  %6.1f  %6d  %6.2f  %8.2f  %9s  %9s  %8d  %8s  %8d  %12s\n",
			s.GetLabel(), s.GetWer()*100, s.GetWhisperCalls(), s.GetRtf(),
			s.GetWhisperAudioSeconds(), lat50, lat95, s.GetPartialRevisions(), safe, maxDrop, verdict)
	}
	printReportWarnings(ctx, report)
	printLengthCurves(ctx, report)
}

func estimatedSecondsLabel(seconds int32) string {
	if seconds <= 0 {
		return ""
	}
	if seconds < 60 {
		return fmt.Sprintf("estimated_seconds=%d", seconds)
	}
	return fmt.Sprintf("estimated_seconds=%d (~%dm%02ds)", seconds, seconds/60, seconds%60)
}

func printReportSummary(ctx cliapp.RunContext, report *evalv1.EvalReport) {
	summary := report.GetSummary()
	if summary == nil || summary.GetRecommendation() == "" {
		return
	}
	confidence := summary.GetConfidence()
	if confidence == "" {
		confidence = "unknown"
	}
	fmt.Fprintf(ctx.Stdout(), "Recommendation: %s (confidence: %s)\n", summary.GetRecommendation(), confidence)
	for _, reason := range summary.GetReasons() {
		fmt.Fprintf(ctx.Stdout(), "  - %s\n", reason)
	}
	for _, note := range summary.GetConfidenceNotes() {
		fmt.Fprintf(ctx.Stdout(), "  - %s\n", note)
	}
	fmt.Fprintln(ctx.Stdout())
}

func printReportWarnings(ctx cliapp.RunContext, report *evalv1.EvalReport) {
	warnings := report.GetWarnings()
	if len(warnings) == 0 {
		return
	}
	fmt.Fprintln(ctx.Stdout(), "\nWarnings:")
	for _, w := range warnings {
		severity := w.GetSeverity()
		if severity == "" {
			severity = "info"
		}
		code := w.GetCode()
		if code == "" {
			code = "warning"
		}
		fmt.Fprintf(ctx.Stdout(), "  - %s/%s: %s\n", severity, code, w.GetMessage())
	}
}

func printLengthCurves(ctx cliapp.RunContext, report *evalv1.EvalReport) {
	hasCurves := false
	for _, s := range report.GetPerStrategy() {
		if len(s.GetLengthCurves()) > 0 {
			hasCurves = true
			break
		}
	}
	if !hasCurves {
		return
	}
	fmt.Fprintln(ctx.Stdout(), "\nLength curves:")
	for _, s := range report.GetPerStrategy() {
		if len(s.GetLengthCurves()) == 0 {
			continue
		}
		fmt.Fprintf(ctx.Stdout(), "  %s\n", s.GetLabel())
		fmt.Fprintf(ctx.Stdout(), "    %-12s  %5s  %6s  %9s  %9s  %8s\n", "BUCKET", "CLIPS", "WER%", "P95", "TTFC", "MAXDROP")
		for _, curve := range s.GetLengthCurves() {
			p95, ttfc := "-", "-"
			if report.GetLatencyMeasured() {
				p95 = fmt.Sprintf("%.0fms", curve.GetFinalizationLatencyP95Ms())
				ttfc = fmt.Sprintf("%.0fms", curve.GetMeanTimeToFirstCommitMs())
			}
			fmt.Fprintf(ctx.Stdout(), "    %-12s  %5d  %6.1f  %9s  %9s  %8d\n",
				curve.GetBucket(), curve.GetClipCount(), curve.GetWer()*100, p95, ttfc, curve.GetMaxDroppedSpanWords())
		}
	}
}

func formatRunStatus(run *experimentv1.ExperimentRun) string {
	if run == nil {
		return "(nil run)"
	}
	condition := strings.TrimSpace(run.GetConditionJson())
	if condition != "" && condition != "{}" {
		return fmt.Sprintf("%s - condition=%s", run.GetStrategy(), condition)
	}
	return fmt.Sprintf("%s - completed", run.GetStrategy())
}

func experimentTerminal(exp *experimentv1.Experiment) bool {
	if exp == nil {
		return false
	}
	switch exp.GetStatus() {
	case experimentv1.ExperimentStatus_EXPERIMENT_STATUS_SUCCEEDED,
		experimentv1.ExperimentStatus_EXPERIMENT_STATUS_FAILED,
		experimentv1.ExperimentStatus_EXPERIMENT_STATUS_CANCELED:
		return true
	default:
		return false
	}
}

// comparisonRow is the per-experiment projection the comparison table renders:
// the experiment, its report (may be nil), and the winning strategy row.
type comparisonRow struct {
	exp    *experimentv1.Experiment
	report *evalv1.EvalReport
	winner *evalv1.StrategyReport
}

func printComparison(ctx cliapp.RunContext, experiments []*experimentv1.ComparedExperiment) {
	if len(experiments) == 0 {
		fmt.Fprintln(ctx.Stdout(), "No experiments returned.")
		return
	}
	rows := make([]comparisonRow, 0, len(experiments))
	bestIdx := -1
	anyMissingReport := false
	for i, ce := range experiments {
		row := comparisonRow{exp: ce.GetExperiment(), report: ce.GetReport()}
		if row.report != nil {
			row.winner = winnerStrategyRow(row.report)
		} else {
			anyMissingReport = true
		}
		rows = append(rows, row)
		if row.winner == nil {
			continue
		}
		if bestIdx == -1 {
			bestIdx = i
			continue
		}
		best := rows[bestIdx].winner
		if row.winner.GetWer() < best.GetWer() ||
			(row.winner.GetWer() == best.GetWer() && row.winner.GetRtf() < best.GetRtf()) {
			bestIdx = i
		}
	}
	var bestWinner *evalv1.StrategyReport
	if bestIdx >= 0 {
		bestWinner = rows[bestIdx].winner
	}

	fmt.Fprintf(ctx.Stdout(), "%-2s %-18s %-38s %-10s %-22s %7s %8s %6s %6s %7s %7s %7s\n",
		"", "NAME", "ID", "STATUS", "WINNER", "WER%", "P95", "CALLS", "RTF", "SAFE", "dWER%", "dRTF")
	for i, row := range rows {
		mark := ""
		if i == bestIdx {
			mark = "*"
		}
		name := row.exp.GetName()
		if name == "" {
			name = "-"
		}
		winnerLabel, werCol, p95Col, callsCol, rtfCol, safeCol, dWerCol, dRtfCol := "-", "-", "-", "-", "-", "-", "-", "-"
		if row.winner != nil {
			winnerLabel = row.winner.GetLabel()
			if winnerLabel == "" {
				winnerLabel = row.winner.GetStrategy()
			}
			werCol = fmt.Sprintf("%.1f", row.winner.GetWer()*100)
			callsCol = fmt.Sprintf("%d", row.winner.GetWhisperCalls())
			rtfCol = fmt.Sprintf("%.2f", row.winner.GetRtf())
			if row.report.GetLatencyMeasured() {
				p95Col = fmt.Sprintf("%.0fms", row.winner.GetFinalizationLatencyP95Ms())
			}
			safeCol = safetyLabel(row.winner.GetSafety())
			if bestWinner != nil {
				dWerCol = fmt.Sprintf("%+.1f", (row.winner.GetWer()-bestWinner.GetWer())*100)
				dRtfCol = fmt.Sprintf("%+.2f", row.winner.GetRtf()-bestWinner.GetRtf())
			}
		}
		fmt.Fprintf(ctx.Stdout(), "%-2s %-18s %-38s %-10s %-22s %7s %8s %6s %6s %7s %7s %7s\n",
			mark, truncate(name, 18), row.exp.GetId(), statusLabel(row.exp.GetStatus()),
			truncate(winnerLabel, 22), werCol, p95Col, callsCol, rtfCol, safeCol, dWerCol, dRtfCol)
	}
	if bestIdx >= 0 {
		fmt.Fprintf(ctx.Stdout(), "\n* best = lowest winner WER (tie-break lower RTF); deltas are vs that experiment.\n")
	}
	if anyMissingReport {
		fmt.Fprintln(ctx.Stdout(), "Rows without a winner have no stored report yet (still running, or failed before reporting).")
	}
	printRecipeDiff(ctx, rows)
	printStrategyAlignment(ctx, rows)
	exps := make([]*experimentv1.Experiment, 0, len(rows))
	for _, row := range rows {
		exps = append(exps, row.exp)
	}
	printExperimentErrors(ctx, exps)
	fmt.Fprintln(ctx.Stdout(), "Use --json for full recipes and per-experiment report payloads.")
}

func printRecipeDiff(ctx cliapp.RunContext, rows []comparisonRow) {
	diffs := recipeDiffLines(rows)
	if len(diffs) == 0 {
		return
	}
	fmt.Fprintln(ctx.Stdout(), "\nRecipe differences:")
	for _, line := range diffs {
		fmt.Fprintf(ctx.Stdout(), "  - %s\n", line)
	}
}

func recipeDiffLines(rows []comparisonRow) []string {
	if len(rows) < 2 {
		return nil
	}
	perExperiment := make([]map[string]string, 0, len(rows))
	fieldSet := map[string]struct{}{}
	for _, row := range rows {
		fields := recipeFields(row.exp.GetRecipe())
		perExperiment = append(perExperiment, fields)
		for field := range fields {
			fieldSet[field] = struct{}{}
		}
	}
	fields := make([]string, 0, len(fieldSet))
	for field := range fieldSet {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	var out []string
	for _, field := range fields {
		first := perExperiment[0][field]
		changed := false
		for _, values := range perExperiment[1:] {
			if values[field] != first {
				changed = true
				break
			}
		}
		if !changed {
			continue
		}
		if len(rows) == 2 {
			out = append(out, fmt.Sprintf("%s: %s -> %s", field, valueOrDash(perExperiment[0][field]), valueOrDash(perExperiment[1][field])))
			continue
		}
		parts := make([]string, 0, len(rows))
		for i, row := range rows {
			parts = append(parts, fmt.Sprintf("%s=%s", experimentShortID(row.exp), valueOrDash(perExperiment[i][field])))
		}
		out = append(out, fmt.Sprintf("%s: %s", field, strings.Join(parts, ", ")))
	}
	return out
}

func recipeFields(recipe *experimentv1.ExperimentRecipe) map[string]string {
	fields := map[string]string{}
	if recipe == nil {
		return fields
	}
	fields["clip_ids"] = strings.Join(recipe.GetClipIds(), ",")
	fields["realtime_repeats"] = strconv.Itoa(int(recipe.GetRealtimeRepeats()))
	fields["chunk_ms"] = strconv.Itoa(int(recipe.GetChunkMs()))
	fields["seed"] = strconv.FormatInt(recipe.GetSeed(), 10)
	fields["dropped_span_threshold_words"] = strconv.Itoa(int(recipe.GetDroppedSpanThresholdWords()))
	fields["latency_tail_seconds"] = strconv.Itoa(int(recipe.GetLatencyTailSeconds()))
	if lf := recipe.GetLongForm(); lf != nil {
		fields["long_form.enabled"] = strconv.FormatBool(lf.GetEnabled())
		fields["long_form.target_duration_seconds"] = strconv.Itoa(int(lf.GetTargetDurationSeconds()))
		fields["long_form.gap_ms"] = strconv.Itoa(int(lf.GetGapMs()))
		fields["long_form.tag_contains"] = lf.GetTagContains()
		fields["long_form.sweep_durations_seconds"] = int32sCSV(lf.GetSweepDurationsSeconds())
	}
	if aug := recipe.GetAugmentation(); aug != nil {
		fields["augmentation.noise_types"] = strings.Join(aug.GetNoiseTypes(), ",")
		fields["augmentation.snr_db"] = float64sCSV(aug.GetSnrDb())
		fields["augmentation.competing_voice_ids"] = strings.Join(aug.GetCompetingVoiceIds(), ",")
		fields["augmentation.competing_text"] = aug.GetCompetingText()
	}
	if speaker := recipe.GetSpeaker(); speaker != nil {
		fields["speaker.target_profile_id"] = speaker.GetTargetProfileId()
		fields["speaker.extraction_enabled"] = strconv.FormatBool(speaker.GetExtractionEnabled())
		fields["speaker.verification_enabled"] = strconv.FormatBool(speaker.GetVerificationEnabled())
		fields["speaker.verification_mode"] = speaker.GetVerificationMode().String()
		fields["speaker.threshold"] = fmt.Sprintf("%.3g", speaker.GetThreshold())
		fields["speaker.fallback_without_verification"] = strconv.FormatBool(speaker.GetFallbackWithoutVerification())
		fields["speaker.ablation_enabled"] = strconv.FormatBool(speaker.GetAblationEnabled())
	}
	for _, strategy := range recipe.GetStrategies() {
		key := strategyDiffKey(strategy, fields)
		fields[key+".kind"] = strategy.GetKind()
		fields[key+".overlap_max_window_ms"] = strconv.Itoa(int(strategy.GetOverlapMaxWindowMs()))
		fields[key+".overlap_max_stall_rejects"] = strconv.Itoa(int(strategy.GetOverlapMaxStallRejects()))
		fields[key+".overlap_window_ms"] = strconv.Itoa(int(strategy.GetOverlapWindowMs()))
		fields[key+".overlap_commit_runs"] = strconv.Itoa(int(strategy.GetOverlapCommitRuns()))
		fields[key+".vad_silence_ms"] = strconv.Itoa(int(strategy.GetVadSilenceMs()))
	}
	return fields
}

func strategyDiffKey(strategy *evalv1.EvalStrategy, fields map[string]string) string {
	base := "strategy." + valueOrDash(strategy.GetKind())
	key := base
	for i := 2; ; i++ {
		if _, exists := fields[key+".kind"]; !exists {
			return key
		}
		key = fmt.Sprintf("%s[%d]", base, i)
	}
}

func printStrategyAlignment(ctx cliapp.RunContext, rows []comparisonRow) {
	keys := alignedStrategyKeys(rows)
	if len(keys) == 0 {
		return
	}
	fmt.Fprintln(ctx.Stdout(), "\nBy-strategy alignment:")
	fmt.Fprintf(ctx.Stdout(), "  %-28s", "STRATEGY")
	for _, row := range rows {
		fmt.Fprintf(ctx.Stdout(), "  %-24s", experimentShortID(row.exp))
	}
	fmt.Fprintln(ctx.Stdout())
	for _, key := range keys {
		fmt.Fprintf(ctx.Stdout(), "  %-28s", truncate(key, 28))
		for _, row := range rows {
			fmt.Fprintf(ctx.Stdout(), "  %-24s", strategyMetricCell(row, key))
		}
		fmt.Fprintln(ctx.Stdout())
	}
}

func alignedStrategyKeys(rows []comparisonRow) []string {
	seen := map[string]struct{}{}
	for _, row := range rows {
		if row.report == nil {
			continue
		}
		for _, strategy := range row.report.GetPerStrategy() {
			seen[strategyAlignmentKey(strategy)] = struct{}{}
		}
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func strategyMetricCell(row comparisonRow, key string) string {
	if row.report == nil {
		return "-"
	}
	for _, strategy := range row.report.GetPerStrategy() {
		if strategyAlignmentKey(strategy) != key {
			continue
		}
		p95 := "-"
		if row.report.GetLatencyMeasured() {
			p95 = fmt.Sprintf("%.0fms", strategy.GetFinalizationLatencyP95Ms())
		}
		return fmt.Sprintf("wer %.1f p95 %s", strategy.GetWer()*100, p95)
	}
	return "-"
}

func strategyAlignmentKey(strategy *evalv1.StrategyReport) string {
	if strategy.GetStrategy() != "" {
		return strings.TrimSpace(strings.SplitN(strategy.GetStrategy(), "/", 2)[0])
	}
	if strategy.GetLabel() != "" {
		return strings.TrimSpace(strings.SplitN(strategy.GetLabel(), "/", 2)[0])
	}
	return "(unknown)"
}

func experimentShortID(exp *experimentv1.Experiment) string {
	if exp == nil {
		return "(nil)"
	}
	if name := strings.TrimSpace(exp.GetName()); name != "" {
		return truncate(name, 18)
	}
	return truncate(exp.GetId(), 18)
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func printExperimentErrors(ctx cliapp.RunContext, experiments []*experimentv1.Experiment) {
	var lines []string
	for _, exp := range experiments {
		if exp == nil {
			continue
		}
		if errMsg := strings.TrimSpace(exp.GetError()); errMsg != "" {
			lines = append(lines, fmt.Sprintf("  - %s: %s", exp.GetId(), errMsg))
		}
	}
	if len(lines) == 0 {
		return
	}
	fmt.Fprintln(ctx.Stdout(), "\nExperiment errors:")
	for _, line := range lines {
		fmt.Fprintln(ctx.Stdout(), line)
	}
}

// winnerStrategyRow returns the report's winning strategy row, preferring the
// summary's declared winner, then a verdict=="winner" row, then lowest WER.
func winnerStrategyRow(report *evalv1.EvalReport) *evalv1.StrategyReport {
	if report == nil {
		return nil
	}
	rows := report.GetPerStrategy()
	if len(rows) == 0 {
		return nil
	}
	if summary := report.GetSummary(); summary != nil {
		if ws := summary.GetWinnerStrategy(); ws != "" {
			for _, r := range rows {
				if r.GetStrategy() == ws {
					return r
				}
			}
		}
		if wl := summary.GetWinnerLabel(); wl != "" {
			for _, r := range rows {
				if r.GetLabel() == wl {
					return r
				}
			}
		}
	}
	for _, r := range rows {
		if r.GetVerdict() == "winner" {
			return r
		}
	}
	best := rows[0]
	for _, r := range rows[1:] {
		if r.GetWer() < best.GetWer() {
			best = r
		}
	}
	return best
}

func safetyLabel(safety *evalv1.SafetyGateReport) string {
	if safety == nil {
		return "-"
	}
	if safety.GetPassed() {
		return "SAFE"
	}
	return "UNSAFE"
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:max]
	}
	return s[:max-1] + "…"
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

func parseNonNegativeIntFlag(name, value string) (int, error) {
	n, err := parseIntFlag(name, value)
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, fmt.Errorf("--%s must be non-negative: %d", name, n)
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

// parseIntCSVFlag parses a comma-separated list of ints into []int32.
func parseIntCSVFlag(name, value string) ([]int32, error) {
	parts := splitCSV(value)
	out := make([]int32, 0, len(parts))
	for _, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("--%s must be comma-separated integers: %q", name, value)
		}
		out = append(out, int32(n))
	}
	return out, nil
}

// optionalBoolFlag reads a tri-state boolean flag. It returns set=false when the
// flag was not provided. When the flag is present with no/empty value
// (e.g. `--long-form` or `--long-form=`) it means true; an explicit
// `--long-form true|false` is honored. (cli-core's parser cannot express a flag
// that is both bare-able and value-accepting, so these stay valued flags; the
// bare-without-`=` form is rejected by the parser before the handler runs.)
func optionalBoolFlag(ctx cliapp.RunContext, name string) (set bool, value bool, err error) {
	if !ctx.BoolFlag(name) {
		return false, false, nil
	}
	raw := strings.TrimSpace(ctx.Flag(name))
	if raw == "" {
		return true, true, nil
	}
	parsed, err := parseBoolFlag(name, raw)
	if err != nil {
		return false, false, err
	}
	return true, parsed, nil
}

func longFormHasContent(lf *experimentv1.LongFormRecipe) bool {
	return lf.GetEnabled() || lf.GetTargetDurationSeconds() > 0 || lf.GetGapMs() > 0 ||
		lf.GetTagContains() != "" || len(lf.GetSweepDurationsSeconds()) > 0
}

func augmentationHasContent(a *experimentv1.AugmentationRecipe) bool {
	return len(a.GetNoiseTypes()) > 0 || len(a.GetCompetingVoiceIds()) > 0 ||
		len(a.GetSnrDb()) > 0 || a.GetCompetingText() != ""
}

func speakerHasContent(s *experimentv1.SpeakerExperimentRecipe) bool {
	return s.GetTargetProfileId() != "" || s.GetExtractionEnabled() || s.GetVerificationEnabled() ||
		s.GetVerificationMode() != sttv1.SpeakerMode_SPEAKER_MODE_UNSPECIFIED || s.GetThreshold() != 0 ||
		s.GetFallbackWithoutVerification() || s.GetAblationEnabled()
}

func strategyUsesOverlap(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "overlap_agree", "overlap":
		return true
	default:
		return false
	}
}

func strategyUsesVADSilence(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "vad_segment", "vad":
		return true
	default:
		return false
	}
}

// loadBaseRecipe unmarshals a full ExperimentRecipe from --recipe-json (inline)
// or --recipe-file (path) via protojson, to be used as the base that individual
// flags then override. Returns an empty recipe when neither is set.
func loadBaseRecipe(ctx cliapp.RunContext) (*experimentv1.ExperimentRecipe, error) {
	inline := strings.TrimSpace(ctx.Flag("recipe-json"))
	file := strings.TrimSpace(ctx.Flag("recipe-file"))
	if inline != "" && file != "" {
		return nil, fmt.Errorf("--recipe-json and --recipe-file are mutually exclusive; pass only one")
	}
	recipe := &experimentv1.ExperimentRecipe{}
	raw := inline
	source := "--recipe-json"
	if file != "" {
		b, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("read --recipe-file: %w", err)
		}
		raw = string(b)
		source = "--recipe-file"
	}
	if strings.TrimSpace(raw) == "" {
		return recipe, nil
	}
	if err := protojson.Unmarshal([]byte(raw), recipe); err != nil {
		return nil, fmt.Errorf("parse %s as ExperimentRecipe JSON: %w", source, err)
	}
	return recipe, nil
}

// flattenCSV splits each argument on commas and concatenates the results.
func flattenCSV(args []string) []string {
	var out []string
	for _, arg := range args {
		out = append(out, splitCSV(arg)...)
	}
	return out
}

// dedupeStrings drops empties and duplicates while preserving first-seen order.
func dedupeStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	var out []string
	for _, v := range in {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func int32sCSV(values []int32) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = strconv.Itoa(int(v))
	}
	return strings.Join(parts, ",")
}

func float64sCSV(values []float64) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = strconv.FormatFloat(v, 'f', -1, 64)
	}
	return strings.Join(parts, ",")
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
