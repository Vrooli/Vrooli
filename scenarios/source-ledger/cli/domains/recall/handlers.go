package recall

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	recallv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall"
	recallconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall/recall_v1connect"
)

type handlers struct {
	client recallconnect.RecallServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	http, base := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: recallconnect.NewRecallServiceClient(http, base)}
}

func intFlag(ctx cliapp.OperationContext, name string) int32 {
	value, err := strconv.ParseInt(ctx.Flag(name), 10, 32)
	if err != nil {
		return 0
	}
	return int32(value)
}

func (h *handlers) recallCall(ctx cliapp.OperationContext) (*recallv1.RecallResponse, error) {
	response, err := h.client.Recall(context.Background(), connect.NewRequest(&recallv1.RecallRequest{Query: ctx.Positional("query"), Scope: ctx.Flag("scope"), Limit: intFlag(ctx, "limit")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("recall source-ledger entries", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) wakeCall(ctx cliapp.OperationContext) (*recallv1.WakeResponse, error) {
	response, err := h.client.Wake(context.Background(), connect.NewRequest(&recallv1.WakeRequest{Scope: ctx.Flag("scope"), LineBudget: intFlag(ctx, "line-budget")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("wake source-ledger scope", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) recallReport(_ cliapp.OperationContext, msg *recallv1.RecallResponse) cliapp.ListReport {
	return report("Recall", msg.GetHits())
}

func (h *handlers) wakeReport(_ cliapp.OperationContext, msg *recallv1.WakeResponse) cliapp.ListReport {
	return report("Wake", msg.GetHits())
}

func report(title string, hits []*recallv1.RecallHit) cliapp.ListReport {
	results := make([]string, 0, len(hits))
	for _, hit := range hits {
		results = append(results, fmt.Sprintf("[%s] %s", hit.GetFacetId(), hit.GetText()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d source-ledger item(s).", len(results))}, ResultsHeading: title, Results: results}
}
