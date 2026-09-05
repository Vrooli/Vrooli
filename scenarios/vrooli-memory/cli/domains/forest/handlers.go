package forest

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	forestv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/forest"
	forestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/forest/forest_v1connect"
)

type handlers struct {
	client forestconnect.ForestServiceClient
}

func (h *handlers) frontierCall(ctx cliapp.OperationContext) (*forestv1.GetFrontierResponse, error) {
	limit, err := frontierLimit(ctx.Flag("limit"))
	if err != nil {
		return nil, err
	}
	r, e := h.client.GetFrontier(context.Background(), connect.NewRequest(&forestv1.GetFrontierRequest{Scope: ctx.Flag("scope"), Limit: limit}))
	if e != nil {
		return nil, cliapp.WrapAPIError("list memory frontier", e, nil)
	}
	return r.Msg, nil
}

func frontierLimit(raw string) (int32, error) {
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("limit must be a non-negative integer")
	}
	return int32(n), nil
}

func (h *handlers) frontierReport(_ cliapp.OperationContext, m *forestv1.GetFrontierResponse) cliapp.ListReport {
	results := make([]string, 0, len(m.GetNodes()))
	for _, node := range m.GetNodes() {
		results = append(results, fmt.Sprintf("%s: facet=%s depth=%d score=%.6f children=%d", node.GetId(), node.GetFacetId(), node.GetDepth(), node.GetCompactionScore(), len(node.GetChildIds())))
	}
	return cliapp.ListReport{Results: results, Summary: []string{fmt.Sprintf("%d eligible node(s) of target %d (showing %d).", m.GetEligibleCount(), m.GetTarget(), len(results))}}
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	http, base := cliapp.NewConnectHTTPClientWithTimeout(core, 0)
	return &handlers{client: forestconnect.NewForestServiceClient(http, base)}
}

func (h *handlers) compactCall(ctx cliapp.OperationContext) (*forestv1.RunCompactionPassResponse, error) {
	r, e := h.client.RunCompactionPass(context.Background(), connect.NewRequest(&forestv1.RunCompactionPassRequest{Scope: ctx.Flag("scope")}))
	if e != nil {
		return nil, cliapp.WrapAPIError("compact memory frontier", e, nil)
	}
	return r.Msg, nil
}

func (h *handlers) compactReport(_ cliapp.OperationContext, m *forestv1.RunCompactionPassResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Created %d summary node(s).", m.GetCompactedCount())}, NextCommand: []string{"`forest compact` — rerun after new episode memories arrive"}}
}

func (h *handlers) rebuildCall(ctx cliapp.OperationContext) (*forestv1.RebuildForestResponse, error) {
	r, e := h.client.RebuildForest(context.Background(), connect.NewRequest(&forestv1.RebuildForestRequest{Scope: ctx.Flag("scope")}))
	if e != nil {
		return nil, cliapp.WrapAPIError("rebuild memory forest", e, nil)
	}
	return r.Msg, nil
}

func (h *handlers) rebuildReport(_ cliapp.OperationContext, m *forestv1.RebuildForestResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Rebuilt forest with %d eligible frontier node(s).", m.GetNodeCount())}}
}
