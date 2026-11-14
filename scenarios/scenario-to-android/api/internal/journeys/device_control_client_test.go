package journeys

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPDeviceClientReadNetworkStateUsesLeaseScopedReadOnlyOperation(t *testing.T) {
	wifi, data, airplane := true, true, false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		require.Equal(t, "/api/v1/devices/android-target/apps", request.URL.Path)
		var payload map[string]any
		require.NoError(t, json.NewDecoder(request.Body).Decode(&payload))
		require.Equal(t, "network-state", payload["operation"])
		require.Equal(t, "lease-token", payload["lease_token"])
		require.Equal(t, false, payload["confirmed"])
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{"result": map[string]any{
			"operation": "network-state", "status": "available",
			"network_state": map[string]any{"wifi_enabled": wifi, "mobile_data_enabled": data, "airplane_mode_enabled": airplane},
		}})
	}))
	defer server.Close()

	state, err := (&HTTPDeviceClient{BaseURL: server.URL}).ReadNetworkState(context.Background(), Lease{DeviceID: "android-target", Token: "lease-token"})
	require.NoError(t, err)
	require.Equal(t, &wifi, state.WifiEnabled)
	require.Equal(t, &data, state.MobileDataEnabled)
	require.Equal(t, &airplane, state.AirplaneModeEnabled)
}

func TestHTTPDeviceClientReadNetworkStateRejectsIncompleteSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"result":{"operation":"network-state","status":"available","network_state":{"wifi_enabled":true,"airplane_mode_enabled":false}}}`))
	}))
	defer server.Close()

	_, err := (&HTTPDeviceClient{BaseURL: server.URL}).ReadNetworkState(context.Background(), Lease{DeviceID: "android-target", Token: "lease-token"})
	require.ErrorContains(t, err, "incomplete")
}

func TestHTTPDeviceClientReadPackageSigningStateRequiresInstalledSigner(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var payload map[string]any
		require.NoError(t, json.NewDecoder(request.Body).Decode(&payload))
		require.Equal(t, "package-signing-state", payload["operation"])
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{"result": map[string]any{
			"operation": "package-signing-state", "package": "com.example.app", "status": "available",
			"installed": true, "signing_certificate_sha256": "abc123",
		}})
	}))
	defer server.Close()

	digest, err := (&HTTPDeviceClient{BaseURL: server.URL}).ReadPackageSigningState(context.Background(), Lease{DeviceID: "android-target", Token: "lease-token"}, "com.example.app")
	require.NoError(t, err)
	require.Equal(t, "abc123", digest)
}

func TestHTTPDeviceClientExecuteReadsArtifactSignerFromConnectJSON(t *testing.T) {
	digest := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	for _, test := range []struct {
		name        string
		disposition string
		wantErr     bool
	}{
		{name: "successful install", disposition: "passed"},
		{name: "partial proof on failed install", disposition: "failed", wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(writer).Encode(map[string]any{"disposition": test.disposition, "chapters": []any{map[string]any{
					"disposition":     test.disposition,
					"artifactInstall": map[string]any{"signingCertificateSha256": digest},
				}}})
			}))
			defer server.Close()

			result, err := (&HTTPDeviceClient{BaseURL: server.URL}).Execute(context.Background(), Lease{DeviceID: "android-target", Token: "lease-token"}, "install-artifact", map[string]string{"step_id": "artifact-install/install"})
			if test.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, digest, result.ArtifactSigningCertificateSHA256)
		})
	}
}
