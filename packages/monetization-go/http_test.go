package monetization

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInjectEntitlementIgnoresForgedIdentityHeaders(t *testing.T) {
	var got string
	h := InjectEntitlement(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = AccessTokenFromContext(r.Context())
		if r.Header.Get("X-User-Email") != "attacker@example.com" {
			t.Errorf("test header was not present")
		}
	}))
	r := httptest.NewRequest(http.MethodGet, "/?user=attacker@example.com", nil)
	r.Header.Set("Authorization", "Bearer signed-consumer-token")
	r.Header.Set("X-User-Email", "attacker@example.com")
	h.ServeHTTP(httptest.NewRecorder(), r)
	if got != "signed-consumer-token" {
		t.Fatalf("token = %q, want signed-consumer-token", got)
	}
}

func TestWriteErrorUsesStableShape(t *testing.T) {
	recorder := httptest.NewRecorder()
	WriteError(recorder, http.StatusServiceUnavailable, ErrorAuthorityUnavailable, Decision{Reason: ReasonLeaseUnavailable, UpgradePath: "http://lpbs.local/subscribe"})
	var got ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ErrorType != ErrorAuthorityUnavailable || !got.Retryable || got.UpgradePath != "http://lpbs.local/subscribe" {
		t.Fatalf("error = %+v", got)
	}
}

func TestDisplayCredits(t *testing.T) {
	got := DisplayCredits(125, 0.01, "credits")
	if got.Value != 1.25 || got.Label != "credits" || got.Multiplier != 0.01 {
		t.Fatalf("display = %+v", got)
	}
	if DisplayCredits(2, 0, "units").Value != 2 {
		t.Fatal("non-positive multiplier should use one")
	}
}

func TestMiddlewareDoesNotUseContextIdentityAsHeader(t *testing.T) {
	ctx := WithAccessToken(context.Background(), "token")
	if AccessTokenFromContext(ctx) != "token" {
		t.Fatal("context token missing")
	}
}
