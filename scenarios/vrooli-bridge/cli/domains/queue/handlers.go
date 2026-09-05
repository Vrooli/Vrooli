package queue

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	queuev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/queue"
	queueconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/queue/queue_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"vrooli-bridge/cli/internal/session"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client queueconnect.QueueServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := session.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: queueconnect.NewQueueServiceClient(httpClient, baseURL),
	}
}

// list prints the live per-node queue (running + queued jobs).
func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListQueue(context.Background(), connect.NewRequest(&queuev1.ListQueueRequest{
		NodeId: ctx.Flag("node"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("list the job queue", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no queue response")
	}

	results := make([]string, 0)
	for _, nq := range resp.Msg.Nodes {
		results = append(results, formatNodeQueue(nq))
		for _, e := range nq.Entries {
			results = append(results, "  "+formatEntry(e))
		}
	}
	if len(results) == 0 {
		results = append(results, "(no running or queued jobs)")
	}

	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d node(s) with queue activity.", len(resp.Msg.Nodes))},
		ResultsHeading: "Job queue",
		Results:        results,
		RetrievalHints: []string{
			"`runs list` — durable run history",
			"`runs abort <id>` — cancel a running or queued job",
		},
	})
}

func formatNodeQueue(nq *queuev1.NodeQueue) string {
	if nq == nil {
		return "(nil)"
	}
	return fmt.Sprintf("node=%s [limit=%d running=%d queued=%d]", nq.NodeId, nq.ConcurrencyLimit, nq.Running, nq.Queued)
}

func formatEntry(e *queuev1.QueueEntry) string {
	if e == nil {
		return "(nil)"
	}
	pos := ""
	if e.State == queuev1.QueueState_QUEUE_STATE_QUEUED {
		pos = fmt.Sprintf(" pos=%d", e.Position)
	}
	return fmt.Sprintf("run=%s [%s]%s verb=%s args=%s", e.RunId, stateLabel(e.State), pos, e.Verb, strings.Join(e.Args, " "))
}

func stateLabel(s queuev1.QueueState) string {
	switch s {
	case queuev1.QueueState_QUEUE_STATE_QUEUED:
		return "queued"
	case queuev1.QueueState_QUEUE_STATE_RUNNING:
		return "running"
	default:
		return "unspecified"
	}
}
