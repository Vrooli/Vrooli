package authoring

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

func TestGatewayAuthorSendsOwnerDerivedRoleAndPrivacyContract(t *testing.T) {
	var received *routingv1.ExecuteRouteRequest
	path, handler := routingconnect.NewRoutingServiceHandler(gatewayRouteStub{
		onExecute: func(request *routingv1.ExecuteRouteRequest) *routingv1.ExecuteRouteResponse {
			received = request
			return &routingv1.ExecuteRouteResponse{
				Valid: true, OutputText: `{"name":"Test","template":"<svg></svg>"}`,
				Evidence: &routingv1.RouteEvidence{SelectedModel: "resolved-code-model"},
			}
		}})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := &GatewayClient{
		HTTPClient: server.Client(),
		Resolve:    func(context.Context) (string, error) { return server.URL, nil },
		Timeout:    45 * time.Second,
	}
	text, model, err := client.Author(context.Background(), "owner brief plus bounded generator schema")
	require.NoError(t, err)
	require.Equal(t, `{"name":"Test","template":"<svg></svg>"}`, text)
	require.Equal(t, "resolved-code-model", model)
	require.NotNil(t, received)
	require.Equal(t, "owner brief plus bounded generator schema", received.GetInputText())
	contract := received.GetRequest()
	require.Equal(t, Role, contract.GetRole())
	require.Equal(t, sharedv1.Profile_PROFILE_REMOTE_ONLY, contract.GetProfile())
	require.Equal(t, sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL, contract.GetPrivacyClass())
	require.Equal(t, privacyProvenance, contract.GetPrivacyProvenance())
	require.Equal(t, "backdrop-studio", contract.GetScenario())
	require.Equal(t, "author_generator", contract.GetOperation())
	require.Equal(t, int32((45 * time.Second).Milliseconds()), contract.GetTimeoutMs())
	require.NotEmpty(t, contract.GetRequestId())
}

type gatewayRouteStub struct {
	routingconnect.UnimplementedRoutingServiceHandler
	onExecute func(*routingv1.ExecuteRouteRequest) *routingv1.ExecuteRouteResponse
}

func (s gatewayRouteStub) ExecuteRoute(_ context.Context, request *connect.Request[routingv1.ExecuteRouteRequest]) (*connect.Response[routingv1.ExecuteRouteResponse], error) {
	return connect.NewResponse(s.onExecute(request.Msg)), nil
}
