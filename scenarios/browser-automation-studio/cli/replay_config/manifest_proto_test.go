package replay_config

import (
	"os"
	"path/filepath"
	"testing"

	replayconfigv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/replay_config"

	"github.com/vrooli/cli-core/cliapp"
)

// TestReplayConfigManifestCoversReplayConfigService asserts the BAS CLI
// manifest binds every RPC declared on ReplayConfigService (Get/Put/Reset).
// Part of the per-domain parity gate for the proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc, Phase 9).
func TestReplayConfigManifestCoversReplayConfigService(t *testing.T) {
	path := filepath.Join("..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, replayconfigv1.File_browser_automation_studio_v1_replay_config_replay_config_proto, "ReplayConfigService")
}
