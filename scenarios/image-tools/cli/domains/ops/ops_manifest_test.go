package ops

import (
	"os"
	"path/filepath"
	"testing"

	opsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops"

	"github.com/vrooli/cli-core/cliapp"
)

// TestOpsManifestCoversOpsService asserts every RPC on OpsService is bound by a
// manifest command (or documented in `omitted`). OpsService has only the
// discovery RPC ListOperations; op EXECUTION is the REST multipart edge, exposed
// via the hand-appended run commands (register.go), not Connect RPCs.
func TestOpsManifestCoversOpsService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, opsv1.File_image_tools_v1_ops_ops_proto, "OpsService")
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
