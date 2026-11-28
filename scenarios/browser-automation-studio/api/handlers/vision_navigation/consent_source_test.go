package vision_navigation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	aihandlers "github.com/vrooli/browser-automation-studio/handlers/ai"
	"github.com/vrooli/browser-automation-studio/internal/testutil"
	"github.com/vrooli/browser-automation-studio/middleware"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
	"github.com/vrooli/browser-automation-studio/services/vision"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai"
	aiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai/aiconnect"
)

func TestStartNavigationHonorsValidatedConsentSourcePreference(t *testing.T) {
	keyStore := &testutil.CredentialStore{}
	credentialAuthority, err := keyStore.Authority()
	require.NoError(t, err)
	identity, err := credentialauthority.ParseIdentity("vrooli/openrouter")
	require.NoError(t, err)
	require.NoError(t, credentialAuthority.Put(identity, "api-key", "synthetic-fixture-key"))

	for _, tc := range []struct {
		name        string
		source      string
		token       string
		withEntitle bool
		wantSource  string
		wantToken   string
		wantReceipt string
		wantRefused bool
		credential  *credentialauthority.Authority
	}{
		{name: "own-key selected with saved key and signed session", source: "own-key", token: "fixture-signed-alice", wantSource: "own-key", wantToken: "fixture-signed-alice", wantReceipt: "opaque-own-key-receipt", credential: credentialAuthority},
		{name: "credits selected with verified feature snapshot", source: "vrooli-credits", token: "fixture-signed-alice", withEntitle: true, wantSource: "vrooli-credits", wantToken: "fixture-signed-alice", wantReceipt: "opaque-vrooli-credits-receipt"},
		{name: "local selection drops account and receipt", source: "local", token: "fixture-signed-alice", withEntitle: true, wantSource: "local"},
		{name: "own-key without saved key is denied", source: "own-key", token: "fixture-signed-alice", wantRefused: true},
		{name: "own-key without signed session is denied", source: "own-key", credential: credentialAuthority, wantRefused: true},
		{name: "credits without signed session is denied", source: "vrooli-credits", withEntitle: true, wantRefused: true},
		{name: "credits without AI feature is denied", source: "vrooli-credits", token: "fixture-signed-alice", wantRefused: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nav := &fakeNavigator{navType: vision.NavigatorPlaywright, available: true, clientSourcePolicy: vision.AllSourcesPolicy(), navigateID: "nav-consent-source"}
			log := logrus.New()
			log.SetOutput(testWriter{t})
			mount := Module(Deps{Logger: log, Registry: newTestRegistry(t, nav), CredentialAuthority: tc.credential})
			router := chi.NewRouter()
			connectx.RegisterChi(router, mount)
			var handler http.Handler = router
			handler = aihandlers.PrivacyConsentContextMiddleware(handler)
			handler = middleware.NewBYOKMiddleware().InjectBYOKKey(handler)
			handler = monetization.InjectEntitlement(handler)
			if tc.withEntitle {
				next := handler
				handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					ctx := monetization.WithSnapshot(r.Context(), &monetization.Snapshot{Features: []string{entitlement.FeatureAI}})
					next.ServeHTTP(w, r.WithContext(ctx))
				})
			}
			server := httptest.NewServer(handler)
			t.Cleanup(server.Close)
			client := aiconnect.NewVisionNavigationServiceClient(server.Client(), server.URL)
			req := connect.NewRequest(&aiv1.StartNavigationRequest{SessionId: "fixture-session", Prompt: "fixture prompt", Model: "fixture-model"})
			req.Header().Set("X-Vrooli-AI-Consent-Source", tc.source)
			req.Header().Set("X-Vrooli-AI-Consent-Grant", "opaque-"+tc.source+"-receipt")
			if tc.token != "" {
				req.Header().Set("Authorization", "Bearer "+tc.token)
			}
			resp, callErr := client.StartNavigation(context.Background(), req)
			if tc.wantRefused {
				require.Error(t, callErr)
				require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(callErr))
				require.Empty(t, nav.lastRequest.SessionID, "refused source/auth must not reach the navigator/sidecar")
				return
			}
			require.NoError(t, callErr)
			require.Equal(t, tc.wantSource, nav.lastRequest.AISource)
			require.Equal(t, tc.wantToken, nav.lastRequest.AccessToken)
			require.Equal(t, tc.wantReceipt, nav.lastRequest.PrivacyConsentGrant)
			require.NotNil(t, resp)
		})
	}
}

func TestStartNavigationRejectsInvalidConsentSourceBeforeNavigator(t *testing.T) {
	nav := &fakeNavigator{navType: vision.NavigatorPlaywright, available: true, clientSourcePolicy: vision.AllSourcesPolicy(), navigateID: "nav-invalid-source"}
	log := logrus.New()
	log.SetOutput(testWriter{t})
	mount := Module(Deps{Logger: log, Registry: newTestRegistry(t, nav)})
	router := chi.NewRouter()
	connectx.RegisterChi(router, mount)
	server := httptest.NewServer(aihandlers.PrivacyConsentContextMiddleware(router))
	t.Cleanup(server.Close)
	client := aiconnect.NewVisionNavigationServiceClient(server.Client(), server.URL)
	req := connect.NewRequest(&aiv1.StartNavigationRequest{SessionId: "fixture-session", Prompt: "fixture prompt", Model: "fixture-model"})
	req.Header().Set("X-Vrooli-AI-Consent-Source", "unexpected-provider")
	_, err := client.StartNavigation(context.Background(), req)
	require.Error(t, err)
	require.Empty(t, nav.lastRequest.SessionID, "invalid source must be refused before navigation begins")
}
