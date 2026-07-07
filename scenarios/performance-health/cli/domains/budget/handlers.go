package budget

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	budgetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/budgets"
	budgetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/budgets/budgets_v1connect"
)

type handlers struct {
	client budgetsconnect.BudgetServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: budgetsconnect.NewBudgetServiceClient(httpClient, baseURL)}
}

// get reads a scenario's declared budget.
func (h *handlers) get(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.GetBudget(context.Background(), connect.NewRequest(&budgetsv1.GetBudgetRequest{Scenario: scenario}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get budget for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no budget response")
	}
	declared := "no budget declared (defaults shown)"
	if resp.Msg.GetDeclared() {
		declared = "declared"
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s: %s.", scenario, declared)},
		ResultsHeading: "Budget",
		Results:        budgetLines(resp.Msg.GetBudget()),
	})
}

// set writes/updates a scenario's budget.
func (h *handlers) set(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	flow := strings.TrimSpace(firstFlag(ctx.FlagValues("flow")))
	b := &budgetsv1.Budget{
		Scenario:                scenario,
		GoBuildMaxMs:            parseInt64(firstFlag(ctx.FlagValues("go-build-max-ms"))),
		UiBuildMaxMs:            parseInt64(firstFlag(ctx.FlagValues("ui-build-max-ms"))),
		BundleMaxBytes:          parseInt64(firstFlag(ctx.FlagValues("bundle-max-bytes"))),
		LcpMaxMs:                parseInt64(firstFlag(ctx.FlagValues("lcp-max-ms"))),
		StartupMaxMs:            parseInt64(firstFlag(ctx.FlagValues("startup-max-ms"))),
		ComponentCommitAvgMaxMs: parseFloat(firstFlag(ctx.FlagValues("component-commit-avg-max-ms"))),
		ComponentCommitMaxMs:    parseFloat(firstFlag(ctx.FlagValues("component-commit-max-ms"))),
		DrawnFpsMin:             parseFloat(firstFlag(ctx.FlagValues("drawn-fps-min"))),
		DroppedFrameRateMax:     parseFloat(firstFlag(ctx.FlagValues("dropped-frame-rate-max"))),
		LongTaskTotalMaxMs:      parseInt64(firstFlag(ctx.FlagValues("long-task-total-max-ms"))),
		LongTaskMaxMs:           parseFloat(firstFlag(ctx.FlagValues("long-task-max-ms"))),
		RasterTotalMaxMs:        parseFloat(firstFlag(ctx.FlagValues("raster-total-max-ms"))),
		LayoutTotalMaxMs:        parseFloat(firstFlag(ctx.FlagValues("layout-total-max-ms"))),
		PaintTotalMaxMs:         parseFloat(firstFlag(ctx.FlagValues("paint-total-max-ms"))),
		InputEventCountMin:      parseInt64(firstFlag(ctx.FlagValues("input-event-count-min"))),
		LoadOnly:                ctx.BoolFlag("load-only"),
		Ratchet:                 ctx.BoolFlag("ratchet"),
	}
	resp, err := h.client.SetBudget(context.Background(), connect.NewRequest(&budgetsv1.SetBudgetRequest{Budget: b, Flow: flow}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("set budget for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no budget response")
	}
	verb := "Set"
	if resp.Msg.GetDryRun() {
		verb = "Validated (dry-run)"
	}
	target := scenario
	if flow != "" {
		target = fmt.Sprintf("%s flow %q", scenario, flow)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("%s budget for %s.", verb, target)},
		Changes: budgetLines(resp.Msg.GetBudget()),
	})
}

