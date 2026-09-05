package settings

import (
	"os"
	"path/filepath"
	"testing"

	settingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/settings"

	"github.com/vrooli/cli-core/cliapp"
)

func TestSettingsManifestCoversSettingsService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, settingsv1.File_flow_verifier_v1_settings_settings_proto, "SettingsService")
}
