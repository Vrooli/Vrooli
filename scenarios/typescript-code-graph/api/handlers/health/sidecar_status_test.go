package health_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"typescript-code-graph/handlers/health"
	"typescript-code-graph/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
)

// TestHandler_AugmentsResponseWithSidecarStatus pins the wiring
// between the health handler and the sidecar status provider. When a
// SidecarStatusProvider is wired, the JSON response carries a
// sidecar_status field reporting the provider's current value.
// Operators reading /health on a degraded box need this field to be
// the first thing they see.
func TestHandler_AugmentsResponseWithSidecarStatus(t *testing.T) {
	h := health.NewHandler(health.Deps{
		Pinger:  &mocks.FakePinger{},
		Service: "tcg-test",
		Version: "1.0.0",
		Sidecar: health.FuncProvider(func() string { return "STATUS_READY" }),
	})

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	resp, err := srv.Client().Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var payload map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	require.Equal(t, "STATUS_READY", payload["sidecar_status"])
	// Standard envelope still intact.
	require.Equal(t, "healthy", payload["status"])
}

// TestHandler_OmitsSidecarFieldWhenNotWired is the negative: the
// health handler with no SidecarStatusProvider behaves exactly like
// the upstream apihealth handler (no sidecar_status key). This keeps
// the template-inherited handler_test.go passing without changes.
func TestHandler_OmitsSidecarFieldWhenNotWired(t *testing.T) {
	h := health.NewHandler(health.Deps{
		Pinger:  &mocks.FakePinger{},
		Service: "tcg-test",
		Version: "1.0.0",
	})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	resp, err := srv.Client().Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	var payload map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	_, ok := payload["sidecar_status"]
	require.False(t, ok, "sidecar_status must be absent when provider is nil")
}
