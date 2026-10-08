package forest

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	forestv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/forest"
	forestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/forest/forest_v1connect"
)

type handlers struct {
	client forestconnect.ForestServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, base := cliapp.NewConnectHTTPClientWithTimeout(core, 0)
	return &handlers{client: forestconnect.NewForestServiceClient(httpClient, base)}
}

func scope(ctx cliapp.OperationContext) string {
	if value := ctx.Flag("scope"); value != "" {
		return value
	}
	return "agent-memory"
}

func number(ctx cliapp.OperationContext, name string) int32 {
	value, _ := strconv.ParseInt(ctx.Flag(name), 10, 32)
	return int32(value)
}

func (h *handlers) frontierCall(ctx cliapp.OperationContext) (*forestv1.GetFrontierResponse, error) {
	response, err := h.client.GetFrontier(context.Background(), connect.NewRequest(&forestv1.GetFrontierRequest{Scope: scope(ctx), Limit: number(ctx, "limit")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("read source-ledger frontier", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) frontierReport(ctx cliapp.OperationContext, msg *forestv1.GetFrontierResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.GetNodes()))
	for _, node := range msg.GetNodes() {
		results = append(results, fmt.Sprintf("%s entry=%s facet=%s score=%.4f", node.GetId(), node.GetEntryId(), node.GetFacetId(), node.GetCompactionScore()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Scope %s frontier: %d node(s), eligible=%d, target=%d.", scope(ctx), len(results), msg.GetEligibleCount(), msg.GetTarget())}, Results: results}
}

func (h *handlers) compactCall(ctx cliapp.OperationContext) (*forestv1.RunCompactionPassResponse, error) {
	response, err := h.client.RunCompactionPass(context.Background(), connect.NewRequest(&forestv1.RunCompactionPassRequest{Scope: scope(ctx), MaxClusters: number(ctx, "max-clusters")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("compact source-ledger forest", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) compactReport(ctx cliapp.OperationContext, msg *forestv1.RunCompactionPassResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Compacted scope %s: %d cluster(s), frontier %d -> %d, target=%d.", scope(ctx), msg.GetCompactedCount(), msg.GetEligibleFrontierBefore(), msg.GetEligibleFrontierAfter(), msg.GetTarget())}}
}

func (h *handlers) rebuildCall(ctx cliapp.OperationContext) (*forestv1.RebuildForestResponse, error) {
	response, err := h.client.RebuildForest(context.Background(), connect.NewRequest(&forestv1.RebuildForestRequest{Scope: scope(ctx)}))
	if err != nil {
		return nil, cliapp.WrapAPIError("rebuild source-ledger forest", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) rebuildReport(ctx cliapp.OperationContext, msg *forestv1.RebuildForestResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Rebuilt scope %s forest; eligible frontier=%d.", scope(ctx), msg.GetNodeCount())}}
}
