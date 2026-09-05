package drills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	drillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/drills"
)

func TestDrillsManifestCoversFailureDrillService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, drillsv1.File_browser_automation_studio_v1_drills_drills_proto, "FailureDrillService")
}
