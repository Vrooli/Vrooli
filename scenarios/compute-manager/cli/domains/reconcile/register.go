package reconcile

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	reconcilev1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/reconcile"
	reconcileconnect "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/reconcile/reconcile_v1connect"
)

type handlers struct {
	client reconcileconnect.ReconcileServiceClient
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	h := &handlers{client: reconcileconnect.NewReconcileServiceClient(httpClient, baseURL)}
	return cliapp.LoadFromManifestPrimitives(manifest, "reconcile", map[string]cliapp.PrimitiveHandler{
		"ReconcileService.RunReconciliation": cliapp.ProtoOperational(h.runCall, h.runReport),
		"ReconcileService.ListFindings":      cliapp.ProtoList(h.listCall, h.listReport),
		"ReconcileService.GetFinding":        cliapp.ProtoMutation(h.getCall, h.getReport),
		"ReconcileService.QuarantineFinding": cliapp.ProtoMutation(h.quarantineCall, h.quarantineReport),
	})
}

func (h *handlers) runCall(ctx cliapp.OperationContext) (*reconcilev1.RunReconciliationResponse, error) {
	response, err := h.client.RunReconciliation(context.Background(), connect.NewRequest(&reconcilev1.RunReconciliationRequest{}))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (h *handlers) runReport(_ cliapp.OperationContext, response *reconcilev1.RunReconciliationResponse) cliapp.OperationalReport {
	return cliapp.OperationalReport{Status: []string{fmt.Sprintf("Reconciliation completed with %d finding(s).", len(response.GetFindings()))}}
}

func (h *handlers) listCall(ctx cliapp.OperationContext) (*reconcilev1.ListFindingsResponse, error) {
	response, err := h.client.ListFindings(context.Background(), connect.NewRequest(&reconcilev1.ListFindingsRequest{Status: ctx.Flag("status")}))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, response *reconcilev1.ListFindingsResponse) cliapp.ListReport {
	results := make([]string, 0, len(response.GetFindings()))
	for _, finding := range response.GetFindings() {
		results = append(results, fmt.Sprintf("%s %s %s %s", finding.GetId(), finding.GetKind().String(), finding.GetProvider(), finding.GetProviderInstanceId()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d reconciliation finding(s).", len(results))}, ResultsHeading: "Findings", Results: results}
}

func (h *handlers) quarantineCall(ctx cliapp.OperationContext) (*reconcilev1.QuarantineFindingResponse, error) {
	response, err := h.client.QuarantineFinding(context.Background(), connect.NewRequest(&reconcilev1.QuarantineFindingRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (h *handlers) getCall(ctx cliapp.OperationContext) (*reconcilev1.GetFindingResponse, error) {
	response, err := h.client.GetFinding(context.Background(), connect.NewRequest(&reconcilev1.GetFindingRequest{Id: ctx.Positional("id")}))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (h *handlers) getReport(_ cliapp.OperationContext, response *reconcilev1.GetFindingResponse) cliapp.MutationReport {
	finding := response.GetFinding()
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("%s %s %s %s", finding.GetId(), finding.GetKind().String(), finding.GetProvider(), finding.GetProviderInstanceId())}}
}

func (h *handlers) quarantineReport(_ cliapp.OperationContext, response *reconcilev1.QuarantineFindingResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Quarantined and destroyed provider orphan %s.", response.GetFinding().GetProviderInstanceId())}}
}
