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
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s: LCP=%dms FCP=%dms long-task=%dms, %d finding(s).", msg.GetScenario(), msg.GetLcpMs(), msg.GetFcpMs(), msg.GetLongTaskMs(), len(msg.GetFindings()))},
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
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s: long-task Δ%+dms, LCP Δ%+dms.", msg.GetScenario(), msg.GetLongTaskDeltaMs(), msg.GetLcpDeltaMs())},
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
