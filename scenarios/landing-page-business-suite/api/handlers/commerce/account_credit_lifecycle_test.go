package billing

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"landing-page-business-suite-api/internal/commerce"
)

func TestAccountCreditLifecycleRejectsCallerAccountAndResolvesDefault(t *testing.T) {
	var requested []string
	status := 0
	deps := AccountCreditLifecycleDependencies{
		Service:      commerce.NewAccountCreditLifecycleService(nil),
		UserID:       func(*http.Request) string { return "verified-user-id" },
		UserIdentity: func(*http.Request) string { return "owner@example.test" },
		ResolveAccount: func(_ *http.Request, selector string) (string, error) {
			requested = append(requested, selector)
			if selector != "" {
				return "selected-account", nil
			}
			return "default-account", nil
		},
		AccountTier: func(*http.Request, string, string) (string, error) { return "pro", nil },
		ResolveApp:  func(*http.Request) (string, error) { return "audio-tools", nil },
		WriteError: func(w http.ResponseWriter, got int, message, _ string) {
			status = got
			w.WriteHeader(got)
			_, _ = w.Write([]byte(message))
		},
	}
	router := mux.NewRouter()
	RegisterAccountCreditLifecycleRoutes(router, deps, func(next http.HandlerFunc) http.HandlerFunc { return next })

	forged := httptest.NewRecorder()
	router.ServeHTTP(forged, httptest.NewRequest(http.MethodPost, "/api/v1/account-credits/reservations", strings.NewReader(`{"operation_id":"00000000-0000-4000-8000-000000000101","amount":10,"business_account_id":"victim"}`)))
	if status != http.StatusBadRequest || len(requested) != 0 {
		t.Fatalf("caller account accepted: status=%d selectors=%v body=%s", status, requested, forged.Body.String())
	}

	status = 0
	valid := httptest.NewRecorder()
	router.ServeHTTP(valid, httptest.NewRequest(http.MethodPost, "/api/v1/account-credits/reservations", strings.NewReader(`{"operation_id":"00000000-0000-4000-8000-000000000101","amount":10}`)))
	if len(requested) != 1 || requested[0] != "" {
		t.Fatalf("account resolver selectors=%v, want only default selection", requested)
	}
	if valid.Code != http.StatusServiceUnavailable {
		t.Fatalf("unavailable owner service status=%d body=%s", valid.Code, valid.Body.String())
	}

	requested = nil
	status = 0
	usd := httptest.NewRecorder()
	router.ServeHTTP(usd, httptest.NewRequest(http.MethodPost, "/api/v1/account-credits/reservations/usd-v1", strings.NewReader(`{"operation_id":"00000000-0000-4000-8000-000000000102","budget_micros":1000}`)))
	if len(requested) != 1 || requested[0] != "" {
		t.Fatalf("USD-v1 account resolver selectors=%v, want default account only", requested)
	}
	if usd.Code != http.StatusServiceUnavailable {
		t.Fatalf("USD-v1 service unavailable status=%d body=%s", usd.Code, usd.Body.String())
	}

	status = 0
	forgedUSD := httptest.NewRecorder()
	router.ServeHTTP(forgedUSD, httptest.NewRequest(http.MethodPost, "/api/v1/account-credits/reservations/usd-v1", strings.NewReader(`{"operation_id":"00000000-0000-4000-8000-000000000103","budget_micros":1000,"business_account_id":"victim","tier":"enterprise","app_bundle_key":"other-app"}`)))
	if status != http.StatusBadRequest {
		t.Fatalf("USD-v1 caller supplied account/policy scope accepted: status=%d body=%s", status, forgedUSD.Body.String())
	}
}

func TestAccountCreditLifecycleRequiresVerifiedHumanIdentityAndAppCapability(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		userID, identity, app string
		want                  int
	}{
		{name: "missing human identity", userID: "user", want: http.StatusUnauthorized},
		{name: "missing app capability", userID: "user", identity: "owner@example.test", want: http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status := 0
			router := mux.NewRouter()
			RegisterAccountCreditLifecycleRoutes(router, AccountCreditLifecycleDependencies{
				Service:        commerce.NewAccountCreditLifecycleService(nil),
				UserID:         func(*http.Request) string { return tc.userID },
				UserIdentity:   func(*http.Request) string { return tc.identity },
				ResolveAccount: func(*http.Request, string) (string, error) { return "acct", nil },
				ResolveApp: func(*http.Request) (string, error) {
					if tc.app == "" {
						return "", http.ErrNoCookie
					}
					return tc.app, nil
				},
				WriteError: func(w http.ResponseWriter, got int, message, _ string) { status = got; w.WriteHeader(got) },
			}, func(next http.HandlerFunc) http.HandlerFunc { return next })
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/account-credits/reservations", strings.NewReader(`{"operation_id":"00000000-0000-4000-8000-000000000101","amount":10}`)))
			if status != tc.want {
				t.Fatalf("status=%d want=%d body=%s", status, tc.want, response.Body.String())
			}
		})
	}
}
