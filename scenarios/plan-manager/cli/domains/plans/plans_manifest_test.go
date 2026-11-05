package plans

import (
	"os"
	"path/filepath"
	"testing"

	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/plans"

	"github.com/vrooli/cli-core/cliapp"
)

// TestPlansManifestCoversPlansService asserts that every RPC declared on
// PlansService has a manifest command binding (across the plans/phase/template
// groups this package owns) or is documented in the manifest's `omitted` array.
// Catches drift between proto and CLI.
func TestPlansManifestCoversPlansService(t *testing.T) {
	manifest := readPlansManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, plansv1.File_plan_manager_v1_plans_plans_proto, "PlansService")
}

func readPlansManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
