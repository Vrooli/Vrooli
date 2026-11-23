package deployments

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/lpbsidentity"
	"github.com/vrooli/api-core/servicetoken"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

func TestHTTPLPBSReleaseClientReadinessAndVerify(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/deploy-readiness" {
			if r.Header.Get("Authorization") != "Bearer secret" || r.Method != http.MethodPost {
				t.Errorf("readiness request = %s auth=%q", r.Method, r.Header.Get("Authorization"))
			}
			_, _ = w.Write([]byte(`{"ready":false,"gates":[{"name":"storage","ready":false,"message":"missing"}]}`))
			return
		}
		if r.URL.Path == "/api/v1/updates/demo/verify" {
			if r.URL.Query().Get("deep") != "true" || r.URL.Query().Get("platform") != "linux" {
				t.Errorf("verify query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"app_key":"demo","channel":"stable","platform":"linux","expected_version":"1","actual_version":"1","actual_sha512":"abc","sha512_match":true,"match":true}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	client, err := NewHTTPLPBSReleaseClient(LPBSClientConfig{BaseURL: server.URL, ServiceSecret: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	readiness, err := client.CheckDeployReadiness(context.Background(), &LPBSReadinessRequest{AppKey: "demo", Channel: "stable"})
	if err != nil || readiness.Ready || len(readiness.Gates) != 1 {
		t.Fatalf("readiness = %#v, %v", readiness, err)
	}
	verified, err := client.Verify(context.Background(), &LPBSVerifyRequest{AppKey: "demo", Channel: "stable", Platform: "linux", ExpectedVersion: "1", ExpectedSHA512: "abc", Deep: true})
	if err != nil || verified == nil || !verified.Match {
		t.Fatalf("verify = %#v, %v", verified, err)
	}
	if !verified.SHA512Match || verified.ObservedSHA512 != "abc" {
		t.Fatalf("verify digest = %#v", verified)
	}
}

func TestHTTPLPBSReleaseClientPrefersScopedTokenAndFailsClosedOnExchangeFailure(t *testing.T) {
	newTokens := func(exchange servicetoken.Exchanger) *servicetoken.Tokens {
		return lpbsidentity.DeploymentManagerRelease.NewTokens("deployment-manager", func(context.Context) (string, error) { return "principal-credential", nil }, servicetoken.WithExchanger(exchange))
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer scoped-token" {
			t.Errorf("Authorization = %q, want scoped bearer", got)
		}
		_, _ = w.Write([]byte(`{"ready":false,"gates":[]}`))
	}))
	defer server.Close()
	tokens := newTokens(func(context.Context, string) (servicetoken.Exchanged, error) {
		return servicetoken.Exchanged{AccessToken: "scoped-token", ExpiresAt: time.Now().Add(time.Minute), Subject: lpbsidentity.DeploymentManagerRelease.Principal("deployment-manager"), Audience: lpbsidentity.DeploymentManagerRelease.Audience, Scopes: []string{lpbsidentity.DeploymentManagerRelease.Scope}}, nil
	})
	client, err := NewHTTPLPBSReleaseClient(LPBSClientConfig{BaseURL: server.URL, ServiceSecret: "legacy-secret", Tokens: tokens})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.CheckDeployReadiness(context.Background(), &LPBSReadinessRequest{AppKey: "demo"}); err != nil {
		t.Fatal(err)
	}

	failing := newTokens(func(context.Context, string) (servicetoken.Exchanged, error) {
		return servicetoken.Exchanged{}, errors.New("authenticator unavailable")
	})
	client, err = NewHTTPLPBSReleaseClient(LPBSClientConfig{BaseURL: server.URL, ServiceSecret: "legacy-secret", Tokens: failing})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.CheckDeployReadiness(context.Background(), &LPBSReadinessRequest{AppKey: "demo"}); err == nil || !strings.Contains(err.Error(), "authenticator unavailable") {
		t.Fatalf("exchange failure should fail closed, got %v", err)
	}
}