// check evaluates a scenario's measurements against its budget (or one flow's
// per-flow budget when --flow is set).
func (h *handlers) check(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	flow := strings.TrimSpace(firstFlag(ctx.FlagValues("flow")))
	resp, err := h.client.CheckBudget(context.Background(), connect.NewRequest(&budgetsv1.CheckBudgetRequest{Scenario: scenario, Flow: flow}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("check budget for %q", scenario), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no check response")
	}
	verdict := "within budget"
	if !resp.Msg.GetPassed() {
		verdict = "OVER BUDGET"
	}
	results := make([]string, 0, len(resp.Msg.GetViolations()))
	for _, v := range resp.Msg.GetViolations() {
		relation := "max"
		if v.GetMode() == "min" {
			relation = "min"
		}
		detail := ""
		if v.GetDetail() != "" {
			detail = " — " + v.GetDetail()
		}
		results = append(results, fmt.Sprintf("%s — measured=%d budget_%s=%d %s%s", v.GetAxis(), v.GetMeasured(), relation, v.GetBudget(), v.GetUnit(), detail))
	}
	if len(results) == 0 {
		results = append(results, "No violations.")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%s: %s.", scenario, verdict)},
		ResultsHeading: "Violations",
		Results:        results,
	})
}

func budgetLines(b *budgetsv1.Budget) []string {
	if b == nil {
		return []string{"(no budget)"}
	}
	lines := []string{
		fmt.Sprintf("go_build_max=%dms", b.GetGoBuildMaxMs()),
		fmt.Sprintf("ui_build_max=%dms", b.GetUiBuildMaxMs()),
		fmt.Sprintf("bundle_max=%dB", b.GetBundleMaxBytes()),
		fmt.Sprintf("lcp_max=%dms", b.GetLcpMaxMs()),
		fmt.Sprintf("startup_max=%dms", b.GetStartupMaxMs()),
		fmt.Sprintf("component_commit_avg_max=%.1fms", b.GetComponentCommitAvgMaxMs()),
		fmt.Sprintf("component_commit_max=%.1fms", b.GetComponentCommitMaxMs()),
		fmt.Sprintf("drawn_fps_min=%.1ffps", b.GetDrawnFpsMin()),
		fmt.Sprintf("dropped_frame_rate_max=%.2f", b.GetDroppedFrameRateMax()),
		fmt.Sprintf("long_task_total_max=%dms", b.GetLongTaskTotalMaxMs()),
		fmt.Sprintf("long_task_max=%.1fms", b.GetLongTaskMaxMs()),
		fmt.Sprintf("raster_total_max=%.1fms", b.GetRasterTotalMaxMs()),
		fmt.Sprintf("layout_total_max=%.1fms", b.GetLayoutTotalMaxMs()),
		fmt.Sprintf("paint_total_max=%.1fms", b.GetPaintTotalMaxMs()),
		fmt.Sprintf("input_event_count_min=%d", b.GetInputEventCountMin()),
		fmt.Sprintf("load_only=%t", b.GetLoadOnly()),
		fmt.Sprintf("ratchet=%t", b.GetRatchet()),
	}
	flows := b.GetFlows()
	if len(flows) > 0 {
		slugs := make([]string, 0, len(flows))
		for slug := range flows {
			slugs = append(slugs, slug)
		}
		sort.Strings(slugs)
		for _, slug := range slugs {
			fb := flows[slug]
			lines = append(lines, fmt.Sprintf("flow[%s]: lcp_max=%dms component_commit_avg_max=%.1fms component_commit_max=%.1fms drawn_fps_min=%.1ffps dropped_frame_rate_max=%.2f long_task_total_max=%dms long_task_max=%.1fms raster_total_max=%.1fms layout_total_max=%.1fms paint_total_max=%.1fms input_event_count_min=%d load_only=%t",
				slug, fb.GetLcpMaxMs(), fb.GetComponentCommitAvgMaxMs(), fb.GetComponentCommitMaxMs(),
				fb.GetDrawnFpsMin(), fb.GetDroppedFrameRateMax(), fb.GetLongTaskTotalMaxMs(), fb.GetLongTaskMaxMs(),
				fb.GetRasterTotalMaxMs(), fb.GetLayoutTotalMaxMs(), fb.GetPaintTotalMaxMs(), fb.GetInputEventCountMin(), fb.GetLoadOnly()))
		}
	}
	return lines
}

func parseInt64(s string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func parseFloat(s string) float64 {
	n, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return n
}

func firstFlag(values []string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
