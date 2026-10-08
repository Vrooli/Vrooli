package gateway

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	gatewayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/gateway"
	gatewayconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/gateway/gateway_v1connect"

	"ai-gateway/cli/domains/internal/gatewayreq"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	client gatewayconnect.GatewayServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: gatewayconnect.NewGatewayServiceClient(httpClient, baseURL)}
}

func (h *handlers) validate(ctx cliapp.RunContext) error {
	req, err := gatewayreq.FromContext(ctx)
	if err != nil {
		return err
	}
	resp, err := h.client.ValidateGatewayRequest(context.Background(), connect.NewRequest(&gatewayv1.ValidateGatewayRequestRequest{Request: req}))
	if err != nil {
		return cliapp.WrapAPIError("validate gateway request", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no validation response")
	}
	results := []string{fmt.Sprintf("valid=%t accepted_profiles=%d", resp.Msg.GetValid(), len(resp.Msg.GetAcceptedProfiles()))}
	for _, issue := range resp.Msg.GetIssues() {
		results = append(results, fmt.Sprintf("%s: %s (%s)", issue.GetField(), issue.GetMessage(), issue.GetCode()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Gateway request validation complete: valid=%t.", resp.Msg.GetValid())},
		ResultsHeading: "Validation",
		Results:        results,
		RetrievalHints: []string{
			"`routing preview --role <role> --kind text` — preview provider routing",
			"`inventory roles` — inspect available provider roles",
		},
	})
}
