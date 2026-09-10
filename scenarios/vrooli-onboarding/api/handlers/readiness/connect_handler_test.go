package readiness

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/readiness"
	readinessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/readiness/readinessv1connect"
	internalreadiness "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/readiness"
)

func TestGetReadinessProjectsTheRichTypedVerdict(t *testing.T) {
	handler := NewConnectHandler(internalreadiness.Service{Evaluate: func(context.Context, string) (internalreadiness.Response, error) {
		return internalreadiness.Response{
			Status: "missing", Scenarios: []string{"alpha"},
			Credentials: []internalreadiness.Credential{{LogicalID: "vrooli/demo", Field: "token", Status: "unconfigured", Required: true}},
			Blockers:    []internalreadiness.CompletionBlocker{{Kind: "credential", Name: "vrooli/demo:token", Reason: "missing", Remediation: "configure it"}},
		}, nil
	}})
	response, err := handler.GetReadiness(context.Background(), connect.NewRequest(&readinessv1.GetReadinessRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetStatus() != readinessv1.ReadinessState_READINESS_STATE_MISSING || len(response.Msg.GetCredentials()) != 1 || len(response.Msg.GetBlockers()) != 1 {
		t.Fatalf("response = %+v", response.Msg)
	}
	if response.Msg.GetCredentials()[0].GetLegacyStatus() != "unconfigured" {
		t.Fatalf("credential status was not preserved: %+v", response.Msg.GetCredentials()[0])
	}
}

func TestAcknowledgeRejectsAnEmptyDigest(t *testing.T) {
	handler := NewConnectHandler(internalreadiness.Service{})
	_, err := handler.AcknowledgeDegradedReadiness(context.Background(), connect.NewRequest(&readinessv1.AcknowledgeDegradedReadinessRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("error = %v, code %v", err, connect.CodeOf(err))
	}
}

func TestAcknowledgeRequiresAuthorizationOffLoopback(t *testing.T) {
	service := internalreadiness.Service{
		Acknowledge: func(_ context.Context, digest string) (internalreadiness.AcknowledgeResult, error) {
			return internalreadiness.AcknowledgeResult{Status: "degraded", ReadinessDigest: digest}, nil
		},
	}
	router := mux.NewRouter()
	Module(service).Mount(router)
	handler := authn.Middleware(authn.Config{Providers: []authn.Provider{testProvider{}}})(router)

	request := func(remoteAddr, authorization string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, readinessconnect.ReadinessServiceAcknowledgeDegradedReadinessProcedure, strings.NewReader(`{"readinessDigest":"digest"}`))
		req.RemoteAddr = remoteAddr
		req.Header.Set("Content-Type", "application/json")
		if authorization != "" {
			req.Header.Set("Authorization", authorization)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}

	if response := request("192.0.2.10:1234", ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized remote response = %d, body = %s", response.Code, response.Body.String())
	}
	if response := request("127.0.0.1:1234", ""); response.Code != http.StatusUnauthorized {
		t.Fatalf("loopback response = %d, body = %s", response.Code, response.Body.String())
	}
	if response := request("192.0.2.10:1234", "Bearer operator-secret"); response.Code != http.StatusOK {
		t.Fatalf("authorized remote response = %d, body = %s", response.Code, response.Body.String())
	}
}

type testProvider struct{}

func (testProvider) Source() identity.AuthSource { return identity.SourceScenarioAuthenticator }

func (testProvider) VerifyRequest(_ context.Context, req *http.Request) (identity.Principal, error) {
	if req.Header.Get("Authorization") != "Bearer operator-secret" {
		return identity.Principal{}, identity.NewFailure(identity.FailureMissing, identity.SourceScenarioAuthenticator)
	}
	return identity.Principal{Kind: identity.ActorHuman, Subject: "operator-1", Verified: true, Source: identity.SourceScenarioAuthenticator, Scopes: []string{"vrooli-onboarding:write"}}, nil
}
