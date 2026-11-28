package runs

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/runs/runs_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client runsconnect.RunsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{core: core, client: runsconnect.NewRunsServiceClient(httpClient, baseURL)}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	req := &runsv1.ListRunsRequest{FlowId: ctx.Flag("flow")}
	if raw := ctx.Flag("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return fmt.Errorf("--limit must be a non-negative integer; got %q", raw)
		}
		req.Limit = uint32(n)
	}
	resp, err := h.client.ListRuns(context.Background(), connect.NewRequest(req))
	if err != nil {
		return cliapp.WrapAPIError("list runs", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Runs))
	for _, r := range resp.Msg.Runs {
		results = append(results, formatRun(r))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d run(s).", len(resp.Msg.Runs))},
		ResultsHeading: "Runs",
		Results:        results,
		RetrievalHints: []string{"`runs show <run-id>` — show one run with full output"},
	})
}

func (h *handlers) show(ctx cliapp.RunContext) error {
	id := ctx.Positional("run-id")
	resp, err := h.client.GetRun(context.Background(), connect.NewRequest(&runsv1.GetRunRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get run %q", id), err, nil)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Run %s", resp.Msg.Run.Id)},
		ResultsHeading: "Detail",
		Results: []string{
			formatRun(resp.Msg.Run),
			"output:\n" + resp.Msg.Run.Output,
		},
	})
}

func formatRun(r *runsv1.Run) string {
	if r == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s | flow=%s | %s | %s | duration=%dms", r.Id, r.FlowId, r.Status, r.Mode, r.DurationMs)
}
