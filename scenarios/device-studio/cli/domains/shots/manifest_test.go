package shots

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
)

func TestManifestCoversShotsService(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil { t.Fatal(err) }
	cliapp.RequireProtoServiceCoverage(t, manifest, shotsv1.File_device_studio_v1_shots_shots_proto, "ShotsService")
}
