package artifacts

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	internalartifacts "vrooli-bridge/internal/artifacts"

	"github.com/stretchr/testify/require"
	transferv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/transfer"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestDeviceSyncDeliveryStreamsDirectedUpload(t *testing.T) {
	var gotToken, gotTarget, gotName, gotPayload string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-Device-Token")
		if gotToken != "hub-token" {
			http.Error(w, "missing test token", http.StatusUnauthorized)
			return
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		gotTarget = r.FormValue("target_device_id")
		gotName = r.FormValue("name")
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		gotPayload = string(data)

		body, err := protojson.Marshal(&transferv1.UploadItemResponse{Item: &transferv1.Item{Id: "item-123"}})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(body)
	}))
	defer server.Close()

	path := filepath.Join(t.TempDir(), "installer.bin")
	require.NoError(t, os.WriteFile(path, []byte("installer-bytes"), 0o600))
	delivery := deviceSyncDelivery{
		endpoint:   server.URL,
		token:      "hub-token",
		targets:    map[string]string{"bridge-node": "hub-device"},
		httpClient: server.Client(),
	}

	got, err := delivery.Deliver(context.Background(), internalartifacts.DeliveryRequest{
		NodeID: "bridge-node", Name: "installer.bin", SourceRef: path,
	})
	require.NoError(t, err)
	require.Equal(t, "dsh://item/item-123", got.Ref)
	require.False(t, got.Delivered)
	require.Equal(t, "hub-token", gotToken)
	require.Equal(t, "hub-device", gotTarget)
	require.Equal(t, "installer.bin", gotName)
	require.Equal(t, "installer-bytes", gotPayload)
}

func TestDeviceSyncDeliveryRefusesMissingTrustConfiguration(t *testing.T) {
	delivery := deviceSyncDelivery{endpoint: "http://127.0.0.1:1"}

	_, err := delivery.Deliver(context.Background(), internalartifacts.DeliveryRequest{
		NodeID: "bridge-node", SourceRef: "file:///tmp/installer.bin",
	})
	require.EqualError(t, err, `device-sync-hub target mapping is not configured for bridge node "bridge-node"`)

	delivery.targets = map[string]string{"bridge-node": "hub-device"}
	_, err = delivery.Deliver(context.Background(), internalartifacts.DeliveryRequest{
		NodeID: "bridge-node", SourceRef: "file:///tmp/installer.bin",
	})
	require.EqualError(t, err, "device-sync-hub device token is not configured")
}

func TestOpenArtifactSourceRejectsUnresolvedContentReference(t *testing.T) {
	_, _, err := openArtifactSource(context.Background(), "blob://builds/installer.bin", "installer.bin", nil)
	require.EqualError(t, err, `artifact source scheme "blob" is unsupported; use a local path, file://, or HTTP(S)`)
}
