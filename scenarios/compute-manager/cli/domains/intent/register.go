package intent

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	intentv1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/intent"
	intentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/intent/intent_v1connect"
)

type handlers struct {
	client intentconnect.IntentServiceClient
}

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	h := &handlers{client: intentconnect.NewIntentServiceClient(httpClient, baseURL)}
	return cliapp.LoadFromManifestPrimitives(manifest, "intent", map[string]cliapp.PrimitiveHandler{
		"IntentService.ListIntents": cliapp.ProtoList(h.listCall, h.listReport),
		"IntentService.GetIntent":   cliapp.ProtoMutation(h.getCall, h.getReport),
	})
}

func (h *handlers) listCall(ctx cliapp.OperationContext) (*intentv1.ListIntentsResponse, error) {
	response, err := h.client.ListIntents(context.Background(), connect.NewRequest(&intentv1.ListIntentsRequest{State: ctx.Flag("state"), Provider: ctx.Flag("provider")}))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, response *intentv1.ListIntentsResponse) cliapp.ListReport {
	results := make([]string, 0, len(response.GetIntents()))
	for _, item := range response.GetIntents() {
		results = append(results, fmt.Sprintf("%s %s %s %s", item.GetId(), item.GetState().String(), item.GetProvider(), item.GetInstanceId()))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d provisioning intent(s).", len(results))}, ResultsHeading: "Intents", Results: results}
}

func (h *handlers) getCall(ctx cliapp.OperationContext) (*intentv1.GetIntentResponse, error) {
	response, err := h.client.GetIntent(context.Background(), connect.NewRequest(&intentv1.GetIntentRequest{IdOrKey: ctx.Positional("id")}))
	if err != nil {
		return nil, err
	}
	return response.Msg, nil
}

func (h *handlers) getReport(_ cliapp.OperationContext, response *intentv1.GetIntentResponse) cliapp.MutationReport {
	item := response.GetIntent()
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("%s %s %s %s", item.GetId(), item.GetState().String(), item.GetProvider(), item.GetInstanceId())}}
}
