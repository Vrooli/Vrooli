package trials

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	trialsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/trials"
	trialsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/trials/trials_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client trialsconnect.TrialsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{core: core, client: trialsconnect.NewTrialsServiceClient(httpClient, baseURL)}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListTrialTasks(context.Background(), connect.NewRequest(&trialsv1.ListTrialTasksRequest{
		Suite: strings.TrimSpace(ctx.Flag("suite")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("trials list", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no tasks response")
	}
	results := make([]string, 0, len(resp.Msg.Tasks))
	for _, t := range resp.Msg.Tasks {
		neg := ""
		if t.GetNegative() {
			neg = " (negative)"
		}
		results = append(results, fmt.Sprintf("%s [%s] %s%s", t.GetId(), t.GetSuite(), t.GetDescription(), neg))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d trial task(s).", len(resp.Msg.Tasks))},
		ResultsHeading: "Trial tasks",
		Results:        results,
		RetrievalHints: []string{"`trials run --task <id>` — dispatch one task (explicit, sandboxed)"},
	})
}

func (h *handlers) run(ctx cliapp.RunContext) error {
	task := strings.TrimSpace(ctx.Flag("task"))
	suite := strings.TrimSpace(ctx.Flag("suite"))
	all := ctx.BoolFlag("all")

	// Single-task by default: trials are expensive, so require an explicit scope
	// rather than silently dispatching the whole suite.
	if task == "" && suite == "" && !all {
		return fmt.Errorf("trials run needs a scope: pass --task <id> for one task, --suite <family> for a family, or --all for the full suite")
	}
	// --all means the whole generated suite: send neither task nor suite (the
	// service runs everything when both are empty). An explicit --task/--suite
	// takes precedence over --all, so no remapping is needed here.

	resp, err := h.client.RunTrials(context.Background(), connect.NewRequest(&trialsv1.RunTrialsRequest{
		Suite:  suite,
		TaskId: task,
	}))
	if err != nil {
		return cliapp.WrapAPIError("trials run", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no runs response")
	}
	results := make([]string, 0, len(resp.Msg.Runs))
	for _, r := range resp.Msg.Runs {
		results = append(results, formatRun(r))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Dispatched/recorded %d run(s). Identical recent runs are reused (idempotent by task+fixture-rev).", len(resp.Msg.Runs))},
		ResultsHeading: "Runs",
		Results:        results,
		RetrievalHints: []string{"`trials show <run-id>` — full run detail", "`trials history` — the trend"},
	})
}

func (h *handlers) history(ctx cliapp.RunContext) error {
	resp, err := h.client.GetTrialHistory(context.Background(), connect.NewRequest(&trialsv1.GetTrialHistoryRequest{
		TaskId: strings.TrimSpace(ctx.Flag("task")),
		Suite:  strings.TrimSpace(ctx.Flag("suite")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("trials history", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no history response")
	}
	results := make([]string, 0, len(resp.Msg.Points))
	for _, p := range resp.Msg.Points {
		results = append(results, fmt.Sprintf("%s — success=%.0f%% median_tokens=%d median_ms=%d runs=%d",
			p.GetAt().AsTime().Format("2006-01-02"), p.GetSuccessRate()*100, p.GetMedianTokens(), p.GetMedianDurationMs(), p.GetRunCount()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d trend point(s), %d recent run(s).", len(resp.Msg.Points), len(resp.Msg.RecentRuns))},
		ResultsHeading: "Trial history",
		Results:        results,
	})
}

func (h *handlers) show(ctx cliapp.RunContext) error {
	id := strings.TrimSpace(ctx.Positional("run-id"))
	resp, err := h.client.GetTrialRun(context.Background(), connect.NewRequest(&trialsv1.GetTrialRunRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("trials show %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Run == nil {
		return fmt.Errorf("server returned no run")
	}
	r := resp.Msg.Run
	results := []string{
		formatRun(r),
		fmt.Sprintf("  sandbox diff: %s", emptyDash(r.GetSandboxDiffRef())),
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Run %s.", r.GetId())},
		ResultsHeading: "Run",
		Results:        results,
	})
}

func (h *handlers) coverage(ctx cliapp.RunContext) error {
	resp, err := h.client.GetGateCoverage(context.Background(), connect.NewRequest(&trialsv1.GetGateCoverageRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("trials coverage", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no coverage response")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{fmt.Sprintf("Guide-gate coverage: %.0f%% (%d/%d Guide tasks have a live gate).",
			resp.Msg.GetGateCoverageRatio()*100, resp.Msg.GetGuideTasksWithGate(), resp.Msg.GetGuideTasksTotal())},
		ResultsHeading: "Gate coverage",
	})
}

func formatRun(r *trialsv1.TrialRun) string {
	return fmt.Sprintf("%s [%s] %s tokens=%d duration_ms=%d @ %s",
		r.GetId(), verdictLabel(r.GetVerdict()), r.GetSuite(), r.GetTokens(), r.GetDurationMs(),
		r.GetAt().AsTime().Format("2006-01-02 15:04"))
}

func verdictLabel(v trialsv1.TrialVerdict) string {
	switch v {
	case trialsv1.TrialVerdict_TRIAL_VERDICT_PASS:
		return "pass"
	case trialsv1.TrialVerdict_TRIAL_VERDICT_FAIL:
		return "fail"
	case trialsv1.TrialVerdict_TRIAL_VERDICT_ERROR:
		return "error"
	default:
		return "?"
	}
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}
