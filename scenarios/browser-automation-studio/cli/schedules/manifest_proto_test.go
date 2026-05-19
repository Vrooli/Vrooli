package schedules

import (
	"os"
	"path/filepath"
	"testing"

	schedulesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/schedules"

	"github.com/vrooli/cli-core/cliapp"
)

// TestSchedulesManifestCoversSchedulesService asserts the BAS CLI manifest
// binds every RPC declared on SchedulesService. Part of the per-domain
// parity gate for the proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc, Phase 9).
func TestSchedulesManifestCoversSchedulesService(t *testing.T) {
	path := filepath.Join("..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, schedulesv1.File_browser_automation_studio_v1_schedules_schedules_proto, "SchedulesService")
}
