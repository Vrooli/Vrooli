package deployments

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
	if _, err := NewHTTPLPBSReleaseClient(LPBSClientConfig{}); err == nil {
		t.Fatal("missing base URL returned nil error")
	}
	client := &HTTPLPBSReleaseClient{baseURL: "http://127.0.0.1:1", httpClient: http.DefaultClient}
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
	client, err := NewHTTPLPBSReleaseClient(LPBSClientConfig{BaseURL: server.URL})
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
