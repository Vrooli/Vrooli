package ai_service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/consumeridentity"
	"github.com/vrooli/browser-automation-studio/config"
	aihandlers "github.com/vrooli/browser-automation-studio/handlers/ai"
	"github.com/vrooli/browser-automation-studio/internal/testutil"
	"github.com/vrooli/browser-automation-studio/middleware"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
	credentialclient "github.com/vrooli/vrooli/packages/credentialclient-go"
	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai"
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai/aiconnect"
)

type attackJourneyIdentitySettings struct{ identity string }

func (s attackJourneyIdentitySettings) GetSetting(context.Context, string) (string, error) {
	return s.identity, nil
}

// TestBASExecutableLeaseAttackJourney composes the production BAS entitlement
// middleware, AIService Connect module, AIElementAnalyzer, Gateway client, and
// file-backed lease store. Only LPBS and Gateway are loopback fixtures.
func TestBASExecutableLeaseAttackJourney(t *testing.T) {
	t.Setenv("APP_DATA_DIR", t.TempDir())
	const identity = "alice@example.test"
	const sessionToken = "fixture-consumer-access-token"
	const keyID = "sp1-journal-key"
	now := time.Now().UTC()

	issuerKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	attackerKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	issuerSet := consumeridentity.NewKeySet(consumeridentity.PublicKey{ID: keyID, Key: &issuerKey.PublicKey})
	issuerJWKS, err := issuerSet.JWKS()
	require.NoError(t, err)

	signLease := func(key *rsa.PrivateKey) string {
		t.Helper()
		token, signErr := entitlementclient.Sign(entitlementclient.Payload{
			UserIdentity: identity, Status: "active", Features: []string{entitlement.FeatureAI},
			NotAfter: now.Add(time.Hour),
		}, keyID, key)
		require.NoError(t, signErr)
		return token
	}
	var leaseMu sync.RWMutex
	currentLease := signLease(attackerKey) // Signed by a key outside the LPBS trust set.
	lpbs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/me":
			if r.Header.Get("Authorization") != "Bearer "+sessionToken {
				http.Error(w, "unauthorized fixture session", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`{"user":{"email":"` + identity + `"}}`))
		case "/api/v1/entitlements":
			if r.URL.Query().Get("user") != identity || r.Header.Get("Authorization") != "Bearer "+sessionToken {
				http.Error(w, "entitlement binding mismatch", http.StatusForbidden)
				return
			}
			leaseMu.RLock()
			lease := currentLease
			leaseMu.RUnlock()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"lease": lease})
		case "/.well-known/jwks.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(issuerJWKS)
		default:
			http.NotFound(w, r)
		}
	}))
	credentials, err := (&testutil.CredentialStore{}).Authority()
	require.NoError(t, err)
	credentialClient, err := credentialclient.NewInProcess(credentialclient.InProcessOptions{Authority: credentials})
	require.NoError(t, err)
	credentialHTTP := &http.Client{Timeout: time.Second}
	newEntitlementService := func() *entitlement.Service {
		service := entitlement.NewService(config.EntitlementConfig{
			RequestTimeout: time.Second, CacheTTL: time.Minute, ServiceURL: lpbs.URL,
		}, logrus.New())
		service.SetSessionResolver(&credentialclient.ConsumerSessionResolver{
			Credentials: credentialClient, LPBSBaseURL: lpbs.URL, HTTPClient: credentialHTTP,
		})
		return service
	}

	var fakeDispatches atomic.Int64
	gateway := &inferenceWireFixture{run: func(_ context.Context, req *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
		if req.Header().Get("Authorization") != "Bearer "+sessionToken || req.Header().Get("X-Vrooli-AI-Source") != "vrooli-credits" {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("fixture Gateway rejected caller identity or source"))
		}
		fakeDispatches.Add(1)
		encoded, encodeErr := json.Marshal("[]")
		require.NoError(t, encodeErr)
		return connect.NewResponse(&inferencev1.RunResponse{ValueJson: string(encoded), Validated: true}), nil
	}}
	gatewayPath, gatewayHandler := inferenceconnect.NewInferenceServiceHandler(gateway)
	gatewayMux := http.NewServeMux()
	gatewayMux.Handle(gatewayPath, gatewayHandler)
	gatewayServer := httptest.NewServer(gatewayMux)
	t.Cleanup(gatewayServer.Close)

	log := logrus.New()
	log.SetOutput(io.Discard)
	modelClient := aihandlers.NewGatewayRolePromptClientWithResolver(
		func(context.Context) (string, error) { return gatewayServer.URL, nil }, gatewayServer.Client(),
	)
	automation := newServiceAutomationRunner()
	screenshot := aihandlers.NewScreenshotHandler(log, aihandlers.WithScreenshotRunner(automation))
	dom := aihandlers.NewDOMHandler(log, aihandlers.WithDOMRunner(automation))
	elementAnalysis := aihandlers.NewElementAnalysisHandler(log, aihandlers.WithElementRunner(automation))
	aiAnalysis := aihandlers.NewAIAnalysisHandler(log, dom,
		aihandlers.WithDOMExtractor(consentFixtureDOM{}),
		aihandlers.WithAIAnalysisModelClient(modelClient),
		aihandlers.WithAIAnalysisTimeout(3*time.Second),
	)
	mount := Module(Deps{
		Screenshot: screenshot, DOM: dom, ElementAnalysis: elementAnalysis, AIAnalysis: aiAnalysis,
		LinkPreview: func(context.Context, string) (*LinkPreviewData, bool, error) { return nil, false, nil }, Logger: log,
	})
	newBASAPI := func(service *entitlement.Service) *httptest.Server {
		router := chi.NewRouter()
		connectx.RegisterChi(router, mount)
		var handler http.Handler = router
		handler = aihandlers.PrivacyConsentContextMiddleware(handler)
		handler = middleware.NewBYOKMiddleware().InjectBYOKKey(handler)
		handler = middleware.NewEntitlementMiddleware(service, log, attackJourneyIdentitySettings{identity: identity}).InjectEntitlement(handler)
		return httptest.NewServer(handler)
	}
	invoke := func(baseURL string) error {
		client := aiconnect.NewAIServiceClient(http.DefaultClient, baseURL)
		request := connect.NewRequest(&aiv1.AIAnalyzeElementsRequest{Url: "https://fixture.invalid", Intent: "find the primary action"})
		request.Header().Set("Authorization", "Bearer "+sessionToken)
		request.Header().Set("X-Vrooli-AI-Consent-Source", "vrooli-credits")
		request.Header().Set("X-Vrooli-AI-Consent-Grant", "fixture-gateway-consent-receipt")
		_, invokeErr := client.AIAnalyzeElements(context.Background(), request)
		return invokeErr
	}

	firstAPI := newBASAPI(newEntitlementService())
	callErr := invoke(firstAPI.URL)
	if callErr == nil || !strings.Contains(callErr.Error(), "ai feature entitlement") {
		t.Fatalf("forged lease was accepted; call error = %v", callErr)
	}
	require.Zero(t, fakeDispatches.Load(), "invalid signature must be refused before Gateway dispatch")

	leaseMu.Lock()
	currentLease = signLease(issuerKey)
	leaseMu.Unlock()
	require.NoError(t, invoke(firstAPI.URL), "valid LPBS lease should pass the production BAS gate")
	require.EqualValues(t, 1, fakeDispatches.Load())
	firstAPI.Close()
	lpbs.Close() // The next API process must rely on the saved signed lease and separate trusted JWKS.

	restartedAPI := newBASAPI(newEntitlementService())
	t.Cleanup(restartedAPI.Close)
	require.NoError(t, invoke(restartedAPI.URL), "fresh API service should restore its signed lease while LPBS is offline")
	require.EqualValues(t, 2, fakeDispatches.Load(), "offline restart must preserve the feature gate result")
}