func TestHTTPLPBSReleaseClientRequiresExactHaltReceipt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/admin/download-channels/halt" || r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("halt request = %s %s auth=%q", r.Method, r.URL.Path, r.Header.Get("Authorization"))
		}
		var request map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request["dry_run"] != true {
			t.Fatalf("halt body = %#v err=%v", request, err)
		}
		_, _ = w.Write([]byte(`{"app_key":"demo","variant_key":"default","revision":4,"halted":true,"outcome":"preview","health":"unknown","dry_run":true}`))
	}))
	defer server.Close()
	client, err := NewHTTPLPBSReleaseClient(LPBSClientConfig{BaseURL: server.URL, ServiceSecret: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := client.HaltChannel(context.Background(), &LPBSRecoveryRequest{AppKey: "demo", VariantKey: "default", ExpectedRevision: 4, Halted: true, DryRun: true})
	if err != nil || receipt == nil || receipt.Outcome != "preview" || !receipt.DryRun {
		t.Fatalf("halt receipt = %#v err=%v", receipt, err)
	}
}

func TestHTTPLPBSReleaseClientRejectsMalformedSuccessfulReadiness(t *testing.T) {
	for name, body := range map[string]string{
		"missing ready":       `{"gates":[]}`,
		"wrong envelope":      `{"data":{"ready":true}}`,
		"invalid json":        `{`,
		"contradictory gates": `{"ready":true,"gates":[{"name":"storage","ready":false}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(body))
			}))
			defer server.Close()
			client, err := NewHTTPLPBSReleaseClient(LPBSClientConfig{BaseURL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := client.CheckDeployReadiness(context.Background(), &LPBSReadinessRequest{AppKey: "demo"}); err == nil {
				t.Fatalf("malformed successful response was accepted: %s", body)
			}
		})
	}
}

func TestHTTPLPBSReleaseClientRejectsVerifyIdentityAndDigestMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"app_key":"other","channel":"stable","platform":"linux","expected_version":"1","actual_version":"1","actual_sha512":"wrong","match":true}`))
	}))
	defer server.Close()
	client, err := NewHTTPLPBSReleaseClient(LPBSClientConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Verify(context.Background(), &LPBSVerifyRequest{AppKey: "demo", Channel: "stable", Platform: "linux", ExpectedVersion: "1", ExpectedSHA512: "abc"}); err == nil {
		t.Fatal("verify accepted a response for a different app")
	}
}

func TestHTTPLPBSReleaseClientRejectsNonCanonicalVerifyEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"app_key":"demo","channel":"stable","platform":"linux","expected_version":"1","observed_version":"1","observed_sha512":"abc","match":true}`))
	}))
	defer server.Close()
	client, err := NewHTTPLPBSReleaseClient(LPBSClientConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Verify(context.Background(), &LPBSVerifyRequest{AppKey: "demo", Channel: "stable", Platform: "linux", ExpectedVersion: "1", ExpectedSHA512: "abc"}); err == nil {
		t.Fatal("non-canonical verify envelope was accepted")
	}
}

func TestHTTPLPBSReleaseClientAcceptsEquivalentDigestEncodings(t *testing.T) {
	payloadDigest := sha512.Sum512([]byte("published artifact"))
	hexDigest := hex.EncodeToString(payloadDigest[:])
	base64Digest := base64.StdEncoding.EncodeToString(payloadDigest[:])
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("deep") != "true" {
			t.Fatalf("expected deep verification query, got %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"app_key":"demo","channel":"stable","platform":"linux","expected_version":"1","actual_version":"1","actual_sha512":"` + base64Digest + `","match":true,"sha512_match":true}`))
	}))
	defer server.Close()
	client, err := NewHTTPLPBSReleaseClient(LPBSClientConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Verify(context.Background(), &LPBSVerifyRequest{
		AppKey: "demo", Channel: "stable", Platform: "linux", ExpectedVersion: "1", ExpectedSHA512: hexDigest, Deep: true,
	})
	if err != nil || result == nil || !result.Match || !result.SHA512Match {
		t.Fatalf("equivalent digest encodings were rejected: %#v, %v", result, err)
	}
}

