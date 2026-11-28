package recall

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	recallv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/recall"
	recallconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/recall/recall_v1connect"
)

type handlers struct {
	client recallconnect.RecallServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	http, base := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: recallconnect.NewRecallServiceClient(http, base)}
}

func intFlag(ctx cliapp.OperationContext, name string) int32 {
	n, err := strconv.ParseInt(ctx.Flag(name), 10, 32)
	if err != nil {
		return 0
	}
	return int32(n)
}

func (h *handlers) recallCall(ctx cliapp.OperationContext) (*recallv1.RecallResponse, error) {
	query, category := ctx.Positional("query"), ctx.Flag("category")
	if strings.TrimSpace(query) == "" && strings.TrimSpace(category) == "" {
		return nil, fmt.Errorf("provide a recall query or --category")
	}
	resp, err := h.client.Recall(context.Background(), connect.NewRequest(&recallv1.RecallRequest{
		Query: query, Category: category, Scope: ctx.Flag("scope"), Limit: intFlag(ctx, "limit"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("recall memory", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) zoomCall(ctx cliapp.OperationContext) (*recallv1.ZoomResponse, error) {
	resp, err := h.client.Zoom(context.Background(), connect.NewRequest(&recallv1.ZoomRequest{
		NodeId: ctx.Positional("node-id"), Scope: ctx.Flag("scope"),
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("zoom memory", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) wakeCall(ctx cliapp.OperationContext) (*recallv1.WakeResponse, error) {
	resp, err := h.client.Wake(context.Background(), connect.NewRequest(&recallv1.WakeRequest{Scope: ctx.Flag("scope"), LineBudget: intFlag(ctx, "budget")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("wake memory", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) siblingsCall(ctx cliapp.OperationContext) (*recallv1.ListSiblingEventsResponse, error) {
	resp, err := h.client.ListSiblingEvents(context.Background(), connect.NewRequest(&recallv1.ListSiblingEventsRequest{Scope: ctx.Flag("scope"), EntryId: ctx.Positional("entry-id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list sibling events", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) recallReport(_ cliapp.OperationContext, msg *recallv1.RecallResponse) cliapp.ListReport {
	out := report("Recall", msg.Hits)
	for _, degradation := range msg.GetDegradations() {
		out.Summary = append(out.Summary, degradation.GetDetail())
	}
	return out
}

func (h *handlers) zoomReport(_ cliapp.OperationContext, msg *recallv1.ZoomResponse) cliapp.ListReport {
	return report("Evidence", msg.Constituents)
}

func (h *handlers) wakeReport(_ cliapp.OperationContext, msg *recallv1.WakeResponse) cliapp.ListReport {
	return report("Wake", msg.Hits)
}

func (h *handlers) siblingsReport(_ cliapp.OperationContext, msg *recallv1.ListSiblingEventsResponse) cliapp.ListReport {
	return report("Sibling events", msg.Entries)
}

func report(title string, hits []*recallv1.RecallHit) cliapp.ListReport {
	out := make([]string, 0, len(hits))
	for _, h := range hits {
		item := fmt.Sprintf("[%s] %s", h.FacetId, h.Text)
		if h.NodeId != "" {
			item += fmt.Sprintf("\n  Evidence  vrooli-memory recall zoom --scope %s %s", h.Scope, h.NodeId)
		}
		if h.Evidence != "" {
			item += "\n  Source  " + h.Evidence
		}
		out = append(out, item)
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d memory item(s).", len(hits))}, ResultsHeading: title, Results: out}
}
