package forest

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	forestv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/forest"
	forestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/forest/forest_v1connect"
)

type handlers struct {
	client forestconnect.ForestServiceClient
}

func (h *handlers) frontierCall(cliapp.OperationContext) (*forestv1.GetFrontierResponse, error) {
	r, e := h.client.GetFrontier(context.Background(), connect.NewRequest(&forestv1.GetFrontierRequest{}))
	if e != nil {
		return nil, cliapp.WrapAPIError("list memory frontier", e, nil)
	}
	return r.Msg, nil
}

func (h *handlers) frontierReport(_ cliapp.OperationContext, m *forestv1.GetFrontierResponse) cliapp.ListReport {
	results := make([]string, 0, len(m.GetNodes()))
	for _, node := range m.GetNodes() {
		results = append(results, fmt.Sprintf("%s: facet=%s depth=%d children=%d", node.GetId(), node.GetFacetId(), node.GetDepth(), len(node.GetChildIds())))
	}
	return cliapp.ListReport{Results: results, Summary: []string{fmt.Sprintf("%d frontier node(s).", len(results))}}
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	http, base := cliapp.NewConnectHTTPClientWithTimeout(core, 0)
	return &handlers{client: forestconnect.NewForestServiceClient(http, base)}
}

func (h *handlers) compactCall(cliapp.OperationContext) (*forestv1.RunCompactionPassResponse, error) {
	r, e := h.client.RunCompactionPass(context.Background(), connect.NewRequest(&forestv1.RunCompactionPassRequest{}))
	if e != nil {
		return nil, cliapp.WrapAPIError("compact memory frontier", e, nil)
	}
	return r.Msg, nil
}

func (h *handlers) compactReport(_ cliapp.OperationContext, m *forestv1.RunCompactionPassResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Created %d summary node(s).", m.GetCompactedCount())}, NextCommand: []string{"`forest compact` — rerun after new episode memories arrive"}}
}

func (h *handlers) rebuildCall(cliapp.OperationContext) (*forestv1.RebuildForestResponse, error) {
	r, e := h.client.RebuildForest(context.Background(), connect.NewRequest(&forestv1.RebuildForestRequest{}))
	if e != nil {
		return nil, cliapp.WrapAPIError("rebuild memory forest", e, nil)
	}
	return r.Msg, nil
}

func (h *handlers) rebuildReport(_ cliapp.OperationContext, m *forestv1.RebuildForestResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Rebuilt forest with %d eligible frontier node(s).", m.GetNodeCount())}}
}
