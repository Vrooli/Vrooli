package consumeridentity

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func testSigner(t *testing.T, id string) (*Signer, *KeySet) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := NewSigner(id, key, "lpbs.test", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	return signer, NewKeySet(PublicKey{ID: id, Key: &key.PublicKey})
}

func TestRS256RoundTripAndJWKS(t *testing.T) {
	signer, keys := testSigner(t, "key-old")
	token, _, err := signer.Sign(Claims{Subject: "user-1", UserID: "user-1", Email: "one@example.test"})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := NewVerifier(keys, "lpbs.test", time.Second).Verify(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Email != "one@example.test" {
		t.Fatalf("claims = %+v", claims)
	}
	body, err := keys.JWKS()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseJWKS(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Keys) != 1 {
		t.Fatalf("parsed keys = %d", len(parsed.Keys))
	}
}

func TestVerifierRejectsTamperingUnknownKeyAndLegacyAlgorithm(t *testing.T) {
	signer, keys := testSigner(t, "key-current")
	token, _, err := signer.Sign(Claims{Subject: "user-1", UserID: "user-1"})
	if err != nil {
		t.Fatal(err)
	}
	parts := splitToken(t, token)
	parts[1] = parts[1] + "x"
	if _, err := NewVerifier(keys, "lpbs.test", time.Second).Verify(parts[0] + "." + parts[1] + "." + parts[2]); !errors.Is(err, ErrSignatureInvalid) && !errors.Is(err, ErrMalformed) {
		t.Fatalf("tampered error = %v", err)
	}
	parts = splitToken(t, token)
	parts[0] = "eyJhbGciOiJIUzI1NiIsImtpZCI6ImtleS1jdXJyZW50In0"
	if _, err := NewVerifier(keys, "lpbs.test", time.Second).Verify(parts[0] + "." + parts[1] + "." + parts[2]); !errors.Is(err, ErrUnsupportedAlgorithm) {
		t.Fatalf("legacy error = %v", err)
	}
	parts = splitToken(t, token)
	parts[0] = "eyJhbGciOiJSUzI1NiIsImtpZCI6Im5vdC1wdWJsaXNoZWQifQ"
	if _, err := NewVerifier(keys, "lpbs.test", time.Second).Verify(parts[0] + "." + parts[1] + "." + parts[2]); !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("unknown key error = %v", err)
	}
}

func TestVerifierAppliesBoundedClockSkew(t *testing.T) {
	signer, keys := testSigner(t, "key-current")
	signer.Now = func() time.Time { return time.Unix(1_000, 0) }
	token, _, err := signer.Sign(Claims{Subject: "user-1", UserID: "user-1"})
	if err != nil {
		t.Fatal(err)
	}
	v := NewVerifier(keys, "lpbs.test", 30*time.Second)
	v.Now = func() time.Time { return time.Unix(1_000, 0).Add(2 * time.Minute) }
	if _, err := v.Verify(token); !errors.Is(err, ErrExpired) {
		t.Fatalf("beyond skew error = %v", err)
	}
	v.Now = func() time.Time { return time.Unix(1_000, 0).Add(45 * time.Second) }
	if _, err := v.Verify(token); err != nil {
		t.Fatalf("within skew error = %v", err)
	}
}

func TestCacheVerifiesWarmWithoutNetwork(t *testing.T) {
	signer, keys := testSigner(t, "key-current")
	token, _, err := signer.Sign(Claims{Subject: "user-1", UserID: "user-1"})
	if err != nil {
		t.Fatal(err)
	}
	body, err := keys.JWKS()
	if err != nil {
		t.Fatal(err)
	}
	fetches := 0
	cache := NewCache("lpbs.test", time.Second, time.Hour, func(*http.Request) ([]byte, error) { fetches++; return body, nil })
	if _, err := cache.Verify(httpRequest(), token); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Verify(httpRequest(), token); err != nil {
		t.Fatal(err)
	}
	if fetches != 1 {
		t.Fatalf("fetches = %d, want one initial fetch", fetches)
	}
}

func TestCacheReportsKeySetUnavailableWhenRefreshFails(t *testing.T) {
	signer, keys := testSigner(t, "key-current")
	token, _, err := signer.Sign(Claims{Subject: "user-1", UserID: "user-1"})
	if err != nil {
		t.Fatal(err)
	}
	body, err := keys.JWKS()
	if err != nil {
		t.Fatal(err)
	}
	cache := NewCache("lpbs.test", 0, time.Hour, func(*http.Request) ([]byte, error) {
		return body, nil
	})
	if _, err := cache.Verify(httpRequest(), token); err != nil {
		t.Fatal(err)
	}
	expired := token
	signer.Now = func() time.Time { return time.Now().Add(-2 * time.Hour) }
	expired, _, err = signer.Sign(Claims{Subject: "user-1", UserID: "user-1"})
	if err != nil {
		t.Fatal(err)
	}
	cache.refreshedAt = time.Now().Add(-2 * time.Hour)
	cache.fetch = func(*http.Request) ([]byte, error) { return nil, fmt.Errorf("authority offline") }
	if _, err := cache.Verify(httpRequest(), expired); !errors.Is(err, ErrKeySetUnavailable) {
		t.Fatalf("offline refresh error = %v", err)
	}
}

func splitToken(t *testing.T, token string) []string {
	t.Helper()
	parts := make([]string, 0, 3)
	raw := []byte(token)
	start := 0
	for i, b := range raw {
		if b == '.' {
			parts = append(parts, string(raw[start:i]))
			start = i + 1
		}
	}
	parts = append(parts, string(raw[start:]))
	if len(parts) != 3 {
		t.Fatal("invalid test token")
	}
	return parts
}
func httpRequest() *http.Request {
	req, _ := http.NewRequest(http.MethodGet, "http://local.test/.well-known/jwks.json", nil)
	return req
}
