package reindex

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	reindexv1 "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/reindex"
	reindexconnect "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/reindex/reindex_v1connect"
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
	msg := resp.Msg
	next := []string{}
	if !msg.GetDryRun() && msg.GetJobId() != "" {
		next = []string{
			fmt.Sprintf("`reindex status %s` — poll job progress", msg.GetJobId()),
			fmt.Sprintf("`reindex cancel %s` — cooperatively cancel the job", msg.GetJobId()),
		}
	}
	result := fmt.Sprintf("Reindex planned: upserts=%d deletes=%d (dry_run=%v)", msg.GetPlannedUpserts(), msg.GetPlannedDeletes(), msg.GetDryRun())
	if !msg.GetDryRun() {
		result = fmt.Sprintf("Started reindex job %s — planned upserts=%d deletes=%d", msg.GetJobId(), msg.GetPlannedUpserts(), msg.GetPlannedDeletes())
	}
	return cliapp.RenderProtoMutation(ctx, msg, cliapp.MutationReport{
		Result:      []string{result},
		NextCommand: next,
	})
}

func (h *handlers) status(ctx cliapp.RunContext) error {
	jobID := ctx.Positional("job_id")
	resp, err := h.client.ReindexStatus(context.Background(), connect.NewRequest(&reindexv1.ReindexStatusRequest{JobId: jobID}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("reindex status %q", jobID), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no status response")
	}
	msg := resp.Msg
	summary := fmt.Sprintf("Job %s: state=%s processed=%d/%d", msg.GetJobId(), msg.GetState(), msg.GetProcessed(), msg.GetTotal())
	if msg.GetError() != "" {
		summary += " error=" + msg.GetError()
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        []string{summary},
		ResultsHeading: "Status",
		Results:        []string{},
	})
}

func (h *handlers) cancel(ctx cliapp.RunContext) error {
	jobID := ctx.Positional("job_id")
	resp, err := h.client.ReindexCancel(context.Background(), connect.NewRequest(&reindexv1.ReindexCancelRequest{JobId: jobID}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("reindex cancel %q", jobID), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no cancel response")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Cancel requested for job %s: cancelled=%v", resp.Msg.GetJobId(), resp.Msg.GetCancelled())},
	})
}
