package recall

import (
	"context"
	"fmt"
	"strconv"

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
	n, _ := strconv.Atoi(ctx.Flag(name))
	return int32(n)
}

func (h *handlers) recallCall(ctx cliapp.OperationContext) (*recallv1.RecallResponse, error) {
	resp, err := h.client.Recall(context.Background(), connect.NewRequest(&recallv1.RecallRequest{Query: ctx.Positional("query"), Limit: intFlag(ctx, "limit")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("recall memory", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) wakeCall(ctx cliapp.OperationContext) (*recallv1.WakeResponse, error) {
	resp, err := h.client.Wake(context.Background(), connect.NewRequest(&recallv1.WakeRequest{TokenBudget: intFlag(ctx, "budget")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("wake memory", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) recallReport(_ cliapp.OperationContext, msg *recallv1.RecallResponse) cliapp.ListReport {
	return report("Recall", msg.Hits)
}

func (h *handlers) wakeReport(_ cliapp.OperationContext, msg *recallv1.WakeResponse) cliapp.ListReport {
	return report("Wake", msg.Hits)
}

func report(title string, hits []*recallv1.RecallHit) cliapp.ListReport {
	out := make([]string, 0, len(hits))
	for _, h := range hits {
		out = append(out, fmt.Sprintf("[%s] %s", h.FacetId, h.Text))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d memory item(s).", len(hits))}, ResultsHeading: title, Results: out}
}
