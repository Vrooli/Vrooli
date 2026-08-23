package analysis

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	analysisv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/analysis"
	analysisconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/analysis/analysis_v1connect"
)

type handlers struct {
	client analysisconnect.AnalysisServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: analysisconnect.NewAnalysisServiceClient(httpClient, baseURL)}
}

// analyze parses one trace into a per-component table plus findings.
func (h *handlers) analyze(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.AnalyzeTrace(context.Background(), connect.NewRequest(&analysisv1.AnalyzeTraceRequest{
		Scenario:      scenario,
		TraceArtifact: firstFlag(ctx.FlagValues("trace")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("analyze trace for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no analysis response")
	}
	msg := resp.Msg
	results := make([]string, 0, len(msg.GetComponents()))
	for _, c := range msg.GetComponents() {
		def := c.GetDefinition()
		if def == "" {
			def = "(definition not located)"
		}
		results = append(results, fmt.Sprintf("%s — count=%d avg=%.1fms max=%.1fms %s", c.GetComponent(), c.GetCommitCount(), c.GetAvgMs(), c.GetMaxMs(), def))
	}
	if len(results) == 0 {
		results = append(results, "No component timings in this trace.")
	}
	frame := msg.GetFrameSummary()
	if frame != nil {
		results = append(results, "", fmt.Sprintf("Frame health — duration=%.1fms begin=%d drawn=%d dropped=%d drawn-fps=%.1f dropped-rate=%.1f%%",
			frame.GetTraceDurationMs(), frame.GetBeginFrameCount(), frame.GetDrawnFrameCount(), frame.GetDroppedFrameCount(), frame.GetApproxDrawnFps(), frame.GetDroppedFrameRate()*100))
	}
	appendEventSummaryRows := func(heading string, rows []*analysisv1.EventSummary) {
		if len(rows) == 0 {
			return
		}
		results = append(results, "", heading+":")
		for _, row := range rows {
			results = append(results, fmt.Sprintf("  %s — count=%d total=%.1fms avg=%.1fms max=%.1fms", row.GetName(), row.GetCount(), row.GetTotalMs(), row.GetAvgMs(), row.GetMaxMs()))
		}
	}
	appendEventSummaryRows("Browser work", msg.GetBrowserWork())
	appendEventSummaryRows("Input events", msg.GetInputEvents())
	if len(msg.GetFindings()) > 0 {
		results = append(results, "", "Deterministic findings (component → file:line):")
		for _, f := range msg.GetFindings() {
			loc := f.GetDefinition()
			if loc == "" {
				loc = "definition not located"
			}
			results = append(results, fmt.Sprintf("  [%s] %s @ %s — %s", f.GetSeverity(), f.GetComponent(), loc, f.GetEvidence()))
		}
	}
	summary := []string{fmt.Sprintf("%s: LCP=%dms FCP=%dms CLS=%.3f long-task=%dms drawn-fps=%.1f dropped=%.1f%%, %d finding(s).", msg.GetScenario(), msg.GetLcpMs(), msg.GetFcpMs(), msg.GetCls(), msg.GetLongTaskMs(), frame.GetApproxDrawnFps(), frame.GetDroppedFrameRate()*100, len(msg.GetFindings()))}
	// Navigation gets its own line: five more numbers on the summary row stops
	// being readable, and the phases are only meaningful read together. Omitted
	// entirely when the observer never fired.
	if msg.GetLoadEventEndMs() > 0 || msg.GetResponseEndMs() > 0 {
		summary = append(summary, fmt.Sprintf("  navigation (%s): response-end=%dms dom-interactive=%dms dom-content-loaded=%dms load-event-end=%dms",
			navigationTypeOrUnknown(msg.GetNavigationType()), msg.GetResponseEndMs(), msg.GetDomInteractiveMs(), msg.GetDomContentLoadedMs(), msg.GetLoadEventEndMs()))
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Component commit profile",
		Results:        results,
	})
}

// compare diffs two traces of the same interaction.
func (h *handlers) compare(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.CompareTraces(context.Background(), connect.NewRequest(&analysisv1.CompareTracesRequest{
		Scenario:          scenario,
		BaselineArtifact:  firstFlag(ctx.FlagValues("baseline")),
		CandidateArtifact: firstFlag(ctx.FlagValues("candidate")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("compare traces for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no comparison response")
	}
	msg := resp.Msg
	results := make([]string, 0, len(msg.GetComponents()))
	for _, d := range msg.GetComponents() {
		results = append(results, fmt.Sprintf(
			"%s — count %d→%d (Δ%+d), avg %.1fms→%.1fms (Δ%+.1fms), max %.1fms→%.1fms (Δ%+.1fms)",
			d.GetComponent(),
			d.GetBaselineCount(), d.GetCandidateCount(), d.GetCountDelta(),
			d.GetBaselineAvgMs(), d.GetCandidateAvgMs(), d.GetDeltaMs(),
			d.GetBaselineMaxMs(), d.GetCandidateMaxMs(), d.GetMaxDeltaMs(),
		))
	}
	if len(results) == 0 {
		results = append(results, "No component deltas.")
	}
	if frame := msg.GetFrameDelta(); frame != nil {
		results = append(results, "", fmt.Sprintf("Frame health Δ — duration=%+.1fms begin=%+d drawn=%+d dropped=%+d drawn-fps=%+.1f dropped-rate=%+.1f%%",
			frame.GetTraceDurationDeltaMs(), frame.GetBeginFrameCountDelta(), frame.GetDrawnFrameCountDelta(), frame.GetDroppedFrameCountDelta(), frame.GetApproxDrawnFpsDelta(), frame.GetDroppedFrameRateDelta()*100))
	}
	appendEventDeltaRows := func(heading string, rows []*analysisv1.EventDelta) {
		if len(rows) == 0 {
			return
		}
		results = append(results, "", heading+":")
		for _, row := range rows {
			results = append(results, fmt.Sprintf("  %s — count %d→%d (Δ%+d), total %.1fms→%.1fms (Δ%+.1fms), avg %.1fms→%.1fms (Δ%+.1fms), max %.1fms→%.1fms (Δ%+.1fms)",
				row.GetName(),
				row.GetBaselineCount(), row.GetCandidateCount(), row.GetCountDelta(),
				row.GetBaselineTotalMs(), row.GetCandidateTotalMs(), row.GetTotalDeltaMs(),
				row.GetBaselineAvgMs(), row.GetCandidateAvgMs(), row.GetAvgDeltaMs(),
				row.GetBaselineMaxMs(), row.GetCandidateMaxMs(), row.GetMaxDeltaMs(),
			))
		}
	}
	appendEventDeltaRows("Browser work deltas", msg.GetBrowserWork())
	appendEventDeltaRows("Input event deltas", msg.GetInputEvents())
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s: long-task Δ%+dms, LCP Δ%+dms, drawn-fps Δ%+.1f.", msg.GetScenario(), msg.GetLongTaskDeltaMs(), msg.GetLcpDeltaMs(), msg.GetFrameDelta().GetApproxDrawnFpsDelta())},
		ResultsHeading: "Component deltas (largest regression first)",
		Results:        results,
	})
}

func firstFlag(values []string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// navigationTypeOrUnknown labels a sample whose navigation type the browser did
// not report, so the line never reads "navigation ()".
func navigationTypeOrUnknown(t string) string {
	if t == "" {
		return "unknown"
	}
	return t
}
