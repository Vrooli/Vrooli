package runs

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs/runs_v1connect"

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

func (h *handlers) get(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetRun(context.Background(), connect.NewRequest(&runsv1.GetRunRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("get run %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Run == nil {
		return fmt.Errorf("server returned no run")
	}
	results := []string{formatRun(resp.Msg.Run)}
	for _, ev := range resp.Msg.Events {
		results = append(results, "  "+formatEvent(ev))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Run %s — %s (%d event(s)).", resp.Msg.Run.Id, statusLabel(resp.Msg.Run.Status), len(resp.Msg.Events))},
		ResultsHeading: "Run",
		Results:        results,
	})
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListRuns(context.Background(), connect.NewRequest(&runsv1.ListRunsRequest{
		NodeId: ctx.Flag("node"),
		Limit:  int32(parseInt(ctx.Flag("limit"))),
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
		Summary:        []string{fmt.Sprintf("%d run(s).", len(resp.Msg.Runs))},
		ResultsHeading: "Runs",
		Results:        results,
		RetrievalHints: []string{
			"`runs get <id>` — show one run with its event history",
			"`runs wait <id>` — block until a run finishes",
		},
	})
}

// wait blocks server-side until the run is terminal (or the wait deadline
// elapses) and returns an error on a non-passing/timed-out run so the process
// exits non-zero, mirroring test-genie's wait verb.
func (h *handlers) wait(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.WaitRun(context.Background(), connect.NewRequest(&runsv1.WaitRunRequest{
		Id:             id,
		TimeoutSeconds: parseInt64(ctx.Flag("timeout")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("wait for run %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Run == nil {
		return fmt.Errorf("server returned no run")
	}
	run := resp.Msg.Run

	if resp.Msg.TimedOut {
		// Surface the snapshot, then fail so automation re-issues the wait.
		_ = cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
			Summary:        []string{fmt.Sprintf("Run %s still %s after the wait window.", run.Id, statusLabel(run.Status))},
			ResultsHeading: "Run",
			Results:        []string{formatRun(run)},
			RetrievalHints: []string{fmt.Sprintf("`runs wait %s` — re-attach and keep waiting (the run is durable)", run.Id)},
		})
		return fmt.Errorf("run %s did not finish within the wait window (still %s)", run.Id, statusLabel(run.Status))
	}

	_ = cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Run %s finished: %s (exit %d).", run.Id, statusLabel(run.Status), run.ExitCode)},
		ResultsHeading: "Run",
		Results:        []string{formatRun(run)},
	})
	if run.Status != runsv1.RunStatus_RUN_STATUS_PASSED {
		return fmt.Errorf("run %s %s (exit %d)", run.Id, statusLabel(run.Status), run.ExitCode)
	}
	return nil
}

func (h *handlers) abort(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.AbortRun(context.Background(), connect.NewRequest(&runsv1.AbortRunRequest{
		Id:     id,
		Reason: ctx.Flag("reason"),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("abort run %q", id), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Run == nil {
		return fmt.Errorf("server returned no run")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Aborted run %s.", resp.Msg.Run.Id)},
		Changes:     []string{formatRun(resp.Msg.Run)},
		NextCommand: []string{"`runs list` — confirm the run state"},
	})
}

// follow streams the run's live event output until the run finishes or the
// stream ends. This is the human follow verb; automation uses wait.
func (h *handlers) follow(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	stream, err := h.client.StreamRunEvents(context.Background(), connect.NewRequest(&runsv1.StreamRunEventsRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("follow run %q", id), err, nil)
	}
	out := ctx.Stdout()
	for stream.Receive() {
		msg := stream.Msg()
		if msg == nil || msg.Event == nil {
			continue
		}
		fmt.Fprintln(out, formatEvent(msg.Event))
	}
	if err := stream.Err(); err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("follow run %q", id), err, nil)
	}
	return nil
}

// ---- formatting helpers ----

func formatRun(r *runsv1.Run) string {
	if r == nil {
		return "(nil)"
	}
	created := ""
	if r.CreatedAt != nil {
		created = r.CreatedAt.AsTime().Format(time.RFC3339)
	}
	cmd := strings.TrimSpace(r.Verb + " " + r.Scenario)
	return fmt.Sprintf("%s — node=%s %q [status=%s exit=%d args=%v created=%s]",
		r.Id, r.NodeId, cmd, statusLabel(r.Status), r.ExitCode, r.Args, created)
}

func formatEvent(ev *channelv1.RunEvent) string {
	if ev == nil {
		return "(nil event)"
	}
	switch ev.Kind {
	case channelv1.RunEventKind_RUN_EVENT_KIND_LOG:
		return strings.TrimRight(ev.LogChunk, "\n")
	case channelv1.RunEventKind_RUN_EVENT_KIND_STATUS:
		return fmt.Sprintf("[status] %s", ev.Status)
	case channelv1.RunEventKind_RUN_EVENT_KIND_EXIT:
		return fmt.Sprintf("[exit] %d", ev.ExitCode)
	case channelv1.RunEventKind_RUN_EVENT_KIND_ARTIFACT_REF:
		return fmt.Sprintf("[artifact] %s", ev.ArtifactRef)
	default:
		return fmt.Sprintf("[event] seq=%d", ev.Sequence)
	}
}

func statusLabel(s runsv1.RunStatus) string {
	switch s {
	case runsv1.RunStatus_RUN_STATUS_QUEUED:
		return "queued"
	case runsv1.RunStatus_RUN_STATUS_RUNNING:
		return "running"
	case runsv1.RunStatus_RUN_STATUS_PASSED:
		return "passed"
	case runsv1.RunStatus_RUN_STATUS_FAILED:
		return "failed"
	case runsv1.RunStatus_RUN_STATUS_ABORTED:
		return "aborted"
	default:
		return "unspecified"
	}
}

func parseInt(raw string) int {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}
	return v
}

func parseInt64(raw string) int64 {
	v, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0
	}
	return v
}
