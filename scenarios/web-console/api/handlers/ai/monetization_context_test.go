package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/lpbsidentity"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/ai"
	internalai "web-console/internal/ai"
)

type monetizationContextGateway struct {
	inferenceconnect.UnimplementedInferenceServiceHandler
	authorization string
	creditCallerAuthorization string
	source        string
}

func (g *monetizationContextGateway) Run(_ context.Context, request *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
	g.authorization = request.Header().Get("Authorization")
	g.creditCallerAuthorization = request.Header().Get(lpbsidentity.CreditCallerAuthorizationHeader)
	g.source = request.Header().Get("X-Vrooli-AI-Source")
	return connect.NewResponse(&inferencev1.RunResponse{ValueJson: `"echo safe"`, Validated: true}), nil
}

type gatewayForwardingService struct {
	provider *internalai.GatewayProvider
}

func (s gatewayForwardingService) Generate(ctx context.Context, prompt, _ string) (string, string, error) {
	value, err := s.provider.Generate(ctx, "system", prompt)
	return value, "vrooli", err
}
func (gatewayForwardingService) Suggest(context.Context, string, string) ([]string, string, error) {
	return nil, "", nil
}
func (gatewayForwardingService) GetConfig(context.Context) ConfigSnapshot { return ConfigSnapshot{} }
func (gatewayForwardingService) UpdateConfig(context.Context, UpdateConfigRequest) (ConfigSnapshot, error) {
	return ConfigSnapshot{}, nil
}
func (gatewayForwardingService) GetHealth(context.Context) []ProviderHealth { return nil }

func TestConnectHandlerPrefersSharedMonetizationContext(t *testing.T) {
	ctx := monetization.WithAccessToken(context.Background(), "middleware-session")
	if got := consumerAuthorization(ctx, "Bearer stale-header"); got != "Bearer middleware-session" {
		t.Fatalf("authorization = %q, want shared session token", got)
	}
	if got := consumerAuthorization(context.Background(), "Bearer direct-connect"); got != "Bearer direct-connect" {
		t.Fatalf("authorization without middleware = %q, want direct Connect header fallback", got)
	}
}

func TestSharedMiddlewareSessionReachesWebConsoleGateway(t *testing.T) {
	fixture := &monetizationContextGateway{}
	path, gatewayHandler := inferenceconnect.NewInferenceServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, gatewayHandler)
	server := httptest.NewServer(mux)
	defer server.Close()

	provider := &internalai.GatewayProvider{
		Source: "vrooli-credits", ProviderName: "vrooli",
		PrivacyClass:  sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		PrivacySource: "web_console.user_command_and_session_context",
		ResolveURL:    func(context.Context) (string, error) { return server.URL, nil },
		ResolveCreditCallerAuthorization: func(context.Context) (string, error) { return "web-console-service-token", nil },
		Client:        server.Client(),
	}
	connectHandler := NewConnectHandler(Deps{Service: gatewayForwardingService{provider: provider}})
	appHandler := monetization.InjectEntitlement(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request := connect.NewRequest(&aiv1.GenerateRequest{Prompt: "list files"})
		// Deliberately model a stale transport header. The request-scoped shared
		// session installed by the outer middleware must be authoritative.
		request.Header().Set("Authorization", "Bearer stale-connect-header")
		response, err := connectHandler.Generate(r.Context(), request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(response.Msg.GetCommand()))
	}))

	request := httptest.NewRequest(http.MethodPost, "/ai", nil)
	request.Header.Set("Authorization", "Bearer shared-middleware-session")
	recorder := httptest.NewRecorder()
	appHandler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "echo safe" {
		t.Fatalf("app response = %d %q", recorder.Code, recorder.Body.String())
	}
	if fixture.authorization != "Bearer shared-middleware-session" || fixture.creditCallerAuthorization != "Bearer web-console-service-token" || fixture.source != "vrooli-credits" {
		t.Fatalf("Gateway request auth=%q credit-caller-auth=%q source=%q", fixture.authorization, fixture.creditCallerAuthorization, fixture.source)
	}
}
