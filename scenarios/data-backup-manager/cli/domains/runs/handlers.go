package runs

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/runs/runs_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client runsconnect.RunsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: runsconnect.NewRunsServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) trigger(ctx cliapp.RunContext) error {
	resp, err := h.client.TriggerRun(context.Background(), connect.NewRequest(&runsv1.TriggerRunRequest{
		PlanId: ctx.Flag("plan"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("trigger run", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Run == nil {
		return fmt.Errorf("server returned no run")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Triggered run %s.", resp.Msg.Run.Id)},
		Changes: []string{formatRun(resp.Msg.Run)},
		NextCommand: []string{
			fmt.Sprintf("`runs get %s` — show run status", resp.Msg.Run.Id),
			"`runs status` — show all target statuses",
		},
	})
}

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetRun(context.Background(), connect.NewRequest(&runsv1.GetRunRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get run %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Run == nil {
		return fmt.Errorf("server returned no run")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Fetched run %s.", resp.Msg.Run.Id)},
		ResultsHeading: "Run",
		Results:        []string{formatRun(resp.Msg.Run)},
	})
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListRuns(context.Background(), connect.NewRequest(&runsv1.ListRunsRequest{
		PlanId: ctx.Flag("plan"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list runs", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no runs response")
	}
	results := make([]string, 0, len(resp.Msg.Runs))
	for _, r := range resp.Msg.Runs {
		results = append(results, formatRun(r))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d run(s).", len(resp.Msg.Runs))},
		ResultsHeading: "Runs",
		Results:        results,
		RetrievalHints: []string{
			"`runs get <id>` — show a single run",
			"`runs trigger --plan <plan-id>` — trigger a run for a plan",
		},
	})
}

func (h *handlers) status(ctx cliapp.RunContext) error {
	resp, err := h.client.ListTargetStatus(context.Background(), connect.NewRequest(&runsv1.ListTargetStatusRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list target status", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no status response")
	}
	results := make([]string, 0, len(resp.Msg.Statuses))
	for _, s := range resp.Msg.Statuses {
		results = append(results, formatTargetStatus(s))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d target status(es).", len(resp.Msg.Statuses))},
		ResultsHeading: "Target Statuses",
		Results:        results,
	})
}

func (h *handlers) browse(ctx cliapp.RunContext) error {
	resp, err := h.client.BrowseSnapshot(context.Background(), connect.NewRequest(&runsv1.BrowseSnapshotRequest{
		DestinationId: ctx.Flag("destination"),
		SnapshotId:    ctx.Flag("snapshot"),
		Path:          ctx.Flag("path"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("browse snapshot", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no browse response")
	}
	results := make([]string, 0, len(resp.Msg.Entries))
	for _, e := range resp.Msg.Entries {
		dirMark := ""
		if e.IsDir {
			dirMark = "/"
		}
		results = append(results, fmt.Sprintf("%s%s (%d bytes)", e.Path, dirMark, e.SizeBytes))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d entries.", len(resp.Msg.Entries))},
		ResultsHeading: "Snapshot Contents",
		Results:        results,
	})
}

func runStatusLabel(s runsv1.RunStatus) string {
	switch s {
	case runsv1.RunStatus_RUN_STATUS_PENDING:
		return "pending"
	case runsv1.RunStatus_RUN_STATUS_CAPTURING:
		return "capturing"
	case runsv1.RunStatus_RUN_STATUS_SNAPSHOTTING:
		return "snapshotting"
	case runsv1.RunStatus_RUN_STATUS_COMPLETED:
		return "completed"
	case runsv1.RunStatus_RUN_STATUS_PARTIAL_FAILED:
		return "partial-failed"
	case runsv1.RunStatus_RUN_STATUS_FAILED:
		return "failed"
	default:
		return "unspecified"
	}
}

func triggerLabel(t runsv1.TriggerSource) string {
	switch t {
	case runsv1.TriggerSource_TRIGGER_SOURCE_SCHEDULER:
		return "scheduler"
	case runsv1.TriggerSource_TRIGGER_SOURCE_MANUAL:
		return "manual"
	default:
		return "unspecified"
	}
}

func formatRun(r *runsv1.Run) string {
	if r == nil {
		return "(nil)"
	}
	started := ""
	if r.StartedAt != nil {
		started = r.StartedAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("%s — plan=%s trigger=%s status=%s outcomes=%d started=%s",
		r.Id, r.PlanId, triggerLabel(r.Trigger), runStatusLabel(r.Status), len(r.Outcomes), started)
}

func formatTargetStatus(s *runsv1.TargetStatus) string {
	if s == nil {
		return "(nil)"
	}
	lastSuccess := "never"
	if s.LastSuccessAt != nil {
		lastSuccess = s.LastSuccessAt.AsTime().Format(time.RFC3339)
	}
	return fmt.Sprintf("target=%s last-success=%s last-run-status=%s",
		s.TargetId, lastSuccess, runStatusLabel(s.LastRunStatus))
}
