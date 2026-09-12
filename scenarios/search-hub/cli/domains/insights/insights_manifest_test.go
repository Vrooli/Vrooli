package insights

import (
	"os"
	"path/filepath"
	"testing"

	metricsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/metrics"

	"github.com/vrooli/cli-core/cliapp"
)

// TestInsightsManifestCoversMetricsService asserts every RPC on MetricsService
// is bound to a CLI command in cli/manifest.json — the CLI-side parity guard
// mirroring the API's TestProtoConnectParity. Adding an RPC to metrics.proto
// without a binding (or omission) fails here.
func TestInsightsManifestCoversMetricsService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, metricsv1.File_search_hub_v1_metrics_metrics_proto, "MetricsService")
}
