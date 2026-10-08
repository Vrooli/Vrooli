package publicproxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDeviceSyncContentProxiesOnlyAuthenticatedItemContent(t *testing.T) {
	itemID := uuid.NewString()
	hub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/transfer/items/"+itemID+"/content", r.URL.Path)
		require.Equal(t, "target-token", r.Header.Get("X-Device-Token"))
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write([]byte("artifact"))
	}))
	defer hub.Close()

	proxy := httptest.NewServer(DeviceSyncContent(hub.URL, hub.Client()))
	defer proxy.Close()
	request, err := http.NewRequest(http.MethodGet, proxy.URL+"/public/device-sync-hub/api/v1/transfer/items/"+itemID+"/content", nil)
	require.NoError(t, err)
	request.Header.Set("X-Device-Token", "target-token")
	response, err := proxy.Client().Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "artifact", string(body))
	require.Equal(t, "application/zip", response.Header.Get("Content-Type"))

	unauthenticated, err := http.Get(proxy.URL + "/public/device-sync-hub/api/v1/transfer/items/" + itemID + "/content")
	require.NoError(t, err)
	defer unauthenticated.Body.Close()
	require.Equal(t, http.StatusUnauthorized, unauthenticated.StatusCode)
}

func TestDeviceSyncContentRejectsArbitraryPaths(t *testing.T) {
	proxy := httptest.NewServer(DeviceSyncContent("http://127.0.0.1:1", nil))
	defer proxy.Close()
	response, err := http.Get(proxy.URL + "/public/device-sync-hub/items/not-a-uuid/content")
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusUnauthorized, response.StatusCode)
}
