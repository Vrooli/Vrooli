package queue

import (
	"os"
	"path/filepath"
	"testing"

	queuev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/queue"

	"github.com/vrooli/cli-core/cliapp"
)

// TestQueueManifestCoversQueueService asserts every RPC on QueueService has a
// manifest command binding (or an `omitted` entry) — catching proto↔CLI drift.
func TestQueueManifestCoversQueueService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, queuev1.File_vrooli_bridge_v1_queue_queue_proto, "QueueService")
}

func readManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
