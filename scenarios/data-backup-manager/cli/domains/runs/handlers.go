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

// trigger enqueues a run. TriggerRun is asynchronous server-side — it returns a
// run in a non-terminal state immediately and the backup executes on a
// background worker — so the standard client timeout is appropriate; poll
// `runs get <id>` for progress and the terminal status.
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
		Result:  []string{fmt.Sprintf("Queued run %s — executing in the background.", resp.Msg.Run.Id)},
		Changes: []string{formatRun(resp.Msg.Run)},
		NextCommand: []string{
			fmt.Sprintf("`runs get %s` — poll run status until it reaches a terminal state", resp.Msg.Run.Id),
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

func (h *handlers) stats(ctx cliapp.RunContext) error {
	resp, err := h.client.GetRunStats(context.Background(), connect.NewRequest(&runsv1.GetRunStatsRequest{
		PlanId: ctx.Flag("plan"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("get run stats", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Stats == nil {
		return fmt.Errorf("server returned no stats")
	}
	s := resp.Msg.Stats
	scope := "all plans"
	if p := ctx.Flag("plan"); p != "" {
		scope = "plan " + p
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Run metrics over %d run(s) (%s).", s.Window, scope)},
		ResultsHeading: "Metrics",
		Results: []string{
			fmt.Sprintf("runs: %d total — %d completed, %d partial-failed, %d failed", s.TotalRuns, s.Completed, s.PartialFailed, s.Failed),
			fmt.Sprintf("success rate: %.0f%%", s.SuccessRate*100),
			fmt.Sprintf("duration: p50 %s, p95 %s", formatMillis(s.P50DurationMs), formatMillis(s.P95DurationMs)),
			fmt.Sprintf("bytes: %s total, %s avg/run", formatBytes(s.TotalBytes), formatBytes(s.AvgBytesPerRun)),
			fmt.Sprintf("throughput: %s/s (logical)", formatBytes(int64(s.AvgThroughputBytesPerSec))),
		},
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

// formatMillis renders a millisecond duration compactly (e.g. 1.4s, 850ms).
func formatMillis(ms int64) string {
	if ms <= 0 {
		return "0ms"
	}
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000.0)
}

// formatBytes renders a byte count in binary units (KiB/MiB/GiB).
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
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
		lastSuccess = fmt.Sprintf("%s (%s ago)", s.LastSuccessAt.AsTime().Format(time.RFC3339), formatAge(s.LastSuccessAgeSeconds))
	}
	freshness := "ok"
	if s.Overdue {
		freshness = "OVERDUE"
	}
	return fmt.Sprintf("target=%s [%s] last-success=%s last-run-status=%s",
		s.TargetId, freshness, lastSuccess, runStatusLabel(s.LastRunStatus))
}

// formatAge renders an age in seconds as a compact human duration.
func formatAge(sec int64) string {
	switch {
	case sec <= 0:
		return "0s"
	case sec < 60:
		return fmt.Sprintf("%ds", sec)
	case sec < 3600:
		return fmt.Sprintf("%dm", sec/60)
	case sec < 86400:
		return fmt.Sprintf("%dh", sec/3600)
	default:
		return fmt.Sprintf("%dd", sec/86400)
	}
}
