package findings

import (
	"os"
	"path/filepath"
	"testing"

	findingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/findings"

	"github.com/vrooli/cli-core/cliapp"
)

// TestFindingsManifestCoversFindingsService asserts that every RPC declared on
// FindingsService either has a manifest command binding or is documented in the
// manifest's `omitted` array. Adding a new RPC without binding/omitting it fails
// here — the CLI↔proto parity gate.
func TestFindingsManifestCoversFindingsService(t *testing.T) {
	manifest := readFindingsManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, findingsv1.File_web_search_v1_findings_findings_proto, "FindingsService")
}

func readFindingsManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
