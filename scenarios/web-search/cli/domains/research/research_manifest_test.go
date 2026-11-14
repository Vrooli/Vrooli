package research

import (
	"os"
	"path/filepath"
	"testing"

	researchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/research"

	"github.com/vrooli/cli-core/cliapp"
)

// TestResearchManifestCoversResearchService asserts every RPC on ResearchService
// has a manifest command binding (or an omitted entry). Adding a new RPC without
// binding/omitting it fails here — the CLI<->proto parity gate.
func TestResearchManifestCoversResearchService(t *testing.T) {
	manifest := readManifest(t)
	cliapp.RequireProtoServiceCoverage(t, manifest, researchv1.File_web_search_v1_research_research_proto, "ResearchService")
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
