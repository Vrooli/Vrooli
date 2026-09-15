package recordings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	recordingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/recordings"
)

// TestRecordingsManifestCoversRecordingsService keeps the UI-owned recordings
// RPC surface explicit: methods must either have a CLI binding or a reasoned
// manifest omission. This prevents new API methods from silently becoming
// unclassified CLI-health contract failures.
func TestRecordingsManifestCoversRecordingsService(t *testing.T) {
	manifestPath := filepath.Join("..", "manifest.json")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read %s: %v", manifestPath, err)
	}
	cliapp.RequireProtoServiceCoverage(t, manifest, recordingsv1.File_browser_automation_studio_v1_recordings_recordings_proto, "RecordingsService")
}
