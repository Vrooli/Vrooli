package domains

import (
	"os"
	"path/filepath"
	"testing"

	domainsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/domains"

	"github.com/vrooli/cli-core/cliapp"
)

// TestManifestCoversDomainsService asserts that every RPC declared on
// DomainsService either has a manifest command binding or is documented in
// the manifest's `omitted` array with a reason. Catches drift between
// proto and CLI: adding a new RPC without binding/omitting it fails here.
func TestManifestCoversDomainsService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, domainsv1.File_architecture_cartographer_v1_domains_domains_proto, "DomainsService")
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
