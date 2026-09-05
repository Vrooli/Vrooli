package uxmetrics

import (
	"os"
	"path/filepath"
	"testing"

	uxmetricsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/uxmetrics"

	"github.com/vrooli/cli-core/cliapp"
)

// TestUXMetricsManifestCoversUXMetricsService asserts the BAS CLI manifest
// binds every RPC declared on UXMetricsService. Part of the per-domain
// parity gate for the proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc, Phase 9).
func TestUXMetricsManifestCoversUXMetricsService(t *testing.T) {
	path := filepath.Join("..", "manifest.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, uxmetricsv1.File_browser_automation_studio_v1_uxmetrics_uxmetrics_proto, "UXMetricsService")
}
