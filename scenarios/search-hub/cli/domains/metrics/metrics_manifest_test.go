package metrics

import (
	"os"
	"path/filepath"
	"testing"

	shmeasuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/measures"

	"github.com/vrooli/cli-core/cliapp"
)

func TestMetricsManifestCoversMeasuresService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, shmeasuresv1.File_search_hub_v1_measures_measures_proto, "MeasuresService")
}
