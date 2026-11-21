package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/consumeridentity"
	"github.com/vrooli/vrooli/packages/entitlementclient-go"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/ai"
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/ai/ai_v1connect"
	internalai "web-console/internal/ai"
)

type webConsoleMonetizationJourneyConfig struct{}

func (webConsoleMonetizationJourneyConfig) GetConfigs(context.Context) []internalai.Config {
	return []internalai.Config{{Name: "vrooli", Enabled: true, Priority: 1, TimeoutSec: 5}}
}

func (webConsoleMonetizationJourneyConfig) UpdateConfig(context.Context, string, bool, int, int, int) bool {
	return false
}
func (webConsoleMonetizationJourneyConfig) RecordSuccess(context.Context, string, time.Duration) {}
func (webConsoleMonetizationJourneyConfig) RecordError(context.Context, string)                  {}
func (webConsoleMonetizationJourneyConfig) GetHealth(context.Context) []internalai.Health        { return nil }
func (webConsoleMonetizationJourneyConfig) GetProviderTimeout(context.Context, string) time.Duration {
	return 5 * time.Second
}
func (webConsoleMonetizationJourneyConfig) IsEnabled(context.Context, string) bool { return true }

func requireJourneyNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

type webConsoleMonetizationJourneyGateway struct {
	inferenceconnect.UnimplementedInferenceServiceHandler
	seenToken      string
	seenSource     string
	dispatches     atomic.Int64
	forgedRefusals atomic.Int64
	unpaidRefusals atomic.Int64
}

func (g *webConsoleMonetizationJourneyGateway) Run(_ context.Context, request *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
	switch request.Header().Get("Authorization") {
	case "Bearer fixture-forged-session":
		g.forgedRefusals.Add(1)
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("fixture Gateway requires a consumer session"))
	case "Bearer fixture-unpaid-session":
		g.unpaidRefusals.Add(1)
		return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("fixture account has no available credits"))
	case "Bearer fixture-valid-session":
	default:
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("fixture Gateway requires a consumer session"))
	}
	g.seenToken = request.Header().Get("Authorization")
	g.seenSource = request.Header().Get("X-Vrooli-AI-Source")
	g.dispatches.Add(1)
	return connect.NewResponse(&inferencev1.RunResponse{ValueJson: `"echo safe"`, Validated: true}), nil
}

