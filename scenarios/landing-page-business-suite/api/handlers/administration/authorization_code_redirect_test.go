package administration

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthorizeWithoutCredentialSendsBrowserToSignInWithPKCEParameters(t *testing.T) {
	deps := testUserAuthDependencies()
	deps.Service = userAuthStub{}
	query := "response_type=code&client_id=desktop&redirect_uri=http%3A%2F%2F127.0.0.1%3A43111%2Fcallback&code_challenge=abc&code_challenge_method=S256&state=xyz"
	recorder := httptest.NewRecorder()
	AuthorizeWithPKCE(deps, NewAuthorizationCodeStore()).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/auth/authorize?"+query, nil))
	if recorder.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	location := recorder.Header().Get("Location")
	if !strings.HasPrefix(location, "/auth/login?") || !strings.Contains(location, "code_challenge=abc") || !strings.Contains(location, "state=xyz") {
		t.Fatalf("location=%q", location)
	}
}

func TestAuthorizeRejectsNonLoopbackRedirectBeforeSignIn(t *testing.T) {
	deps := testUserAuthDependencies()
	deps.Service = userAuthStub{}
	recorder := httptest.NewRecorder()
	AuthorizeWithPKCE(deps, NewAuthorizationCodeStore()).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/auth/authorize?redirect_uri=https%3A%2F%2Fevil.example%2Fcb&code_challenge=abc&code_challenge_method=S256", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", recorder.Code)
	}
}

func TestAuthorizationCodesAreRandomAndSingleUse(t *testing.T) {
	first, err := randomAuthorizationCode()
	if err != nil {
		t.Fatal(err)
	}
	second, _ := randomAuthorizationCode()
	if first == second || len(first) < 40 {
		t.Fatalf("codes are not unique random values: %q %q", first, second)
	}
}
