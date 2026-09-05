package onboard

import (
	"os"
	"path/filepath"
	"testing"

	onboardv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/onboard"

	"github.com/vrooli/cli-core/cliapp"
)

// TestOnboardManifestCoversOnboardService asserts every RPC on OnboardService
// has a manifest command binding (there is no node-facing RPC to omit here) —
// catching proto↔CLI drift.
func TestOnboardManifestCoversOnboardService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, onboardv1.File_vrooli_bridge_v1_onboard_onboard_proto, "OnboardService")
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
