package journeys

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPWebViewAttacherUsesLeasedDeviceAndReturnsEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/devices/device-1/webview/attach", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"endpoint":{"cdp_endpoint":"http://127.0.0.1:43123","renderer_id":"renderer-1","renderer_url":"http://localhost:17827/"}}`))
	}))
	defer server.Close()

	endpoint, err := (HTTPWebViewAttacher{BaseURL: server.URL, Actor: "scenario-to-android"}).AttachWebView(context.Background(), Lease{DeviceID: "device-1", Token: "lease-token"}, "com.example.hello")
	require.NoError(t, err)
	require.Equal(t, WebViewAttachment{CDPEndpoint: "http://127.0.0.1:43123", RendererID: "renderer-1", RendererURL: "http://localhost:17827/"}, endpoint)
}

func TestHTTPWebViewAttacherRequiresDeviceIdentity(t *testing.T) {
	_, err := (HTTPWebViewAttacher{BaseURL: "http://device-control"}).AttachWebView(context.Background(), Lease{Token: "lease-token"}, "com.example.hello")
	require.ErrorContains(t, err, "leased device identity")
}
