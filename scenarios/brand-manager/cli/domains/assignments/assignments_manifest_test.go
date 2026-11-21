package assignments

import (
	"os"
	"path/filepath"
	"testing"

	assignmentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assignments"

	"github.com/vrooli/cli-core/cliapp"
)

// TestAssignmentsManifestCoversAssignmentsService asserts that every RPC
// declared on AssignmentsService either has a manifest command binding or is
// documented in the manifest's `omitted` array with a reason. Catches drift
// between proto and CLI: adding a new RPC without binding/omitting it fails here.
func TestAssignmentsManifestCoversAssignmentsService(t *testing.T) {
	manifest := readAssignmentsManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, assignmentsv1.File_brand_manager_v1_assignments_assignments_proto, "AssignmentsService")
}

func readAssignmentsManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
