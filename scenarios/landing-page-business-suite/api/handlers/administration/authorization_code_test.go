package administration

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	admin "landing-page-business-suite-api/internal/administration"
)

func TestAuthorizationCodeStorePKCEOneUse(t *testing.T) {
	verifier := "native-verifier-with-enough-entropy"
	digest := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(digest[:])
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	store := NewAuthorizationCodeStore()
	store.now = func() time.Time { return now }
	pair := &admin.TokenPair{AccessToken: "access", RefreshToken: "refresh", ExpiresAt: now.Add(time.Hour), TokenType: "Bearer"}
	if err := store.Issue("code", pair, &admin.User{Email: "user@example.com"}, challenge, "http://127.0.0.1:43210/callback", time.Minute); err != nil {
		t.Fatal(err)
	}
	got, _, err := store.Exchange("code", verifier, "http://127.0.0.1:43210/callback")
	if err != nil || got != pair {
		t.Fatalf("exchange = %v, %v", got, err)
	}
	if _, _, err := store.Exchange("code", verifier, "http://127.0.0.1:43210/callback"); err != errAuthorizationCodeUsed {
		t.Fatalf("replay error = %v", err)
	}
}

func TestAuthorizationCodeStoreRejectsWrongVerifierAndNonLoopback(t *testing.T) {
	store := NewAuthorizationCodeStore()
	if err := store.Issue("code", &admin.TokenPair{AccessToken: "access"}, nil, "challenge", "http://127.0.0.1:1/callback", time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Exchange("code", "wrong", "http://127.0.0.1:1/callback"); err != errInvalidCodeVerifier {
		t.Fatalf("wrong verifier error = %v", err)
	}
	for _, redirect := range []string{"vrooli://auth/callback", "http://192.168.1.4:1234/callback", "https://127.0.0.1:1234/callback"} {
		if validLoopbackRedirect(redirect) {
			t.Fatalf("redirect %q unexpectedly accepted", redirect)
		}
	}
}
