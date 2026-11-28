package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbeUsesBasicAuthenticationWithoutMutatingProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", request.Method)
		}
		accountSID, token, ok := request.BasicAuth()
		if !ok || accountSID != "AC123" || token != "token" {
			t.Fatalf("unexpected basic auth: %q %q %t", accountSID, token, ok)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	status, err := Probe(context.Background(), server.Client(), server.URL, "AC123", "token")
	if err != nil || status != http.StatusOK {
		t.Fatalf("Probe() = %d, %v", status, err)
	}
}

func TestProbeRejectsMissingCredentials(t *testing.T) {
	if _, err := Probe(context.Background(), nil, "https://example.test", "", ""); err == nil {
		t.Fatal("expected missing credentials to fail")
	}
}

func TestProbeReportsCredentialRejection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	if status, err := Probe(context.Background(), server.Client(), server.URL, "AC123", "bad"); err == nil || status != http.StatusForbidden {
		t.Fatalf("Probe() = %d, %v; want forbidden error", status, err)
	}
}

func TestProbeReportsProviderFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	if status, err := Probe(context.Background(), server.Client(), server.URL, "AC123", "token"); err == nil || status != http.StatusServiceUnavailable {
		t.Fatalf("Probe() = %d, %v; want provider failure", status, err)
	}
}
