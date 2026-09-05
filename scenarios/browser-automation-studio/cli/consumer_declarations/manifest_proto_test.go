package consumerdeclarations

import (
	"os"
	"path/filepath"
	"testing"

	consumerdeclarationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/consumer_declarations"
	"github.com/vrooli/cli-core/cliapp"
)

func TestManifestCoversConsumerDeclarationsService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "manifest.json"))
	if err != nil { t.Fatal(err) }
	cliapp.RequireProtoServiceCoverage(t, raw, consumerdeclarationsv1.File_browser_automation_studio_v1_consumer_declarations_consumer_declarations_proto, "ConsumerDeclarationsService")
}
