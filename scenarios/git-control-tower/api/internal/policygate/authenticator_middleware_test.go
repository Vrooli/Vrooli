package policygate

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestMiddlewareAttachesVerifiedPrincipalAndIgnoresCallerHeader(t *testing.T) {
	var observed Principal
	handler := Middleware(PrincipalVerifierFunc(func(_ context.Context, token string) (Principal, error) {
		if token != "signed-token" {
			t.Fatalf("token=%q", token)
		}
		return Principal{Kind: cliutil.CallerKindHuman, Subject: "human-1", Verified: true}, nil
	}))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observed, _ = PrincipalFromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/repo/status", nil)
	req.Header.Set("Authorization", "Bearer signed-token")
	req.Header.Set(cliutil.HeaderCaller, "vrooli-agent")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent || observed.Subject != "human-1" || observed.Kind != cliutil.CallerKindHuman {
		t.Fatalf("status=%d principal=%#v", res.Code, observed)
	}
}

func TestMiddlewareKeepsReadOnlyAvailableButRecordsInvalidCredential(t *testing.T) {
	seenFailure := false
	handler := Middleware(PrincipalVerifierFunc(func(_ context.Context, _ string) (Principal, error) {
		return Principal{}, errors.New("bad token")
	}))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenFailure = AuthFailureFromContext(r.Context()) != nil
		if _, ok := PrincipalFromContext(r.Context()); ok {
			t.Error("invalid credential created a principal")
		}
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/repo/status", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK || !seenFailure {
		t.Fatalf("status=%d failure=%t", res.Code, seenFailure)
	}
}
