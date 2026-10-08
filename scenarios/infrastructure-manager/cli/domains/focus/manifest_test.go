package focus

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	focusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/focus"
)

func TestManifestCoversFocusService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, focusv1.File_infrastructure_manager_v1_focus_focus_proto, "FocusService")
}
