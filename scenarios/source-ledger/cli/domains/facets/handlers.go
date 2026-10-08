package facets

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	facetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/facets"
	facetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/facets/facets_v1connect"
)

type handlers struct {
	client facetsconnect.FacetsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, base := cliapp.NewConnectHTTPClientWithTimeout(core, 0)
	return &handlers{client: facetsconnect.NewFacetsServiceClient(httpClient, base)}
}

func scope(ctx cliapp.OperationContext) string {
	if value := ctx.Flag("scope"); value != "" {
		return value
	}
	return "agent-memory"
}

func int32Flag(ctx cliapp.OperationContext, name string) int32 {
	value, _ := strconv.ParseInt(ctx.Flag(name), 10, 32)
	return int32(value)
}

func (h *handlers) listCall(ctx cliapp.OperationContext) (*facetsv1.ListFacetsResponse, error) {
	response, err := h.client.ListFacets(context.Background(), connect.NewRequest(&facetsv1.ListFacetsRequest{Scope: scope(ctx)}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list source-ledger facets", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) listReport(ctx cliapp.OperationContext, msg *facetsv1.ListFacetsResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.GetFacets()))
	for _, facet := range msg.GetFacets() {
		results = append(results, fmt.Sprintf("%s retention=%s compactable=%t resident=%d", facet.GetId(), facet.GetRetentionPolicy(), facet.GetCompactionEligible(), facet.GetResidentBudget()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Scope %s has %d facet(s).", scope(ctx), len(results))}, Results: results}
}

func (h *handlers) unassignedCall(ctx cliapp.OperationContext) (*facetsv1.CountUnassignedResponse, error) {
	response, err := h.client.CountUnassigned(context.Background(), connect.NewRequest(&facetsv1.CountUnassignedRequest{Scope: scope(ctx)}))
	if err != nil {
		return nil, cliapp.WrapAPIError("count unassigned source-ledger entries", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) unassignedReport(ctx cliapp.OperationContext, msg *facetsv1.CountUnassignedResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Scope %s has %d unassigned journal entry(ies).", scope(ctx), msg.GetCount())}, Results: []string{fmt.Sprintf("scope=%s unassigned=%d", scope(ctx), msg.GetCount())}}
}

func (h *handlers) deleteCall(ctx cliapp.OperationContext) (*facetsv1.DeleteFacetResponse, error) {
	response, err := h.client.DeleteFacet(context.Background(), connect.NewRequest(&facetsv1.DeleteFacetRequest{Scope: scope(ctx), FacetId: ctx.Flag("facet")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("delete source-ledger facet", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) deleteReport(ctx cliapp.OperationContext, _ *facetsv1.DeleteFacetResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Deleted unassigned facet %s from scope %s.", ctx.Flag("facet"), scope(ctx))}}
}

func (h *handlers) setCall(ctx cliapp.OperationContext) (*facetsv1.SetFacetPolicyResponse, error) {
	response, err := h.client.SetFacetPolicy(context.Background(), connect.NewRequest(&facetsv1.SetFacetPolicyRequest{Scope: scope(ctx), FacetId: ctx.Flag("facet"), RetentionPolicy: ctx.Flag("retention-policy"), CompactionEligible: ctx.Flag("compaction-eligible") == "true", ResidentBudget: int32Flag(ctx, "resident-budget")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("set source-ledger facet policy", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) setReport(ctx cliapp.OperationContext, msg *facetsv1.SetFacetPolicyResponse) cliapp.MutationReport {
	facet := msg.GetFacet()
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Updated scope %s facet %s: retention=%s compactable=%t resident=%d.", scope(ctx), facet.GetId(), facet.GetRetentionPolicy(), facet.GetCompactionEligible(), facet.GetResidentBudget())}}
}

func (h *handlers) assignCall(ctx cliapp.OperationContext) (*facetsv1.AssignFacetResponse, error) {
	response, err := h.client.AssignFacet(context.Background(), connect.NewRequest(&facetsv1.AssignFacetRequest{Scope: scope(ctx), EntryId: ctx.Flag("entry-id"), FacetId: ctx.Flag("facet")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("assign source-ledger facet", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) assignReport(ctx cliapp.OperationContext, _ *facetsv1.AssignFacetResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Assigned scope %s entry %s to facet %s.", scope(ctx), ctx.Flag("entry-id"), ctx.Flag("facet"))}}
}

func (h *handlers) pinCall(ctx cliapp.OperationContext) (*facetsv1.SetPinResponse, error) {
	response, err := h.client.SetPin(context.Background(), connect.NewRequest(&facetsv1.SetPinRequest{Scope: scope(ctx), EntryId: ctx.Flag("entry-id"), Pinned: ctx.Flag("pinned") == "true"}))
	if err != nil {
		return nil, cliapp.WrapAPIError("set source-ledger pin", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) pinReport(ctx cliapp.OperationContext, _ *facetsv1.SetPinResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Updated pin in scope %s for entry %s.", scope(ctx), ctx.Flag("entry-id"))}}
}

func (h *handlers) supersedeCall(ctx cliapp.OperationContext) (*facetsv1.MarkSupersededResponse, error) {
	response, err := h.client.MarkSuperseded(context.Background(), connect.NewRequest(&facetsv1.MarkSupersededRequest{Scope: scope(ctx), EntryId: ctx.Flag("entry-id"), ReplacementEntryId: ctx.Flag("replacement-entry-id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("mark source-ledger entry superseded", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) supersedeReport(ctx cliapp.OperationContext, _ *facetsv1.MarkSupersededResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Marked entry %s superseded in scope %s.", ctx.Flag("entry-id"), scope(ctx))}}
}

func (h *handlers) resolveCall(ctx cliapp.OperationContext) (*facetsv1.ResolveThreadResponse, error) {
	response, err := h.client.ResolveThread(context.Background(), connect.NewRequest(&facetsv1.ResolveThreadRequest{Scope: scope(ctx), EntryId: ctx.Flag("entry-id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("resolve source-ledger thread", err, nil)
	}
	return response.Msg, nil
}

func (h *handlers) resolveReport(ctx cliapp.OperationContext, _ *facetsv1.ResolveThreadResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Resolved thread entry %s in scope %s.", ctx.Flag("entry-id"), scope(ctx))}}
}
