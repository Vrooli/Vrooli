package meter

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	meterv1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/meter"
	meterconnect "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/meter/meter_v1connect"
)

type handlers struct {
	client meterconnect.MeterServiceClient
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	h := &handlers{client: meterconnect.NewMeterServiceClient(httpClient, baseURL)}
	return cliapp.LoadFromManifestPrimitives(manifest, "meter", map[string]cliapp.PrimitiveHandler{
		"MeterService.Usage":        cliapp.ProtoList(h.usageCall, h.usageReport),
		"MeterService.Reservations": cliapp.ProtoList(h.reservationsCall, h.reservationsReport),
		"MeterService.Ceiling":      cliapp.ProtoMutation(h.ceilingCall, h.ceilingReport),
	})
}

func (h *handlers) usageCall(ctx cliapp.OperationContext) (*meterv1.UsageResponse, error) {
	response, err := h.client.Usage(context.Background(), connect.NewRequest(&meterv1.UsageRequest{InstanceId: ctx.Flag("instance-id"), Tenant: ctx.Flag("tenant")}))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (h *handlers) usageReport(_ cliapp.OperationContext, response *meterv1.UsageResponse) cliapp.ListReport {
	results := make([]string, 0, len(response.GetUsage()))
	for _, item := range response.GetUsage() {
		results = append(results, fmt.Sprintf("%s %s %d", item.GetInstanceId(), item.GetTenant(), item.GetQuantity()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d usage record(s).", len(results))}, ResultsHeading: "Usage", Results: results}
}

func (h *handlers) reservationsCall(ctx cliapp.OperationContext) (*meterv1.ReservationsResponse, error) {
	response, err := h.client.Reservations(context.Background(), connect.NewRequest(&meterv1.ReservationsRequest{InstanceId: ctx.Flag("instance-id"), State: ctx.Flag("state")}))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (h *handlers) reservationsReport(_ cliapp.OperationContext, response *meterv1.ReservationsResponse) cliapp.ListReport {
	results := make([]string, 0, len(response.GetReservations()))
	for _, item := range response.GetReservations() {
		results = append(results, fmt.Sprintf("%s %s %s %d", item.GetId(), item.GetState(), item.GetInstanceId(), item.GetQuantity()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d reservation(s).", len(results))}, ResultsHeading: "Reservations", Results: results}
}

func (h *handlers) ceilingCall(ctx cliapp.OperationContext) (*meterv1.CeilingResponse, error) {
	response, err := h.client.Ceiling(context.Background(), connect.NewRequest(&meterv1.CeilingRequest{Tenant: ctx.Flag("tenant")}))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (h *handlers) ceilingReport(_ cliapp.OperationContext, response *meterv1.CeilingResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Tenant ceiling usage: %d; limit: %d.", response.GetUsed(), response.GetLimit())}}
}
