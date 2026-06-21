package budget

import (
	"context"
	"fmt"
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
	b := &budgetsv1.Budget{
		Scenario:                scenario,
		GoBuildMaxMs:            parseInt64(firstFlag(ctx.FlagValues("go-build-max-ms"))),
		UiBuildMaxMs:            parseInt64(firstFlag(ctx.FlagValues("ui-build-max-ms"))),
		BundleMaxBytes:          parseInt64(firstFlag(ctx.FlagValues("bundle-max-bytes"))),
		LcpMaxMs:                parseInt64(firstFlag(ctx.FlagValues("lcp-max-ms"))),
		P95MaxMs:                parseInt64(firstFlag(ctx.FlagValues("p95-max-ms"))),
		StartupMaxMs:            parseInt64(firstFlag(ctx.FlagValues("startup-max-ms"))),
		ComponentCommitAvgMaxMs: parseFloat(firstFlag(ctx.FlagValues("component-commit-avg-max-ms"))),
		ComponentCommitMaxMs:    parseFloat(firstFlag(ctx.FlagValues("component-commit-max-ms"))),
		Ratchet:                 ctx.BoolFlag("ratchet"),
	}
	resp, err := h.client.SetBudget(context.Background(), connect.NewRequest(&budgetsv1.SetBudgetRequest{Budget: b}))
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
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("%s budget for %s.", verb, scenario)},
		Changes: budgetLines(resp.Msg.GetBudget()),
	})
}

// check evaluates a scenario's measurements against its budget.
func (h *handlers) check(ctx cliapp.RunContext) error {
	scenario := ctx.Positional("scenario")
	resp, err := h.client.CheckBudget(context.Background(), connect.NewRequest(&budgetsv1.CheckBudgetRequest{Scenario: scenario}))
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
		results = append(results, fmt.Sprintf("%s — measured=%d budget=%d %s", v.GetAxis(), v.GetMeasured(), v.GetBudget(), v.GetUnit()))
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
	return []string{
		fmt.Sprintf("go_build_max=%dms", b.GetGoBuildMaxMs()),
		fmt.Sprintf("ui_build_max=%dms", b.GetUiBuildMaxMs()),
		fmt.Sprintf("bundle_max=%dB", b.GetBundleMaxBytes()),
		fmt.Sprintf("lcp_max=%dms", b.GetLcpMaxMs()),
		fmt.Sprintf("p95_max=%dms", b.GetP95MaxMs()),
		fmt.Sprintf("startup_max=%dms", b.GetStartupMaxMs()),
		fmt.Sprintf("component_commit_avg_max=%.1fms", b.GetComponentCommitAvgMaxMs()),
		fmt.Sprintf("component_commit_max=%.1fms", b.GetComponentCommitMaxMs()),
		fmt.Sprintf("ratchet=%t", b.GetRatchet()),
	}
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
