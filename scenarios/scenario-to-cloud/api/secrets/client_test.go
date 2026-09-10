package secrets

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/api-core/discovery"
)

func TestFetchBundleSecretsSendsScopedServiceToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer deployment-token" {
			t.Fatalf("authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"scenario":"demo","tier":"tier-4-saas","secrets":[]}`))
	}))
	defer server.Close()

	client := &Client{
		httpClient:   &http.Client{},
		serviceToken: "deployment-token",
		resolver:     discovery.NewStaticResolver(server.URL),
	}
	if _, err := client.FetchBundleSecrets(context.Background(), "demo", "tier-4-saas", nil); err != nil {
		t.Fatalf("FetchBundleSecrets() error = %v", err)
	}
}

func TestNewClientDoesNotReadLegacyDeploymentTokenEnvironment(t *testing.T) {
	t.Setenv("SECRETS_MANAGER_DEPLOYMENT_TOKEN", "legacy-token")

	client := NewClient()
	if client.serviceToken != "" {
		t.Fatalf("NewClient() read the legacy deployment token environment variable")
	}
}
