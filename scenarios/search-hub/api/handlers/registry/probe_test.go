package registry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

type probeResolver string

func (p probeResolver) ResolveScenarioURL(context.Context, string) (string, error) {
	return string(p), nil
}

func TestHTTPProbeRejectsFourHundredProviderContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("unexpected probe request: %s %s", r.Method, r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()
	descriptor := &registryv1.ProviderDescriptor{Endpoint: &registryv1.Endpoint{Kind: &registryv1.Endpoint_HttpJson{HttpJson: &registryv1.HttpJsonEndpoint{
		ScenarioId: "fixture", Path: "/search", Method: registryv1.HttpMethod_HTTP_METHOD_POST,
		BodyTemplate: `{"query":"{{query}}","limit":{{limit}}}`,
	}}}}
	err := (HTTPProbe{Resolver: probeResolver(server.URL)}).Probe(context.Background(), descriptor)
	require.Error(t, err)
	require.Contains(t, err.Error(), "HTTP 400")
}

func TestProbeBodySubstitutesTemplateWithoutInvalidJSON(t *testing.T) {
	body := probeBody(`{"query":"{{query}}","limit":{{limit}},"scope":"{{scope}}"}`)
	require.JSONEq(t, `{"query":"__search_hub_registration_probe__","limit":1,"scope":""}`, string(body))
}
