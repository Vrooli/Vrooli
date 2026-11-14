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
func controlToken(ctx cliapp.OperationContext) string {
	if t := strings.TrimSpace(ctx.Flag("control-token")); t != "" {
		return t
	}
	return strings.TrimSpace(os.Getenv(controlTokenEnv))
}

// runCall calls the shared SearchControlService.Reindex method. The legacy
// --scenario flag maps onto the contract's provider-defined `scope` filter. It
// is the operation half of the proto_mutation primitive; runReport renders it.
func (h *handlers) runCall(ctx cliapp.OperationContext) (*controlv1.ReindexResponse, error) {
	resp, err := h.client.Reindex(context.Background(), connect.NewRequest(&controlv1.ReindexRequest{
		Scope:        ctx.Flag("scenario"),
		DryRun:       ctx.BoolFlag("dry-run"),
		ControlToken: controlToken(ctx),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("reindex run", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no reindex response")
	}
	return resp.Msg, nil
}

// runReport maps the reindex response to the human MutationReport.
func (h *handlers) runReport(_ cliapp.OperationContext, msg *controlv1.ReindexResponse) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Started reindex job %s (dry_run=%v).", msg.JobId, msg.DryRun)},
		Changes: []string{
			fmt.Sprintf("planned_upserts=%d", msg.PlannedUpserts),
			fmt.Sprintf("planned_deletes=%d", msg.PlannedDeletes),
		},
		NextCommand: []string{
			fmt.Sprintf("`reindex status %s` — poll job progress", msg.JobId),
			fmt.Sprintf("`reindex cancel %s` — cooperatively cancel the job", msg.JobId),
		},
	}
}

// statusCall runs SearchControlService.ReindexStatus (operation half of the
// proto_list primitive); statusReport renders it.
func (h *handlers) statusCall(ctx cliapp.OperationContext) (*controlv1.ReindexStatusResponse, error) {
	jobID := ctx.Positional("job_id")
	resp, err := h.client.ReindexStatus(context.Background(), connect.NewRequest(&controlv1.ReindexStatusRequest{
		JobId:        jobID,
		ControlToken: controlToken(ctx),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError(fmt.Sprintf("reindex status %q", jobID), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no status response")
	}
	return resp.Msg, nil
}

// statusReport maps the status response to the human ListReport.
func (h *handlers) statusReport(_ cliapp.OperationContext, msg *controlv1.ReindexStatusResponse) cliapp.ListReport {
	return cliapp.ListReport{
		Summary: []string{
			fmt.Sprintf("Job %s: state=%s processed=%d/%d", msg.JobId, msg.State, msg.Processed, msg.Total),
		},
		ResultsHeading: "Status",
		Results:        []string{},
	}
}

// cancelCall runs SearchControlService.ReindexCancel (operation half of the
// proto_mutation primitive); cancelReport renders it.
func (h *handlers) cancelCall(ctx cliapp.OperationContext) (*controlv1.ReindexCancelResponse, error) {
	jobID := ctx.Positional("job_id")
	resp, err := h.client.ReindexCancel(context.Background(), connect.NewRequest(&controlv1.ReindexCancelRequest{
		JobId:        jobID,
		ControlToken: controlToken(ctx),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError(fmt.Sprintf("reindex cancel %q", jobID), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no cancel response")
	}
	return resp.Msg, nil
}

// cancelReport maps the cancel response to the human MutationReport.
func (h *handlers) cancelReport(_ cliapp.OperationContext, msg *controlv1.ReindexCancelResponse) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Cancel job %s: cancelled=%v", msg.JobId, msg.Cancelled)},
	}
}
