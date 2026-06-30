package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
	evalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval/eval_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client evalconnect.EvalServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: evalconnect.NewEvalServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) run(ctx cliapp.RunContext) error {
	req := &evalv1.RunEvalRequest{}
	for _, kind := range splitCSV(ctx.Flag("strategies")) {
		req.Strategies = append(req.Strategies, &evalv1.EvalStrategy{Kind: kind, OverlapMaxStallRejects: -1})
	}
	if v := strings.TrimSpace(ctx.Flag("overlap-max-window-ms")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("--overlap-max-window-ms must be an integer: %q", v)
		}
		ensureDefaultStrategies(req)
		for _, s := range req.Strategies {
			s.OverlapMaxWindowMs = int32(n)
		}
	}
	req.ClipIds = splitCSV(ctx.Flag("clip-ids"))
	if v := strings.TrimSpace(ctx.Flag("realtime-repeats")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("--realtime-repeats must be an integer: %q", v)
		}
		req.RealtimeRepeats = int32(n)
	}
	if v := strings.TrimSpace(ctx.Flag("chunk-ms")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("--chunk-ms must be an integer: %q", v)
		}
		req.ChunkMs = int32(n)
	}

	resp, err := h.client.RunEval(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("run-eval", err, nil)
	}
	report := resp.Msg.GetReport()

	if out := strings.TrimSpace(ctx.Flag("output")); out != "" {
		data, jerr := json.MarshalIndent(report, "", "  ")
		if jerr != nil {
			return fmt.Errorf("marshal report: %w", jerr)
		}
		if werr := os.WriteFile(out, data, 0o644); werr != nil {
			return fmt.Errorf("write report: %w", werr)
		}
		fmt.Fprintf(ctx.Stdout(), "Wrote report JSON to %s.\n", out)
	}

	if strings.EqualFold(ctx.Flag("format"), "json") {
		data, jerr := json.MarshalIndent(report, "", "  ")
		if jerr != nil {
			return fmt.Errorf("marshal report: %w", jerr)
		}
		fmt.Fprintln(ctx.Stdout(), string(data))
		return nil
	}

	printReportTable(ctx, report)
	return nil
}

func ensureDefaultStrategies(req *evalv1.RunEvalRequest) {
	if len(req.GetStrategies()) > 0 {
		return
	}
	req.Strategies = []*evalv1.EvalStrategy{
		{Kind: "batch", OverlapMaxStallRejects: -1},
		{Kind: "vad_segment", OverlapMaxStallRejects: -1},
		{Kind: "overlap_agree", OverlapMaxStallRejects: -1},
	}
}

func printReportTable(ctx cliapp.RunContext, report *evalv1.EvalReport) {
	if report == nil || len(report.GetPerStrategy()) == 0 {
		fmt.Fprintln(ctx.Stdout(), "No strategies evaluated.")
		return
	}
	fmt.Fprintf(ctx.Stdout(), "%-24s  %6s  %6s  %6s  %8s  %9s  %9s  %8s\n",
		"STRATEGY", "WER%", "CALLS", "RTF", "AUDIO_S", "LAT_P50", "LAT_P95", "REVISES")
	for _, s := range report.GetPerStrategy() {
		lat50, lat95 := "-", "-"
		if report.GetLatencyMeasured() {
			lat50 = fmt.Sprintf("%.0fms", s.GetFinalizationLatencyP50Ms())
			lat95 = fmt.Sprintf("%.0fms", s.GetFinalizationLatencyP95Ms())
		}
		fmt.Fprintf(ctx.Stdout(), "%-24s  %6.1f  %6d  %6.2f  %8.2f  %9s  %9s  %8d\n",
			s.GetLabel(), s.GetWer()*100, s.GetWhisperCalls(), s.GetRtf(),
			s.GetWhisperAudioSeconds(), lat50, lat95, s.GetPartialRevisions())
	}
	if !report.GetLatencyMeasured() {
		fmt.Fprintln(ctx.Stdout(), "\n(latency columns omitted — pass --realtime-repeats N to measure finalization latency)")
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
