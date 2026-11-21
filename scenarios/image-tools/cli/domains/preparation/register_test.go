package preparation

import (
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/preparation"
)

func TestPreparationManifestCoversNativeService(t *testing.T) {
	raw, err := os.ReadFile("../../manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, v1.File_image_tools_v1_preparation_preparation_proto, "PreparationService")
}
