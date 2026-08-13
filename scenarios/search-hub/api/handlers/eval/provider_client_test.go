package eval

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	internaleval "search-hub/internal/eval"
	"search-hub/internal/testutil/mocks"

	aisearch "github.com/vrooli/ai-go/search"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

type staticResolver struct{ url string }

func (s staticResolver) ResolveScenarioURL(context.Context, string) (string, error) {
	return s.url, nil
}

func httpJSONDescriptor() *registryv1.ProviderDescriptor {
	return &registryv1.ProviderDescriptor{
		ProviderId: "cli-health.commands",
		Type:       "command",
		Endpoint: &registryv1.Endpoint{Kind: &registryv1.Endpoint_HttpJson{HttpJson: &registryv1.HttpJsonEndpoint{
			ScenarioId:   "cli-health",
			Path:         "/search",
			Method:       registryv1.HttpMethod_HTTP_METHOD_POST,
			BodyTemplate: `{"query":"{{query}}","limit":{{limit}}}`,
		}}},
		ResultMapping: &registryv1.ResultMapping{
			ResultsPath: "results", IdField: "name", TitleField: "name", ScoreField: "score",
		},
	}
}

func TestSearchForwardsOverrideHeaders(t *testing.T) {
	doer := &mocks.FakeDoer{}
	doer.AddResponse(http.StatusOK, []byte(`{"results":[]}`))
	c := newHTTPProviderClient(staticResolver{url: "http://provider.test"}, doer)

	ov := aisearch.SearchOverrides{
		RerankEnabled:   aisearch.OverrideBool(true),
		RerankShortlist: aisearch.OverrideInt(40),
	}
	_, err := c.Search(context.Background(), httpJSONDescriptor(), "restart", 10,
		internaleval.SearchCallOptions{Overrides: &ov, ControlToken: "tok-xyz"})
	require.NoError(t, err)
	require.Len(t, doer.Requests, 1)

	req := doer.Requests[0]
	require.NotEmpty(t, req.Header.Get(aisearch.OverridesHeader), "override header must be set")
	require.Equal(t, "tok-xyz", req.Header.Get(aisearch.ControlTokenHeader), "control token must be forwarded")

	// The header value round-trips back to the overrides we sent.
	got, perr := aisearch.ParseOverridesHeader(req.Header.Get(aisearch.OverridesHeader))
	require.NoError(t, perr)
	require.NotNil(t, got.RerankEnabled)
	require.True(t, *got.RerankEnabled)
	require.NotNil(t, got.RerankShortlist)
	require.Equal(t, 40, *got.RerankShortlist)
}

func TestSearchBaselineSendsNoOverrideHeaders(t *testing.T) {
	doer := &mocks.FakeDoer{}
	doer.AddResponse(http.StatusOK, []byte(`{"results":[]}`))
	c := newHTTPProviderClient(staticResolver{url: "http://provider.test"}, doer)

	_, err := c.Search(context.Background(), httpJSONDescriptor(), "restart", 10,
		internaleval.SearchCallOptions{})
	require.NoError(t, err)
	require.Len(t, doer.Requests, 1)

	req := doer.Requests[0]
	require.Empty(t, req.Header.Get(aisearch.OverridesHeader), "baseline call must not set the override header")
	require.Empty(t, req.Header.Get(aisearch.ControlTokenHeader), "baseline call must not leak a token header")
}

func TestSearchRendersScopePlaceholders(t *testing.T) {
	doer := &mocks.FakeDoer{}
	doer.AddResponse(http.StatusOK, []byte(`{"results":[]}`))
	c := newHTTPProviderClient(staticResolver{url: "http://provider.test"}, doer)
	desc := httpJSONDescriptor()
	desc.Endpoint.GetHttpJson().BodyTemplate = `{"query":"{{query}}","limit":{{limit}},"scope":"{{scope_kind}}","target":"{{scope_value}}"}`

	_, err := c.Search(context.Background(), desc, "architecture", 10,
		internaleval.SearchCallOptions{Scope: "scenario:cli-health"})
	require.NoError(t, err)
	require.Len(t, doer.Requests, 1)

	require.JSONEq(t,
		`{"query":"architecture","limit":10,"scope":"scenario","target":"cli-health"}`,
		readRequestBody(t, doer.Requests[0]),
	)
}

func TestApplyOverrideHeadersTokenOmittedWhenEmpty(t *testing.T) {
	// Overrides present but no token: the override header rides, the token header
	// does not (the provider gate will then reject — by design).
	req, err := http.NewRequest(http.MethodPost, "http://x/search", http.NoBody)
	require.NoError(t, err)
	ov := aisearch.SearchOverrides{RerankBlend: aisearch.OverrideBool(false)}
	require.NoError(t, applyOverrideHeaders(req, internaleval.SearchCallOptions{Overrides: &ov}))
	require.NotEmpty(t, req.Header.Get(aisearch.OverridesHeader))
	require.Empty(t, req.Header.Get(aisearch.ControlTokenHeader))
}

func TestApplyOverrideHeadersNoopForZeroOverrides(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "http://x/search", http.NoBody)
	require.NoError(t, err)
	// nil overrides.
	require.NoError(t, applyOverrideHeaders(req, internaleval.SearchCallOptions{ControlToken: "tok"}))
	require.Empty(t, req.Header.Get(aisearch.OverridesHeader))
	require.Empty(t, req.Header.Get(aisearch.ControlTokenHeader), "token without overrides is never sent")
	// explicitly-zero overrides.
	require.NoError(t, applyOverrideHeaders(req, internaleval.SearchCallOptions{Overrides: &aisearch.SearchOverrides{}, ControlToken: "tok"}))
	require.Empty(t, req.Header.Get(aisearch.OverridesHeader))
}

func TestParseIndexTimestampUsesDeclaredDirectField(t *testing.T) {
	got := parseIndexTimestamp([]byte(`{"lastReconcileAt":"2026-08-13T11:34:20.777491506Z"}`), "lastReconcileAt")
	require.Equal(t, "2026-08-13T11:34:20.777491506Z", got.UTC().Format(time.RFC3339Nano))
}

func TestParseIndexTimestampUsesDeclaredNestedField(t *testing.T) {
	got := parseIndexTimestamp([]byte(`{"metrics":{"last_indexed_at":"2026-08-13T11:34:20.777491506Z"}}`), "metrics.last_indexed_at")
	require.Equal(t, "2026-08-13T11:34:20.777491506Z", got.UTC().Format(time.RFC3339Nano))
}

func readRequestBody(t *testing.T, req *http.Request) string {
	t.Helper()
	raw, err := req.GetBody()
	if err != nil {
		t.Fatalf("GetBody: %v", err)
	}
	defer raw.Close()
	buf := new(strings.Builder)
	if _, err := io.Copy(buf, raw); err != nil {
		t.Fatalf("read body: %v", err)
	}
	return buf.String()
}
