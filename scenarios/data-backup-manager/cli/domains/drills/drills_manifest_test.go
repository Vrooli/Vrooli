package drills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	drillsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/drills"
)

func TestDrillsManifestCoversRecoveryDrillsService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, drillsv1.File_data_backup_manager_v1_drills_drills_proto, "RecoveryDrillsService")
}