func TestHTTPLPBSReleaseClientHandlesConfigurationAndResponses(t *testing.T) {
	t.Setenv("LPBS_BASE_URL", "")
	t.Setenv("LPBS_SERVICE_SECRET", "")
	client, err := NewHTTPLPBSReleaseClient(LPBSClientConfig{
		ResolveBaseURL: func(context.Context) (string, error) { return "", errors.New("discovery unavailable") },
	})
	if err != nil {
		t.Fatalf("deferred discovery should not fail construction: %v", err)
	}
	if _, err := client.CheckDeployReadiness(context.Background(), &LPBSReadinessRequest{AppKey: "demo"}); err == nil {
		t.Fatal("undiscoverable base URL returned nil operation error")
	}
	client = &HTTPLPBSReleaseClient{baseURL: "http://127.0.0.1:1", httpClient: http.DefaultClient}
	if _, err := client.Verify(context.Background(), &LPBSVerifyRequest{}); err == nil || !strings.Contains(err.Error(), "app_key") {
		t.Fatalf("missing app key error = %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/deploy-readiness" {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("down"))
			return
		}
		if r.URL.Path == "/api/v1/updates/demo/verify" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("missing"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	client, err = NewHTTPLPBSReleaseClient(LPBSClientConfig{BaseURL: server.URL, ServiceSecret: "synthetic-test-token"})
	if err != nil {
		t.Fatal(err)
	}
	readiness, err := client.CheckDeployReadiness(context.Background(), &LPBSReadinessRequest{AppKey: "demo"})
	if err != nil || readiness.Ready || readiness.Error == "" {
		t.Fatalf("failed readiness = %#v, %v", readiness, err)
	}
	verified, err := client.Verify(context.Background(), &LPBSVerifyRequest{AppKey: "demo"})
	if err != nil || verified.Match || verified.Error == "" {
		t.Fatalf("failed verify = %#v, %v", verified, err)
	}
	badJSON := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("not-json")) }))
	defer badJSON.Close()
	client.baseURL = badJSON.URL
	if _, err := client.Verify(context.Background(), &LPBSVerifyRequest{AppKey: "demo"}); err == nil {
		t.Fatal("invalid verify JSON returned nil error")
	}
}

func TestNewHTTPLPBSReleaseClientDefersBaseURLResolutionUntilEachOperation(t *testing.T) {
	t.Setenv("LPBS_BASE_URL", "")
	var requestsA, requestsB int
	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestsA++
		_, _ = w.Write([]byte(`{"ready":false}`))
	}))
	defer serverA.Close()
	serverB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestsB++
		_, _ = w.Write([]byte(`{"ready":false}`))
	}))
	defer serverB.Close()
	resolutions := 0
	client, err := NewHTTPLPBSReleaseClient(LPBSClientConfig{
		ResolveBaseURL: func(context.Context) (string, error) {
			resolutions++
			if resolutions == 1 {
				return serverA.URL + "/", nil
			}
			return serverB.URL, nil
		},
		ResolveServiceSecret: func() (string, error) { return "authority-secret", nil },
	})
	if err != nil {
		t.Fatalf("constructor error: %v", err)
	}
	if resolutions != 0 {
		t.Fatalf("constructor resolutions = %d, want 0", resolutions)
	}
	if client.baseURL != "" || client.resolveBaseURL == nil {
		t.Fatalf("dynamic address state = base %q, resolver configured %v", client.baseURL, client.resolveBaseURL != nil)
	}
	if client.serviceSecret != "" {
		t.Fatalf("serviceSecret = %q, want no eager secret resolution", client.serviceSecret)
	}
	for range 2 {
		if _, err := client.CheckDeployReadiness(context.Background(), &LPBSReadinessRequest{AppKey: "demo"}); err != nil {
			t.Fatal(err)
		}
	}
	if resolutions != 2 || requestsA != 1 || requestsB != 1 {
		t.Fatalf("resolutions=%d requests A/B=%d/%d, want 2 and 1/1", resolutions, requestsA, requestsB)
	}
}

func TestNewHTTPLPBSReleaseClientDefersLegacyServiceSecretResolution(t *testing.T) {
	t.Setenv("LPBS_BASE_URL", "")
	t.Setenv("LPBS_SERVICE_SECRET", "")
	resolved := false
	client, err := NewHTTPLPBSReleaseClient(LPBSClientConfig{
		BaseURL: "http://lpbs.test",
		ResolveServiceSecret: func() (string, error) {
			resolved = true
			return "authority-secret", nil
		},
	})
	if err != nil {
		t.Fatalf("constructor error: %v", err)
	}
	if resolved || client.serviceSecret != "" {
		t.Fatalf("legacy authority secret was read before authorization: resolved=%v secret=%q", resolved, client.serviceSecret)
	}
}

