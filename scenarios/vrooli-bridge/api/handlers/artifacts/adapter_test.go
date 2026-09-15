package artifacts

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	internalartifacts "vrooli-bridge/internal/artifacts"
	"vrooli-bridge/internal/channelsign"
	"vrooli-bridge/internal/cpkeys"
	"vrooli-bridge/internal/presence"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/scheduletest"
	transferv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/transfer"
	"google.golang.org/protobuf/encoding/protojson"
)

type recordingArtifactPusher struct {
	nodeID, distributionID, itemID, name, destinationPath string
	calls                                                 int
}

func (p *recordingArtifactPusher) PushArtifact(_ context.Context, nodeID, distributionID, itemID, name, destinationPath string) (int, error) {
	p.nodeID = nodeID
	p.distributionID = distributionID
	p.itemID = itemID
	p.name = name
	p.destinationPath = destinationPath
	p.calls++
	return 1, nil
}

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

func TestDeviceSyncDeliveryQueuesSignedPlacementInstruction(t *testing.T) {
	var gotToken string
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
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()
		_, err = io.Copy(io.Discard, file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		body, err := protojson.Marshal(&transferv1.UploadItemResponse{Item: &transferv1.Item{Id: "item-456"}})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(body)
	}))
	defer server.Close()

	source := filepath.Join(t.TempDir(), "installer.bin")
	require.NoError(t, os.WriteFile(source, []byte("installer-bytes"), 0o600))
	pusher := &recordingArtifactPusher{}
	delivery := deviceSyncDelivery{
		endpoint:   server.URL,
		token:      "hub-token",
		targets:    map[string]string{"bridge-node": "hub-device"},
		pusher:     pusher,
		httpClient: server.Client(),
	}

	got, err := delivery.Deliver(context.Background(), internalartifacts.DeliveryRequest{
		DistributionID: "dist-123", NodeID: "bridge-node", Name: "installer.bin", SourceRef: source,
		DestinationPath: "/opt/vrooli/installer.bin",
	})
	require.NoError(t, err)
	require.False(t, got.Delivered, "the node has not acknowledged placement yet")
	require.Equal(t, "accepted by device-sync-hub; signed target placement instruction queued", got.Detail)
	require.Equal(t, "hub-token", gotToken)
	require.Equal(t, 1, pusher.calls)
	require.Equal(t, "bridge-node", pusher.nodeID)
	require.Equal(t, "dist-123", pusher.distributionID)
	require.Equal(t, "item-456", pusher.itemID)
	require.Equal(t, "installer.bin", pusher.name)
	require.Equal(t, "/opt/vrooli/installer.bin", pusher.destinationPath)
}

func TestArtifactPlacementPusherSignsTypedDeliveryFrame(t *testing.T) {
	hub := presence.NewHub(scheduletest.New(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)))
	conn := hub.Connect("bridge-node")
	defer conn.Close()

	key, err := cpkeys.LoadOrCreate(t.TempDir())
	require.NoError(t, err)
	pusher := NewArtifactPlacementPusher(hub, key)

	require.Equal(t, 1, mustPushArtifact(t, pusher, "bridge-node", "dist-123", "item-456", "installer.bin", "/opt/vrooli/installer.bin"))

	select {
	case payload := <-conn.Out():
		frame, err := channelsign.Open(key.PublicKey(), payload)
		require.NoError(t, err)
		delivery := frame.GetArtifactDelivery()
		require.NotNil(t, delivery)
		require.Equal(t, "dist-123", delivery.GetDistributionId())
		require.Equal(t, "item-456", delivery.GetItemId())
		require.Equal(t, "installer.bin", delivery.GetName())
		require.Equal(t, "/opt/vrooli/installer.bin", delivery.GetDestinationPath())
	case <-time.After(time.Second):
		t.Fatal("no artifact placement frame pushed to the node channel")
	}
}

func mustPushArtifact(t *testing.T, pusher ArtifactPlacementPusher, nodeID, distributionID, itemID, name, destinationPath string) int {
	t.Helper()
	delivered, err := pusher.PushArtifact(context.Background(), nodeID, distributionID, itemID, name, destinationPath)
	require.NoError(t, err)
	return delivered
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

func TestDeviceSyncTokenSelectionPrefersAuthorityAndRetainsCompatibilityFallback(t *testing.T) {
	require.Equal(t, "authority-token", selectDeviceSyncToken(" authority-token ", "compatibility-token"))
	require.Equal(t, "compatibility-token", selectDeviceSyncToken("", " compatibility-token "))
	require.Equal(t, "", selectDeviceSyncToken(" ", " "))
}

func TestOpenArtifactSourceRejectsUnresolvedContentReference(t *testing.T) {
	_, _, err := openArtifactSource(context.Background(), "blob://builds/installer.bin", "installer.bin", nil)
	require.EqualError(t, err, `artifact source scheme "blob" is unsupported; use a local path, file://, or HTTP(S)`)
}
