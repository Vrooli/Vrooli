package authn

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestForwardedNativeAudienceIsBoundToTheImmediateCreateRunHop(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(jwksHandler(key))
	defer server.Close()
	now := time.Now()
	cfg := Config{Providers: []Provider{NewScenarioAuthenticatorProvider(JWTConfig{Audience: "scenario-authenticator:agent-manager", JWKSURL: server.URL})}}.WithCreateRunCallerAudience("swarm-manager")
	for _, tc := range []struct {
		audience, path string
		allowed        bool
	}{
		{"swarm-manager", "/api/v1/runs", true},
		{"agent-manager", "/api/v1/runs", true},
		{"other-scenario", "/api/v1/runs", false},
		{"swarm-manager", "/api/v1/authorizations/request/approve", false},
		{"swarm-manager", "/api/v1/profiles", false},
	} {
		token := signJWT(t, key, map[string]any{"user_id": "owner", "iss": "scenario-authenticator", "aud": "scenario-authenticator:" + tc.audience, "scope": []string{"agent-manager:write"}, "exp": now.Add(time.Hour).Unix()})
		r := httptest.NewRequest(http.MethodPost, tc.path, nil)
		r.Header.Set("Authorization", "Bearer "+token)
		p, err := cfg.Authenticate(r.Context(), r)
		if (err == nil) != tc.allowed {
			t.Fatalf("audience=%s path=%s error=%v", tc.audience, tc.path, err)
		}
		if err == nil && (p.Subject != "owner" || p.ExpiresAt.After(now.Add(time.Hour))) {
			t.Fatal("adapter widened the verified owner or deadline")
		}
	}
}