// TestWebConsoleOfflineLeaseCannotGrantMeteredAI composes the production
// setupRoutes/Handler assembly, AI Connect handler, GatewayProvider, and
// shared request-context middleware. A cached lease survives an API-side
// restart, but D8 keeps AI authorization at Gateway: local identity/lease
// hints cannot substitute for the signed consumer session.
func TestWebConsoleOfflineLeaseCannotGrantMeteredAI(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("APP_DATA_DIR", dataDir)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	requireJourneyNoError(t, err)
	keys := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: "web-console-journey", Key: &key.PublicKey})
	jwks, err := keys.JWKS()
	requireJourneyNoError(t, err)
	lease, err := entitlementclient.Sign(entitlementclient.Payload{
		UserIdentity: "alice@example.test", Status: "active", Features: []string{"ai"},
		NotAfter: time.Now().UTC().Add(time.Hour),
	}, "web-console-journey", key)
	requireJourneyNoError(t, err)
	firstStore, err := newWebConsoleLeaseStore()
	requireJourneyNoError(t, err)
	requireJourneyNoError(t, firstStore.Save("alice@example.test", entitlementclient.LeaseRecord{Lease: lease}))

	// A new app-side client/store models a process restart with LPBS offline;
	// trusted issuer keys remain in the separate credential authority fixture.
	restartedStore, err := newWebConsoleLeaseStore()
	requireJourneyNoError(t, err)
	leaseClient := entitlementclient.NewClient("https://lpbs.example.test", nil, nil)
	leaseClient.SetStore(restartedStore)
	leaseClient.SetIssuerKeyStore(&webConsoleTestIssuerKeys{jwks: jwks})
	if restored, restoreErr := leaseClient.CachedAt("alice@example.test", time.Now().UTC()); restoreErr != nil || restored.Status != "active" {
		t.Fatalf("offline lease restore after restart = %+v, %v", restored, restoreErr)
	}

	gateway := &webConsoleMonetizationJourneyGateway{}
	path, gatewayHandler := inferenceconnect.NewInferenceServiceHandler(gateway)
	gatewayMux := http.NewServeMux()
	gatewayMux.Handle(path, gatewayHandler)
	gatewayServer := httptest.NewServer(gatewayMux)
	t.Cleanup(gatewayServer.Close)
	var resolverCalls atomic.Int64
	provider := &internalai.GatewayProvider{
		Source: "vrooli-credits", ProviderName: "vrooli",
		PrivacyClass:  sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL,
		PrivacySource: "web_console.user_command_and_session_context",
		ResolveURL: func(context.Context) (string, error) {
			resolverCalls.Add(1)
			return gatewayServer.URL, nil
		},
		Client: gatewayServer.Client(),
	}
	server := newFakeTestServer()
	server.aiChain = internalai.NewChain(provider)
	server.aiConfig = webConsoleMonetizationJourneyConfig{}
	server.ai = internalai.NewService(server.aiChain, server.aiConfig, &internalai.SystemContext{}, nil,
		&server.metrics.AIGenerations, &server.metrics.AISuggestions)
	server.setupRoutes()
	app := httptest.NewServer(server.Handler())
	t.Cleanup(app.Close)
	client := aiconnect.NewAIServiceClient(app.Client(), app.URL)

	attack := connect.NewRequest(&aiv1.GenerateRequest{Prompt: "run an account-backed operation"})
	attack.Header().Set("X-User-Identity", "alice@example.test")
	_, err = client.Generate(context.Background(), attack)
	if connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("offline cached lease plus local identity hint decision = %v, want fail-closed credits_required", err)
	}
	if got := resolverCalls.Load(); got != 0 {
		t.Fatalf("metered request without a consumer session resolved Gateway %d times", got)
	}
	if got := gateway.dispatches.Load(); got != 0 {
		t.Fatalf("local cached entitlement authorized %d Gateway dispatches", got)
	}

	// A forged signed-session claim reaches the shared authority and is refused
	// there; a valid identity with no available credits is likewise refused
	// before the fake model adapter dispatches. Web Console deliberately leaves
	// the meter decision to AI Gateway rather than its local lease cache.
	for _, token := range []string{"fixture-forged-session", "fixture-unpaid-session"} {
		request := connect.NewRequest(&aiv1.GenerateRequest{Prompt: "run an account-backed operation"})
		request.Header().Set("Authorization", "Bearer "+token)
		if _, err := client.Generate(context.Background(), request); err == nil {
			t.Fatalf("session %q unexpectedly passed the Gateway authority", token)
		}
	}
	if got := gateway.forgedRefusals.Load(); got != 1 {
		t.Fatalf("forged-session refusals = %d, want 1", got)
	}
	if got := gateway.unpaidRefusals.Load(); got != 1 {
		t.Fatalf("unpaid-session refusals = %d, want 1", got)
	}
	if got := gateway.dispatches.Load(); got != 0 {
		t.Fatalf("forged/unpaid sessions reached model adapter %d times", got)
	}

	authorized := connect.NewRequest(&aiv1.GenerateRequest{Prompt: "run an account-backed operation"})
	authorized.Header().Set("Authorization", "Bearer fixture-valid-session")
	response, err := client.Generate(context.Background(), authorized)
	requireJourneyNoError(t, err)
	if response.Msg.GetCommand() != "echo safe" {
		t.Fatalf("command = %q, want %q", response.Msg.GetCommand(), "echo safe")
	}
	if got := resolverCalls.Load(); got != 3 {
		t.Fatalf("Gateway URL resolver calls = %d, want forged/unpaid refusals plus one granted dispatch", got)
	}
	if got := gateway.dispatches.Load(); got != 1 {
		t.Fatalf("Gateway dispatches = %d, want 1", got)
	}
	if gateway.seenToken != "Bearer fixture-valid-session" || gateway.seenSource != "vrooli-credits" {
		t.Fatalf("Gateway observed authorization=%q source=%q", gateway.seenToken, gateway.seenSource)
	}
}
