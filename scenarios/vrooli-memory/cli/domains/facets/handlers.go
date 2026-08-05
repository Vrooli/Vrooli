package facets

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	facetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/facets"
	facetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/facets/facets_v1connect"
)

type handlers struct{ client facetsconnect.FacetsServiceClient }
func newHandlers(core *cliapp.ScenarioApp) *handlers { http, base := cliapp.NewConnectHTTPClient(core); return &handlers{client: facetsconnect.NewFacetsServiceClient(http, base)} }
func (h *handlers) listCall(cliapp.OperationContext) (*facetsv1.ListFacetsResponse, error) { r, err := h.client.ListFacets(context.Background(), connect.NewRequest(&facetsv1.ListFacetsRequest{})); if err != nil { return nil, cliapp.WrapAPIError("list facets", err, nil) }; return r.Msg, nil }
func (h *handlers) listReport(_ cliapp.OperationContext, m *facetsv1.ListFacetsResponse) cliapp.ListReport { results := make([]string, 0, len(m.GetFacets())); for _, f := range m.GetFacets() { results = append(results, fmt.Sprintf("%s: %s (%s)", f.GetId(), f.GetLabel(), f.GetRetentionPolicy())) }; return cliapp.ListReport{Results: results, Summary: []string{fmt.Sprintf("%d facet(s).", len(results))}} }
func (h *handlers) assignCall(ctx cliapp.OperationContext) (*facetsv1.AssignFacetResponse, error) { r, err := h.client.AssignFacet(context.Background(), connect.NewRequest(&facetsv1.AssignFacetRequest{EntryId: ctx.Positional("entry-id"), FacetId: ctx.Flag("facet")})); if err != nil { return nil, cliapp.WrapAPIError("correct memory facet", err, nil) }; return r.Msg, nil }
func (h *handlers) pinCall(ctx cliapp.OperationContext) (*facetsv1.SetPinResponse, error) { pinned := !ctx.BoolFlag("unpin"); r, err := h.client.SetPin(context.Background(), connect.NewRequest(&facetsv1.SetPinRequest{EntryId: ctx.Positional("entry-id"), Pinned: pinned})); if err != nil { return nil, cliapp.WrapAPIError("change memory pin", err, nil) }; return r.Msg, nil }
func (h *handlers) unpinCall(ctx cliapp.OperationContext) (*facetsv1.SetPinResponse, error) { r, err := h.client.SetPin(context.Background(), connect.NewRequest(&facetsv1.SetPinRequest{EntryId: ctx.Positional("entry-id"), Pinned: false})); if err != nil { return nil, cliapp.WrapAPIError("unpin memory", err, nil) }; return r.Msg, nil }
func (h *handlers) proposalsCall(cliapp.OperationContext) (*facetsv1.ListPinProposalsResponse, error) { r, err := h.client.ListPinProposals(context.Background(), connect.NewRequest(&facetsv1.ListPinProposalsRequest{})); if err != nil { return nil, cliapp.WrapAPIError("list pin proposals", err, nil) }; return r.Msg, nil }
func (h *handlers) proposalsReport(_ cliapp.OperationContext, m *facetsv1.ListPinProposalsResponse) cliapp.ListReport { results := make([]string, 0, len(m.GetProposals())); for _, p := range m.GetProposals() { results = append(results, fmt.Sprintf("%s: %s (%d entries)", p.GetId(), p.GetRationale(), len(p.GetEntryIds()))) }; return cliapp.ListReport{Results: results, Summary: []string{fmt.Sprintf("%d pin proposal(s).", len(results))}} }
func (h *handlers) resolveProposalCall(ctx cliapp.OperationContext) (*facetsv1.ResolvePinProposalResponse, error) { r, err := h.client.ResolvePinProposal(context.Background(), connect.NewRequest(&facetsv1.ResolvePinProposalRequest{ProposalId: ctx.Positional("proposal-id"), Accept: ctx.BoolFlag("accept")})); if err != nil { return nil, cliapp.WrapAPIError("resolve pin proposal", err, nil) }; return r.Msg, nil }
func (h *handlers) supersedeCall(ctx cliapp.OperationContext) (*facetsv1.MarkSupersededResponse, error) { r, err := h.client.MarkSuperseded(context.Background(), connect.NewRequest(&facetsv1.MarkSupersededRequest{EntryId: ctx.Positional("entry-id"), ReplacementEntryId: ctx.Flag("replacement-entry-id")})); if err != nil { return nil, cliapp.WrapAPIError("mark memory superseded", err, nil) }; return r.Msg, nil }
func (h *handlers) resolveThreadCall(ctx cliapp.OperationContext) (*facetsv1.ResolveThreadResponse, error) { r, err := h.client.ResolveThread(context.Background(), connect.NewRequest(&facetsv1.ResolveThreadRequest{EntryId: ctx.Positional("entry-id")})); if err != nil { return nil, cliapp.WrapAPIError("resolve memory thread", err, nil) }; return r.Msg, nil }
func (h *handlers) assignReport(_ cliapp.OperationContext, _ *facetsv1.AssignFacetResponse) cliapp.MutationReport { return cliapp.MutationReport{Result: []string{"Memory facet correction appended."}} }
func (h *handlers) pinReport(_ cliapp.OperationContext, _ *facetsv1.SetPinResponse) cliapp.MutationReport { return cliapp.MutationReport{Result: []string{"Memory pin state updated."}} }
func (h *handlers) proposalReport(_ cliapp.OperationContext, _ *facetsv1.ResolvePinProposalResponse) cliapp.MutationReport { return cliapp.MutationReport{Result: []string{"Pin proposal resolved."}} }
func (h *handlers) supersedeReport(_ cliapp.OperationContext, _ *facetsv1.MarkSupersededResponse) cliapp.MutationReport { return cliapp.MutationReport{Result: []string{"Supersession mark appended."}} }
func (h *handlers) threadReport(_ cliapp.OperationContext, _ *facetsv1.ResolveThreadResponse) cliapp.MutationReport { return cliapp.MutationReport{Result: []string{"Thread resolution mark appended."}} }
