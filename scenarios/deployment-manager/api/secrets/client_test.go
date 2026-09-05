package secrets

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchBundleSecretsUsesTierAndParsesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/deployment/secrets/demo" || r.URL.Query().Get("tier") != "desktop" || r.URL.Query().Get("include_optional") != "true" {
			t.Errorf("request = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"bundle_secrets":[{"id":"key","required":true,"target":{"type":"env","name":"KEY"}}]}`))
	}))
	defer server.Close()
	t.Setenv("SECRETS_MANAGER_URL", server.URL)
	got, err := NewClient().FetchBundleSecrets(context.Background(), "demo", "desktop")
	if err != nil || len(got) != 1 || got[0].ID != "key" || got[0].Target.Name != "KEY" {
		t.Fatalf("secrets = %#v, %v", got, err)
	}
}

func TestFetchBundleSecretsReportsHTTPAndDecodeErrors(t *testing.T) {
	for name, body := range map[string]string{
		"http error":   "status:503 body:down",
		"decode error": "not-json",
	} {
		t.Run(name, func(t *testing.T) {
			status := http.StatusOK
			response := body
			if name == "http error" {
				status = http.StatusServiceUnavailable
				response = "down"
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(response))
			}))
			defer server.Close()
			t.Setenv("SECRETS_MANAGER_URL", server.URL)
			_, err := NewClient().FetchBundleSecrets(context.Background(), "demo", "desktop")
			if err == nil || !strings.Contains(err.Error(), "secrets-manager") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
