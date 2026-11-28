package report

import (
	"os"
	"path/filepath"
	"testing"

	reportv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/report"

	"github.com/vrooli/cli-core/cliapp"
)

func TestReportManifestCoversReportService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, reportv1.File_development_toolchain_validator_v1_report_report_proto, "ReportService")
}
