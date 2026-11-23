package consumeridentity

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/identity"
)

func TestRequestProviderVerifiesBearerAndCookieRequestsConcurrently(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := NewSigner("request-provider", key, "lpbs.test", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := signer.Sign(Claims{Subject: "subject", UserID: "account", Email: " test@example.test "})
	if err != nil {
		t.Fatal(err)
	}
	jwks, err := NewKeySet(PublicKey{ID: signer.KeyID, Key: &key.PublicKey}).JWKS()
	if err != nil {
		t.Fatal(err)
	}
	cache := NewCache(signer.Issuer, 0, time.Hour, func(*http.Request) ([]byte, error) { return jwks, nil })
	provider := NewRequestProvider(cache)

	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if i%2 == 0 {
				req.Header.Set("Authorization", "Bearer "+token)
			} else {
				req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
			}
			principal, verifyErr := provider.VerifyRequest(context.Background(), req)
			if verifyErr != nil || principal.Subject != "account" || principal.Email != "test@example.test" || !principal.Verified || principal.Kind != identity.ActorHuman {
				t.Errorf("VerifyRequest() principal=%+v err=%v", principal, verifyErr)
			}
		}(i)
	}
	wg.Wait()

	_, err = provider.VerifyRequest(context.Background(), httptest.NewRequest(http.MethodGet, "/", nil))
	if failure, ok := identity.FailureFromError(err); !ok || failure.Class != identity.FailureMissing {
		t.Fatalf("missing credential failure=%v; want FailureMissing", err)
	}
}
