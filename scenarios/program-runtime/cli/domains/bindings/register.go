package bindings

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	bindingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings"
	bindingsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/bindings/bindings_v1connect"
)

const GroupName = "bindings"

type handlers struct {
	client bindingsconnect.BindingRegistryServiceClient
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	h := &handlers{client: bindingsconnect.NewBindingRegistryServiceClient(httpClient, baseURL)}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"BindingRegistryService.ListBindings":    cliapp.ProtoList(h.list, h.listReport),
		"BindingRegistryService.ListUnbound":     cliapp.ProtoList(h.unbound, h.unboundReport),
		"BindingRegistryService.ResolveActCells": cliapp.ProtoList(h.act, h.actReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("bindings: load from manifest: %w", err)
	}
	return group, nil
}

func (h *handlers) list(ctx cliapp.OperationContext) (*bindingsv1.ListBindingsResponse, error) {
	r, err := h.client.ListBindings(context.Background(), connect.NewRequest(&bindingsv1.ListBindingsRequest{Scenario: ctx.Flag("scenario"), Group: ctx.Flag("group")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list bindings", err, nil)
	}
	return r.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, r *bindingsv1.ListBindingsResponse) cliapp.ListReport {
	items := make([]string, 0, len(r.GetBindings()))
	for _, b := range r.GetBindings() {
		items = append(items, fmt.Sprintf("%s — %s.%s [%s]", b.GetId(), b.GetService(), b.GetMethod(), b.GetEffect()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d governed binding(s).", len(items))}, ResultsHeading: "Bindings", Results: items}
}

func (h *handlers) unbound(ctx cliapp.OperationContext) (*bindingsv1.ListUnboundResponse, error) {
	r, err := h.client.ListUnbound(context.Background(), connect.NewRequest(&bindingsv1.ListUnboundRequest{Scenario: ctx.Flag("scenario")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list unbound capabilities", err, nil)
	}
	return r.Msg, nil
}

func (h *handlers) unboundReport(_ cliapp.OperationContext, r *bindingsv1.ListUnboundResponse) cliapp.ListReport {
	items := make([]string, 0, len(r.GetCapabilities()))
	for _, c := range r.GetCapabilities() {
		items = append(items, fmt.Sprintf("%s/%s — %s (%s)", c.GetScenario(), c.GetCommand(), c.GetReason().String(), c.GetDetail()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d unbound capability record(s).", len(items))}, ResultsHeading: "Unbound", Results: items}
}

func (h *handlers) act(_ cliapp.OperationContext) (*bindingsv1.ResolveActCellsResponse, error) {
	r, err := h.client.ResolveActCells(context.Background(), connect.NewRequest(&bindingsv1.ResolveActCellsRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("resolve Act cells", err, nil)
	}
	return r.Msg, nil
}

func (h *handlers) actReport(_ cliapp.OperationContext, r *bindingsv1.ResolveActCellsResponse) cliapp.ListReport {
	items := make([]string, 0, len(r.GetCells()))
	for _, c := range r.GetCells() {
		items = append(items, fmt.Sprintf("%s — %s", c.GetId(), c.GetVerdict().String()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Act cells: %d (confidence=%s).", len(items), r.GetDenominatorConfidence())}, ResultsHeading: "Act", Results: items}
}
