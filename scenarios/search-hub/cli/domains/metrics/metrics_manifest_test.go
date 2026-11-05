package metrics

import (
	"os"
	"path/filepath"
	"strings"
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

func TestValidateWindowTokenRejectsUnknownToken(t *testing.T) {
	if err := validateWindowToken("bogus"); err == nil || !strings.Contains(err.Error(), "invalid --window") {
		t.Fatalf("validateWindowToken(bogus) = %v", err)
	}
	for _, token := range []string{"", "this_week", "last_7d", "last_30d", "this_month", "last_month", "this_quarter"} {
		if err := validateWindowToken(token); err != nil {
			t.Fatalf("validateWindowToken(%q) = %v", token, err)
		}
	}
}
