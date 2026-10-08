package administration

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	admin "landing-page-business-suite-api/internal/administration"
)

func TestSetAndClearAuthCookiesPreserveSecurityAttributes(t *testing.T) {
	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	pair := &admin.TokenPair{AccessToken: "access", RefreshToken: "refresh", ExpiresAt: now.Add(time.Hour)}
	recorder := httptest.NewRecorder()
	SetAuthCookies(recorder, pair, true, now)
	cookies := recorder.Result().Cookies()
	if len(cookies) != 3 {
		t.Fatalf("cookies = %d, want 3", len(cookies))
	}
	if cookies[0].Name != "__Host-access_token" || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("access cookie = %#v", cookies[0])
	}
	if cookies[1].Name != "__Secure-refresh_token" || cookies[1].Path != "/api/v1/auth" || !cookies[1].HttpOnly || !cookies[1].Secure {
		t.Fatalf("refresh cookie = %#v", cookies[1])
	}

	recorder = httptest.NewRecorder()
	ClearAuthCookies(recorder, true)
	cleared := recorder.Result().Cookies()
	if len(cleared) != 3 {
		t.Fatalf("cleared cookies = %d, want 3", len(cleared))
	}
	for _, cookie := range cleared {
		if cookie.Name == "__Host-lpbs_session_hint" {
			if cookie.MaxAge >= 0 || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
				t.Fatalf("cleared hint cookie = %#v", cookie)
			}
			continue
		}
		if cookie.MaxAge >= 0 || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
			t.Fatalf("cleared cookie = %#v", cookie)
		}
	}
}

func TestFormatNullableTime(t *testing.T) {
	if got := FormatNullableTime(nil); got != nil {
		t.Fatalf("nil time = %v", got)
	}
	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	if got := FormatNullableTime(&now); got != now.Format(time.RFC3339) {
		t.Fatalf("formatted time = %v", got)
	}
}

func TestRequestMagicLinkRejectsInvalidEmailBeforeService(t *testing.T) {
	called, status := false, 0
	deps := testUserAuthDependencies()
	deps.Service = userAuthStub{request: func(admin.SignInRequest) error { called = true; return nil }}
	deps.WriteError = func(_ http.ResponseWriter, got int, _, _ string) { status = got }
	RequestMagicLink(deps).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/auth/magic-link", strings.NewReader(`{"email":"bad"}`)))
	if status != http.StatusBadRequest || called {
		t.Fatalf("status=%d called=%t", status, called)
	}
}

func TestRequestMagicLinkRateLimitsBeforeService(t *testing.T) {
	called := false
	deps := testUserAuthDependencies()
	deps.Service = userAuthStub{request: func(admin.SignInRequest) error { called = true; return nil }}
	deps.Throttle = throttleStub(false)
	recorder := httptest.NewRecorder()
	RequestMagicLink(deps).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/auth/magic-link", strings.NewReader(`{"email":"user@example.test"}`)))
	if recorder.Code != http.StatusTooManyRequests || called {
		t.Fatalf("status=%d called=%t", recorder.Code, called)
	}
	if recorder.Header().Get("Retry-After") == "" || !strings.Contains(recorder.Body.String(), `"reason":"rate_limited"`) {
		t.Fatalf("rate-limit response lacks retry guidance: headers=%v body=%s", recorder.Header(), recorder.Body.String())
	}
}

func TestRequestMagicLinkReportsDeliveryFailureTruthfully(t *testing.T) {
	deps := testUserAuthDependencies()
	deps.Service = userAuthStub{request: func(admin.SignInRequest) error {
		return errors.Join(admin.ErrDeliveryUnavailable, errors.New("provider down"))
	}}
	recorder := httptest.NewRecorder()
	RequestMagicLink(deps).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/auth/magic-link", strings.NewReader(`{"email":"USER@example.test"}`)))
	if recorder.Code != http.StatusServiceUnavailable || !strings.Contains(recorder.Body.String(), `"reason":"delivery_unavailable"`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "provider down") {
		t.Fatalf("provider detail leaked: %s", recorder.Body.String())
	}
}

func TestRequestMagicLinkForwardsNormalizedEmailBindingAndContext(t *testing.T) {
	var got admin.SignInRequest
	deps := testUserAuthDependencies()
	deps.Service = userAuthStub{request: func(request admin.SignInRequest) error { got = request; return nil }}
	recorder := httptest.NewRecorder()
	body := `{"email":" User@Example.test ","browser_binding":"binding-binding-binding","context":{"redirect_uri":"http://127.0.0.1:4000/cb"}}`
	RequestMagicLink(deps).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/auth/magic-link", strings.NewReader(body)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if got.Email != "user@example.test" || got.BrowserBinding != "binding-binding-binding" || !strings.Contains(string(got.Context), "127.0.0.1:4000") {
		t.Fatalf("forwarded request = %#v", got)
	}
}

