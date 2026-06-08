package reindex

import (
	"context"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control"
	controlconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control/control_v1connect"
)

// controlTokenEnv is the operator escape hatch for the control token. The token
// is normally minted by search-hub and held in the API process; an operator who
// drives reindex locally supplies it via --control-token or this env var. Without
// a matching token the server rejects the call — the control token is the only
// gate (there is no token-free control verb).
const controlTokenEnv = "CLI_HEALTH_SEARCH_CONTROL_TOKEN"

type handlers struct {
	core   *cliapp.ScenarioApp
	client controlconnect.SearchControlServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: controlconnect.NewSearchControlServiceClient(httpClient, baseURL),
	}
}

// controlToken resolves the control token from the --control-token flag, falling
// back to the CLI_HEALTH_SEARCH_CONTROL_TOKEN env var.
func controlToken(ctx cliapp.RunContext) string {
	if t := strings.TrimSpace(ctx.Flag("control-token")); t != "" {
		return t
	}
	return strings.TrimSpace(os.Getenv(controlTokenEnv))
}

// run calls the shared SearchControlService.Reindex method. The legacy
// --scenario flag maps onto the contract's provider-defined `scope` filter.
func (h *handlers) run(ctx cliapp.RunContext) error {
	resp, err := h.client.Reindex(context.Background(), connect.NewRequest(&controlv1.ReindexRequest{
		Scope:        ctx.Flag("scenario"),
		DryRun:       ctx.BoolFlag("dry-run"),
		ControlToken: controlToken(ctx),
	}))
	if err != nil {
		return cliapp.WrapAPIError("reindex run", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no reindex response")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Started reindex job %s (dry_run=%v).", resp.Msg.JobId, resp.Msg.DryRun)},
		Changes: []string{
			fmt.Sprintf("planned_upserts=%d", resp.Msg.PlannedUpserts),
			fmt.Sprintf("planned_deletes=%d", resp.Msg.PlannedDeletes),
		},
		NextCommand: []string{
			fmt.Sprintf("`reindex status %s` — poll job progress", resp.Msg.JobId),
			fmt.Sprintf("`reindex cancel %s` — cooperatively cancel the job", resp.Msg.JobId),
		},
	})
}

// status calls the shared SearchControlService.ReindexStatus method.
func (h *handlers) status(ctx cliapp.RunContext) error {
	jobID := ctx.Positional("job_id")
	resp, err := h.client.ReindexStatus(context.Background(), connect.NewRequest(&controlv1.ReindexStatusRequest{
		JobId:        jobID,
		ControlToken: controlToken(ctx),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("reindex status %q", jobID), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no status response")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Job %s: state=%s processed=%d/%d", resp.Msg.JobId, resp.Msg.State, resp.Msg.Processed, resp.Msg.Total),
		},
		ResultsHeading: "Status",
		Results:        []string{},
	})
}

// cancel calls the shared SearchControlService.ReindexCancel method.
func (h *handlers) cancel(ctx cliapp.RunContext) error {
	jobID := ctx.Positional("job_id")
	resp, err := h.client.ReindexCancel(context.Background(), connect.NewRequest(&controlv1.ReindexCancelRequest{
		JobId:        jobID,
		ControlToken: controlToken(ctx),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("reindex cancel %q", jobID), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no cancel response")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Cancel job %s: cancelled=%v", resp.Msg.JobId, resp.Msg.Cancelled)},
	})
}