func TestNewHTTPLPBSReleaseClientPrefersExplicitConfigOverOwners(t *testing.T) {
	t.Setenv("LPBS_BASE_URL", "")
	t.Setenv("LPBS_SERVICE_SECRET", "")
	client, err := NewHTTPLPBSReleaseClient(LPBSClientConfig{
		BaseURL:       "http://explicit.test",
		ServiceSecret: "explicit-secret",
		ResolveBaseURL: func(context.Context) (string, error) {
			t.Fatal("discovery resolver must not run when BaseURL is explicit")
			return "", nil
		},
		ResolveServiceSecret: func() (string, error) {
			t.Fatal("authority resolver must not run when ServiceSecret is explicit")
			return "", nil
		},
	})
	if err != nil {
		t.Fatalf("constructor error: %v", err)
	}
	if client.baseURL != "http://explicit.test" || client.serviceSecret != "explicit-secret" {
		t.Fatalf("explicit config not preferred: base=%q secret=%q", client.baseURL, client.serviceSecret)
	}
	if client.resolveBaseURL != nil {
		t.Fatal("explicit LPBS URL must remain pinned")
	}
}

func TestLegacyServiceSecretEnvironmentIsIgnored(t *testing.T) {
	t.Setenv("LPBS_BASE_URL", "")
	t.Setenv("LPBS_SERVICE_SECRET", "must-not-be-used")
	client, err := NewHTTPLPBSReleaseClient(LPBSClientConfig{
		BaseURL: "http://lpbs.test",
	})
	if err != nil {
		t.Fatalf("constructor error: %v", err)
	}
	if client.serviceSecret != "" {
		t.Fatalf("serviceSecret = %q, want empty", client.serviceSecret)
	}
}

func TestLegacyServiceSecretResolvedOnlyForConfirmedUnissuedPrincipal(t *testing.T) {
	t.Setenv("LPBS_BASE_URL", "")
	resolutions := 0
	role := lpbsidentity.DeploymentManagerRelease
	tokens := servicetoken.New(servicetoken.Config{
		Name:   "deployment-manager " + role.Label,
		Expect: servicetoken.ExpectIdentity(role.Principal("deployment-manager"), role.Audience, role.Scope),
		Credential: func(context.Context) (string, error) {
			return "", credentialauthority.ErrUnconfigured
		},
		Describe: func(kind, cause error) error {
			return fmt.Errorf("%w: %w", kind, cause)
		},
	})
	client, err := NewHTTPLPBSReleaseClient(LPBSClientConfig{
		BaseURL: "http://lpbs.test",
		Tokens:  tokens,
		ResolveServiceSecret: func() (string, error) {
			resolutions++
			return "legacy-recovery-token", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolutions != 0 {
		t.Fatal("legacy credential resolved before the scoped identity check")
	}
	req, err := http.NewRequest(http.MethodGet, "http://lpbs.test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.authorize(context.Background(), req); err != nil {
		t.Fatalf("confirmed-unissued fallback failed: %v", err)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer legacy-recovery-token" || resolutions != 1 {
		t.Fatalf("authorization=%q resolutions=%d", got, resolutions)
	}
}

func TestLazyLPBSReleaseClientRetriesResolutionOnLaterUse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ready": true,
			"gates": []map[string]any{{"name": "download_storage", "ready": true}},
		})
	}))
	defer server.Close()

	attempts := 0
	client := NewLazyLPBSReleaseClient(LPBSClientConfig{
		ResolveBaseURL: func(context.Context) (string, error) {
			attempts++
			if attempts == 1 {
				return "", errors.New("scenario_not_running")
			}
			return server.URL, nil
		},
		ResolveServiceSecret: func() (string, error) { return "secret", nil },
	})

	if _, err := client.CheckDeployReadiness(context.Background(), &LPBSReadinessRequest{AppKey: "web-console"}); err == nil {
		t.Fatal("expected the first resolution to fail while LPBS was not running")
	}
	result, err := client.CheckDeployReadiness(context.Background(), &LPBSReadinessRequest{AppKey: "web-console"})
	if err != nil {
		t.Fatalf("second call returned error: %v", err)
	}
	if result == nil || !result.Ready {
		t.Fatalf("result = %+v, want a ready result after retry", result)
	}
	if attempts != 2 {
		t.Fatalf("resolve attempts = %d, want 2 (resolution retried)", attempts)
	}
}
