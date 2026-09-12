package relay

import (
	"os"
	"path/filepath"
	"testing"

	relayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRelayManifestCoversRelayService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, relayv1.File_vrooli_bridge_v1_relay_relay_proto, "RelayService")
}
