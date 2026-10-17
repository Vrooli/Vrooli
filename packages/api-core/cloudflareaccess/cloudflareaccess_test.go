package cloudflareaccess

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/identity"
)

func TestVerifierAcceptsValidApplicationUserToken(t *testing.T) {
	key, server, now := testKeyServer(t)
	const teamDomain = "https://team.example.test"
	v, err := NewVerifier(Config{TeamDomain: teamDomain, Audience: "gct-aud", JWKSURL: server.URL, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	token := testToken(t, key, "k1", map[string]any{"iss": teamDomain, "aud": []string{"gct-aud"}, "exp": now.Add(time.Minute).Unix(), "iat": now.Add(-time.Minute).Unix(), "nbf": now.Add(-time.Minute).Unix(), "type": "app", "sub": "operator-1", "email": "operator@example.test"})
	principal, err := v.VerifyRequest(t.Context(), httptest.NewRequest(http.MethodGet, "/", strings.NewReader("")))
	if err == nil {
		t.Fatal("missing assertion unexpectedly succeeded")
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(assertionHeader, token)
	principal, err = v.VerifyRequest(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if !principal.IsHuman() || principal.Subject != "operator-1" || principal.Email != "operator@example.test" || principal.Source != "cloudflare_access" {
		t.Fatalf("principal=%#v", principal)
	}
}

func TestVerifierRejectsClaimAndServiceFailures(t *testing.T) {
	key, server, now := testKeyServer(t)
	const teamDomain = "https://team.example.test"
	v, err := NewVerifier(Config{TeamDomain: teamDomain, Audience: "gct-aud", JWKSURL: server.URL, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		mutate func(map[string]any)
		class  string
	}{
		{"wrong issuer", func(c map[string]any) { c["iss"] = "https://other.example" }, "invalid"},
		{"wrong audience", func(c map[string]any) { c["aud"] = "other" }, "invalid"},
		{"expired", func(c map[string]any) { c["exp"] = now.Add(-time.Minute).Unix() }, "expired"},
		{"future nbf", func(c map[string]any) { c["nbf"] = now.Add(time.Minute).Unix() }, "invalid"},
		{"future iat", func(c map[string]any) { c["iat"] = now.Add(time.Minute).Unix() }, "invalid"},
		{"wrong type", func(c map[string]any) { c["type"] = "service" }, "invalid"},
		{"missing email", func(c map[string]any) { delete(c, "email") }, "invalid"},
		{"service token status", func(c map[string]any) { c["service_token_status"] = true }, "service"},
		{"service id", func(c map[string]any) { c["service_token_id"] = "svc-1" }, "service"},
		{"common name", func(c map[string]any) { c["common_name"] = "svc.example" }, "service"},
		{"empty subject", func(c map[string]any) { c["sub"] = "" }, "service"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			claims := map[string]any{"iss": teamDomain, "aud": "gct-aud", "exp": now.Add(time.Minute).Unix(), "iat": now.Add(-time.Minute).Unix(), "type": "app", "sub": "operator-1", "email": "operator@example.test"}
			tc.mutate(claims)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(assertionHeader, testToken(t, key, "k1", claims))
			_, err := v.VerifyRequest(t.Context(), req)
			failure, ok := failureClass(err)
			if !ok || failure != tc.class {
				t.Fatalf("failure=%v class=%q want %q", err, failure, tc.class)
			}
		})
	}
	serviceReq := httptest.NewRequest(http.MethodGet, "/", nil)
	serviceReq.Header.Set("CF-Access-Client-Id", "client")
	if _, err := v.VerifyRequest(t.Context(), serviceReq); !hasFailure(err, "service") {
		t.Fatalf("service headers err=%v", err)
	}
	forged := httptest.NewRequest(http.MethodGet, "/", nil)
	forged.Header.Set("Cf-Access-Authenticated-User-Email", "operator@example.test")
	if _, err := v.VerifyRequest(t.Context(), forged); !hasFailure(err, "missing") {
		t.Fatalf("forged header err=%v", err)
	}
}

func TestVerifierRejectsMalformedAndWrongAlgorithmAssertions(t *testing.T) {
	key, server, now := testKeyServer(t)
	v, err := NewVerifier(Config{TeamDomain: "https://team.example.test", Audience: "aud", JWKSURL: server.URL, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	claims := map[string]any{"iss": "https://team.example.test", "aud": "aud", "exp": now.Add(time.Minute).Unix(), "type": "app", "sub": "operator", "email": "operator@example.test"}
	cases := []struct {
		name  string
		token string
	}{
		{name: "malformed", token: "not-a-jwt"},
		{name: "wrong algorithm", token: testTokenWithAlgorithm(t, key, "k1", "HS256", claims)},
		{name: "missing kid", token: testTokenWithAlgorithm(t, key, "", "RS256", claims)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(assertionHeader, tc.token)
			_, verifyErr := v.VerifyRequest(t.Context(), req)
			if !hasFailure(verifyErr, "invalid") {
				t.Fatalf("error=%v", verifyErr)
			}
			if strings.Contains(verifyErr.Error(), tc.token) {
				t.Fatal("verification error leaked the assertion")
			}
			if tc.name == "wrong algorithm" && strings.Contains(verifyErr.Error(), "HS256") {
				t.Fatal("verification error leaked the algorithm-bearing token")
			}
		})
	}
}

func TestVerifierHonorsConfiguredClockSkew(t *testing.T) {
	key, server, now := testKeyServer(t)
	v, err := NewVerifier(Config{
		TeamDomain: "https://team.example.test",
		Audience:   "aud",
		JWKSURL:    server.URL,
		ClockSkew:  30 * time.Second,
		Now:        func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	claims := map[string]any{
		"iss":   "https://team.example.test",
		"aud":   "aud",
		"exp":   now.Add(-10 * time.Second).Unix(),
		"iat":   now.Add(10 * time.Second).Unix(),
		"nbf":   now.Add(10 * time.Second).Unix(),
		"type":  "app",
		"sub":   "operator",
		"email": "operator@example.test",
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(assertionHeader, testToken(t, key, "k1", claims))
	principal, err := v.VerifyRequest(t.Context(), req)
	if err != nil || !principal.IsHuman() {
		t.Fatalf("principal=%#v err=%v", principal, err)
	}
}

func TestVerifierRefreshesOnceForRotatedKeyAndNeverLeaksToken(t *testing.T) {
	first, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	second, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	current := map[string]*rsa.PrivateKey{"first": first}
	fetches := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		fetches++
		keys := make([]map[string]string, 0, len(current))
		for kid, key := range current {
			keys = append(keys, jwkForTest(kid, key))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": keys})
	}))
	defer server.Close()
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	const teamDomain = "https://team.example.test"
	v, err := NewVerifier(Config{TeamDomain: teamDomain, Audience: "aud", JWKSURL: server.URL, Now: func() time.Time { return now }, JWKSCacheTTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	claims := func() map[string]any {
		return map[string]any{"iss": teamDomain, "aud": "aud", "exp": now.Add(time.Minute).Unix(), "iat": now.Add(-time.Minute).Unix(), "type": "app", "sub": "operator", "email": "operator@example.test"}
	}
	if _, err := v.Verify(t.Context(), testToken(t, first, "first", claims())); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	current = map[string]*rsa.PrivateKey{"second": second, "first": first}
	mu.Unlock()
	token := testToken(t, second, "second", claims())
	if _, err := v.Verify(t.Context(), token); err != nil {
		t.Fatal(err)
	}
	if _, err := v.Verify(t.Context(), token); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	gotFetches := fetches
	mu.Unlock()
	if gotFetches != 2 {
		t.Fatalf("JWKS fetches=%d want 2", gotFetches)
	}
	bad := token[:len(token)-1] + "x"
	_, err = v.Verify(t.Context(), bad)
	if err != nil && strings.Contains(err.Error(), bad) {
		t.Fatal("verification error leaked the assertion")
	}
}

func TestVerifierCoalescesConcurrentUnknownKeyRefresh(t *testing.T) {
	first, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	second, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	current := map[string]*rsa.PrivateKey{"first": first}
	fetches := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(10 * time.Millisecond)
		mu.Lock()
		defer mu.Unlock()
		fetches++
		keys := make([]map[string]string, 0, len(current))
		for kid, key := range current {
			keys = append(keys, jwkForTest(kid, key))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": keys})
	}))
	defer server.Close()
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	const teamDomain = "https://team.example.test"
	v, err := NewVerifier(Config{TeamDomain: teamDomain, Audience: "aud", JWKSURL: server.URL, Now: func() time.Time { return now }, JWKSCacheTTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	claims := map[string]any{"iss": teamDomain, "aud": "aud", "exp": now.Add(time.Minute).Unix(), "iat": now.Add(-time.Minute).Unix(), "type": "app", "sub": "operator", "email": "operator@example.test"}
	if _, err := v.Verify(t.Context(), testToken(t, first, "first", claims)); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	current = map[string]*rsa.PrivateKey{"second": second, "first": first}
	mu.Unlock()
	tokens := make([]string, 8)
	for i := range tokens {
		tokens[i] = testToken(t, second, "second", claims)
	}
	start := make(chan struct{})
	results := make(chan error, len(tokens))
	var wg sync.WaitGroup
	for _, token := range tokens {
		wg.Add(1)
		go func(token string) {
			defer wg.Done()
			<-start
			_, verifyErr := v.Verify(t.Context(), token)
			results <- verifyErr
		}(token)
	}
	close(start)
	wg.Wait()
	close(results)
	for verifyErr := range results {
		if verifyErr != nil {
			t.Fatal(verifyErr)
		}
	}
	mu.Lock()
	gotFetches := fetches
	mu.Unlock()
	if gotFetches != 2 {
		t.Fatalf("concurrent JWKS fetches=%d want 2", gotFetches)
	}
}

func TestVerifierFailsClosedWhenJWKSRefreshFails(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	available := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if !available {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{jwkForTest("k1", key)}})
	}))
	defer server.Close()
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	const teamDomain = "https://team.example.test"
	v, err := NewVerifier(Config{TeamDomain: teamDomain, Audience: "aud", JWKSURL: server.URL, Now: func() time.Time { return now }, JWKSCacheTTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	claims := map[string]any{"iss": teamDomain, "aud": "aud", "exp": now.Add(time.Minute).Unix(), "iat": now.Add(-time.Minute).Unix(), "type": "app", "sub": "operator", "email": "operator@example.test"}
	if _, err := v.Verify(t.Context(), testToken(t, key, "k1", claims)); err != nil {
		t.Fatal(err)
	}
	available = false
	unknown := testToken(t, key, "unknown", claims)
	_, err = v.Verify(t.Context(), unknown)
	if failure, ok := failureClass(err); !ok || failure != "unavailable" {
		t.Fatalf("failure=%v class=%q", err, failure)
	}
}

func testKeyServer(t *testing.T) (*rsa.PrivateKey, *httptest.Server, time.Time) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{jwkForTest("k1", key)}})
	}))
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	t.Cleanup(server.Close)
	return key, server, now
}

func jwkForTest(kid string, key *rsa.PrivateKey) map[string]string {
	return map[string]string{"kty": "RSA", "alg": "RS256", "kid": kid, "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1})}
}

func testToken(t *testing.T, key *rsa.PrivateKey, kid string, claims map[string]any) string {
	return testTokenWithAlgorithm(t, key, kid, "RS256", claims)
}

func testTokenWithAlgorithm(t *testing.T, key *rsa.PrivateKey, kid, algorithm string, claims map[string]any) string {
	t.Helper()
	encode := func(value any) string {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return base64.RawURLEncoding.EncodeToString(data)
	}
	input := encode(map[string]any{"alg": algorithm, "kid": kid}) + "." + encode(claims)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func failureClass(err error) (string, bool) {
	if err == nil {
		return "", false
	}
	if failure, ok := err.(*identity.Failure); ok {
		return string(failure.Class), true
	}
	return "", false
}

func hasFailure(err error, class string) bool {
	got, ok := failureClass(err)
	return ok && got == class
}
