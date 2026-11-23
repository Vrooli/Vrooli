package authn

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/identity"
)

func TestCloudflareProviderLossAndRecoveryRetainsSavedDraftAcrossKeyRotation(t *testing.T) {
	oldKey := cloudflareTestRSAKey(t)
	newKey := cloudflareTestRSAKey(t)
	const (
		teamDomain = "https://team.example.test"
		audience   = "onboarding-test"
	)
	var keyMu sync.RWMutex
	keyUnavailable := false
	keys := map[string]*rsa.PrivateKey{"old": oldKey}
	keyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		keyMu.RLock()
		defer keyMu.RUnlock()
		if keyUnavailable {
			http.Error(w, "provider unavailable", http.StatusServiceUnavailable)
			return
		}
		jwkSet := make([]map[string]string, 0, len(keys))
		for kid, key := range keys {
			jwkSet = append(jwkSet, map[string]string{
				"kty": "RSA", "alg": "RS256", "kid": kid,
				"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
				"e": base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}),
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": jwkSet})
	}))
	defer keyServer.Close()

	values := map[string]string{
		"VROOLI_AUTH_PROVIDERS":                "cloudflare_access",
		"VROOLI_CLOUDFLARE_ACCESS_TEAM_DOMAIN": teamDomain,
		"VROOLI_CLOUDFLARE_ACCESS_AUDIENCE":    audience,
		"VROOLI_CLOUDFLARE_ACCESS_JWKS_URL":    keyServer.URL,
	}
	cfg, err := FromEnvironment(func(key string) string { return values[key] })
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Providers) != 1 || cfg.Providers[0].Source() != identity.SourceCloudflareAccess {
		t.Fatalf("configured provider chain = %v", configuredSources(cfg.Providers))
	}

	type savedDraft struct {
		Owner   string `json:"owner"`
		Answers string `json:"answers"`
	}
	draftPath := filepath.Join(t.TempDir(), "draft.json")
	owner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, err := RequireHuman(r.Context())
		if err != nil {
			status, _ := identity.StatusFromContext(r.Context())
			if status.FailureClass == identity.FailureUnavailable {
				http.Error(w, "identity provider unavailable", http.StatusServiceUnavailable)
				return
			}
			http.Error(w, "human identity required", http.StatusUnauthorized)
			return
		}
		if r.Method == http.MethodPost {
			body, readErr := io.ReadAll(r.Body)
			if readErr != nil {
				http.Error(w, "invalid draft", http.StatusBadRequest)
				return
			}
			stored, marshalErr := json.Marshal(savedDraft{Owner: principal.Subject, Answers: string(body)})
			if marshalErr != nil {
				http.Error(w, "invalid draft", http.StatusBadRequest)
				return
			}
			if writeErr := os.WriteFile(draftPath, stored, 0600); writeErr != nil {
				http.Error(w, "draft storage unavailable", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		stored, readErr := os.ReadFile(draftPath)
		if readErr != nil {
			http.Error(w, "draft unavailable", http.StatusNotFound)
			return
		}
		var saved savedDraft
		if unmarshalErr := json.Unmarshal(stored, &saved); unmarshalErr != nil {
			http.Error(w, "draft unavailable", http.StatusInternalServerError)
			return
		}
		if saved.Owner != principal.Subject {
			http.Error(w, "draft owner mismatch", http.StatusForbidden)
			return
		}
		_, _ = io.WriteString(w, saved.Answers)
	})
	app := httptest.NewServer(Middleware(cfg)(owner))
	defer app.Close()
	oldToken := cloudflareTestToken(t, oldKey, "old", teamDomain, audience)
	newToken := cloudflareTestToken(t, newKey, "new", teamDomain, audience)

	requestDraft := func(method, token, body string) *http.Response {
		t.Helper()
		req, err := http.NewRequest(method, app.URL+"/draft", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Cf-Access-Jwt-Assertion", token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}
	readBody := func(resp *http.Response) string {
		t.Helper()
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		return string(body)
	}

	resp := requestDraft(http.MethodPost, oldToken, "saved setup answers")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("initial draft save status=%d body=%s", resp.StatusCode, readBody(resp))
	}
	_ = resp.Body.Close()
	savedBytes, err := os.ReadFile(draftPath)
	if err != nil {
		t.Fatal(err)
	}

	keyMu.Lock()
	keyUnavailable = true
	keyMu.Unlock()
	resp = requestDraft(http.MethodPost, newToken, "replacement must not commit")
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("rotated-key save during provider loss status=%d body=%s", resp.StatusCode, readBody(resp))
	}
	if body := readBody(resp); !strings.Contains(body, "identity provider unavailable") {
		t.Fatalf("provider-loss response = %q", body)
	}
	retainedBytes, err := os.ReadFile(draftPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(retainedBytes, savedBytes) {
		t.Fatal("provider-loss mutation changed saved draft bytes")
	}

	resp = requestDraft(http.MethodGet, oldToken, "")
	body := readBody(resp)
	if resp.StatusCode != http.StatusOK || body != "saved setup answers" {
		t.Fatalf("saved draft during key-provider loss status=%d body=%q", resp.StatusCode, body)
	}

	keyMu.Lock()
	keyUnavailable = false
	keys = map[string]*rsa.PrivateKey{"new": newKey}
	keyMu.Unlock()
	resp = requestDraft(http.MethodGet, newToken, "")
	body = readBody(resp)
	if resp.StatusCode != http.StatusOK || body != "saved setup answers" {
		t.Fatalf("saved draft after provider recovery status=%d body=%q", resp.StatusCode, body)
	}
	recoveredBytes, err := os.ReadFile(draftPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(recoveredBytes, savedBytes) {
		t.Fatal("provider recovery changed saved draft bytes without a user write")
	}
}

func cloudflareTestRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func cloudflareTestToken(t *testing.T, key *rsa.PrivateKey, kid, issuer, audience string) string {
	t.Helper()
	encode := func(value any) string {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return base64.RawURLEncoding.EncodeToString(data)
	}
	header := encode(map[string]string{"alg": "RS256", "kid": kid, "typ": "JWT"})
	now := time.Now().UTC()
	claims := encode(map[string]any{
		"iss": issuer, "aud": []string{audience}, "exp": now.Add(10 * time.Minute).Unix(),
		"iat": now.Add(-time.Minute).Unix(), "type": "app", "sub": "synthetic-owner", "email": "synthetic@example.test",
	})
	input := header + "." + claims
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}
