package reindex

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	reindexv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/reindex"
	reindexconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/reindex/reindex_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client reindexconnect.ReindexServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: reindexconnect.NewReindexServiceClient(httpClient, baseURL),
	}
}

// run calls the generated Connect ReindexService.Reindex method.
func (h *handlers) run(ctx cliapp.RunContext) error {
	resp, err := h.client.Reindex(context.Background(), connect.NewRequest(&reindexv1.ReindexRequest{
		Scenario: ctx.Flag("scenario"),
		DryRun:   ctx.BoolFlag("dry-run"),
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

// status calls the generated Connect ReindexService.ReindexStatus method.
func (h *handlers) status(ctx cliapp.RunContext) error {
	jobID := ctx.Positional("job_id")
	resp, err := h.client.ReindexStatus(context.Background(), connect.NewRequest(&reindexv1.ReindexStatusRequest{
		JobId: jobID,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("reindex status %q", jobID), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no status response")
	}
	summary := []string{
		fmt.Sprintf("Job %s: state=%s processed=%d/%d", resp.Msg.JobId, resp.Msg.State, resp.Msg.Processed, resp.Msg.Total),
	}
	if resp.Msg.Error != "" {
		summary = append(summary, fmt.Sprintf("error: %s", resp.Msg.Error))
	}
	results := []string{}
	for _, w := range resp.Msg.GetWarnings() {
		results = append(results, "[WARN] "+w)
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Status",
		Results:        results,
	})
}

// cancel calls the generated Connect ReindexService.ReindexCancel method.
func (h *handlers) cancel(ctx cliapp.RunContext) error {
	jobID := ctx.Positional("job_id")
	resp, err := h.client.ReindexCancel(context.Background(), connect.NewRequest(&reindexv1.ReindexCancelRequest{
		JobId: jobID,
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
