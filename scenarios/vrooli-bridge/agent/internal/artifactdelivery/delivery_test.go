package artifactdelivery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeliverDownloadsAndAtomicallyPlacesDirectedItem(t *testing.T) {
	const payload = "signed installer bytes"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(deviceTokenHeader) != "target-token" || r.URL.Path != "/api/v1/transfer/items/item-1/content" {
			http.Error(w, "unexpected request", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(payload))
	}))
	defer server.Close()

	destination := filepath.Join(t.TempDir(), "nested", "installer.bin")
	got, err := Deliver(context.Background(), server.Client(), Config{BaseURL: server.URL, DeviceToken: "target-token"}, Request{
		ItemID: "item-1", Name: "installer.bin", DestinationPath: destination,
	})
	require.NoError(t, err)
	require.Equal(t, destination, got.Path)
	require.Equal(t, int64(len(payload)), got.SizeBytes)
	sum := sha256.Sum256([]byte(payload))
	require.Equal(t, hex.EncodeToString(sum[:]), got.SHA256)
	data, readErr := os.ReadFile(destination)
	require.NoError(t, readErr)
	require.Equal(t, payload, string(data))
}

func TestDeliverRejectsRelativeDestinationWithoutWorkDir(t *testing.T) {
	_, err := Deliver(context.Background(), nil, Config{BaseURL: "http://hub", DeviceToken: "token"}, Request{ItemID: "item", DestinationPath: "installer.bin"})
	require.EqualError(t, err, "relative artifact destination requires the agent work directory")
}

func TestResolveDestinationRejectsWorkDirEscape(t *testing.T) {
	_, err := resolveDestination(t.TempDir(), "../outside.bin")
	require.EqualError(t, err, "relative artifact destination escapes the agent work directory")
}
