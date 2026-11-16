package ai_service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	aihandlers "github.com/vrooli/browser-automation-studio/handlers/ai"
	"github.com/vrooli/browser-automation-studio/middleware"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai"
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai/aiconnect"
)

// This composes the real BAS AIService Connect handler, its production
// AIElementAnalyzer and gatewayRolePromptClient, and a generated Gateway Run
// handler. It isolates the inter-service wire contract without relying on a
// running BAS/Gateway process or any provider credentials.
func TestAIServiceForwardsConsentToGatewayRun(t *testing.T) {
	log := logrus.New()
	log.SetOutput(io.Discard)
	var observed []*inferencev1.RunRequest
	var observedHeaders []http.Header
	var fakeRemoteDispatches atomic.Int64
	gatewayService := &inferenceWireFixture{run: func(_ context.Context, req *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
		observed = append(observed, req.Msg)
		observedHeaders = append(observedHeaders, req.Header().Clone())
		if req.Header().Get("X-Vrooli-AI-Source") != "local" {
			token := strings.TrimPrefix(req.Header().Get("Authorization"), "Bearer ")
			if token != "fixture.signed.alice" || req.Header().Get("X-Vrooli-AI-Consent-Grant") != "opaque-fixture-own-key-receipt" || req.Header().Get("X-Vrooli-AI-Source") != "own-key" {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.New("fixture Gateway consent authority refused before provider dispatch"))
			}
			fakeRemoteDispatches.Add(1)
		}
		encoded, err := json.Marshal("[]")
		require.NoError(t, err)
		return connect.NewResponse(&inferencev1.RunResponse{ValueJson: string(encoded), Validated: true}), nil
	}}
	gatewayPath, gatewayHandler := inferenceconnect.NewInferenceServiceHandler(gatewayService)
	gatewayMux := http.NewServeMux()
	gatewayMux.Handle(gatewayPath, gatewayHandler)
	gatewayServer := httptest.NewServer(gatewayMux)
	t.Cleanup(gatewayServer.Close)

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
		Screenshot: screenshot, DOM: dom, ElementAnalysis: elementAnalysis,
		AIAnalysis: aiAnalysis, LinkPreview: func(context.Context, string) (*LinkPreviewData, bool, error) { return nil, false, nil }, Logger: log,
	})

	router := chi.NewRouter()
	connectx.RegisterChi(router, mount)
	var handler http.Handler = router
	handler = aihandlers.PrivacyConsentContextMiddleware(handler)
	handler = middleware.NewBYOKMiddleware().InjectBYOKKey(handler)
	handler = monetization.InjectEntitlement(handler)
	basServer := httptest.NewServer(handler)
	t.Cleanup(basServer.Close)
	client := aiconnect.NewAIServiceClient(basServer.Client(), basServer.URL)

	remoteCall := func(token, receipt string) error {
		req := connect.NewRequest(&aiv1.AIAnalyzeElementsRequest{Url: "https://fixture.invalid", Intent: "find the search button"})
		if token != "" {
			req.Header().Set("Authorization", "Bearer "+token)
		}
		req.Header().Set(middleware.BYOKHeaderName, "selector-only-value-must-not-forward")
		req.Header().Set("X-Vrooli-AI-Consent-Source", "own-key")
		req.Header().Set("X-Vrooli-AI-Consent-Grant", receipt)
		req.Header().Set("Idempotency-Key", "consent-ai-service-fixture-operation")
		_, err := client.AIAnalyzeElements(context.Background(), req)
		return err
	}
	for _, attempt := range []struct {
		name, token, receipt string
	}{
		{"missing", "fixture.signed.alice", ""},
		{"forged", "fixture.signed.alice", "fixture-forged"},
		{"wrong user", "fixture.signed.bob", "opaque-fixture-own-key-receipt"},
		{"wrong source", "fixture.signed.alice", "fixture-vrooli-credits-receipt"},
		{"expired", "fixture.signed.alice", "fixture-expired-receipt"},
		{"revoked", "fixture.signed.alice", "fixture-revoked-receipt"},
	} {
		t.Run(attempt.name+" reaches refusal before fake dispatch", func(t *testing.T) {
			before := fakeRemoteDispatches.Load()
			callErr := remoteCall(attempt.token, attempt.receipt)
			require.Error(t, callErr)
			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(callErr))
			require.Equal(t, before, fakeRemoteDispatches.Load())
		})
	}
	t.Run("selected source and active grant reach production caller", func(t *testing.T) {
		before := fakeRemoteDispatches.Load()
		require.NoError(t, remoteCall("fixture.signed.alice", "opaque-fixture-own-key-receipt"))
		require.Equal(t, before+1, fakeRemoteDispatches.Load(), "valid exact-source consent dispatches once")
		idx := len(observed) - 1
		require.Equal(t, "own-key", observedHeaders[idx].Get("X-Vrooli-AI-Source"))
		require.Equal(t, "Bearer fixture.signed.alice", observedHeaders[idx].Get("Authorization"))
		require.Equal(t, "opaque-fixture-own-key-receipt", observedHeaders[idx].Get("X-Vrooli-AI-Consent-Grant"))
		require.Equal(t, "", observedHeaders[idx].Get(middleware.BYOKHeaderName), "raw source selector must not be forwarded")
		require.Equal(t, sharedv1.Profile_PROFILE_REMOTE_ONLY, observed[idx].GetProfile())
		require.Equal(t, sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL, observed[idx].GetPrivacyClass())
		require.Equal(t, "browser_automation_studio.workflow_prompt_and_owner_context", observed[idx].GetPrivacyProvenance())
		require.NotEmpty(t, observed[idx].GetRootOperationId())
		require.Contains(t, observed[idx].GetSource(), "find the search button")
		require.NotContains(t, observed[idx].GetSource(), "selector-only-value-must-not-forward")
	})

	t.Run("local request carries neither account identity nor receipt", func(t *testing.T) {
		req := connect.NewRequest(&aiv1.AIAnalyzeElementsRequest{Url: "https://fixture.invalid", Intent: "local only"})
		req.Header().Set("X-Vrooli-AI-Consent-Source", "local")
		req.Header().Set("X-Vrooli-AI-Consent-Grant", "must-not-travel-with-local-source")
		_, err := client.AIAnalyzeElements(context.Background(), req)
		require.NoError(t, err)
		require.Len(t, observed, 8)
		idx := len(observed) - 1
		require.Equal(t, "local", observedHeaders[idx].Get("X-Vrooli-AI-Source"))
		require.Empty(t, observedHeaders[idx].Get("Authorization"))
		require.Empty(t, observedHeaders[idx].Get("X-Vrooli-AI-Consent-Grant"))
		require.Equal(t, sharedv1.Profile_PROFILE_LOCAL_ONLY, observed[idx].GetProfile())
		require.Equal(t, int64(1), fakeRemoteDispatches.Load(), "local requests must not dispatch remotely")
	})
}

type consentFixtureDOM struct{}

func (consentFixtureDOM) ExtractDOMTree(context.Context, string) (string, error) {
	return `<html><body><button id="search">Search</button></body></html>`, nil
}

type inferenceWireFixture struct {
	inferenceconnect.UnimplementedInferenceServiceHandler
	run func(context.Context, *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error)
}

func (f *inferenceWireFixture) Run(ctx context.Context, req *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
	return f.run(ctx, req)
}
