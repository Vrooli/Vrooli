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
	if len(cookies) != 2 {
		t.Fatalf("cookies = %d, want 2", len(cookies))
	}
	if cookies[0].Name != "access_token" || !cookies[0].HttpOnly || !cookies[0].Secure || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("access cookie = %#v", cookies[0])
	}
	if cookies[1].Name != "refresh_token" || cookies[1].Path != "/api/v1/auth" || !cookies[1].HttpOnly || !cookies[1].Secure {
		t.Fatalf("refresh cookie = %#v", cookies[1])
	}

	recorder = httptest.NewRecorder()
	ClearAuthCookies(recorder, true)
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.MaxAge >= 0 || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
			t.Fatalf("cleared cookie = %#v", cookie)
		}
	}
}

func TestRedirectWithTokensUsesURLFragment(t *testing.T) {
	pair := &admin.TokenPair{AccessToken: "access", RefreshToken: "refresh", ExpiresAt: time.Now().UTC(), TokenType: "Bearer"}
	recorder := httptest.NewRecorder()
	deps := UserAuthDependencies{LogError: func(string, map[string]any) {}}
	RedirectWithTokens(recorder, httptest.NewRequest(http.MethodGet, "/", nil), "https://app.example.test/success", pair, deps)
	response := recorder.Result()
	if response.StatusCode != http.StatusFound {
		t.Fatalf("status = %d", response.StatusCode)
	}
	location := response.Header.Get("Location")
	if location == "" || !strings.Contains(location, "access_token=access") || !strings.Contains(location, "refresh_token=refresh") {
		t.Fatalf("location = %q", location)
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
	deps.Service = userAuthStub{request: func(context.Context, string, string, string) error { called = true; return nil }}
	deps.WriteError = func(_ http.ResponseWriter, got int, _, _ string) { status = got }
	RequestMagicLink(deps).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/auth/magic-link", strings.NewReader(`{"email":"bad"}`)))
	if status != http.StatusBadRequest || called {
		t.Fatalf("status=%d called=%t", status, called)
	}
}

func TestRequestMagicLinkRateLimitsBeforeService(t *testing.T) {
	called, status := false, 0
	deps := testUserAuthDependencies()
	deps.Service = userAuthStub{request: func(context.Context, string, string, string) error { called = true; return nil }}
	deps.RateLimiter = rateLimiterStub(false)
	deps.WriteError = func(_ http.ResponseWriter, got int, _, _ string) { status = got }
	RequestMagicLink(deps).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/auth/magic-link", strings.NewReader(`{"email":"user@example.test"}`)))
	if status != http.StatusTooManyRequests || called {
		t.Fatalf("status=%d called=%t", status, called)
	}
}

func TestRequestMagicLinkDoesNotExposeServiceFailure(t *testing.T) {
	deps := testUserAuthDependencies()
	deps.Service = userAuthStub{request: func(context.Context, string, string, string) error { return errors.New("mail provider unavailable") }}
	recorder := httptest.NewRecorder()
	RequestMagicLink(deps).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/auth/magic-link", strings.NewReader(`{"email":"USER@example.test"}`)))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "Check your email") {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func testUserAuthDependencies() UserAuthDependencies {
	return UserAuthDependencies{
		ClientIP:   func(*http.Request) string { return "127.0.0.1" },
		WriteError: func(http.ResponseWriter, int, string, string) {},
		Log:        func(string, map[string]any) {},
		LogError:   func(string, map[string]any) {},
	}
}

type rateLimiterStub bool

func (r rateLimiterStub) Allow(string) bool { return bool(r) }

type userAuthStub struct {
	request func(context.Context, string, string, string) error
}

func (s userAuthStub) RequestMagicLink(ctx context.Context, email, ip, userAgent string) error {
	if s.request == nil {
		return nil
	}
	return s.request(ctx, email, ip, userAgent)
}

func (userAuthStub) VerifyMagicLink(context.Context, string, string, string) (*admin.TokenPair, *admin.User, error) {
	return nil, nil, nil
}
func (userAuthStub) RefreshTokens(context.Context, string) (*admin.TokenPair, error) { return nil, nil }
func (userAuthStub) Logout(context.Context, string) error                            { return nil }
func (userAuthStub) GetUserByID(context.Context, string) (*admin.User, error)        { return nil, nil }
