package livesearch

import (
	"os"
	"path/filepath"
	"testing"

	livesearchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/livesearch"

	"github.com/vrooli/cli-core/cliapp"
)

// TestLiveSearchManifestCoversLiveSearchService asserts that every RPC declared
// on LiveSearchService either has a manifest command binding or is documented in
// the manifest's `omitted` array. Adding a new RPC without binding/omitting it
// fails here — the CLI↔proto parity gate.
func TestLiveSearchManifestCoversLiveSearchService(t *testing.T) {
	manifest := readLiveSearchManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, livesearchv1.File_web_search_v1_livesearch_livesearch_proto, "LiveSearchService")
}

func readLiveSearchManifest(t *testing.T) []byte {
	t.Helper()
	path := filepath.Join("..", "..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}
