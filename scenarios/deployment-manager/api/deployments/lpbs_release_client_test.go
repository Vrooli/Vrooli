package deployments

import (
	"context"
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
			_, _ = w.Write([]byte(`{"app_key":"demo","channel":"stable","platform":"linux","expected_version":"1","observed_version":"1","sha512_match":true,"match":true}`))
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
	if err != nil || !readiness.Ready || len(readiness.Gates) != 1 {
		t.Fatalf("readiness = %#v, %v", readiness, err)
	}
	verified, err := client.Verify(context.Background(), &LPBSVerifyRequest{AppKey: "demo", Channel: "stable", Platform: "linux", ExpectedVersion: "1", Deep: true})
	if err != nil || verified == nil || !verified.Match {
		t.Fatalf("verify = %#v, %v", verified, err)
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