func TestVerifySignInCodeThrottlesGuessesBeforeService(t *testing.T) {
	called := false
	deps := testUserAuthDependencies()
	deps.Service = userAuthStub{verify: func(admin.SignInVerification) (*admin.SignInResult, error) { called = true; return nil, nil }}
	deps.Throttle = throttleStub(false)
	recorder := httptest.NewRecorder()
	VerifySignInCode(deps).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/auth/verify-code", strings.NewReader(`{"email":"user@example.test","code":"123456","browser_binding":"binding-binding-binding"}`)))
	if recorder.Code != http.StatusTooManyRequests || called {
		t.Fatalf("status=%d called=%t", recorder.Code, called)
	}
}

func TestVerifySignInCodeMapsWrongCodeToRecoverableReason(t *testing.T) {
	deps := testUserAuthDependencies()
	deps.Service = userAuthStub{verify: func(admin.SignInVerification) (*admin.SignInResult, error) { return nil, admin.ErrCodeInvalid }}
	recorder := httptest.NewRecorder()
	VerifySignInCode(deps).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/auth/verify-code", strings.NewReader(`{"email":"user@example.test","code":"123 456","browser_binding":"binding-binding-binding"}`)))
	if recorder.Code != http.StatusUnauthorized || !strings.Contains(recorder.Body.String(), `"reason":"code_invalid"`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestVerifyMagicLinkReturnsContextOnlyWithResult(t *testing.T) {
	deps := testUserAuthDependencies()
	deps.SecureCookies = func() bool { return true }
	deps.Now = time.Now
	deps.Service = userAuthStub{verify: func(v admin.SignInVerification) (*admin.SignInResult, error) {
		if v.Token != "tok" || v.BrowserBinding != "binding-binding-binding" {
			t.Fatalf("verification = %#v", v)
		}
		return &admin.SignInResult{
			Tokens: &admin.TokenPair{AccessToken: "a", RefreshToken: "r", ExpiresAt: time.Now().Add(time.Minute), TokenType: "Bearer"},
			User:   &admin.User{ID: "u", Email: "user@example.test"}, SameBrowser: true, Context: []byte(`{"app":"x"}`),
		}, nil
	}}
	recorder := httptest.NewRecorder()
	VerifyMagicLink(deps).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/auth/verify", strings.NewReader(`{"token":"tok","browser_binding":"binding-binding-binding"}`)))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"context":{"app":"x"}`) || len(recorder.Result().Cookies()) != 3 {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func testUserAuthDependencies() UserAuthDependencies {
	return UserAuthDependencies{
		ClientIP:   func(*http.Request) string { return "127.0.0.1" },
		WriteError: func(w http.ResponseWriter, status int, _, _ string) { w.WriteHeader(status) },
		Log:        func(string, map[string]any) {},
		LogError:   func(string, map[string]any) {},
	}
}

type throttleStub bool

func (t throttleStub) Allow(context.Context, string, admin.ThrottleRule) (bool, error) {
	return bool(t), nil
}

type userAuthStub struct {
	request func(admin.SignInRequest) error
	verify  func(admin.SignInVerification) (*admin.SignInResult, error)
}

func (s userAuthStub) RequestSignIn(_ context.Context, request admin.SignInRequest) (*admin.SignInStarted, error) {
	if s.request != nil {
		if err := s.request(request); err != nil {
			return nil, err
		}
	}
	return &admin.SignInStarted{ExpiresAt: time.Now().Add(15 * time.Minute)}, nil
}

func (userAuthStub) PreviewSignIn(context.Context, string, string) (*admin.SignInPreview, error) {
	return &admin.SignInPreview{}, nil
}

func (s userAuthStub) VerifySignIn(_ context.Context, v admin.SignInVerification) (*admin.SignInResult, error) {
	if s.verify == nil {
		return nil, admin.ErrTokenInvalid
	}
	return s.verify(v)
}
func (userAuthStub) RefreshTokens(context.Context, string) (*admin.TokenPair, error) { return nil, nil }
func (userAuthStub) RefreshTokensFromSource(context.Context, string, admin.RefreshSource) (*admin.TokenPair, error) {
	return nil, nil
}
func (userAuthStub) Logout(context.Context, string) error                     { return nil }
func (userAuthStub) GetUserByID(context.Context, string) (*admin.User, error) { return nil, nil }
