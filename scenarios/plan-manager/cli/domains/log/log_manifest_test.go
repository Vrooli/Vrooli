package log

import (
	"os"
	"path/filepath"
	"testing"

	logv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/log"

	"github.com/vrooli/cli-core/cliapp"
)

// TestLogManifestCoversLogService asserts that every RPC declared on LogService
// has a manifest command binding in the `log` group this package owns, or is
// documented in the manifest's `omitted` array. Catches drift between proto and
// CLI.
func TestLogManifestCoversLogService(t *testing.T) {
	manifest := readLogManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, logv1.File_plan_manager_v1_log_log_proto, "LogService")
}

func readLogManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
